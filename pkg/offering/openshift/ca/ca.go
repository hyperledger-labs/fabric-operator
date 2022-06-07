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

package openshiftca

import (
	"context"
	"fmt"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	baseca "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca"
	basecaoverride "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/ca/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/version"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("openshift_ca")

type Override interface {
	baseca.Override
	CARoute(object v1.Object, route *routev1.Route, action resources.Action) error
	OperationsRoute(object v1.Object, route *routev1.Route, action resources.Action) error
}

var _ baseca.IBPCA = &CA{}

type CA struct {
	*baseca.CA

	CARouteManager         resources.Manager
	OperationsRouteManager resources.Manager

	Override Override
}

func New(client k8sclient.Client, scheme *runtime.Scheme, config *config.Config) *CA {
	o := &override.Override{
		Override: basecaoverride.Override{
			Client: client,
		},
	}
	ca := &CA{
		CA:       baseca.New(client, scheme, config, o),
		Override: o,
	}
	ca.CreateManagers()
	return ca
}

func (ca *CA) CreateManagers() {
	resourceManager := resourcemanager.New(ca.Client, ca.Scheme)
	ca.CARouteManager = resourceManager.CreateRouteManager("ca", ca.Override.CARoute, ca.GetLabels, ca.Config.CAInitConfig.RouteFile)
	ca.OperationsRouteManager = resourceManager.CreateRouteManager("operations", ca.Override.OperationsRoute, ca.GetLabels, ca.Config.CAInitConfig.RouteFile)
}

func (ca *CA) Reconcile(instance *current.IBPCA, update baseca.Update) (common.Result, error) {

	var err error

	versionSet, err := ca.SetVersion(instance)
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

	instanceUpdated, err := ca.PreReconcileChecks(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed pre reconcile checks")
	}

	if instanceUpdated {
		log.Info("Updating instance after pre reconcile checks")
		err := ca.Client.Patch(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPCA{},
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

	err = ca.AddTLSCryptoIfMissing(instance, ca.GetEndpointsDNS(instance))
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to generate tls crypto")
	}

	err = ca.Initialize(instance, update)
	if err != nil {
		return common.Result{}, operatorerrors.Wrap(err, operatorerrors.CAInitilizationFailed, "failed to initialize ca")
	}

	err = ca.ReconcileManagers(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to reconcile managers")
	}

	if update.CATagUpdated() {
		if err := ca.ReconcileFabricCAMigration(instance); err != nil {
			return common.Result{}, operatorerrors.Wrap(err, operatorerrors.FabricCAMigrationFailed, "failed to migrate fabric ca versions")
		}
	}

	err = ca.UpdateConnectionProfile(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to create connection profile")
	}

	err = ca.CheckStates(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check and restore state")
	}

	status, err := ca.CheckCertificates(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check for expiring certificates")
	}

	if update.CACryptoUpdated() {
		err = ca.Restart.ForTLSReenroll(instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	err = ca.HandleActions(instance, update)
	if err != nil {
		return common.Result{}, err
	}

	err = ca.HandleRestart(instance, update)
	if err != nil {
		return common.Result{}, err
	}

	return common.Result{
		Status: status,
	}, nil
}

func (ca *CA) ReconcileManagers(instance *current.IBPCA, update baseca.Update) error {
	err := ca.CA.ReconcileManagers(instance, update)
	if err != nil {
		return err
	}

	err = ca.CARouteManager.Reconcile(instance, update.SpecUpdated())
	if err != nil {
		return errors.Wrap(err, "failed CA Route reconciliation")
	}

	err = ca.OperationsRouteManager.Reconcile(instance, update.SpecUpdated())
	if err != nil {
		return errors.Wrap(err, "failed Operations Route reconciliation")
	}

	return nil
}
