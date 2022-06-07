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

package k8sconsole

import (
	"context"
	"fmt"
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	baseconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console"
	baseconsoleoverride "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/console/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("k8s_console")

type Override interface {
	baseconsole.Override
	Ingress(v1.Object, *networkingv1.Ingress, resources.Action) error
	Ingressv1beta1(v1.Object, *networkingv1beta1.Ingress, resources.Action) error
}

type Console struct {
	*baseconsole.Console

	IngressManager        resources.Manager
	Ingressv1beta1Manager resources.Manager

	Override Override
}

func New(client k8sclient.Client, scheme *runtime.Scheme, config *config.Config) *Console {
	o := &override.Override{
		Override: baseconsoleoverride.Override{},
	}

	console := &Console{
		Console:  baseconsole.New(client, scheme, config, o),
		Override: o,
	}

	console.CreateManagers()
	return console
}

func (c *Console) CreateManagers() {
	override := c.Override
	resourceManager := resourcemanager.New(c.Client, c.Scheme)
	c.IngressManager = resourceManager.CreateIngressManager("", override.Ingress, c.GetLabels, c.Config.ConsoleInitConfig.IngressFile)
	c.Ingressv1beta1Manager = resourceManager.CreateIngressv1beta1Manager("", override.Ingressv1beta1, c.GetLabels, c.Config.ConsoleInitConfig.Ingressv1beta1File)
}

func (c *Console) Reconcile(instance *current.IBPConsole, update baseconsole.Update) (common.Result, error) {
	var err error

	versionSet, err := c.SetVersion(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, fmt.Sprintf("failed updating CR '%s' to version '%s'", instance.Name, version.Operator))
	}
	if versionSet {
		log.Info("Instance version updated, requeuing request...")
		return common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
		}, nil
	}

	instanceUpdated, err := c.PreReconcileChecks(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed pre reconcile checks")
	}

	if instanceUpdated {
		log.Info("Updating instance after pre reconcile checks")
		err := c.Client.Patch(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPConsole{},
				Strategy: client.MergeFrom,
			},
		})
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update instance")
		}

		log.Info("Instance updated, requeuing request...")
		return common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
		}, nil
	}

	err = c.ReconcileManagers(instance, update.SpecUpdated())
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to reconcile managers")
	}

	err = c.CheckStates(instance, update.SpecUpdated())
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check and restore state")
	}

	err = c.CheckForConfigMapUpdates(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check for config map updates")
	}

	err = c.HandleActions(instance, update)
	if err != nil {
		return common.Result{}, err
	}

	if err := c.HandleRestart(instance, update); err != nil {
		return common.Result{}, err
	}

	return common.Result{}, nil
}

func (c *Console) ReconcileManagers(instance *current.IBPConsole, update bool) error {
	var err error

	err = c.Console.ReconcileManagers(instance, update)
	if err != nil {
		return err
	}

	err = c.ReconcileIngressManager(instance, update)
	if err != nil {
		return err
	}

	err = c.NetworkPolicyReconcile(instance)
	if err != nil {
		return errors.Wrap(err, "failed Network Policy reconciliation")
	}

	return nil
}

func (c *Console) ReconcileIngressManager(instance *current.IBPConsole, update bool) error {
	if c.Config.Operator.Globals.AllowKubernetesEighteen == "true" {
		// check k8s version
		version, err := util.GetServerVersion()
		if err != nil {
			return err
		}
		if strings.Compare(version.Minor, "19") < 0 { // v1beta
			err = c.Ingressv1beta1Manager.Reconcile(instance, update)
			if err != nil {
				return errors.Wrap(err, "failed Ingressv1beta1 reconciliation")
			}
		} else {
			err = c.IngressManager.Reconcile(instance, update)
			if err != nil {
				return errors.Wrap(err, "failed Ingress reconciliation")
			}
		}
	} else {
		err := c.IngressManager.Reconcile(instance, update)
		if err != nil {
			return errors.Wrap(err, "failed Ingress reconciliation")
		}
	}
	return nil
}
