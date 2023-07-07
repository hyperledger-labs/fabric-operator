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

package k8speer

import (
	"context"
	"fmt"
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	commoninit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	controllerclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	basepeer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer"
	basepeeroverride "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/peer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("k8s_peer")

type Override interface {
	basepeer.Override
	Ingress(v1.Object, *networkingv1.Ingress, resources.Action) error
	Ingressv1beta1(v1.Object, *networkingv1beta1.Ingress, resources.Action) error
}

var _ basepeer.IBPPeer = &Peer{}

type Peer struct {
	*basepeer.Peer

	IngressManager        resources.Manager
	Ingressv1beta1Manager resources.Manager

	Override Override
}

func New(client controllerclient.Client, scheme *runtime.Scheme, config *config.Config) *Peer {
	o := &override.Override{
		Override: basepeeroverride.Override{
			Client:                        client,
			DefaultCouchContainerFile:     config.PeerInitConfig.CouchContainerFile,
			DefaultCouchInitContainerFile: config.PeerInitConfig.CouchInitContainerFile,
			DefaultCCLauncherFile:         config.PeerInitConfig.CCLauncherFile,
		},
	}

	p := &Peer{
		Peer:     basepeer.New(client, scheme, config, o),
		Override: o,
	}

	p.CreateManagers()
	return p
}

func (p *Peer) CreateManagers() {
	resourceManager := resourcemanager.New(p.Client, p.Scheme)
	p.IngressManager = resourceManager.CreateIngressManager("", p.Override.Ingress, p.GetLabels, p.Config.PeerInitConfig.IngressFile)
	p.Ingressv1beta1Manager = resourceManager.CreateIngressv1beta1Manager("", p.Override.Ingressv1beta1, p.GetLabels, p.Config.PeerInitConfig.Ingressv1beta1File)
}

func (p *Peer) ReconcileManagers(instance *current.IBPPeer, update basepeer.Update) error {
	err := p.Peer.ReconcileManagers(instance, update)
	if err != nil {
		return err
	}

	err = p.ReconcileIngressManager(instance, update.SpecUpdated())
	if err != nil {
		return errors.Wrap(err, "failed Ingress reconciliation")
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

	instanceUpdated, err := p.PreReconcileChecks(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed pre reconcile checks")
	}

	// We do not have to wait for service to get the external endpoint
	// thus we call UpdateExternalEndpoint in reconcile before reconcile managers
	externalEndpointUpdated := p.UpdateExternalEndpoint(instance)

	hostAPI := fmt.Sprintf("%s-%s-peer.%s", instance.Namespace, instance.Name, instance.Spec.Domain)
	hostOperations := fmt.Sprintf("%s-%s-operations.%s", instance.Namespace, instance.Name, instance.Spec.Domain)
	hostGrpcWeb := fmt.Sprintf("%s-%s-grpcweb.%s", instance.Namespace, instance.Name, instance.Spec.Domain)
	legacyHostAPI := fmt.Sprintf("%s-%s.%s", instance.Namespace, instance.Name, instance.Spec.Domain)
	hosts := []string{hostAPI, hostOperations, hostGrpcWeb, legacyHostAPI, "127.0.0.1"}
	csrHostUpdated := p.CheckCSRHosts(instance, hosts)

	if instanceUpdated || externalEndpointUpdated || csrHostUpdated {
		log.Info(fmt.Sprintf("Updating instance after pre reconcile checks: %t, updating external endpoint: %t, csr host updated: %t", instanceUpdated, externalEndpointUpdated, csrHostUpdated))
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

func (p *Peer) ReconcileIngressManager(instance *current.IBPPeer, update bool) error {
	if p.Config.Operator.Globals.AllowKubernetesEighteen == "true" {
		// check k8s version
		version, err := util.GetServerVersion()
		if err != nil {
			return err
		}
		if strings.Compare(version.Minor, "19") < 0 { // v1beta
			err = p.Ingressv1beta1Manager.Reconcile(instance, update)
			if err != nil {
				return errors.Wrap(err, "failed Ingressv1beta1 reconciliation")
			}
		} else {
			err = p.IngressManager.Reconcile(instance, update)
			if err != nil {
				return errors.Wrap(err, "failed Ingress reconciliation")
			}
		}
	} else {
		err := p.IngressManager.Reconcile(instance, update)
		if err != nil {
			return errors.Wrap(err, "failed Ingress reconciliation")
		}
	}
	return nil
}
