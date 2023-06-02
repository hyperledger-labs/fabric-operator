/*
 * Copyright contributors to the Hyperledger Fabric Operator project
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 * 	  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package staggerrestarts

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/action"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/configmap"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("stagger_restart_service")

type Instance interface {
	v1.Object
	GetMSPID() string
}

type StaggerRestartsService struct {
	Client           k8sclient.Client
	ConfigMapManager *configmap.Manager
	Timeout          time.Duration
}

func New(client k8sclient.Client, timeout time.Duration) *StaggerRestartsService {
	return &StaggerRestartsService{
		Client:           client,
		Timeout:          timeout,
		ConfigMapManager: configmap.NewManager(client),
	}
}

// Restart is called by the restart manager.
// For CA/Peer/Orderer: adds component to the queue for restart.
// For Console: 		restarts the component directly as there is only one ibpconsole
//
//	instance per network. We bypass the queue logic for ibpconsoles.
func (s *StaggerRestartsService) Restart(instance Instance, reason string) error {
	switch instance.(type) {
	case *current.IBPConsole:
		if err := s.RestartImmediately("console", instance, reason); err != nil {
			return errors.Wrapf(err, "failed to restart %s", instance.GetName())
		}
	default:
		if err := s.AddToQueue(instance, reason); err != nil {
			return errors.Wrapf(err, "failed to add restart request to queue for %s", instance.GetName())
		}
	}

	return nil
}

// AddToQueue is called by the restart manager and handles adding the
// restart request to the queue associated with the instance's MSPID
// in the <ca/peer/orderer>-restart-config CM.
func (s *StaggerRestartsService) AddToQueue(instance Instance, reason string) error {
	var componentType string
	switch instance.(type) {
	case *current.IBPCA:
		componentType = "ca"
	case *current.IBPOrderer:
		componentType = "orderer"
	case *current.IBPPeer:
		componentType = "peer"

	}

	err := wait.Poll(time.Second, 3*time.Second, func() (bool, error) {
		err := s.addToQueue(componentType, instance, reason)
		if err != nil {
			log.Error(err, "failed to add to queue")
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return errors.Wrapf(err, "failed to add %s to queue", instance.GetName())
	}

	return nil
}

func (s *StaggerRestartsService) addToQueue(componentType string, instance Instance, reason string) error {
	component := &Component{
		CRName: instance.GetName(),
		Reason: reason,
		Status: Pending,
	}

	restartConfig, err := s.GetConfig(componentType, instance.GetNamespace())
	if err != nil {
		return err
	}

	// Add component to queue
	restartConfig.AddToQueue(instance.GetMSPID(), component)

	err = s.UpdateConfig(componentType, instance.GetNamespace(), restartConfig)
	if err != nil {
		return err
	}

	return nil
}

func (s *StaggerRestartsService) RestartImmediately(componentType string, instance Instance, reason string) error {
	log.Info(fmt.Sprintf("Restarting %s...", instance.GetName()))
	err := s.RestartDeployment(instance.GetName(), instance.GetNamespace())
	if err != nil {
		return err
	}

	component := &Component{
		CRName:               instance.GetName(),
		Reason:               reason,
		Status:               Restarted,
		LastCheckedTimestamp: time.Now().UTC().String(),
	}

	restartConfig, err := s.GetConfig(componentType, instance.GetNamespace())
	if err != nil {
		return err
	}
	restartConfig.AddToLog(component)

	err = s.UpdateConfig(componentType, instance.GetNamespace(), restartConfig)
	if err != nil {
		return err
	}

	return nil
}

// Reconcile is called by the ca/peer/orderer reconcile loops via the restart
// manager when an update to the <ca/peer/orderer>-restart-config CM is detected
// and handles the different states of the first component of each queue.
//
// Returns true if the controller needs to requeue the request to reconcile the restart manager.
func (s *StaggerRestartsService) Reconcile(componentType, namespace string) (bool, error) {
	requeue := false

	restartConfig, err := s.GetConfig(componentType, namespace)
	if err != nil {
		return requeue, err
	}

	updated := false
	// Check front component of each queue
	for mspid, queue := range restartConfig.Queues {
		if len(queue) == 0 {
			// queue is empty - do nothing
			continue
		}

		component := queue[0]
		name := component.CRName

		switch component.Status {
		case Pending:
			log.Info(fmt.Sprintf("%s in pending status, restarting deployment", component.CRName))

			// Save pod name
			pods, err := s.GetRunningPods(name, namespace)
			if err != nil {
				return requeue, errors.Wrapf(err, "failed to get running pods for %s", name)
			}

			if len(pods) > 0 {
				component.PodName = pods[0].Name
			}

			// Restart component
			err = s.RestartDeployment(name, namespace)
			if err != nil {
				return requeue, errors.Wrapf(err, "failed to restart deployment %s", name)
			}

			// Update config
			component.Status = Waiting
			component.LastCheckedTimestamp = time.Now().UTC().String()
			component.CheckUntilTimestamp = time.Now().Add(s.Timeout).UTC().String()

			updated = true

		case Waiting:
			pods, err := s.GetRunningPods(name, namespace)
			if err != nil {
				return requeue, errors.Wrapf(err, "failed to get running pods for %s", name)
			}

			// Scenario 1: the pod has restarted
			if len(pods) == 1 {
				if component.PodName != pods[0].Name {
					// Pod has restarted as the old pod has disappeared
					log.Info(fmt.Sprintf("%s in completed status, removing from %s restart queue", component.CRName, mspid))
					component.Status = Completed

					restartConfig.AddToLog(component)
					restartConfig.PopFromQueue(mspid)

					log.Info(fmt.Sprintf("Remaining restart queue(s) to reconcile: %s", queuesToString(restartConfig.Queues)))
					updated = true

					continue
				}
			}

			// Scenario 2: the pod has not restarted and the wait period has timed out
			checkUntil, err := parseTime(component.CheckUntilTimestamp)
			if err != nil {
				return requeue, errors.Wrap(err, "failed to parse checkUntilTimestamp")
			}
			if time.Now().UTC().After(checkUntil) {
				log.Info(fmt.Sprintf("%s in expired status, has not restarted within %s", component.CRName, s.Timeout.String()))
				// Pod has not restarted within s.timeout, move to log
				component.Status = Expired

				restartConfig.AddToLog(component)
				restartConfig.PopFromQueue(mspid)

				log.Info(fmt.Sprintf("Remaining restart queue(s) to reconcile: %s", queuesToString(restartConfig.Queues)))
				updated = true

				continue
			}

			// Scenario 3: the pod has not yet restarted but there is still time remaining
			// to wait for the pod to restart.

			// To prevent the restart manager from overwritting the config map and losing
			// data, the config map updates that trigger reconciles only occur every 10-30
			// seconds. If the lastCheckedInterval amount of time has not yet passed since
			// the lastCheckedTimestamp, then we return true to tell the controllers to
			// requeue the request to reconcile the restart config map to ensure that
			// a reconcile will occur again even when the config map is not updated.

			lastCheckedInterval := time.Duration(randomInt(10, 30)) * time.Second
			lastChecked, err := parseTime(component.LastCheckedTimestamp)
			if err != nil {
				return requeue, errors.Wrap(err, "failed to parse lastCheckedTimestamp")
			}

			if lastChecked.Add(lastCheckedInterval).Before(time.Now()) {
				component.LastCheckedTimestamp = time.Now().UTC().String()
				updated = true
			} else {
				requeue = true
			}

		default:
			// Expired or Completed status - should not reach this case as Waiting case handles moving components to log
			log.Info(fmt.Sprintf("%s restart status is %s, removing from %s restart queue", component.CRName, component.Status, mspid))

			restartConfig.AddToLog(component)
			restartConfig.PopFromQueue(mspid)

			updated = true
		}
	}

	if updated {
		err = s.UpdateConfig(componentType, namespace, restartConfig)
		if err != nil {
			return requeue, err
		}
	}

	return requeue, nil
}

func (s *StaggerRestartsService) GetConfig(componentType, namespace string) (*RestartConfig, error) {
	cmName := fmt.Sprintf("%s-restart-config", componentType)

	cfg := &RestartConfig{
		Queues: map[string][]*Component{},
	}
	err := s.ConfigMapManager.GetRestartConfigFrom(cmName, namespace, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (s *StaggerRestartsService) UpdateConfig(componentType, namespace string, cfg *RestartConfig) error {
	cmName := fmt.Sprintf("%s-restart-config", componentType)
	return s.ConfigMapManager.UpdateConfig(cmName, namespace, cfg)
}

func (s *StaggerRestartsService) RestartDeployment(name, namespace string) error {
	log.Info(fmt.Sprintf("Restarting deployment %s", name))

	err := action.Restart(s.Client, name, namespace)
	if err != nil {
		return err
	}

	return nil
}

func (s *StaggerRestartsService) GetRunningPods(name, namespace string) ([]corev1.Pod, error) {
	pods := []corev1.Pod{}

	labelSelector, err := labels.Parse(fmt.Sprintf("app=%s", name))
	if err != nil {
		return pods, errors.Wrap(err, "failed to parse label selector for app name")
	}

	listOptions := &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     namespace,
	}

	podList := &corev1.PodList{}
	err = s.Client.List(context.TODO(), podList, listOptions)
	if err != nil {
		log.Error(err, "failed to get pod list for %s", name)
		// return empty pods list
		// NOTE: decided not to return error here since this funtion will be called multiple
		// times throughout the process of old pods terminating and new pods starting up.
		// We don't want to error out prematurely if this client call isn't able to retrieve
		// a list of pods during the restart process.
		return pods, nil
	}

	for _, pod := range podList.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			containerStatuses := pod.Status.ContainerStatuses

			readyContainers := 0
			numContainers := len(containerStatuses)

			for _, status := range containerStatuses {
				// TODO: is it required to check status.Ready?
				if status.Ready && status.State.Running != nil {
					readyContainers++
				}
			}
			if readyContainers == numContainers {
				pods = append(pods, pod)
			}
		}
	}

	return pods, nil
}

func queuesToString(queues map[string][]*Component) string {
	lst := []string{}
	for org, queue := range queues {
		str := org + ": [ "
		if org == "" {
			// This is a ca queue
			str = "[ "
		}
		for _, comp := range queue {
			str += comp.CRName + " "
		}
		str += " ]"

		lst = append(lst, str)
	}

	return strings.Join(lst, ",")
}

func parseTime(t string) (time.Time, error) {
	format := "2006-01-02 15:04:05.999999999 -0700 MST"
	return time.Parse(format, t)
}

// Returns a random integer between min and max.
func randomInt(min, max int) int {
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	return int(randomNum.Int64()) + min
}
