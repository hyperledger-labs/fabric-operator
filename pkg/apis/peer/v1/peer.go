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
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/docker/docker/api/types/container"
)

type Core struct {
	Peer       Peer       `json:"peer,omitempty"`
	Chaincode  Chaincode  `json:"chaincode,omitempty"`
	Operations Operations `json:"operations,omitempty"`
	Metrics    Metrics    `json:"metrics,omitempty"`
	VM         VM         `json:"vm,omitempty"`
	Ledger     Ledger     `json:"ledger,omitempty"`

	// Not Fabric - this is for deployment
	MaxNameLength *int `json:"maxnamelength,omitempty"`
}

type Peer struct {
	ID                     string         `json:"id,omitempty"`
	NetworkID              string         `json:"networkId,omitempty"`
	ListenAddress          string         `json:"listenAddress,omitempty"`
	ChaincodeListenAddress string         `json:"chaincodeListenAddress,omitempty"`
	ChaincodeAddress       string         `json:"chaincodeAddress,omitempty"`
	Address                string         `json:"address,omitempty"`
	AddressAutoDetect      *bool          `json:"addressAutoDetect,omitempty"`
	Keepalive              KeepAlive      `json:"keepalive,omitempty"`
	Gossip                 Gossip         `json:"gossip,omitempty"`
	TLS                    TLS            `json:"tls,omitempty"`
	Authentication         Authentication `json:"authentication,omitempty"`
	FileSystemPath         string         `json:"fileSystemPath,omitempty"`
	BCCSP                  *common.BCCSP  `json:"BCCSP,omitempty"`
	MspConfigPath          string         `json:"mspConfigPath,omitempty"`
	LocalMspId             string         `json:"localMspId,omitempty"`
	Client                 Client         `json:"client,omitempty"`
	DeliveryClient         DeliveryClient `json:"deliveryclient,omitempty"`
	LocalMspType           string         `json:"localMspType,omitempty"`
	Profile                Profile        `json:"profile,omitempty"`
	AdminService           AdminService   `json:"adminService,omitempty"`
	Handlers               HandlersConfig `json:"handlers,omitempty"`
	ValidatorPoolSize      int            `json:"validatorPoolSize,omitempty"`
	Discovery              Discovery      `json:"discovery,omitempty"`
	Limits                 Limits         `json:"limits,omitempty"`
}

type PluginMapping map[string]HandlerConfig

type HandlersConfig struct {
	AuthFilters []HandlerConfig `json:"authFilters"`
	Decorators  []HandlerConfig `json:"decorators"`
	Endorsers   PluginMapping   `json:"endorsers"`
	Validators  PluginMapping   `json:"validators"`
}

type HandlerConfig struct {
	Name    string `json:"name"`
	Library string `json:"library"`
}

type KeepAlive struct {
	MinInterval    common.Duration `json:"minInterval,omitempty"`
	Client         KeepAliveClient `json:"client,omitempty"`
	DeliveryClient KeepAliveClient `json:"deliveryClient,omitempty"`
}

type KeepAliveClient struct {
	Interval common.Duration `json:"interval,omitempty"`
	Timeout  common.Duration `json:"timeout,omitempty"`
}

type KeepAliveDeliveryClient struct {
	Interval common.Duration `json:"interval,omitempty"`
	Timeout  common.Duration `json:"timeout,omitempty"`
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
	Election                   Election        `json:"election,omitempty"`
	PvtData                    PVTData         `json:"pvtData,omitempty"`
	State                      State           `json:"state,omitempty"`
	MaxConnectionAttempts      int             `json:"maxConnectionAttempts,omitempty"`
	MsgExpirationFactor        int             `json:"msgExpirationFactor,omitempty"`
}

type Election struct {
	StartupGracePeriod       common.Duration `json:"startupGracePeriod,omitempty"`
	MembershipSampleInterval common.Duration `json:"membershipSampleInterval,omitempty"`
	LeaderElectionDuration   common.Duration `json:"leaderElectionDuration,omitempty"`
	LeaderAliveThreshold     common.Duration `json:"leaderAliveThreshold,omitempty"`
}

