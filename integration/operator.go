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

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"

	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	cainit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	ordererinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer"
	peerinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	uzap "go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// GetOperatorConfig returns the operator configuration with the default templating files population
// and with default versions set for components.
func GetOperatorConfig(configs, caFiles, peerFiles, ordererFiles, consoleFiles string) *config.Config {
	ulevel := uzap.NewAtomicLevelAt(2)
	if os.Getenv("LOG_LEVEL") == "debug" {
		ulevel = uzap.NewAtomicLevelAt(-1)
	}
	level := zap.Level(&ulevel)
	logger := zap.New(zap.Opts(level))

	cfg := &config.Config{
		CAInitConfig: &cainit.Config{
			CADefaultConfigPath:    filepath.Join(configs, "ca/ca.yaml"),
			TLSCADefaultConfigPath: filepath.Join(configs, "ca/tlsca.yaml"),
			DeploymentFile:         filepath.Join(caFiles, "deployment.yaml"),
			PVCFile:                filepath.Join(caFiles, "pvc.yaml"),
			ServiceFile:            filepath.Join(caFiles, "service.yaml"),
			RoleFile:               filepath.Join(caFiles, "role.yaml"),
			ServiceAccountFile:     filepath.Join(caFiles, "serviceaccount.yaml"),
			RoleBindingFile:        filepath.Join(caFiles, "rolebinding.yaml"),
			ConfigMapFile:          filepath.Join(caFiles, "configmap-caoverride.yaml"),
			IngressFile:            filepath.Join(caFiles, "ingress.yaml"),
			Ingressv1beta1File:     filepath.Join(caFiles, "ingressv1beta1.yaml"),
			RouteFile:              filepath.Join(caFiles, "route.yaml"),
			SharedPath:             "/tmp/data",
		},
		PeerInitConfig: &peerinit.Config{
			CorePeerFile:           filepath.Join(configs, "peer/core.yaml"),
			CorePeerV2File:         filepath.Join(configs, "peer/v2/core.yaml"),
			CorePeerV25File:        filepath.Join(configs, "peer/v25/core.yaml"),
			OUFile:                 filepath.Join(configs, "peer/ouconfig.yaml"),
			InterOUFile:            filepath.Join(configs, "peer/ouconfig-inter.yaml"),
			DeploymentFile:         filepath.Join(peerFiles, "deployment.yaml"),
			PVCFile:                filepath.Join(peerFiles, "pvc.yaml"),
			CouchDBPVCFile:         filepath.Join(peerFiles, "couchdb-pvc.yaml"),
			ServiceFile:            filepath.Join(peerFiles, "service.yaml"),
			RoleFile:               filepath.Join(peerFiles, "role.yaml"),
			ServiceAccountFile:     filepath.Join(peerFiles, "serviceaccount.yaml"),
			RoleBindingFile:        filepath.Join(peerFiles, "rolebinding.yaml"),
			FluentdConfigMapFile:   filepath.Join(peerFiles, "fluentd-configmap.yaml"),
			CouchContainerFile:     filepath.Join(peerFiles, "couchdb.yaml"),
			CouchInitContainerFile: filepath.Join(peerFiles, "couchdb-init.yaml"),
			IngressFile:            filepath.Join(peerFiles, "ingress.yaml"),
			Ingressv1beta1File:     filepath.Join(peerFiles, "ingressv1beta1.yaml"),
			CCLauncherFile:         filepath.Join(peerFiles, "chaincode-launcher.yaml"),
			RouteFile:              filepath.Join(peerFiles, "route.yaml"),
			StoragePath:            "/tmp/peerinit",
		},
		OrdererInitConfig: &ordererinit.Config{
			OrdererV2File:      filepath.Join(configs, "orderer/v2/orderer.yaml"),
			OrdererV24File:     filepath.Join(configs, "orderer/v24/orderer.yaml"),
			OrdererV25File:     filepath.Join(configs, "orderer/v25/orderer.yaml"),
			OrdererFile:        filepath.Join(configs, "orderer/orderer.yaml"),
			ConfigTxFile:       filepath.Join(configs, "orderer/configtx.yaml"),
			OUFile:             filepath.Join(configs, "orderer/ouconfig.yaml"),
			InterOUFile:        filepath.Join(configs, "orderer/ouconfig-inter.yaml"),
			DeploymentFile:     filepath.Join(ordererFiles, "deployment.yaml"),
			PVCFile:            filepath.Join(ordererFiles, "pvc.yaml"),
			ServiceFile:        filepath.Join(ordererFiles, "service.yaml"),
			CMFile:             filepath.Join(ordererFiles, "configmap.yaml"),
			RoleFile:           filepath.Join(ordererFiles, "role.yaml"),
			ServiceAccountFile: filepath.Join(ordererFiles, "serviceaccount.yaml"),
			RoleBindingFile:    filepath.Join(ordererFiles, "rolebinding.yaml"),
			IngressFile:        filepath.Join(ordererFiles, "ingress.yaml"),
			Ingressv1beta1File: filepath.Join(ordererFiles, "ingressv1beta1.yaml"),
			RouteFile:          filepath.Join(ordererFiles, "route.yaml"),
			StoragePath:        "/tmp/ordererinit",
		},
		ConsoleInitConfig: &config.ConsoleConfig{
			DeploymentFile:           filepath.Join(consoleFiles, "deployment.yaml"),
			PVCFile:                  filepath.Join(consoleFiles, "pvc.yaml"),
			ServiceFile:              filepath.Join(consoleFiles, "service.yaml"),
			CMFile:                   filepath.Join(consoleFiles, "configmap.yaml"),
			ConsoleCMFile:            filepath.Join(consoleFiles, "console-configmap.yaml"),
			DeployerCMFile:           filepath.Join(consoleFiles, "deployer-configmap.yaml"),
			RoleFile:                 filepath.Join(consoleFiles, "role.yaml"),
			RoleBindingFile:          filepath.Join(consoleFiles, "rolebinding.yaml"),
			ServiceAccountFile:       filepath.Join(consoleFiles, "serviceaccount.yaml"),
			IngressFile:              filepath.Join(consoleFiles, "ingress.yaml"),
			Ingressv1beta1File:       filepath.Join(consoleFiles, "ingressv1beta1.yaml"),
			NetworkPolicyIngressFile: filepath.Join(consoleFiles, "networkpolicy-ingress.yaml"),
			NetworkPolicyDenyAllFile: filepath.Join(consoleFiles, "networkpolicy-denyall.yaml"),
		},
		Logger: &logger,
		Operator: config.Operator{
			Restart: config.Restart{
				Timeout: common.MustParseDuration("5m"),
			},
		},
	}

	setDefaultVersions(cfg)
	return cfg
}

