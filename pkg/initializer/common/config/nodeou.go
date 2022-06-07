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

package config

import "sigs.k8s.io/yaml"

type NodeOUConfig struct {
	NodeOUs NodeOUs
}

type NodeOUs struct {
	Enable              bool
	ClientOUIdentifier  Identifier
	PeerOUIdentifier    Identifier
	AdminOUIdentifier   Identifier
	OrdererOUIdentifier Identifier
}

type Identifier struct {
	Certificate                  string
	OrganizationalUnitIdentifier string
}

func NodeOUConfigFromBytes(nodeOU []byte) (*NodeOUConfig, error) {
	nodeOUConfig := &NodeOUConfig{}
	err := yaml.Unmarshal(nodeOU, nodeOUConfig)
	if err != nil {
		return nil, err
	}

	return nodeOUConfig, nil
}

func NodeOUConfigToBytes(config *NodeOUConfig) ([]byte, error) {
	nodeOUBytes, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	return nodeOUBytes, nil
}
