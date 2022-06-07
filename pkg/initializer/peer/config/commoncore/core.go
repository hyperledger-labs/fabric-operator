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
package commoncore

import (
	"sigs.k8s.io/yaml"
)

func ConvertBootstrapToArray(intf interface{}) ([]byte, error) {
	if intf == nil {
		return nil, nil
	}

	if coreBytes, ok := intf.([]byte); ok {
		return convertBootstrapToArray(coreBytes)
	}

	bytes, err := yaml.Marshal(intf)
	if err != nil {
		return nil, err
	}

	return convertBootstrapToArray(bytes)
}

// convertBootstrapToArray returns an updated core config where peer.gossip.bootstrap is
// an array of strings ([]string) instead of a string.
//
// Peer.gossip.bootstrap can be passed to the operator as a string or []string in the peer's
// core config; however, the operator defines the field in the Core config struct definition
// as a []string due to how Fabric parses the field
// (https://github.com/hyperledger/fabric/blob/release-1.4/peer/node/start.go#L897).
func convertBootstrapToArray(coreBytes []byte) ([]byte, error) {
	if coreBytes == nil {
		return nil, nil
	}

	type Core map[string]interface{}

	coreObj := Core{}
	err := yaml.Unmarshal(coreBytes, &coreObj)
	if err != nil {
		return nil, err
	}

	peer, ok := coreObj["peer"].(map[string]interface{})
	if peer == nil {
		// If peer not found, simply return original config
		return coreBytes, nil
	}

	gossip, ok := peer["gossip"].(map[string]interface{})
	if !ok {
		// If peer.gossip not found, simply return original config
		return coreBytes, nil
	}

	bootstrap, ok := gossip["bootstrap"].(string)
	if !ok {
		// If peer.gossip.bootstrap not found or unable to be converted
		// into a string, simply return original config
		return coreBytes, nil
	}

	if bootstrap == "" {
		gossip["bootstrap"] = nil
	} else {
		gossip["bootstrap"] = []string{bootstrap}
	}

	newCore, err := yaml.Marshal(coreObj)
	if err != nil {
		return nil, err
	}

	return newCore, nil
}