func setDefaultVersions(operatorCfg *config.Config) {
	operatorCfg.Operator.Versions = &deployer.Versions{
		CA: map[string]deployer.VersionCA{
			FabricCAVersion + "-1": {
				Default: true,
				Version: FabricCAVersion + "-1",
				Image: deployer.CAImages{
					CAInitImage: InitImage,
					CAInitTag:   InitTag,
					CAImage:     CaImage,
					CATag:       CaTag,
				},
			},
		},
		Peer: map[string]deployer.VersionPeer{
			FabricVersion + "-1": {
				Default: true,
				Version: FabricVersion + "-1",
				Image: deployer.PeerImages{
					PeerInitImage: InitImage,
					PeerInitTag:   InitTag,
					PeerImage:     PeerImage,
					PeerTag:       PeerTag,
					CouchDBImage:  CouchdbImage,
					CouchDBTag:    CouchdbTag,
					GRPCWebImage:  GrpcwebImage,
					GRPCWebTag:    GrpcwebTag,
				},
			},
		},
		Orderer: map[string]deployer.VersionOrderer{
			FabricVersion + "-1": {
				Default: true,
				Version: FabricVersion + "-1",
				Image: deployer.OrdererImages{
					OrdererInitImage: InitImage,
					OrdererInitTag:   InitTag,
					OrdererImage:     OrdererImage,
					OrdererTag:       OrdererTag,
					GRPCWebImage:     GrpcwebImage,
					GRPCWebTag:       GrpcwebTag,
				},
			},
		},
	}
}

type Operator struct {
	NativeResourcePoller
}

func (o *Operator) GetPod() (*corev1.Pod, error) {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("name=%s", o.Name),
	}
	podList, err := o.Client.CoreV1().Pods(o.Namespace).List(context.TODO(), opts)
	if err != nil {
		return nil, err
	}
	return &podList.Items[0], nil
}

func (o *Operator) Restart() error {
	pod, err := o.GetPod()
	if err != nil {
		return err
	}

	err = o.Client.CoreV1().Pods(o.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
