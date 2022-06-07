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

package v1

import (
	"io/ioutil"
	"path/filepath"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/commoncore"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

func ReadCoreFile(path string) (*Core, error) {
	core, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	return coreFromBytes(core)
}

func ReadFrom(from *[]byte) (*Core, error) {
	return coreFromBytes(*from)
}

func ReadCoreFromBytes(core []byte) (*Core, error) {
	return coreFromBytes(core)
}

func coreFromBytes(coreBytes []byte) (*Core, error) {
	coreConfig := &Core{}
	err := yaml.Unmarshal(coreBytes, coreConfig)
	if err != nil {
		// Check if peer.gossip.bootstrap needs to be converted
		updatedCore, err := commoncore.ConvertBootstrapToArray(coreBytes)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert peer.gossip.bootstrap to string array")
		}
		err = yaml.Unmarshal(updatedCore, coreConfig)
		if err != nil {
			return nil, err
		}
	}

	return coreConfig, nil
}