type PVTData struct {
	PullRetryThreshold                         common.Duration `json:"pullRetryThreshold,omitempty"`
	TransientstoreMaxBlockRetention            int             `json:"transientstoreMaxBlockRetention,omitempty"`
	PushAckTimeout                             common.Duration `json:"pushAckTimeout,omitempty"`
	BtlPullMargin                              int             `json:"btlPullMargin,omitempty"`
	ReconcileBatchSize                         int             `json:"reconcileBatchSize,omitempty"`
	ReconcileSleepInterval                     common.Duration `json:"reconcileSleepInterval,omitempty"`
	ReconciliationEnabled                      *bool           `json:"reconciliationEnabled,omitempty"`
	SkipPullingInvalidTransactionsDuringCommit *bool           `json:"skipPullingInvalidTransactionsDuringCommit,omitempty"`
}

type State struct {
	Enabled         *bool           `json:"enabled,omitempty"`
	CheckInterval   common.Duration `json:"checkInterval,omitempty"`
	ResponseTimeout common.Duration `json:"responseTimeout,omitempty"`
	BatchSize       int             `json:"batchSize,omitempty"`
	BlockBufferSize int             `json:"blockBufferSize,omitempty"`
	MaxRetries      int             `json:"maxRetries,omitempty"`
}

type TLS struct {
	Enabled            *bool         `json:"enabled,omitempty"`
	ClientAuthRequired *bool         `json:"clientAuthRequired,omitempty"`
	Cert               Cert          `json:"cert,omitempty"`
	Key                Key           `json:"key,omitempty"`
	RootCert           Cert          `json:"rootCert,omitempty"`
	ClientRootCAs      ClientRootCAs `json:"clientRootCas,omitempty"`
	ClientKey          Key           `json:"clientKey,omitempty"`
	ClientCert         Cert          `json:"clientCert,omitempty"`
}

type Cert struct {
	File string `json:"file,omitempty"`
}

type Key struct {
	File string `json:"file,omitempty"`
}

type ClientRootCAs struct {
	Files []string `json:"files,omitempty"`
}

type Authentication struct {
	Timewindow common.Duration `json:"timewindow,omitempty"`
}

type Client struct {
	ConnTimeout common.Duration `json:"connTimeout,omitempty"`
}

type AddressOverride struct {
	From        string `json:"from"`
	To          string `json:"to"`
	CACertsFile string `json:"caCertsFile"`
}

type DeliveryClient struct {
	ReconnectTotalTimeThreshold common.Duration   `json:"reconnectTotalTimeThreshold,omitempty"`
	ConnTimeout                 common.Duration   `json:"connTimeout,omitempty"`
	ReConnectBackoffThreshold   common.Duration   `json:"reConnectBackoffThreshold,omitempty"`
	AddressOverrides            []AddressOverride `json:"addressOverrides,omitempty"`
}

type Profile struct {
	Enabled       *bool  `json:"enabled,omitempty"`
	ListenAddress string `json:"listenAddress,omitempty"`
}

type AdminService struct {
	ListenAddress string `json:"listenAddress,omitempty"`
}

type Discovery struct {
	Enabled                      *bool   `json:"enabled,omitempty"`
	AuthCacheEnabled             *bool   `json:"authCacheEnabled,omitempty"`
	AuthCacheMaxSize             int     `json:"authCacheMaxSize,omitempty"`
	AuthCachePurgeRetentionRatio float64 `json:"authCachePurgeRetentionRatio,omitempty"`
	OrgMembersAllowedAccess      *bool   `json:"orgMembersAllowedAccess,omitempty"`
}

type Limits struct {
	Concurrency Concurrency `json:"concurrency,omitempty"`
}

type Concurrency struct {
	Qscc int `json:"qscc,omitempty"`
}

// Operations configures the operations endpont for the peer.
type Operations struct {
	ListenAddress string        `json:"listenAddress,omitempty"`
	TLS           OperationsTLS `json:"tls,omitempty"`
}

