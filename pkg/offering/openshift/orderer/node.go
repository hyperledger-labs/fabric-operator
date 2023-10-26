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

package openshiftorderer

import (
	"context"
	"fmt"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commoninit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/orderer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/version"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Override interface {
	baseorderer.Override
	OrdererRoute(object v1.Object, route *routev1.Route, action resources.Action) error
	OperationsRoute(object v1.Object, route *routev1.Route, action resources.Action) error
	AdminRoute(object v1.Object, route *routev1.Route, action resources.Action) error
	OrdererGRPCRoute(object v1.Object, route *routev1.Route, action resources.Action) error
}

var _ baseorderer.IBPOrderer = &Node{}

type Node struct {
	*baseorderer.Node

	RouteManager           resources.Manager
	OperationsRouteManager resources.Manager
	AdminRouteManager      resources.Manager
	GRPCRouteManager       resources.Manager

	Override Override
}

func NewNode(basenode *baseorderer.Node) *Node {
	node := &Node{
		Node:     basenode,
		Override: &override.Override{},
	}
	node.CreateManagers()
	return node
}

func (n *Node) CreateManagers() {
	resourceManager := resourcemanager.New(n.Node.Client, n.Node.Scheme)
	n.RouteManager = resourceManager.CreateRouteManager("", n.Override.OrdererRoute, n.GetLabels, n.Config.OrdererInitConfig.RouteFile)
	n.OperationsRouteManager = resourceManager.CreateRouteManager("", n.Override.OperationsRoute, n.GetLabels, n.Config.OrdererInitConfig.RouteFile)
	n.AdminRouteManager = resourceManager.CreateRouteManager("", n.Override.AdminRoute, n.GetLabels, n.Config.OrdererInitConfig.RouteFile)
	n.GRPCRouteManager = resourceManager.CreateRouteManager("", n.Override.OrdererGRPCRoute, n.GetLabels, n.Config.OrdererInitConfig.RouteFile)
}

func (n *Node) Reconcile(instance *current.IBPOrderer, update baseorderer.Update) (common.Result, error) {
	var err error
	var status *current.CRStatus

	versionSet, err := n.SetVersion(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, fmt.Sprintf("failed updating CR '%s' to version '%s'", instance.Name, version.Operator))
	}
	if versionSet {
		log.Info("Instance version updated, requeuing request...")
		return common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
			OverrideUpdateStatus: true,
		}, nil
	}

	instanceUpdated, err := n.PreReconcileChecks(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed pre reconcile checks")
	}
	externalEndpointUpdated := n.UpdateExternalEndpoint(instance)

	if instanceUpdated || externalEndpointUpdated {
		log.Info(fmt.Sprintf("Updating instance after pre reconcile checks: %t, updating external endpoint: %t",
			instanceUpdated, externalEndpointUpdated))

		err = n.Client.Patch(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPOrderer{},
				Strategy: client.MergeFrom,
			},
		})
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update instance")
		}

		log.Info("Instance updated during reconcile checks, request will be requeued...")
		return common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
			Status: &current.CRStatus{
				Type:    current.Initializing,
				Reason:  "Setting default values for either zone, region, and/or external endpoint",
				Message: "Operator has updated spec with defaults as part of initialization",
			},
			OverrideUpdateStatus: true,
		}, nil
	}

	err = n.Initialize(instance, update)
	if err != nil {
		return common.Result{}, operatorerrors.Wrap(err, operatorerrors.OrdererInitilizationFailed, "failed to initialize orderer node")
	}

	if update.OrdererTagUpdated() {
		if err := n.ReconcileFabricOrdererMigration(instance); err != nil {
			return common.Result{}, operatorerrors.Wrap(err, operatorerrors.FabricOrdererMigrationFailed, "failed to migrate fabric orderer versions")
		}
	}

	if update.MigrateToV2() {
		if err := n.FabricOrdererMigrationV2_0(instance); err != nil {
			return common.Result{}, operatorerrors.Wrap(err, operatorerrors.FabricOrdererMigrationFailed, "failed to migrate fabric orderer to version v2.x")
		}
	}

	if update.MigrateToV24() {
		if err := n.FabricOrdererMigrationV2_4(instance); err != nil {
			return common.Result{}, operatorerrors.Wrap(err, operatorerrors.FabricOrdererMigrationFailed, "failed to migrate fabric orderer to version v2.4.x")
		}
	}

	if update.MigrateToV25() {
		if err := n.FabricOrdererMigrationV2_5(instance); err != nil {
			return common.Result{}, operatorerrors.Wrap(err, operatorerrors.FabricOrdererMigrationFailed, "failed to migrate fabric orderer to version v2.5.x")
		}
	}

	err = n.ReconcileManagers(instance, update, nil)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to reconcile managers")
	}

	err = n.UpdateConnectionProfile(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to create connection profile")
	}

	err = n.CheckStates(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check and restore state")
	}

	err = n.UpdateParentStatus(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to update parent's status")
	}

	status, result, err := n.CustomLogic(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to run custom offering logic		")
	}
	if result != nil {
		log.Info(fmt.Sprintf("Finished reconciling '%s' with Custom Logic result", instance.GetName()))
		return *result, nil
	}

	if update.EcertUpdated() {
		log.Info("Ecert was updated")
		// Request deployment restart for tls cert update
		err = n.Restart.ForCertUpdate(commoninit.ECERT, instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	if update.TLSCertUpdated() {
		log.Info("TLS cert was updated")
		// Request deployment restart for ecert update
		err = n.Restart.ForCertUpdate(commoninit.TLS, instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	if update.MSPUpdated() {
		if err = n.UpdateMSPCertificates(instance); err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update certificates passed in MSP spec")
		}
	}

	if err := n.HandleActions(instance, update); err != nil {
		return common.Result{}, err
	}

	if err := n.HandleRestart(instance, update); err != nil {
		return common.Result{}, err
	}

	return common.Result{
		Status: status,
	}, nil
}

func (n *Node) ReconcileManagers(instance *current.IBPOrderer, updated baseorderer.Update, genesisBlock []byte) error {
	var err error

	err = n.Node.ReconcileManagers(instance, updated, genesisBlock)
	if err != nil {
		return err
	}

	update := updated.SpecUpdated()

	err = n.RouteManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Orderer Route reconciliation")
	}

	err = n.OperationsRouteManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Operations Route reconciliation")
	}

	err = n.GRPCRouteManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Orderer GRPC Route reconciliation")
	}

	currentVer := version.String(instance.Spec.FabricVersion)
	if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_4_1) {
		err = n.AdminRouteManager.Reconcile(instance, update)
		if err != nil {
			return errors.Wrap(err, "failed Orderer Admin Route reconciliation")
		}
	}

	return nil
}
