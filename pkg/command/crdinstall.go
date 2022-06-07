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

package command

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/crd"
	"github.com/IBM-Blockchain/fabric-operator/pkg/k8s/clientset"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
)

func CRDInstall(dir string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	err = CRDInstallUsingConfig(config, dir)
	if err != nil {
		return errors.Wrap(err, "failed to install CRDs")
	}

	return nil
}

func CRDInstallUsingConfig(config *rest.Config, dir string) error {
	clientSet, err := clientset.New(config)
	if err != nil {
		return errors.Wrap(err, "failed to get client")
	}

	crds := crd.GetCRDListFromDir(dir)
	manager, err := crd.NewManager(clientSet, crds...)
	if err != nil {
		return errors.Wrap(err, "failed to create CRD manager")
	}

	err = manager.Create()
	if err != nil {
		return errors.Wrap(err, "failed to create CRDs")
	}

	return nil
}
