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

package job

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/errors"

	controller "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type Status string

const (
	FAILED    Status = "failed"
	COMPLETED Status = "completed"
	UNKNOWN   Status = "unknown"
)

var log = logf.Log.WithName("job_resource")

type Timeouts struct {
	WaitUntilActive, WaitUntilFinished time.Duration
}

func jobIDGenerator() string {
	charset := "0123456789abcdefghijklmnopqrstuvwxyz"

	randString1 := make([]byte, 10)
	for i := range randString1 {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		randString1[i] = charset[num.Int64()]
	}

	randString2 := make([]byte, 5)
	for i := range randString2 {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		randString2[i] = charset[num.Int64()]
	}

	return string(randString1) + "-" + string(randString2)
}

func New(job *v1.Job, timeouts *Timeouts) *Job {
	if job != nil {
		job.Name = fmt.Sprintf("%s-%s", job.GetName(), jobIDGenerator())
	}

	return &Job{
		Job:      job,
		Timeouts: timeouts,
	}
}

func NewWithDefaults(job *v1.Job) *Job {
	if job != nil {
		job.Name = fmt.Sprintf("%s-%s", job.GetName(), jobIDGenerator())
	}

	return &Job{
		Job: job,
		Timeouts: &Timeouts{
			WaitUntilActive:   60 * time.Second,
			WaitUntilFinished: 60 * time.Second,
		},
	}
}

func NewWithDefaultsUseExistingName(job *v1.Job) *Job {
	return &Job{
		Job: job,
		Timeouts: &Timeouts{
			WaitUntilActive:   60 * time.Second,
			WaitUntilFinished: 60 * time.Second,
		},
	}
}

type Job struct {
	*v1.Job

	Timeouts *Timeouts
}

func (j *Job) MustGetContainer(name string) container.Container {
	cont, _ := j.GetContainer(name)
	return cont
}

func (j *Job) GetContainer(name string) (cont container.Container, err error) {
	for i, c := range j.Spec.Template.Spec.Containers {
		if c.Name == name {
			cont = container.Container{Container: &j.Spec.Template.Spec.Containers[i]}
			return
		}
	}
	for i, c := range j.Spec.Template.Spec.InitContainers {
		if c.Name == name {
			cont = container.Container{Container: &j.Spec.Template.Spec.InitContainers[i]}
			return
		}
	}
	return cont, fmt.Errorf("container '%s' not found", name)
}

func (j *Job) AddContainer(add container.Container) {
	j.Spec.Template.Spec.Containers = util.AppendContainerIfMissing(j.Spec.Template.Spec.Containers, *add.Container)
}

func (j *Job) AddInitContainer(add container.Container) {
	j.Spec.Template.Spec.InitContainers = util.AppendContainerIfMissing(j.Spec.Template.Spec.InitContainers, *add.Container)
}

func (j *Job) AppendVolumeIfMissing(volume corev1.Volume) {
	j.Spec.Template.Spec.Volumes = util.AppendVolumeIfMissing(j.Spec.Template.Spec.Volumes, volume)
}

func (j *Job) AppendPullSecret(imagePullSecret corev1.LocalObjectReference) {
	j.Spec.Template.Spec.ImagePullSecrets = util.AppendImagePullSecretIfMissing(j.Spec.Template.Spec.ImagePullSecrets, imagePullSecret)
}

// UpdateSecurityContextForAllContainers updates the security context for all containers defined
// in the job
func (j *Job) UpdateSecurityContextForAllContainers(sc container.SecurityContext) {
	for i := range j.Spec.Template.Spec.InitContainers {
		container.UpdateSecurityContext(&j.Spec.Template.Spec.InitContainers[i], sc)
	}

	for i := range j.Spec.Template.Spec.Containers {
		container.UpdateSecurityContext(&j.Spec.Template.Spec.Containers[i], sc)
	}
}

func (j *Job) Delete(client controller.Client) error {
	if err := client.Delete(context.TODO(), j.Job); err != nil {
		return errors.Wrap(err, "failed to delete")
	}

	// TODO: Need to investigate why job is not adding controller reference to job pod,
	// this manual cleanup should not be required after deleting job
	podList := &corev1.PodList{}
	if err := client.List(context.TODO(), podList, k8sclient.MatchingLabels{"job-name": j.GetName()}); err != nil {
		return errors.Wrap(err, "failed to list job pods")
	}

	for _, pod := range podList.Items {
		podListItem := pod
		if err := client.Delete(context.TODO(), &podListItem); err != nil {
			return errors.Wrapf(err, "failed to delete pod '%s'", podListItem.Name)
		}
	}

	return nil
}

