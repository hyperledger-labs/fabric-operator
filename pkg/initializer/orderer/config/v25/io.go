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

package v25

import (
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

func ReadOrdererFile(path string) (*Orderer, error) {
	config, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	orderer := &Orderer{}
	err = yaml.Unmarshal(config, orderer)
	if err != nil {
		return nil, err
	}

	return orderer, nil
}

func ReadOrdererFromBytes(config []byte) (*Orderer, error) {
	orderer := &Orderer{}
	err := yaml.Unmarshal(config, orderer)
	if err != nil {
		return nil, err
	}

	return orderer, nil
}

func ReadFrom(from *[]byte) (*Orderer, error) {
	ordererConfig := &Orderer{}
	err := yaml.Unmarshal(*from, ordererConfig)
	if err != nil {
		return nil, err
	}

	return ordererConfig, nil
}
