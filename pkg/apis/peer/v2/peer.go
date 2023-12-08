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

package v2

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

type Core struct {
	Peer       Peer          `json:"peer,omitempty"`
	Chaincode  Chaincode     `json:"chaincode,omitempty"`
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
	Keepalive              KeepAlive         `json:"keepalive,omitempty"`
	Gossip                 Gossip            `json:"gossip,omitempty"`
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
	Limits                 Limits            `json:"limits,omitempty"`
	MaxRecvMsgSize         int               `json:"maxRecvMsgSize,omitempty"`
	MaxSendMsgSize         int               `json:"maxSendMsgSize,omitempty"`
}

type Gossip struct {
	Bootstrap                  []string        `json:"bootstrap,omitempty"`
	UseLeaderElection          *bool           `json:"useLeaderElection,omitempty"`
	OrgLeader                  *bool           `json:"orgLeader,omitempty"`
	MembershipTrackerInterval  common.Duration `json:"membershipTrackerInterval,omitempty"`
	Endpoint                   string          `json:"endpoint,omitempty"`
	MaxBlockCountToStore       int             `json:"maxBlockCountToStore,omitempty"`
	MaxPropagationBurstLatency common.Duration `json:"maxPropagationBurstLatency,omitempty"`
	MaxPropagationBurstSize    int             `json:"maxPropagationBurstSize,omitempty"`
	PropagateIterations        int             `json:"propagateIterations,omitempty"`
	PropagatePeerNum           int             `json:"propagatePeerNum,omitempty"`
	PullInterval               common.Duration `json:"pullInterval,omitempty"`
	PullPeerNum                int             `json:"pullPeerNum,omitempty"`
	RequestStateInfoInterval   common.Duration `json:"requestStateInfoInterval,omitempty"`
	PublishStateInfoInterval   common.Duration `json:"publishStateInfoInterval,omitempty"`
	StateInfoRetentionInterval common.Duration `json:"stateInfoRetentionInterval,omitempty"`
	PublishCertPeriod          common.Duration `json:"publishCertPeriod,omitempty"`
	SkipBlockVerification      *bool           `json:"skipBlockVerification,omitempty"`
	DialTimeout                common.Duration `json:"dialTimeout,omitempty"`
	ConnTimeout                common.Duration `json:"connTimeout,omitempty"`
	RecvBuffSize               int             `json:"recvBuffSize,omitempty"`
	SendBuffSize               int             `json:"sendBuffSize,omitempty"`
	DigestWaitTime             common.Duration `json:"digestWaitTime,omitempty"`
	RequestWaitTime            common.Duration `json:"requestWaitTime,omitempty"`
	ResponseWaitTime           common.Duration `json:"responseWaitTime,omitempty"`
	AliveTimeInterval          common.Duration `json:"aliveTimeInterval,omitempty"`
	AliveExpirationTimeout     common.Duration `json:"aliveExpirationTimeout,omitempty"`
	ReconnectInterval          common.Duration `json:"reconnectInterval,omitempty"`
	ExternalEndpoint           string          `json:"externalEndpoint,omitempty"`
	Election                   v1.Election     `json:"election,omitempty"`
	PvtData                    PVTData         `json:"pvtData,omitempty"`
	State                      v1.State        `json:"state,omitempty"`
	MaxConnectionAttempts      int             `json:"maxConnectionAttempts,omitempty"`
	MsgExpirationFactor        int             `json:"msgExpirationFactor,omitempty"`
}

type PVTData struct {
	PullRetryThreshold                         common.Duration                       `json:"pullRetryThreshold,omitempty"`
	TransientstoreMaxBlockRetention            int                                   `json:"transientstoreMaxBlockRetention,omitempty"`
	PushAckTimeout                             common.Duration                       `json:"pushAckTimeout,omitempty"`
	BtlPullMargin                              int                                   `json:"btlPullMargin,omitempty"`
	ReconcileBatchSize                         int                                   `json:"reconcileBatchSize,omitempty"`
	ReconcileSleepInterval                     common.Duration                       `json:"reconcileSleepInterval,omitempty"`
	ReconciliationEnabled                      *bool                                 `json:"reconciliationEnabled,omitempty"`
	SkipPullingInvalidTransactionsDuringCommit *bool                                 `json:"skipPullingInvalidTransactionsDuringCommit,omitempty"`
	ImplicitCollectionDisseminationPolicy      ImplicitCollectionDisseminationPolicy `json:"implicitCollectionDisseminationPolicy,omitempty"`
}

type AddressOverride struct {
	From        string `json:"from"`
	To          string `json:"to"`
	CACertsFile string `json:"caCertsFile"`
	certBytes   []byte
}

