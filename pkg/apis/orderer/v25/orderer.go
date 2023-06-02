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
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v1"
	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v24"
)

type Orderer struct {
	General              v2.General              `json:"general,omitempty"`
	FileLedger           v2.FileLedger           `json:"fileLedger,omitempty"`
	Debug                v1.Debug                `json:"debug,omitempty"`
	Consensus            interface{}             `json:"consensus,omitempty"`
	Operations           v1.Operations           `json:"operations,omitempty"`
	Metrics              v1.Metrics              `json:"metrics,omitempty"`
	Admin                v2.Admin                `json:"admin,omitempty"`
	ChannelParticipation v2.ChannelParticipation `json:"channelParticipation,omitempty"`
}
