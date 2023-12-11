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
	consolev1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/console/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
// IBPConsoleSpec defines the desired state of IBPConsole
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type IBPConsoleSpec struct {
	// License should be accepted by the user to be able to setup console
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	License License `json:"license"`

	// Images (Optional) lists the images to be used for console's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Images *ConsoleImages `json:"images,omitempty"`

	// ImagePullSecrets (Optional) is the list of ImagePullSecrets to be used for console's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`

	// Replicas (Optional - default 1) is the number of console replicas to be setup
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources (Optional) is the amount of resources to be provided to console deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Resources *ConsoleResources `json:"resources,omitempty"`

	// Service (Optional) is the override object for console's service
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Service *Service `json:"service,omitempty"`

	// ServiceAccountName defines serviceaccount used for console deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Storage (Optional - uses default storageclass if not provided) is the override object for CA's PVC config
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Storage *ConsoleStorage `json:"storage,omitempty"`

	// NetworkInfo is object for network overrides
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NetworkInfo *NetworkInfo `json:"networkinfo,omitempty"`

	// Ingress (Optional) is ingress object for ingress overrides
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Ingress Ingress `json:"ingress,omitempty"`

	/* console settings */
	// AuthScheme is auth scheme for console access
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	AuthScheme string `json:"authScheme,omitempty"`

	// AllowDefaultPassword, if true, will bypass the password reset flow
	// on the first connection to the console GUI.  By default (false), all
	// consoles require a password reset at the first login.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	AllowDefaultPassword bool `json:"allowDefaultPassword,omitempty"`

	// Components is database name used for components
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Components string `json:"components,omitempty"`

	// ClusterData is object cluster data information
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ClusterData *consolev1.IBPConsoleClusterData `json:"clusterdata,omitempty"`

	// ConfigtxlatorURL is url for configtxlator server
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConfigtxlatorURL string `json:"configtxlator,omitempty"`

	// ConnectionString is connection url for backend database
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConnectionString string `json:"connectionString,omitempty"`

	// DeployerTimeout is timeout value for deployer calls
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DeployerTimeout int32 `json:"deployerTimeout,omitempty"`

	// DeployerURL is url for deployer server
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DeployerURL string `json:"deployerUrl,omitempty"`

	// Email is the email used for initial access
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Email string `json:"email,omitempty"`

	// FeatureFlags is object for feature flag settings
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	FeatureFlags *consolev1.FeatureFlags `json:"featureflags,omitempty"`

	IAMApiKey       string           `json:"iamApiKey,omitempty"`
	SegmentWriteKey string           `json:"segmentWriteKey,omitempty"`
	IBMID           *consolev1.IBMID `json:"ibmid,omitempty"`
	Proxying        *bool            `json:"proxying,omitempty"`

	// Password is initial password to access console
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Password string `json:"password,omitempty"`

	// PasswordSecretName is secretname where password is stored
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PasswordSecretName string `json:"passwordSecretName,omitempty"`

	// Sessions is sessions database name to use
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Sessions string `json:"sessions,omitempty"`

	// System is system database name to use
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	System string `json:"system,omitempty"`

	// SystemChannel is default systemchannel name
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	SystemChannel string `json:"systemChannel,omitempty"`

	// TLSSecretName is secret name to load custom tls certs
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLSSecretName string `json:"tlsSecretName,omitempty"`

	CRN                  *CRN      `json:"crn,omitempty"`
	Kubeconfig           *[]byte   `json:"kubeconfig,omitempty"`
	KubeconfigSecretName string    `json:"kubeconfigsecretname,omitempty"`
	Versions             *Versions `json:"versions,omitempty"`
	KubeconfigNamespace  string    `json:"kubeconfignamespace,omitempty"`

	// RegistryURL is registry url used to pull images
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	RegistryURL string `json:"registryURL,omitempty"`

	// Deployer is object for deployer configs
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Deployer *Deployer `json:"deployer,omitempty"`

	// Arch (Optional) is the architecture of the nodes where console should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Arch []string `json:"arch,omitempty"`

	// Region (Optional) is the region of the nodes where the console should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Region string `json:"region,omitempty"`

	// Zone (Optional) is the zone of the nodes where the console should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Zone string `json:"zone,omitempty"`

	// ConfigOverride (Optional) is the object to provide overrides
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConfigOverride *ConsoleOverrides `json:"configoverride,omitempty"`

	// Action (Optional) is action object for trigerring actions
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Action ConsoleAction `json:"action,omitempty"`

	// Version (Optional) is version for the console
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Version string `json:"version"`

	// UseTags (Optional) is a flag to switch between image digests and tags
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	UseTags *bool `json:"usetags"`
}

// +k8s:deepcopy-gen=true
// ConsoleOverrides is the overrides to console configuration
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type ConsoleOverrides struct {
	// Console is the overrides to console configuration
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Console *runtime.RawExtension `json:"console,omitempty"`

	// Deployer is the overrides to deployer configuration
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Deployer *runtime.RawExtension `json:"deployer,omitempty"`

	// MaxNameLength (Optional) is the maximum length of the name that the console can have
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	MaxNameLength *int `json:"maxnamelength,omitempty"`
}

