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

package initializer

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	configv1 "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
	configv2 "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v2"
	configv25 "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v25"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate counterfeiter -o mocks/client.go -fake-name Client ../../k8s/controllerclient Client

type CoreConfigMap struct {
	Config    *Config
	Scheme    *runtime.Scheme
	GetLabels func(instance metav1.Object) map[string]string
	Client    k8sclient.Client
}

func (c *CoreConfigMap) GetCoreConfig(instance *current.IBPPeer) (*corev1.ConfigMap, error) {
	return common.GetConfigFromConfigMap(c.Client, instance)
}

func (c *CoreConfigMap) CreateOrUpdate(instance *current.IBPPeer, peer CoreConfig) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", instance.GetName()),
			Namespace: instance.GetNamespace(),
			Labels:    c.GetLabels(instance),
		},
		BinaryData: map[string][]byte{},
	}

	existing, err := c.GetCoreConfig(instance)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	if existing != nil {
		cm.BinaryData = existing.BinaryData
	}

	peerBytes, err := peer.ToBytes()
	if err != nil {
		return err
	}
	cm.BinaryData["core.yaml"] = peerBytes

	err = c.addNodeOU(instance, cm)
	if err != nil {
		return err
	}

	err = c.Client.CreateOrUpdate(context.TODO(), cm, k8sclient.CreateOrUpdateOption{
		Owner:  instance,
		Scheme: c.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create or update Peer config map")
	}

	return nil

}

func (c *CoreConfigMap) AddNodeOU(instance *current.IBPPeer) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", instance.GetName()),
			Namespace: instance.GetNamespace(),
			Labels:    c.GetLabels(instance),
		},
		BinaryData: map[string][]byte{},
	}

	existing, err := c.GetCoreConfig(instance)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	if existing != nil {
		cm.BinaryData = existing.BinaryData
	}

	err = c.addNodeOU(instance, cm)
	if err != nil {
		return err
	}

	err = c.Client.CreateOrUpdate(context.TODO(), cm, k8sclient.CreateOrUpdateOption{
		Owner:  instance,
		Scheme: c.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create or update Peer config map")
	}

	return nil
}

func (c *CoreConfigMap) addNodeOU(instance *current.IBPPeer, cm *corev1.ConfigMap) error {
	if !instance.Spec.NodeOUDisabled() {
		configFilePath := c.Config.OUFile

		// Check if both intermediate ecerts and tlscerts exists
		if util.IntermediateSecretExists(c.Client, instance.Namespace, fmt.Sprintf("ecert-%s-intercerts", instance.Name)) &&
			util.IntermediateSecretExists(c.Client, instance.Namespace, fmt.Sprintf("tls-%s-intercerts", instance.Name)) {
			configFilePath = c.Config.InterOUFile
		}

		ouBytes, err := ioutil.ReadFile(filepath.Clean(configFilePath))
		if err != nil {
			return errors.Wrapf(err, "failed to read OU config file from '%s'", configFilePath)
		}

		cm.BinaryData["config.yaml"] = ouBytes
	} else {
		// Set enabled to false in config
		nodeOUConfig, err := config.NodeOUConfigFromBytes(cm.BinaryData["config.yaml"])
		if err != nil {
			return err
		}

		nodeOUConfig.NodeOUs.Enable = false
		ouBytes, err := config.NodeOUConfigToBytes(nodeOUConfig)
		if err != nil {
			return err
		}

		cm.BinaryData["config.yaml"] = ouBytes
	}

	return nil
}

func GetCoreFromConfigMap(client k8sclient.Client, instance *current.IBPPeer) (*corev1.ConfigMap, error) {
	return common.GetConfigFromConfigMap(client, instance)
}

func GetCoreConfigFromBytes(instance *current.IBPPeer, bytes []byte) (CoreConfig, error) {
	switch version.GetMajorReleaseVersion(instance.Spec.FabricVersion) {
	case version.V2:
		peerversion := version.String(instance.Spec.FabricVersion)
		if peerversion.EqualWithoutTag(version.V2_5_1) || peerversion.GreaterThan(version.V2_5_1) {
			v25config, err := configv25.ReadCoreFromBytes(bytes)
			if err != nil {
				return nil, err
			}
			return v25config, nil
		} else {
			v2config, err := configv2.ReadCoreFromBytes(bytes)
			if err != nil {
				return nil, err
			}
			return v2config, nil
		}
	case version.V1:
		fallthrough
	default:
		// Choosing to default to v1.4 to not break backwards comptability, if coming
		// from a previous version of operator the 'FabricVersion' field would not be set and would
		// result in an error.
		v1config, err := configv1.ReadCoreFromBytes(bytes)
		if err != nil {
			return nil, err
		}
		return v1config, nil
	}
}

func GetCoreConfigFromFile(instance *current.IBPPeer, file string) (CoreConfig, error) {
	switch version.GetMajorReleaseVersion(instance.Spec.FabricVersion) {
	case version.V2:
		log.Info("v2 Fabric Peer requested")
		peerversion := version.String(instance.Spec.FabricVersion)
		if peerversion.EqualWithoutTag(version.V2_5_1) || peerversion.GreaterThan(version.V2_5_1) {
			v25config, err := configv25.ReadCoreFile(file)
			if err != nil {
				return nil, err
			}
			return v25config, nil
		} else {
			v2config, err := configv2.ReadCoreFile(file)
			if err != nil {
				return nil, err
			}
			return v2config, nil
		}
	case version.V1:
		fallthrough
	default:
		// Choosing to default to v1.4 to not break backwards comptability, if coming
		// from a previous version of operator the 'FabricVersion' field would not be set and would
		// result in an error. // TODO: Determine if we want to throw error or handle setting
		// FabricVersion as part of migration logic.
		log.Info("v1 Fabric Peer requested")
		pconfig, err := configv1.ReadCoreFile(file)
		if err != nil {
			return nil, err
		}
		return pconfig, nil
	}
}
