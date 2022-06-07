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

package migrator

import (
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/global"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = logf.Log

type Migrator struct {
	Client    k8sclient.Client
	Reader    client.Reader
	Config    *config.Config
	Namespace string
}

func New(mgr manager.Manager, cfg *config.Config, namespace string) *Migrator {
	client := k8sclient.New(mgr.GetClient(), &global.ConfigSetter{})
	reader := mgr.GetAPIReader()
	return &Migrator{
		Client:    client,
		Reader:    reader,
		Config:    cfg,
		Namespace: namespace,
	}
}

func (m *Migrator) Migrate() error {

	// No-op

	return nil
}
