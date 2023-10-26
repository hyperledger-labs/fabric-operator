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

package openshiftpeer

import (
	"context"
	"fmt"
	"regexp"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	commoninit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	controllerclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	basepeer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer"
	basepeeroverride "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/peer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/version"
	openshiftv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("openshift_peer")

type Override interface {
	basepeer.Override
	PeerRoute(object v1.Object, route *routev1.Route, action resources.Action) error
	OperationsRoute(object v1.Object, route *routev1.Route, action resources.Action) error
	PeerGRPCRoute(object v1.Object, route *routev1.Route, action resources.Action) error
}

var _ basepeer.IBPPeer = &Peer{}

type Peer struct {
	*basepeer.Peer

	RouteManager           resources.Manager
	OperationsRouteManager resources.Manager
	GRPCRouteManager       resources.Manager
	RestClient             *clientset.Clientset

	Override Override
}

func New(client controllerclient.Client, scheme *runtime.Scheme, config *config.Config, restclient *clientset.Clientset) *Peer {
	o := &override.Override{
		Override: basepeeroverride.Override{
			Client:                        client,
			DefaultCouchContainerFile:     config.PeerInitConfig.CouchContainerFile,
			DefaultCouchInitContainerFile: config.PeerInitConfig.CouchInitContainerFile,
			DefaultCCLauncherFile:         config.PeerInitConfig.CCLauncherFile,
		},
	}

	peer := &Peer{
		Peer:       basepeer.New(client, scheme, config, o),
		Override:   o,
		RestClient: restclient,
	}

	peer.CreateManagers()
	return peer
}

func (p *Peer) CreateManagers() {
	resourceManager := resourcemanager.New(p.Client, p.Scheme)
	p.RouteManager = resourceManager.CreateRouteManager("peer", p.Override.PeerRoute, p.GetLabels, p.Config.PeerInitConfig.RouteFile)
	p.OperationsRouteManager = resourceManager.CreateRouteManager("operations", p.Override.OperationsRoute, p.GetLabels, p.Config.PeerInitConfig.RouteFile)
	p.GRPCRouteManager = resourceManager.CreateRouteManager("grpcweb", p.Override.PeerGRPCRoute, p.GetLabels, p.Config.PeerInitConfig.RouteFile)
}

func (p *Peer) ReconcileManagers(instance *current.IBPPeer, update basepeer.Update) error {
	err := p.Peer.ReconcileManagers(instance, update)
	if err != nil {
		return err
	}

	err = p.RouteManager.Reconcile(instance, update.SpecUpdated())
	if err != nil {
		return errors.Wrap(err, "failed Peer Route reconciliation")
	}

	err = p.OperationsRouteManager.Reconcile(instance, update.SpecUpdated())
	if err != nil {
		return errors.Wrap(err, "failed Operations Route reconciliation")
	}

	err = p.GRPCRouteManager.Reconcile(instance, update.SpecUpdated())
	if err != nil {
		return errors.Wrap(err, "failed Peer GRPC Route reconciliation")
	}

	return nil
}

func (p *Peer) Reconcile(instance *current.IBPPeer, update basepeer.Update) (common.Result, error) {
	var err error
	var status *current.CRStatus

	versionSet, err := p.SetVersion(instance)
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

	updatecr, err := p.SelectDinDArgs(instance)
	if err != nil {
		log.Info("Cannot get cluster version. Ignoring openshift cluster version")
	}

	update.SetDindArgsUpdated(updatecr)
	instanceUpdated, err := p.PreReconcileChecks(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed pre reconcile checks")
	}

	// We do not have to wait for service to get the external endpoint
	// thus we call UpdateExternalEndpoint in reconcile before reconcile managers
	externalEndpointUpdated := p.UpdateExternalEndpoint(instance)

	hostAPI := fmt.Sprintf("%s-%s-peer.%s", instance.Namespace, instance.Name, instance.Spec.Domain)
	hostOperations := fmt.Sprintf("%s-%s-operations.%s", instance.Namespace, instance.Name, instance.Spec.Domain)
	hostGrpc := fmt.Sprintf("%s-%s-grpcweb.%s", instance.Namespace, instance.Name, instance.Spec.Domain)
	legacyHostAPI := fmt.Sprintf("%s-%s.%s", instance.Namespace, instance.Name, instance.Spec.Domain)
	hosts := []string{hostAPI, hostOperations, hostGrpc, legacyHostAPI, "127.0.0.1"}
	csrHostUpdated := p.CheckCSRHosts(instance, hosts)

	if instanceUpdated || externalEndpointUpdated || csrHostUpdated {
		log.Info(fmt.Sprintf("Updating instance after pre reconcile checks: %t, updating external endpoint: %t, csrhost Updated: %t", instanceUpdated, externalEndpointUpdated, csrHostUpdated))
		err := p.Client.Patch(context.TODO(), instance, nil, controllerclient.PatchOption{
			Resilient: &controllerclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPPeer{},
				Strategy: k8sclient.MergeFrom,
			},
		})
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update instance after prereconcile checks")
		}

		log.Info("Instance updated, requeuing request...")
		return common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
		}, nil
	}

	jobRunning, err := p.HandleMigrationJobs(k8sclient.MatchingLabels{
		"owner":    instance.GetName(),
		"job-name": fmt.Sprintf("%s-dbmigration", instance.GetName()),
	}, instance)
	if jobRunning {
		log.Info(fmt.Sprintf("Requeuing request until job completes"))
		return common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
		}, nil
	}
	if err != nil {
		return common.Result{}, err
	}

	err = p.Initialize(instance, update)
	if err != nil {
		return common.Result{}, operatorerrors.Wrap(err, operatorerrors.PeerInitilizationFailed, "failed to initialize peer")
	}

	if update.PeerTagUpdated() {
		if err := p.ReconcileFabricPeerMigrationV1_4(instance); err != nil {
			return common.Result{}, operatorerrors.Wrap(err, operatorerrors.FabricPeerMigrationFailed, "failed to migrate fabric peer versions")
		}
	}

	if update.MigrateToV2() {
		if err := p.ReconcileFabricPeerMigrationV2_0(instance); err != nil {
			return common.Result{}, operatorerrors.Wrap(err, operatorerrors.FabricPeerMigrationFailed, "failed to migrate fabric peer to version v2.0.x")
		}
	}

	if update.MigrateToV24() {
		if err := p.ReconcileFabricPeerMigrationV2_4(instance); err != nil {
			return common.Result{}, operatorerrors.Wrap(err, operatorerrors.FabricPeerMigrationFailed, "failed to migrate fabric peer to version v2.4.x")
		}
	}

	if update.MigrateToV25() {
		if err := p.ReconcileFabricPeerMigrationV2_5(instance); err != nil {
			return common.Result{}, operatorerrors.Wrap(err, operatorerrors.FabricPeerMigrationFailed, "failed to migrate fabric peer to version v2.5.x")
		}
	}

	err = p.ReconcileManagers(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to reconcile managers")
	}

	err = p.UpdateConnectionProfile(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to create connection profile")
	}

	err = p.CheckStates(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check and restore state")
	}

	// custom product logic can be implemented here
	// No-Op atm
	status, result, err := p.CustomLogic(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to run custom offering logic")
	}
	if result != nil {
		log.Info(fmt.Sprintf("Finished reconciling '%s' with Custom Logic result", instance.GetName()))
		return *result, nil
	}

	if update.EcertUpdated() {
		log.Info("Ecert was updated")
		// Request deployment restart for tls cert update
		err = p.Restart.ForCertUpdate(commoninit.ECERT, instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	if update.TLSCertUpdated() {
		log.Info("TLS cert was updated")
		// Request deployment restart for ecert update
		err = p.Restart.ForCertUpdate(commoninit.TLS, instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	if update.MSPUpdated() {
		err = p.UpdateMSPCertificates(instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update certificates passed in MSP spec")
		}
	}

	if err := p.HandleActions(instance, update); err != nil {
		return common.Result{}, err
	}

	// If configs were update during initialize, need to restart pods to pick up new
	// config changes. This should be done as the last the step, specifically after ReconcileManagers,
	// to allow all any updates to the deployment to be completed before restarting.
	// Trigger deployment restart by deleting deployment
	if err := p.HandleRestart(instance, update); err != nil {
		return common.Result{}, err
	}

	return common.Result{
		Status: status,
	}, nil
}

func (p *Peer) SelectDinDArgs(instance *current.IBPPeer) (bool, error) {

	if len(instance.Spec.DindArgs) != 0 {
		return false, nil
	}

	clusterversion := openshiftv1.ClusterVersion{}

	err := p.RestClient.RESTClient().Get().
		AbsPath("apis", "config.openshift.io", "v1", "clusterversions", "version").
		Do(context.TODO()).
		Into(&clusterversion)

	if err != nil {
		return false, err
	}

	dindargs := []string{"--log-driver", "fluentd", "--log-opt", "fluentd-address=localhost:9880", "--mtu", "1400", "--iptables=true"}

	re := regexp.MustCompile(`4\.[0-9]\.[0-9]`)
	if re.MatchString(clusterversion.Status.Desired.Version) {
		dindargs = []string{"--log-driver", "fluentd", "--log-opt", "fluentd-address=localhost:9880", "--mtu", "1400", "--iptables=false"}
	}

	instance.Spec.DindArgs = dindargs

	return true, nil
}
