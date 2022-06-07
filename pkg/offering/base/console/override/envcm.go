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
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *Override) CM(object v1.Object, cm *corev1.ConfigMap, action resources.Action, options map[string]interface{}) error {
	instance := object.(*current.IBPConsole)
	switch action {
	case resources.Create:
		return o.CreateCM(instance, cm)
	case resources.Update:
		return o.UpdateCM(instance, cm)
	}

	return nil
}

func (o *Override) CreateCM(instance *current.IBPConsole, cm *corev1.ConfigMap) error {
	cm.Data["HOST_URL"] = "https://" + instance.Spec.NetworkInfo.Domain + ":443"

	err := o.CommonCM(instance, cm)
	if err != nil {
		return err
	}

	return nil
}

func (o *Override) UpdateCM(instance *current.IBPConsole, cm *corev1.ConfigMap) error {
	err := o.CommonCM(instance, cm)
	if err != nil {
		return err
	}

	return nil
}

func (o *Override) CommonCM(instance *current.IBPConsole, cm *corev1.ConfigMap) error {
	if instance.Spec.ConnectionString != "" {
		connectionString := instance.Spec.ConnectionString
		cm.Data["DB_CONNECTION_STRING"] = connectionString
	}

	if instance.Spec.System != "" {
		system := instance.Spec.System
		cm.Data["DB_SYSTEM"] = system
	}

	if instance.Spec.TLSSecretName != "" {
		cm.Data["KEY_FILE_PATH"] = "/certs/tls/tls.key"
		cm.Data["PEM_FILE_PATH"] = "/certs/tls/tls.crt"
	} else {
		cm.Data["KEY_FILE_PATH"] = "/certs/tls.key"
		cm.Data["PEM_FILE_PATH"] = "/certs/tls.crt"
	}

	return nil
}
