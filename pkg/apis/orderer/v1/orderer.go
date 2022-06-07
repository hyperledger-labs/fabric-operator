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
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
)

type Orderer struct {
	General    General     `json:"general,omitempty"`
	FileLedger FileLedger  `json:"fileLedger,omitempty"`
	Debug      Debug       `json:"debug,omitempty"`
	Consensus  interface{} `json:"consensus,omitempty"`
	Operations Operations  `json:"operations,omitempty"`
	Metrics    Metrics     `json:"metrics,omitempty"`
}

// General contains config which should be common among all orderer types.
type General struct {
	LedgerType        string             `json:"ledgerType,omitempty"`
	ListenAddress     string             `json:"listenAddress,omitempty"`
	ListenPort        uint16             `json:"listenPort,omitempty"`
	TLS               TLS                `json:"tls,omitempty"`
	Cluster           Cluster            `json:"cluster,omitempty"`
	Keepalive         Keepalive          `json:"keepalive,omitempty"`
	ConnectionTimeout commonapi.Duration `json:"connectionTimeout,omitempty"`
	GenesisMethod     string             `json:"genesisMethod,omitempty"`
	GenesisFile       string             `json:"genesisFile,omitempty"` // For compatibility only, will be replaced by BootstrapFile
	BootstrapFile     string             `json:"bootstrapFile,omitempty"`
	Profile           Profile            `json:"profile,omitempty"`
	LocalMSPDir       string             `json:"localMspDir,omitempty"`
	LocalMSPID        string             `json:"localMspId,omitempty"`
	BCCSP             *commonapi.BCCSP   `json:"BCCSP,omitempty"`
	Authentication    Authentication     `json:"authentication,omitempty"`
}

type Cluster struct {
	ListenAddress                        string             `json:"listenAddress,omitempty"`
	ListenPort                           uint16             `json:"listenPort,omitempty"`
	ServerCertificate                    string             `json:"serverCertificate,omitempty"`
	ServerPrivateKey                     string             `json:"serverPrivateKey,omitempty"`
	ClientCertificate                    string             `json:"clientCertificate,omitempty"`
	ClientPrivateKey                     string             `json:"clientPrivateKey,omitempty"`
	RootCAs                              []string           `json:"rootCas,omitempty"`
	DialTimeout                          commonapi.Duration `json:"dialTimeout,omitempty"`
	RPCTimeout                           commonapi.Duration `json:"rpcTimeout,omitempty"`
	ReplicationBufferSize                int                `json:"replicationBufferSize,omitempty"`
	ReplicationPullTimeout               commonapi.Duration `json:"replicationPullTimeout,omitempty"`
	ReplicationRetryTimeout              commonapi.Duration `json:"replicationRetryTimeout,omitempty"`
	ReplicationBackgroundRefreshInterval commonapi.Duration `json:"replicationBackgroundRefreshInterval,omitempty"`
	ReplicationMaxRetries                int                `json:"replicationMaxRetries,omitempty"`
	SendBufferSize                       int                `json:"sendBufferSize,omitempty"`
	CertExpirationWarningThreshold       commonapi.Duration `json:"certExpirationWarningThreshold,omitempty"`
	TLSHandshakeTimeShift                commonapi.Duration `json:"tlsHandshakeTimeShift,omitempty"`
}

// Keepalive contains configuration for gRPC servers.
type Keepalive struct {
	ServerMinInterval commonapi.Duration `json:"serverMinInterval,omitempty"`
	ServerInterval    commonapi.Duration `json:"serverInterval,omitempty"`
	ServerTimeout     commonapi.Duration `json:"serverTimeout,omitempty"`
}

// TLS contains configuration for TLS connections.
type TLS struct {
	Enabled            *bool    `json:"enabled,omitempty"`
	PrivateKey         string   `json:"privateKey,omitempty"`
	Certificate        string   `json:"certificate,omitempty"`
	RootCAs            []string `json:"rootCas,omitempty"`
	ClientAuthRequired *bool    `json:"clientAuthRequired,omitempty"`
	ClientRootCAs      []string `json:"clientRootCas,omitempty"`
}