// +k8s:deepcopy-gen=true
type ConsoleOverridesConsole struct {
	HostURL                    string `json:"hostURL,omitempty"`
	ActivityTrackerConsolePath string `json:"activityTrackerConsolePath,omitempty"`
	ActivityTrackerHostPath    string `json:"activityTrackerHostPath,omitempty"`
	HSM                        string `json:"hsm"`
}

// +k8s:deepcopy-gen=true
type ConsoleOverridesDeployer struct {
	Timeouts *DeployerTimeouts `json:"timeouts,omitempty"`
}

// +k8s:deepcopy-gen=true
type Versions struct {
	CA      map[string]VersionCA      `json:"ca"`
	Peer    map[string]VersionPeer    `json:"peer"`
	Orderer map[string]VersionOrderer `json:"orderer"`
}

type VersionCA struct {
	Default bool     `json:"default"`
	Version string   `json:"version"`
	Image   CAImages `json:"image,omitempty"`
}

type VersionOrderer struct {
	Default bool          `json:"default"`
	Version string        `json:"version"`
	Image   OrdererImages `json:"image,omitempty"`
}
type VersionPeer struct {
	Default bool       `json:"default"`
	Version string     `json:"version"`
	Image   PeerImages `json:"image,omitempty"`
}

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
// IBPConsoleStatus defines the observed state of IBP Console
// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
type IBPConsoleStatus struct {
	CRStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen=true
// The Console is used to deploy and manage the CA, peer, ordering nodes.
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="IBP Console"
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Deployments,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Ingresses,v1beta1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`PersistentVolumeClaim,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Role,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`RoleBinding,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Route,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Services,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`ServiceAccounts,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`ConfigMaps,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Secrets,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Pods,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Replicasets,v1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`IBPCA,v1beta1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`IBPPeer,v1beta1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`IBPOrderer,v1beta1,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`IBPConsole,v1beta1,""`
type IBPConsole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Spec IBPConsoleSpec `json:"spec,omitempty"`

	// Status is the observed state of IBPConsole
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Status IBPConsoleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// IBPConsoleList contains a list of IBP Console
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type IBPConsoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IBPConsole `json:"items"`
}

// +k8s:deepcopy-gen=true
// ConsoleResources is the overrides to the resources of the Console
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type ConsoleResources struct {
	// Init is the resources provided to the init container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Init *corev1.ResourceRequirements `json:"init,omitempty"`

	// CouchDB is the resources provided to the couchdb container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CouchDB *corev1.ResourceRequirements `json:"couchdb,omitempty"`

	// Console is the resources provided to the console container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Console *corev1.ResourceRequirements `json:"console,omitempty"`

	// Deployer is the resources provided to the deployer container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Deployer *corev1.ResourceRequirements `json:"deployer,omitempty"`

	// Configtxlator is the resources provided to the configtxlator container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Configtxlator *corev1.ResourceRequirements `json:"configtxlator,omitempty"`
}

// +k8s:deepcopy-gen=true
// ConsoleStorage is the overrides to the storage of the console
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type ConsoleStorage struct {
	// Console is the configuration of the storage of the console
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Console *StorageSpec `json:"console,omitempty"`
}

// +k8s:deepcopy-gen=true
// ConsoleImages is the list of images to be used in console deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type ConsoleImages struct {
	// ConsoleInitImage is the name of the console init image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConsoleInitImage string `json:"consoleInitImage,omitempty"`

	// ConsoleInitTag is the tag of the console init image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConsoleInitTag string `json:"consoleInitTag,omitempty"`

	// ConsoleImage is the name of the console image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConsoleImage string `json:"consoleImage,omitempty"`

	// ConsoleTag is the tag of the console image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConsoleTag string `json:"consoleTag,omitempty"`

	// ConfigtxlatorImage is the name of the configtxlator image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConfigtxlatorImage string `json:"configtxlatorImage,omitempty"`

	// ConfigtxlatorTag is the tag of the configtxlator image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConfigtxlatorTag string `json:"configtxlatorTag,omitempty"`

	// DeployerImage is the name of the deployer image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DeployerImage string `json:"deployerImage,omitempty"`

	// DeployerTag is the tag of the deployer image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DeployerTag string `json:"deployerTag,omitempty"`

	// CouchDBImage is the name of the couchdb image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CouchDBImage string `json:"couchdbImage,omitempty"`

	// CouchDBTag is the tag of the couchdb image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CouchDBTag string `json:"couchdbTag,omitempty"`

	// MustgatherImage is the name of the mustgather image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	MustgatherImage string `json:"mustgatherImage,omitempty"`

	// MustgatherTag is the tag of the mustgatherTag image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	MustgatherTag string `json:"mustgatherTag,omitempty"`
}

type Deployer struct {
	Domain           string `json:"domain,omitempty"`
	ConnectionString string `json:"connectionstring,omitempty"`
	ComponentsDB     string `json:"components_db,omitempty"`
	CreateDB         bool   `json:"create_db,omitempty"`
}

type DeployerTimeouts struct {
	Deployment int `json:"componentDeploy"`
	APIServer  int `json:"apiServer"`
}

// ConsoleAction contains actions that can be performed on console
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type ConsoleAction struct {
	Restart bool `json:"restart,omitempty"`
}
