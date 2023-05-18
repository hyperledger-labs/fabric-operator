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

package v1beta1

import (
	"encoding/json"
	"strings"
)

func (s *IBPConsole) ResetRestart() {
	s.Spec.Action.Restart = false
}

// GetMSPID returns empty string as no orgs are
// associated with console (implemented for the
// restart manager logic)
func (s *IBPConsole) GetMSPID() string {
	// no-op
	return ""
}

func (s *IBPConsole) UseTags() bool {
	useTags := false
	if s.Spec.UseTags != nil && *(s.Spec.UseTags) {
		useTags = *s.Spec.UseTags
	}
	return useTags
}

func (s *IBPConsoleSpec) GetOverridesConsole() (*ConsoleOverridesConsole, error) {
	override := &ConsoleOverridesConsole{}
	if s.ConfigOverride != nil && s.ConfigOverride.Console != nil {
		err := json.Unmarshal(s.ConfigOverride.Console.Raw, override)
		if err != nil {
			return nil, err
		}
	}
	return override, nil
}

func (s *IBPConsoleSpec) GetOverridesDeployer() (*ConsoleOverridesDeployer, error) {
	override := &ConsoleOverridesDeployer{}
	if s.ConfigOverride != nil && s.ConfigOverride.Deployer != nil {
		err := json.Unmarshal(s.ConfigOverride.Deployer.Raw, override)
		if err != nil {
			return nil, err
		}
	}
	return override, nil
}

func (s *IBPConsoleSpec) UsingRemoteDB() bool {
	if strings.Contains(s.ConnectionString, "localhost") || s.ConnectionString == "" {
		return false
	}

	return true
}

func (v *Versions) Override(requestedVersions *Versions, registryURL string, arch string) {
	if requestedVersions == nil {
		return
	}

	if len(requestedVersions.CA) != 0 {
		CAVersions := map[string]VersionCA{}
		for key := range requestedVersions.CA {
			var caConfig VersionCA
			requestedCAVersion := requestedVersions.CA[key]
			caConfig.Image.Override(&requestedCAVersion.Image, registryURL, arch)
			caConfig.Default = requestedCAVersion.Default
			caConfig.Version = requestedCAVersion.Version
			CAVersions[key] = caConfig
		}
		v.CA = CAVersions
	}

	if len(requestedVersions.Peer) != 0 {
		PeerVersions := map[string]VersionPeer{}
		for key := range requestedVersions.Peer {
			var peerConfig VersionPeer
			requestedPeerVersion := requestedVersions.Peer[key]
			peerConfig.Image.Override(&requestedPeerVersion.Image, registryURL, arch)
			peerConfig.Default = requestedPeerVersion.Default
			peerConfig.Version = requestedPeerVersion.Version
			PeerVersions[key] = peerConfig
		}
		v.Peer = PeerVersions
	}

	if len(requestedVersions.Orderer) != 0 {
		OrdererVersions := map[string]VersionOrderer{}
		for key := range requestedVersions.Orderer {
			var ordererConfig VersionOrderer
			requestedOrdererVersion := requestedVersions.Orderer[key]
			ordererConfig.Image.Override(&requestedOrdererVersion.Image, registryURL, arch)
			ordererConfig.Default = requestedOrdererVersion.Default
			ordererConfig.Version = requestedOrdererVersion.Version
			OrdererVersions[key] = ordererConfig
		}
		v.Orderer = OrdererVersions
	}
}

func init() {
	SchemeBuilder.Register(&IBPConsole{}, &IBPConsoleList{})
}

func (c *IBPConsoleStatus) HasType() bool {

	return c.CRStatus.Type != ""
}