// SASLPlain contains configuration for SASL/PLAIN authentication
type SASLPlain struct {
	Enabled  *bool  `json:"enabled,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

// Authentication contains configuration parameters related to authenticating
// client messages.
type Authentication struct {
	TimeWindow         commonapi.Duration `json:"timeWindow,omitempty"`
	NoExpirationChecks *bool              `json:"noExpirationChecks,omitempty"`
}

// Profile contains configuration for Go pprof profiling.
type Profile struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Address string `json:"address,omitempty"`
}

// FileLedger contains configuration for the file-based ledger.
type FileLedger struct {
	Location string `json:"location,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
}

// Retry contains configuration related to retries and timeouts when the
// connection to the Kafka cluster cannot be established, or when Metadata
// requests needs to be repeated (because the cluster is in the middle of a
// leader election).
type Retry struct {
	ShortInterval   commonapi.Duration `json:"shortInterval,omitempty"`
	ShortTotal      commonapi.Duration `json:"shortTotal,omitempty"`
	LongInterval    commonapi.Duration `json:"longInterval,omitempty"`
	LongTotal       commonapi.Duration `json:"longTotal,omitempty"`
	NetworkTimeouts NetworkTimeouts    `json:"networkTimeouts,omitempty"`
	Metadata        Metadata           `json:"metadata,omitempty"`
	Producer        Producer           `json:"producer,omitempty"`
	Consumer        Consumer           `json:"consumer,omitempty"`
}

// NetworkTimeouts contains the socket timeouts for network requests to the
// Kafka cluster.
type NetworkTimeouts struct {
	DialTimeout  commonapi.Duration `json:"dialTimeout,omitempty"`
	ReadTimeout  commonapi.Duration `json:"readTimeout,omitempty"`
	WriteTimeout commonapi.Duration `json:"writeTimeout,omitempty"`
}

// Metadata contains configuration for the metadata requests to the Kafka
// cluster.
type Metadata struct {
	RetryMax     int                `json:"retryMax,omitempty"`
	RetryBackoff commonapi.Duration `json:"retryBackoff,omitempty"`
}

// Producer contains configuration for the producer's retries when failing to
// post a message to a Kafka partition.
type Producer struct {
	RetryMax     int                `json:"retryMax,omitempty"`
	RetryBackoff commonapi.Duration `json:"retryBackoff,omitempty"`
}

// Consumer contains configuration for the consumer's retries when failing to
// read from a Kafa partition.
type Consumer struct {
	RetryBackoff commonapi.Duration `json:"retryBackoff,omitempty"`
}

// Topic contains the settings to use when creating Kafka topics
type Topic struct {
	ReplicationFactor int16 `json:"replicationFactor,omitempty"`
}

// Debug contains configuration for the orderer's debug parameters.
type Debug struct {
	BroadcastTraceDir string `json:"broadcastTraceDir,omitempty"`
	DeliverTraceDir   string `json:"deliverTraceDir,omitempty"`
}

// Operations configures the operations endpont for the orderer.
type Operations struct {
	ListenAddress string `json:"listenAddress,omitempty"`
	TLS           TLS    `json:"tls,omitempty"`
}

// Operations confiures the metrics provider for the orderer.
type Metrics struct {
	Provider string `json:"provider,omitempty"`
	Statsd   Statsd `json:"statsd,omitempty"`
}

// Statsd provides the configuration required to emit statsd metrics from the orderer.
type Statsd struct {
	Network       string             `json:"network,omitempty"`
	Address       string             `json:"address,omitempty"`
	WriteInterval commonapi.Duration `json:"writeInterval,omitempty"`
	Prefix        string             `json:"prefix,omitempty"`
}