// TLS contains configuration for TLS connections.
type OperationsTLS struct {
	Enabled            *bool `json:"enabled,omitempty"`
	PrivateKey         File  `json:"key,omitempty"`
	Certificate        File  `json:"cert,omitempty"`
	ClientAuthRequired *bool `json:"clientAuthRequired,omitempty"`
	ClientRootCAs      Files `json:"clientRootCas,omitempty"`
}

type File struct {
	File string `json:"file,omitempty"`
}

type Files struct {
	Files []string `json:"files,omitempty"`
}

// Metrics confiures the metrics provider for the peer.
type Metrics struct {
	Provider string `json:"provider,omitempty"`
	Statsd   Statsd `json:"statsd,omitempty"`
}

// Statsd provides the configuration required to emit statsd metrics from the peer.
type Statsd struct {
	Network       string          `json:"network,omitempty"`
	Address       string          `json:"address,omitempty"`
	WriteInterval common.Duration `json:"writeInterval,omitempty"`
	Prefix        string          `json:"prefix,omitempty"`
}

type Chaincode struct {
	ID             ID                `json:"id,omitempty"`
	Builder        string            `json:"builder,omitempty"`
	Pull           *bool             `json:"pull.omitempty"`
	Golang         Golang            `json:"golang,omitempty"`
	Java           Java              `json:"java,omitempty"`
	Node           Node              `json:"node,omitempty"`
	StartupTimeout common.Duration   `json:"startuptimeout,omitempty"`
	ExecuteTimeout common.Duration   `json:"executetimeout,omitempty"`
	InstallTimeout common.Duration   `json:"installTimeout,omitempty"`
	Mode           string            `json:"mode,omitempty"`
	KeepAlive      common.Duration   `json:"keepalive,omitempty"`
	System         map[string]string `json:"system,omitempty"`
	Logging        Logging           `json:"logging,omitempty"`
	SystemPlugins  []SystemPlugin    `json:"systemPlugins,omitempty"`
}

type SystemPlugin struct {
	Enabled           *bool  `json:"enabled"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	InvokableExternal *bool  `json:"invokableExternal"`
	InvokableCC2CC    *bool  `json:"invokableCC2CC"`
}

type ID struct {
	Path string `json:"path,omitempty"`
	Name string `json:"name,omitempty"`
}

type Golang struct {
	Runtime     string `json:"runtime,omitempty"`
	DynamicLink *bool  `json:"dynamicLink,omitempty"`
}

type Java struct {
	Runtime string `json:"runtime,omitempty"`
}

type Node struct {
	Runtime string `json:"runtime,omitempty"`
}

type Logging struct {
	Level  string `json:"level,omitempty"`
	Shim   string `json:"shim,omitempty"`
	Format string `json:"format,omitempty"`
}

type VM struct {
	Endpoint string   `json:"endpoint,omitempty"`
	Docker   VMDocker `json:"docker,omitempty"`
}

type VMDocker struct {
	TLS          DockerTLS            `json:"tls,omitempty"`
	AttachStdout *bool                `json:"attachStdout,omitempty"`
	HostConfig   container.HostConfig `json:"hostConfig,omitempty"`
}

type DockerTLS struct {
	Enabled *bool `json:"enabled,omitempty"`
	CA      File  `json:"ca,omitempty"`
	Cert    File  `json:"cert,omitempty"`
	Key     File  `json:"key,omitempty"`
}

type Ledger struct {
	State   LedgerState   `json:"state,omitempty"`
	History LedgerHistory `json:"history,omitempty"`
}

type LedgerState struct {
	StateDatabase   string        `json:"stateDatabase,omitempty"`
	TotalQueryLimit int           `json:"totalQueryLimit,omitempty"`
	CouchdbConfig   CouchdbConfig `json:"couchDBConfig,omitempty"`
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
}

type LedgerHistory struct {
	EnableHistoryDatabase *bool `json:"enableHistoryDatabase,omitempty"`
}
