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

package v25

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/action"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	v2peer "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v2"
	v25peer "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v25"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	v25config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v25"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	ver "github.com/IBM-Blockchain/fabric-operator/version"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

var log = logf.Log.WithName("peer_fabric_migrator")

//go:generate counterfeiter -o mocks/configmapmanager.go -fake-name ConfigMapManager . ConfigMapManager
type ConfigMapManager interface {
	GetCoreConfig(*current.IBPPeer) (*corev1.ConfigMap, error)
	CreateOrUpdate(*current.IBPPeer, initializer.CoreConfig) error
}

//go:generate counterfeiter -o mocks/deploymentmanager.go -fake-name DeploymentManager . DeploymentManager
type DeploymentManager interface {
	Get(metav1.Object) (client.Object, error)
	Delete(metav1.Object) error
	DeploymentStatus(metav1.Object) (appsv1.DeploymentStatus, error)
	GetScheme() *runtime.Scheme
}

type Migrate struct {
	DeploymentManager DeploymentManager
	ConfigMapManager  ConfigMapManager
	Client            k8sclient.Client
}

func (m *Migrate) MigrationNeeded(instance metav1.Object) bool {
	// Check for DinD container, if DinD container not found this is
	// v25 fabric IBP instance
	obj, err := m.DeploymentManager.Get(instance)
	if err != nil {
		// If deployment does not exist, this instance is not a healthy
		// state and migration should be avoided
		return false
	}

	var deploymentUpdated bool
	var configUpdated bool

	dep := obj.(*appsv1.Deployment)
	for _, cont := range dep.Spec.Template.Spec.Containers {
		if strings.ToLower(cont.Name) == "dind" {
			// DinD container found, instance is not at v25
			deploymentUpdated = false
		}
	}

	cm, err := m.ConfigMapManager.GetCoreConfig(instance.(*current.IBPPeer))
	if err != nil {
		// If config map does not exist, this instance is not a healthy
		// state and migration should be avoided
		return false
	}

	v1corebytes := cm.BinaryData["core.yaml"]

	core := &v25config.Core{}
	err = yaml.Unmarshal(v1corebytes, core)
	if err != nil {
		return false
	}

	configUpdated = configHasBeenUpdated(core)

	return !deploymentUpdated || !configUpdated
}

func (m *Migrate) UpgradeDBs(instance metav1.Object, timeouts config.DBMigrationTimeouts) error {
	log.Info(fmt.Sprintf("Resetting Peer '%s'", instance.GetName()))
	return action.UpgradeDBs(m.DeploymentManager, m.Client, instance.(*current.IBPPeer), timeouts)
}

func (m *Migrate) UpdateConfig(instance metav1.Object, version string) error {
	log.Info("Updating config to v25")
	cm, err := m.ConfigMapManager.GetCoreConfig(instance.(*current.IBPPeer))
	if err != nil {
		return errors.Wrap(err, "failed to get config map")
	}
	v1corebytes := cm.BinaryData["core.yaml"]

	core := &v25config.Core{}
	err = yaml.Unmarshal(v1corebytes, core)
	if err != nil {
		return err
	}

	// resetting VM endpoint
	// VM and Ledger structs been added to Peer. endpoint is not required for v25 peer as there is no DinD
	core.VM.Endpoint = ""

	core.Chaincode.ExternalBuilders = []v2peer.ExternalBuilder{
		v2peer.ExternalBuilder{
			Name: "ibp-builder",
			Path: "/usr/local",
			EnvironmentWhiteList: []string{
				"IBP_BUILDER_ENDPOINT",
				"IBP_BUILDER_SHARED_DIR",
			},
			PropogateEnvironment: []string{
				"IBP_BUILDER_ENDPOINT",
				"IBP_BUILDER_SHARED_DIR",
				"PEER_NAME",
			},
		},
	}

	core.Chaincode.InstallTimeout = common.MustParseDuration("300s")
	if core.Chaincode.System == nil {
		core.Chaincode.System = make(map[string]string)
	}
	core.Chaincode.System["_lifecycle"] = "enable"

	core.Peer.Limits.Concurrency.DeliverService = 2500
	core.Peer.Limits.Concurrency.EndorserService = 2500

	core.Peer.Gossip.PvtData.ImplicitCollectionDisseminationPolicy.RequiredPeerCount = 0
	core.Peer.Gossip.PvtData.ImplicitCollectionDisseminationPolicy.MaxPeerCount = 1

	currentVer := ver.String(version)

	trueVal := true

	if currentVer.EqualWithoutTag(ver.V2_5_1) || currentVer.GreaterThan(ver.V2_5_1) {
		core.Peer.Gateway = v25peer.Gateway{
			Enabled:            &trueVal,
			EndorsementTimeout: common.MustParseDuration("30s"),
			DialTimeout:        common.MustParseDuration("120s"),
			BroadcastTimeout:   common.MustParseDuration("30s"),
		}
		core.Peer.Limits.Concurrency.GatewayService = 500
		core.Ledger.State.SnapShots = v2peer.SnapShots{
			RootDir: "/data/peer/ledgersData/snapshots/",
		}

		core.Ledger.PvtDataStore = v25peer.PvtDataStore{
			CollElgProcMaxDbBatchSize:           500,
			CollElgProcDbBatchesInterval:        1000,
			DeprioritizedDataReconcilerInterval: common.MustParseDuration("3600s"),
			PurgeInterval:                       100,
			PurgedKeyAuditLogging:               &trueVal,
		}
	}

	core.Ledger.State.CouchdbConfig.CacheSize = 64
	core.Ledger.State.CouchdbConfig.MaxRetries = 10

	err = m.ConfigMapManager.CreateOrUpdate(instance.(*current.IBPPeer), core)
	if err != nil {
		return err
	}

	return nil
}