func (j *Job) Status(client controller.Client) (Status, error) {
	k8sJob, err := j.get(client)
	if err != nil {
		return UNKNOWN, err
	}

	if k8sJob.Status.Failed >= int32(1) {
		return FAILED, nil
	}

	pods, err := j.getPods(client)
	if err != nil {
		return UNKNOWN, err
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodSucceeded {
			return FAILED, nil
		}
	}

	return COMPLETED, nil
}

func (j *Job) ContainerStatus(client controller.Client, contName string) (Status, error) {
	pods, err := j.getPods(client)
	if err != nil {
		return UNKNOWN, err
	}

	for _, pod := range pods.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == contName {
				if containerStatus.State.Terminated != nil {
					if containerStatus.State.Terminated.ExitCode == int32(0) {
						return COMPLETED, nil
					}
					return FAILED, nil
				}
			}
		}
	}

	return UNKNOWN, nil
}

func (j *Job) WaitUntilActive(client controller.Client) error {
	err := wait.Poll(500*time.Millisecond, j.Timeouts.WaitUntilActive, func() (bool, error) {
		log.Info(fmt.Sprintf("Waiting for job '%s' to start in namespace '%s'", j.GetName(), j.GetNamespace()))

		k8sJob, err := j.get(client)
		if err != nil {
			return false, err
		}

		if k8sJob.Status.Active >= int32(1) || k8sJob.Status.Succeeded >= int32(1) {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "job failed to start")
	}
	return nil
}

func (j *Job) WaitUntilFinished(client controller.Client) error {

	err := wait.Poll(2*time.Second, j.Timeouts.WaitUntilFinished, func() (bool, error) {
		log.Info(fmt.Sprintf("Waiting for job pod '%s' to finish", j.GetName()))

		pods, err := j.getPods(client)
		if err != nil {
			log.Info(fmt.Sprintf("get job pod err: %s", err))
			return false, nil
		}

		if len(pods.Items) == 0 {
			return false, nil
		}

		return j.podsTerminated(pods), nil
	})
	if err != nil {
		return errors.Wrapf(err, "pod for job '%s' failed to finish", j.GetName())
	}

	return nil
}

func (j *Job) podsTerminated(pods *corev1.PodList) bool {
	for _, pod := range pods.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Terminated == nil {
				return false
			}
		}
	}

	return true
}

func (j *Job) WaitUntilContainerFinished(client controller.Client, contName string) error {

	err := wait.Poll(2*time.Second, j.Timeouts.WaitUntilFinished, func() (bool, error) {
		log.Info(fmt.Sprintf("Waiting for job pod '%s' to finish", j.GetName()))

		pods, err := j.getPods(client)
		if err != nil {
			log.Info(fmt.Sprintf("get job pod err: %s", err))
			return false, nil
		}

		if len(pods.Items) == 0 {
			return false, nil
		}

		return j.containerTerminated(pods, contName), nil
	})
	if err != nil {
		return errors.Wrapf(err, "pod for job '%s' failed to finish", j.GetName())
	}

	return nil
}

func (j *Job) ContainerFinished(client controller.Client, contName string) (bool, error) {
	pods, err := j.getPods(client)
	if err != nil {
		return false, err
	}

	return j.containerTerminated(pods, contName), nil
}

func (j *Job) containerTerminated(pods *corev1.PodList, contName string) bool {
	for _, pod := range pods.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == contName {
				if containerStatus.State.Terminated == nil {
					return false
				}
			}
		}
	}

	return true
}

func (j *Job) getPods(client controller.Client) (*corev1.PodList, error) {
	labelSelector, err := labels.Parse(fmt.Sprintf("job-name=%s", j.GetName()))
	if err != nil {
		return nil, err
	}

	opts := &k8sclient.ListOptions{
		LabelSelector: labelSelector,
	}

	pods := &corev1.PodList{}
	if err := client.List(context.TODO(), pods, opts); err != nil {
		return nil, err
	}

	return pods, nil
}

func (j *Job) get(client controller.Client) (*v1.Job, error) {
	k8sJob := &v1.Job{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: j.GetName(), Namespace: j.GetNamespace()}, k8sJob)
	if err != nil {
		return nil, err
	}

	return k8sJob, nil
}
