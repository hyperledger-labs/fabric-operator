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

package configmap

import (
	"context"
	"encoding/json"

	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Manager struct {
	Client k8sclient.Client
}

func NewManager(client k8sclient.Client) *Manager {
	return &Manager{
		Client: client,
	}
}

func (c *Manager) GetRestartConfigFrom(cmName string, namespace string, into interface{}) error {
	cm := &corev1.ConfigMap{}
	n := types.NamespacedName{
		Name:      cmName,
		Namespace: namespace,
	}

	err := c.Client.Get(context.TODO(), n, cm)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get %s config map", cmName)
		}

		// If config map doesn't exist yet, keep into cfg empty
		return nil
	}

	if cm.BinaryData["restart-config.yaml"] == nil {
		return nil
	}

	err = json.Unmarshal(cm.BinaryData["restart-config.yaml"], into)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal %s config map", cmName)
	}

	return nil
}

func (c *Manager) UpdateConfig(cmName string, namespace string, cfg interface{}) error {
	bytes, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
		},
		BinaryData: map[string][]byte{
			"restart-config.yaml": bytes,
		},
	}

	err = c.Client.CreateOrUpdate(context.TODO(), cm)
	if err != nil {
		return errors.Wrapf(err, "failed to create or update %s config map", cmName)
	}

	return nil
}
