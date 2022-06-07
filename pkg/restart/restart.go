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

package restart

import (
	"fmt"
	"strings"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/configmap"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/staggerrestarts"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("restart_manager")

type RestartManager struct {
	Client                 k8sclient.Client
	Timers                 map[string]*time.Timer
	WaitTime               time.Duration
	ConfigMapManager       *configmap.Manager
	StaggerRestartsService *staggerrestarts.StaggerRestartsService
}

func New(client k8sclient.Client, waitTime, timeout time.Duration) *RestartManager {
	r := &RestartManager{
		Client:                 client,
		Timers:                 map[string]*time.Timer{},
		WaitTime:               waitTime,
		ConfigMapManager:       configmap.NewManager(client),
		StaggerRestartsService: staggerrestarts.New(client, timeout),
	}

	return r
}

func (r *RestartManager) ForAdminCertUpdate(instance v1.Object) error {
	return r.updateConfigFor(instance, ADMINCERT)
}

func (r *RestartManager) ForCertUpdate(certType common.SecretType, instance v1.Object) error {
	var err error
	switch certType {
	case common.TLS:
		err = r.ForTLSReenroll(instance)
	case common.ECERT:
		err = r.ForEcertReenroll(instance)
	}

	if err != nil {
		return err
	}

	return nil
}

func (r *RestartManager) ForEcertReenroll(instance v1.Object) error {
	return r.updateConfigFor(instance, ECERTUPDATE)
}

func (r *RestartManager) ForTLSReenroll(instance v1.Object) error {
	return r.updateConfigFor(instance, TLSUPDATE)
}

func (r *RestartManager) ForConfigOverride(instance v1.Object) error {
	return r.updateConfigFor(instance, CONFIGOVERRIDE)
}

func (r *RestartManager) ForMigration(instance v1.Object) error {
	return r.updateConfigFor(instance, MIGRATION)
}

func (r *RestartManager) ForNodeOU(instance v1.Object) error {
	return r.updateConfigFor(instance, NODEOU)
}

func (r *RestartManager) ForConfigMapUpdate(instance v1.Object) error {
	return r.updateConfigFor(instance, CONFIGMAPUPDATE)
}

func (r *RestartManager) ForRestartAction(instance v1.Object) error {
	return r.updateConfigFor(instance, RESTARTACTION)
}

// Updates the operator-config for the given reason by setting the request
// status to 'pending' and request timestamp to the current time:
//
// instances[instance_name].Requests[reason].Status = "pending"
func (r *RestartManager) updateConfigFor(instance v1.Object, reason Reason) error {
	cfg, err := r.GetConfig(instance)
	if err != nil {
		return err
	}

	if cfg.Instances == nil {
		cfg.Instances = map[string]*Restart{}
	}
	_, ok := cfg.Instances[instance.GetName()]
	if !ok {
		cfg.Instances[instance.GetName()] = &Restart{}
	}

	restart := cfg.Instances[instance.GetName()]
	updateRestartRequest(restart, reason)

	log.Info(fmt.Sprintf("Updating operator-config map, %s restart requested due to %s", instance.GetName(), reason))
	err = r.UpdateConfigMap(cfg, instance)
	if err != nil {
		return err
	}

	return nil
}

func updateRestartRequest(restart *Restart, reason Reason) {
	if restart.Requests == nil {
		restart.Requests = map[Reason]*Request{}
	}

	if restart.Requests[reason] == nil {
		restart.Requests[reason] = &Request{}
	}

	// Set request time
	req := restart.Requests[reason]
	if req.Status != Pending {
		req.Status = Pending
		req.RequestTimestamp = time.Now().UTC().Format(time.RFC3339)
	}
}

type Instance interface {
	v1.Object
	GetMSPID() string
}

// TriggerIfNeeded checks operator-config for any pending restarts, sets a timer to restart
// the deployment if required, and restarts the deployment.
func (r *RestartManager) TriggerIfNeeded(instance Instance) error {
	var trigger bool

	cfg, err := r.GetConfig(instance)
	if err != nil {
		return err
	}

	restart := cfg.Instances[instance.GetName()]
	if restart == nil || restart.Requests == nil {
		// Do nothing if restart doesn't have any pending requests
		return nil
	}

	reasonList := []string{}
	for reason, req := range restart.Requests {
		if req != nil {
			if req.Status == Pending {
				reasonList = append(reasonList, string(reason))
				if r.triggerRestart(req) {
					trigger = true
				}
			}

		}
	}
	reasonString := strings.Join(reasonList, ",")

	if trigger {
		err = r.RestartDeployment(instance, reasonString)
		if err != nil {
			return err
		}
	} else if r.pendingRequests(restart) {
		err = r.SetTimer(instance, reasonString)
		if err != nil {
			return errors.Wrap(err, "failed to set timer to restart deployment")
		}
	}

	return nil
}

func (r *RestartManager) triggerRestart(req *Request) bool {
	if req != nil {
		if req.Status == Pending {
			if req.LastActionTimestamp == "" { // no previous restart has occurred
				return true
			}

			lastRestart, err := time.Parse(time.RFC3339, req.LastActionTimestamp)
			if err != nil {
				return true
			}

			requestedRestart, err := time.Parse(time.RFC3339, req.RequestTimestamp)
			if err != nil {
				return true
			}

			if requestedRestart.Sub(lastRestart) >= r.WaitTime {
				return true
			}
		}
	}

	return false
}

func (r *RestartManager) pendingRequests(restart *Restart) bool {
	for _, req := range restart.Requests {
		if req.Status == Pending {
			return true
		}
	}
	return false
}

func (r *RestartManager) SetTimer(instance Instance, reason string) error {
	cfg, err := r.GetConfig(instance)
	if err != nil {
		return err
	}

	restart := cfg.Instances[instance.GetName()]

	oldestRequestTime := time.Now().UTC()
	lastActionTime := ""
	// Want to set timer duration based on oldest pending request
	for _, req := range restart.Requests {
		if req != nil {
			requestTime, err := time.Parse(time.RFC3339, req.RequestTimestamp)
			if err == nil {
				if requestTime.Before(oldestRequestTime) {
					oldestRequestTime = requestTime
					lastActionTime = req.LastActionTimestamp
				}
			}
		}
	}

	// Set timer if not already running
	if r.Timers[instance.GetName()] == nil {
		dur := r.getTimerDuration(lastActionTime, oldestRequestTime)
		log.Info(fmt.Sprintf("Setting timer to restart %s in %f minutes", instance.GetName(), dur.Minutes()))

		r.Timers[instance.GetName()] = time.AfterFunc(dur, func() {
			err := r.RestartDeployment(instance, reason)
			if err != nil {
				log.Error(err, fmt.Sprintf("failed to restart deployment for %s", instance.GetName()))
			}
		})
	} else {
		log.Info(fmt.Sprintf("Timer already set to restart %s shortly", instance.GetName()))
	}

	return nil
}

// If lastRestartTime was less than 10 min (or value of WaitTime) ago, calculate how much
// time remains before WaitTime has passed to trigger next restart
func (r *RestartManager) getTimerDuration(actionTime string, requestTime time.Time) time.Duration {
	lastRestartTime, err := time.Parse(time.RFC3339, actionTime)
	if err != nil {
		// Default to WaitTime
		return r.WaitTime
	}
	timePassed := requestTime.Sub(lastRestartTime)
	return r.WaitTime - timePassed
}

// RestartDeployment adds the instance to the queue to stagger restarts
func (r *RestartManager) RestartDeployment(instance Instance, reason string) error {
	log.Info(fmt.Sprintf("Queuing instance %s for restart", instance.GetName()))

	err := r.ClearRestartConfigForInstance(instance)
	if err != nil {
		return errors.Wrap(err, "failed to clear restart config")
	}

	err = r.StaggerRestartsService.Restart(instance, reason)
	if err != nil {
		return errors.Wrap(err, "failed to add restart request to queue")
	}

	return nil
}

func (r *RestartManager) ClearRestartConfigForInstance(instance v1.Object) error {
	cfg, err := r.GetConfig(instance)
	if err != nil {
		return err
	}

	if cfg.Instances == nil || cfg.Instances[instance.GetName()] == nil {
		return nil
	}

	for _, req := range cfg.Instances[instance.GetName()].Requests {
		if req != nil && req.Status == Pending {
			clearRestart(req)
		}
	}

	// Stop timer if previously set
	if r.Timers[instance.GetName()] != nil {
		r.Timers[instance.GetName()].Stop()
		r.Timers[instance.GetName()] = nil
	}

	err = r.UpdateConfigMap(cfg, instance)
	if err != nil {
		return err
	}

	return nil
}

func clearRestart(req *Request) {
	req.LastActionTimestamp = time.Now().UTC().Format(time.RFC3339)
	req.RequestTimestamp = ""
	req.Status = Complete
}

func (r *RestartManager) GetConfig(instance v1.Object) (*Config, error) {
	cmName := "operator-config"

	cfg := &Config{}
	err := r.ConfigMapManager.GetRestartConfigFrom(cmName, instance.GetNamespace(), cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (r *RestartManager) UpdateConfigMap(cfg *Config, instance v1.Object) error {
	cmName := "operator-config"

	return r.ConfigMapManager.UpdateConfig(cmName, instance.GetNamespace(), cfg)
}
