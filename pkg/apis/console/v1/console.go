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

type DBCustomNames struct {
	Components string `json:"DB_COMPONENTS"`
	Sessions   string `json:"DB_SESSIONS"`
	System     string `json:"DB_SYSTEM"`
}

type FabricCapabilites struct {
	Application []string `json:"application"`
	Channel     []string `json:"channel"`
	Orderer     []string `json:"orderer"`
}

type IBMID struct {
	URL          string `json:"url,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

// IBPConsoleStructureData provides the clsuter info the console
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
type IBPConsoleClusterData struct {
	// Zones provides the zones available
	Zones []string `json:"zones,omitempty"`

	// Type provides the type of cluster
	Type string `json:"type,omitempty"`

	Namespace string `json:"namespace,omitempty"`
}

// +k8s:deepcopy-gen=true
type InfraImportOptions struct {
	Platform          string   `json:"platform,omitempty"`
	SupportedCAs      []string `json:"supported_cas,omitempty"`
	SupportedOrderers []string `json:"supported_orderers,omitempty"`
	SupportedPeers    []string `json:"supported_peers,omitempty"`
}

// +k8s:deepcopy-gen=true
type FeatureFlags struct {
	ImportOnlyEnabled       *bool               `json:"import_only_enabled,omitempty"`
	ReadOnlyEnabled         *bool               `json:"read_only_enabled,omitempty"`
	CreateChannelEnabled    bool                `json:"create_channel_enabled,omitempty"`
	RemotePeerConfigEnabled bool                `json:"remote_peer_config_enabled,omitempty"`
	SaasEnabled             bool                `json:"saas_enabled,omitempty"`
	TemplatesEnabled        bool                `json:"templates_enabled,omitempty"`
	CapabilitiesEnabled     bool                `json:"capabilities_enabled,omitempty"`
	HighAvailability        bool                `json:"high_availability,omitempty"`
	EnableNodeOU            bool                `json:"enable_ou_identifier,omitempty"`
	HSMEnabled              bool                `json:"hsm_enabled,omitempty"`
	ScaleRaftNodesEnabled   bool                `json:"scale_raft_nodes_enabled,omitempty"`
	InfraImportOptions      *InfraImportOptions `json:"infra_import_options,omitempty"`
	Lifecycle20Enabled      bool                `json:"lifecycle2_0_enabled,omitempty"`
	Patch14to20Enabled      bool                `json:"patch_1_4to2_x_enabled,omitempty"`
	DevMode                 bool                `json:"dev_mode,omitempty"`
	MustgatherEnabled       bool                `json:"mustgather_enabled,omitempty"`
}

// Added here to avoid the Circular dependency
type CRN struct {
	Version      string `json:"version,omitempty"`
	CName        string `json:"c_name,omitempty"`
	CType        string `json:"c_type,omitempty"`
	Servicename  string `json:"service_name,omitempty"`
	Location     string `json:"location,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
	InstanceID   string `json:"instance_id,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
}

type ConsoleSettingsConfig struct {
	Version              string                 `json:"version"`
	Email                string                 `json:"initial_admin"`
	AuthScheme           string                 `json:"auth_scheme,omitempty"`
	AllowDefaultPassword bool                   `json:"allow_default_password"`
	Configtxlator        string                 `json:"configtxlator"`
	DeployerURL          string                 `json:"deployer_url"`
	DeployerTimeout      int32                  `json:"deployer_timeout"`
	HSM                  string                 `json:"hsm"`
	SegmentWriteKey      string                 `json:"segment_write_key"`
	DBCustomNames        DBCustomNames          `json:"db_custom_names"`
	EnforceBackendSSL    bool                   `json:"enforce_backend_ssl"`
	SystemChannelID      string                 `json:"system_channel_id"`
	DynamicTLS           bool                   `json:"dynamic_tls"`
	DynamicConfig        bool                   `json:"dynamic_config"`
	Zone                 string                 `json:"zone"`
	Infrastructure       string                 `json:"infrastructure"`
	FabricCapabilites    FabricCapabilites      `json:"fabric_capabilities"`
	ClusterData          *IBPConsoleClusterData `json:"cluster_data"`
	ProxyTLSReqs         string                 `json:"proxy_tls_fabric_reqs"`
	ProxyTLSUrl          string                 `json:"proxy_tls_ws_url"`
	Featureflags         *FeatureFlags          `json:"feature_flags"`
	IBMID                *IBMID                 `json:"ibmid,omitempty"`
	IAMApiKey            string                 `json:"iam_api_key,omitempty"`
	CRN                  *CRN                   `json:"crn,omitempty"`
	CRNString            string                 `json:"crn_string,omitempty"`
	ActivityTrackerPath  string                 `json:"activity_tracker_path,omitempty"`
	TrustProxy           string                 `json:"trust_proxy,omitempty"`
}
