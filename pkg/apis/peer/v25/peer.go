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
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v1"
	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v2"
)

type Core struct {
	Peer       Peer          `json:"peer,omitempty"`
	Chaincode  v2.Chaincode  `json:"chaincode,omitempty"`
	Operations v1.Operations `json:"operations,omitempty"`
	Metrics    v1.Metrics    `json:"metrics,omitempty"`
	VM         v1.VM         `json:"vm,omitempty"`
	Ledger     Ledger        `json:"ledger,omitempty"`
	// Not Fabric - this is for deployment
	MaxNameLength *int `json:"maxnamelength,omitempty"`
}

type Peer struct {
	ID                     string            `json:"id,omitempty"`
	NetworkID              string            `json:"networkId,omitempty"`
	ListenAddress          string            `json:"listenAddress,omitempty"`
	ChaincodeListenAddress string            `json:"chaincodeListenAddress,omitempty"`
	ChaincodeAddress       string            `json:"chaincodeAddress,omitempty"`
	Address                string            `json:"address,omitempty"`
	AddressAutoDetect      *bool             `json:"addressAutoDetect,omitempty"`
	Gateway                Gateway           `json:"gateway,omitempty"`
	Keepalive              v2.KeepAlive      `json:"keepalive,omitempty"`
	Gossip                 v2.Gossip         `json:"gossip,omitempty"`
	TLS                    v1.TLS            `json:"tls,omitempty"`
	Authentication         v1.Authentication `json:"authentication,omitempty"`
	FileSystemPath         string            `json:"fileSystemPath,omitempty"`
	BCCSP                  *common.BCCSP     `json:"BCCSP,omitempty"`
	MspConfigPath          string            `json:"mspConfigPath,omitempty"`
	LocalMspId             string            `json:"localMspId,omitempty"`
	Client                 v1.Client         `json:"client,omitempty"`
	DeliveryClient         v1.DeliveryClient `json:"deliveryclient,omitempty"`
	LocalMspType           string            `json:"localMspType,omitempty"`
	Profile                v1.Profile        `json:"profile,omitempty"`
	AdminService           v1.AdminService   `json:"adminService,omitempty"`
	Handlers               v1.HandlersConfig `json:"handlers,omitempty"`
	ValidatorPoolSize      int               `json:"validatorPoolSize,omitempty"`
	Discovery              v1.Discovery      `json:"discovery,omitempty"`
	Limits                 v2.Limits         `json:"limits,omitempty"`
	MaxRecvMsgSize         int               `json:"maxRecvMsgSize,omitempty"`
	MaxSendMsgSize         int               `json:"maxSendMsgSize,omitempty"`
}

type Ledger struct {
	State        v2.LedgerState   `json:"state,omitempty"`
	History      v1.LedgerHistory `json:"history,omitempty"`
	PvtDataStore PvtDataStore     `json:"pvtdataStore,omitempty"`
}

type PvtDataStore struct {
	CollElgProcMaxDbBatchSize           int             `json:"collElgProcMaxDbBatchSize,omitempty"`
	CollElgProcDbBatchesInterval        int             `json:"collElgProcDbBatchesInterval,omitempty"`
	DeprioritizedDataReconcilerInterval common.Duration `json:"deprioritizedDataReconcilerInterval,omitempty"`
	PurgeInterval                       int             `json:"purgeInterval,omitempty"`
	PurgedKeyAuditLogging               *bool           `json:"purgedKeyAuditLogging,omitempty"`
}

type Gateway struct {
	Enabled            *bool           `json:"enabled,omitempty"`
	EndorsementTimeout common.Duration `json:"endorsementTimeout,omitempty"`
	DialTimeout        common.Duration `json:"dialTimeout,omitempty"`
	BroadcastTimeout   common.Duration `json:"broadcastTimeout,omitempty"`
}
