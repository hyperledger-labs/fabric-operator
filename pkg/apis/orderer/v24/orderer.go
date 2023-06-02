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

package v24

import (
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v1"
)

type Orderer struct {
	General              General              `json:"general,omitempty"`
	FileLedger           FileLedger           `json:"fileLedger,omitempty"`
	Debug                v1.Debug             `json:"debug,omitempty"`
	Consensus            interface{}          `json:"consensus,omitempty"`
	Operations           v1.Operations        `json:"operations,omitempty"`
	Metrics              v1.Metrics           `json:"metrics,omitempty"`
	Admin                Admin                `json:"admin,omitempty"`
	ChannelParticipation ChannelParticipation `json:"channelParticipation,omitempty"`
}

type General struct {
	ListenAddress     string             `json:"listenAddress,omitempty"`
	ListenPort        uint16             `json:"listenPort,omitempty"`
	TLS               v1.TLS             `json:"tls,omitempty"`
	Cluster           v1.Cluster         `json:"cluster,omitempty"`
	Keepalive         v1.Keepalive       `json:"keepalive,omitempty"`
	ConnectionTimeout commonapi.Duration `json:"connectionTimeout,omitempty"`
	GenesisFile       string             `json:"genesisFile,omitempty"` // For compatibility only, will be replaced by BootstrapFile
	BootstrapFile     string             `json:"bootstrapFile,omitempty"`
	BootstrapMethod   string             `json:"bootstrapMethod,omitempty"`
	Profile           v1.Profile         `json:"profile,omitempty"`
	LocalMSPDir       string             `json:"localMspDir,omitempty"`
	LocalMSPID        string             `json:"localMspId,omitempty"`
	BCCSP             *commonapi.BCCSP   `json:"BCCSP,omitempty"`
	Authentication    v1.Authentication  `json:"authentication,omitempty"`
	MaxRecvMsgSize    int                `json:"maxRecvMsgSize,omitempty"`
	MaxSendMsgSize    int                `json:"maxSendMsgSize,omitempty"`
}

// FileLedger contains configuration for the file-based ledger.
type FileLedger struct {
	Location string `json:"location,omitempty"`
}

type Admin struct {
	ListenAddress string `json:"listenAddress,omitempty"`
	TLs           v1.TLS `json:"tls,omitempty"`
}

type ChannelParticipation struct {
	Enabled            *bool  `json:"enabled,omitempty"`
	MaxRequestBodySize uint32 `json:"maxRequestBodySize,omitempty"`
}
