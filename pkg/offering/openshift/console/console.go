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

package openshiftconsole

import (
	"context"
	"fmt"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	baseconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console"
	baseconsoleoverride "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/console/override"
	"github.com/IBM-Blockchain/fabric-operator/version"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultRoute = "./definitions/console/route.yaml"
)

var log = logf.Log.WithName("openshift_console")

type Override interface {
	baseconsole.Override
	ConsoleRoute(object v1.Object, route *routev1.Route, action resources.Action) error
	ProxyRoute(object v1.Object, route *routev1.Route, action resources.Action) error
}

var _ baseconsole.IBPConsole = &Console{}

type Console struct {
	*baseconsole.Console

	RouteManager      resources.Manager
	ProxyRouteManager resources.Manager

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
	resourceManager := resourcemanager.New(c.Client, c.Scheme)
	c.RouteManager = resourceManager.CreateRouteManager("console", c.Override.ConsoleRoute, c.GetLabels, c.Config.ConsoleInitConfig.RouteFile)
	c.ProxyRouteManager = resourceManager.CreateRouteManager("console-proxy", c.Override.ProxyRoute, c.GetLabels, c.Config.ConsoleInitConfig.RouteFile)
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
		err = c.Client.Patch(context.TODO(), instance, nil, k8sclient.PatchOption{
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
	err := c.Console.ReconcileManagers(instance, update)
	if err != nil {
		return err
	}

	err = c.RouteManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Console Route reconciliation")
	}

	err = c.ProxyRouteManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Proxy Route reconciliation")
	}

	err = c.NetworkPolicyReconcile(instance)
	if err != nil {
		return errors.Wrap(err, "failed Network Policy reconciliation")
	}
	return nil
}
