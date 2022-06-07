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
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *Override) StateDBPVC(object v1.Object, pvc *corev1.PersistentVolumeClaim, action resources.Action) error {
	instance := object.(*current.IBPPeer)
	switch action {
	case resources.Create:
		return o.CreateStateDBPVC(instance, pvc)
	case resources.Update:
		return o.UpdateStateDBPVC(instance, pvc)
	}

	return nil
}

func (o *Override) CreateStateDBPVC(instance *current.IBPPeer, pvc *corev1.PersistentVolumeClaim) error {
	storage := instance.Spec.Storage
	if storage != nil {
		stateDBStorage := storage.StateDB
		if stateDBStorage != nil {
			if stateDBStorage.Class != "" {
				pvc.Spec.StorageClassName = &stateDBStorage.Class
			}
			if stateDBStorage.Size != "" {
				quantity, err := resource.ParseQuantity(stateDBStorage.Size)
				if err != nil {
					return err
				}
				resourceMap := pvc.Spec.Resources.Requests
				if resourceMap == nil {
					resourceMap = corev1.ResourceList{}
				}
				resourceMap[corev1.ResourceStorage] = quantity
				pvc.Spec.Resources.Requests = resourceMap
			}
		}
	}

	if pvc.ObjectMeta.Labels == nil {
		pvc.ObjectMeta.Labels = map[string]string{}
	}
	if instance.Spec.Zone != "" {
		pvc.ObjectMeta.Labels["zone"] = instance.Spec.Zone
	}

	if instance.Spec.Region != "" {
		pvc.ObjectMeta.Labels["region"] = instance.Spec.Region
	}

	return nil
}

func (o *Override) UpdateStateDBPVC(instance *current.IBPPeer, cm *corev1.PersistentVolumeClaim) error {
	return nil
}
