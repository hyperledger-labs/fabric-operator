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

func (o *Override) Service(object v1.Object, service *corev1.Service, action resources.Action) error {
	instance := object.(*current.IBPPeer)
	switch action {
	case resources.Create:
		return o.CreateService(instance, service)
	case resources.Update:
		return o.UpdateService(instance, service)
	}

	return nil
}

func (o *Override) CreateService(instance *current.IBPPeer, service *corev1.Service) error {
	if instance.Spec.Service != nil {
		serviceType := instance.Spec.Service.Type
		if serviceType != "" {
			service.Spec.Type = serviceType
		}
	}

	return nil
}

func (o *Override) UpdateService(instance *current.IBPPeer, service *corev1.Service) error {
	return nil
}
