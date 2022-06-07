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
	"strconv"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *Override) OrdererNode(object v1.Object, orderernode *current.IBPOrderer, action resources.Action) error {
	instance := object.(*current.IBPOrderer)
	switch action {
	case resources.Create:
		return o.CreateOrderernode(instance, orderernode)
	case resources.Update:
		return o.UpdateOrderernode(instance, orderernode)
	}

	return nil
}

func (o *Override) CreateOrderernode(instance *current.IBPOrderer, orderernode *current.IBPOrderer) error {
	if instance.Spec.ClusterLocation != nil && instance.Spec.ClusterSize != 0 {
		offset := *instance.Spec.NodeNumber - 1
		instance.Spec.Region = instance.Spec.ClusterLocation[offset].Region
		instance.Spec.Zone = instance.Spec.ClusterLocation[offset].Zone

		if instance.Spec.Zone != "" && instance.Spec.Region == "" {
			instance.Spec.Region = "select"
		}
	}

	if !instance.Spec.License.Accept {
		return errors.New("user must accept license before continuing")
	}

	name := instance.GetName()
	instance.DeepCopyInto(orderernode)
	orderernode.Name = name + "node" + strconv.Itoa(*instance.Spec.NodeNumber)
	orderernode.ResourceVersion = ""
	orderernode.Labels = map[string]string{
		"parent": name,
	}

	return nil
}

func (o *Override) UpdateOrderernode(instance *current.IBPOrderer, deployment *current.IBPOrderer) error {
	return nil
}