// SetChaincodeLauncherResourceOnCR will update the peer's CR by adding chaincode launcher
// resources. The default resources are defined in deployer's config map, which is part
// IBPConsole resource. The default resources are extracted for the chaincode launcher
// by reading the deployer's config map and updating the CR.
func (m *Migrate) SetChaincodeLauncherResourceOnCR(instance metav1.Object) error {
	log.Info("Setting chaincode launcher resource on CR")
	cr := instance.(*current.IBPPeer)

	if cr.Spec.Resources != nil && cr.Spec.Resources.CCLauncher != nil {
		// No need to proceed further if Chaincode launcher resources already set
		return nil
	}

	consoleList := &current.IBPConsoleList{}
	if err := m.Client.List(context.TODO(), consoleList); err != nil {
		return err
	}
	consoles := consoleList.Items

	// If no consoles found, set default resource for chaincode launcher container
	rr := &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("0.1"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}

	if len(consoles) > 0 {
		log.Info("Setting chaincode launcher resource on CR based on deployer config from config map")
		// Get config map associated with console
		cm := &corev1.ConfigMap{}
		nn := types.NamespacedName{
			Name:      fmt.Sprintf("%s-deployer", consoles[0].GetName()),
			Namespace: instance.GetNamespace(),
		}
		if err := m.Client.Get(context.TODO(), nn, cm); err != nil {
			return err
		}

		settingsBytes := []byte(cm.Data["settings.yaml"])
		settings := &deployer.Config{}
		if err := yaml.Unmarshal(settingsBytes, settings); err != nil {
			return err
		}

		if settings.Defaults != nil && settings.Defaults.Resources != nil &&
			settings.Defaults.Resources.Peer != nil && settings.Defaults.Resources.Peer.CCLauncher != nil {

			rr = settings.Defaults.Resources.Peer.CCLauncher
		}
	}

	log.Info(fmt.Sprintf("Setting chaincode launcher resource on CR to %+v", rr))
	if cr.Spec.Resources == nil {
		cr.Spec.Resources = &current.PeerResources{}
	}
	cr.Spec.Resources.CCLauncher = rr
	if err := m.Client.Update(context.TODO(), cr); err != nil {
		return err
	}

	return nil
}

// Updates required from v1.4 to v25.x:
// - External builders
// - Limits
// - Install timeout
// - Implicit collection dissemination policy
func configHasBeenUpdated(core *v25config.Core) bool {
	if len(core.Chaincode.ExternalBuilders) == 0 {
		return false
	}
	if core.Chaincode.ExternalBuilders[0].Name != "ibp-builder" {
		return false
	}

	// Check if install timeout was set
	if reflect.DeepEqual(core.Chaincode.InstallTimeout, common.Duration{}) {
		return false
	}

	if core.Peer.Limits.Concurrency.DeliverService != 2500 {
		return false
	}

	if core.Peer.Limits.Concurrency.EndorserService != 2500 {
		return false
	}

	return true
}