type Limits struct {
	Concurrency Concurrency `json:"concurrency,omitempty"`
}

type Concurrency struct {
	EndorserService int `json:"endorserService,omitempty"`
	DeliverService  int `json:"deliverService,omitempty"`
	GatewayService  int `json:"gatewayService,omitempty"`
}

type ImplicitCollectionDisseminationPolicy struct {
	RequiredPeerCount int `json:"requiredPeerCount,omitempty"`
	MaxPeerCount      int `json:"maxPeerCount,omitempty"`
}

type Chaincode struct {
	ID               v1.ID             `json:"id,omitempty"`
	Builder          string            `json:"builder,omitempty"`
	Pull             *bool             `json:"pull,omitempty"`
	Golang           v1.Golang         `json:"golang,omitempty"`
	Java             v1.Java           `json:"java,omitempty"`
	Node             v1.Node           `json:"node,omitempty"`
	StartupTimeout   common.Duration   `json:"startuptimeout,omitempty"`
	ExecuteTimeout   common.Duration   `json:"executetimeout,omitempty"`
	Mode             string            `json:"mode,omitempty"`
	KeepAlive        common.Duration   `json:"keepalive,omitempty"`
	System           map[string]string `json:"system,omitempty"`
	Logging          v1.Logging        `json:"logging,omitempty"`
	ExternalBuilders []ExternalBuilder `json:"externalBuilders,omitempty"`
	InstallTimeout   common.Duration   `json:"installTimeout,omitempty"`
}

type ExternalBuilder struct {
	Path                 string   `json:"path,omitempty"`
	Name                 string   `json:"name,omitempty"`
	EnvironmentWhiteList []string `json:"environmentWhiteList,omitempty"`
	PropogateEnvironment []string `json:"propagateEnvironment,omitempty"`
}

type Ledger struct {
	State        LedgerState      `json:"state,omitempty"`
	History      v1.LedgerHistory `json:"history,omitempty"`
	PvtDataStore PvtDataStore     `json:"pvtdataStore,omitempty"`
}

type LedgerState struct {
	StateDatabase   string        `json:"stateDatabase,omitempty"`
	TotalQueryLimit int           `json:"totalQueryLimit,omitempty"`
	CouchdbConfig   CouchdbConfig `json:"couchDBConfig,omitempty"`
	SnapShots       SnapShots     `json:"SnapShots,omitempty"`
}

type CouchdbConfig struct {
	CouchDBAddress          string          `json:"couchDBAddress,omitempty"`
	Username                string          `json:"username,omitempty"`
	Password                string          `json:"password,omitempty"`
	MaxRetries              int             `json:"maxRetries,omitempty"`
	MaxRetriesOnStartup     int             `json:"maxRetriesOnStartup,omitempty"`
	RequestTimeout          common.Duration `json:"requestTimeout,omitempty"`
	QueryLimit              int             `json:"internalQueryLimit,omitempty"`
	MaxBatchUpdateSize      int             `json:"maxBatchUpdateSize,omitempty"`
	WarmIndexesAfterNBlocks int             `json:"warmIndexesAfterNBlocks,omitempty"`
	CreateGlobalChangesDB   *bool           `json:"createGlobalChangesDB,omitempty"`
	CacheSize               int             `json:"cacheSize,omitempty"`
}

type SnapShots struct {
	RootDir string `json:"rootDir,omitempty"`
}

type PvtDataStore struct {
	CollElgProcMaxDbBatchSize           int             `json:"collElgProcMaxDbBatchSize,omitempty"`
	CollElgProcDbBatchesInterval        int             `json:"collElgProcDbBatchesInterval,omitempty"`
	DeprioritizedDataReconcilerInterval common.Duration `json:"deprioritizedDataReconcilerInterval,omitempty"`
}

type Gateway struct {
	Enabled            *bool           `json:"enabled,omitempty"`
	EndorsementTimeout common.Duration `json:"endorsementTimeout,omitempty"`
	DialTimeout        common.Duration `json:"dialTimeout,omitempty"`
}

type KeepAlive struct {
	Interval       common.Duration    `json:"interval,omitempty"`
	Timeout        common.Duration    `json:"timeout,omitempty"`
	MinInterval    common.Duration    `json:"minInterval,omitempty"`
	Client         v1.KeepAliveClient `json:"client,omitempty"`
	DeliveryClient v1.KeepAliveClient `json:"deliveryClient,omitempty"`
}

func (a *AddressOverride) CACertsFileToBytes() ([]byte, error) {
	data, err := util.Base64ToBytes(a.CACertsFile)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (a *AddressOverride) GetCertBytes() []byte {
	return a.certBytes
}
