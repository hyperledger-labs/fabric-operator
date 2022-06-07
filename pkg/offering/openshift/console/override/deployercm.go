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

package override

import (
	"errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
	baseconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console/override"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func (o *Override) DeployerCM(object v1.Object, cm *corev1.ConfigMap, action resources.Action, options map[string]interface{}) error {
	instance := object.(*current.IBPConsole)
	switch action {
	case resources.Create:
		return o.CreateDeployerCM(instance, cm, options)
	case resources.Update:
		return o.UpdateDeployerCM(instance, cm, options)
	}

	return nil
}

func (o *Override) CreateDeployerCM(instance *current.IBPConsole, cm *corev1.ConfigMap, options map[string]interface{}) error {
	data := cm.Data["settings.yaml"]

	config := &deployer.Config{}
	err := yaml.Unmarshal([]byte(data), config)
	if err != nil {
		return err
	}

	if instance.Spec.NetworkInfo == nil || instance.Spec.NetworkInfo.Domain == "" {
		return errors.New("domain not provided")
	}

	err = baseconsole.CommonDeployerCM(instance, config, options)
	if err != nil {
		return err
	}

	config.ClusterType = offering.OPENSHIFT.String()
	config.ServiceConfig.Type = corev1.ServiceTypeClusterIP

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	cm.Data["settings.yaml"] = string(bytes)

	return nil
}
