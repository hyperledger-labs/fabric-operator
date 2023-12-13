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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
// IBPPeerSpec defines the desired state of IBPPeer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type IBPPeerSpec struct {
	// License should be accepted by the user to be able to setup Peer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	License License `json:"license"`

	/* generic configs - images/resources/storage/servicetype/version/replicas */

	// Images (Optional) lists the images to be used for peer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Images *PeerImages `json:"images,omitempty"`

	// RegistryURL is registry url used to pull images
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	RegistryURL string `json:"registryURL,omitempty"`

	// ImagePullSecrets (Optional) is the list of ImagePullSecrets to be used for peer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`

	// Replicas (Optional - default 1) is the number of peer replicas to be setup
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources (Optional) is the amount of resources to be provided to peer deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Resources *PeerResources `json:"resources,omitempty"`

	// Service (Optional) is the override object for peer's service
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Service *Service `json:"service,omitempty"`

	// Storage (Optional - uses default storageclass if not provided) is the override object for peer's PVC config
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Storage *PeerStorages `json:"storage,omitempty"`

	/* peer specific configs */
	// MSPID is the msp id of the peer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	MSPID string `json:"mspID,omitempty"`

	// StateDb (Optional) is the statedb used for peer, can be couchdb or leveldb
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	StateDb string `json:"stateDb,omitempty"`

	// ConfigOverride (Optional) is the object to provide overrides to core yaml config
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	ConfigOverride *runtime.RawExtension `json:"configoverride,omitempty"`

	// HSM (Optional) is DEPRECATED
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HSM *HSM `json:"hsm,omitempty"`

	// DisableNodeOU (Optional) is used to switch nodeou on and off
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DisableNodeOU *bool `json:"disablenodeou,omitempty"`

	// CustomNames (Optional) is to use pre-configured resources for peer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CustomNames PeerCustomNames `json:"customNames,omitempty"`

	// FabricVersion (Optional) is fabric version for the peer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	FabricVersion string `json:"version"`

	// NumSecondsWarningPeriod (Optional - default 30 days) is used to define certificate expiry warning period.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NumSecondsWarningPeriod int64 `json:"numSecondsWarningPeriod,omitempty"`

	/* msp data can be passed in secret on in spec */
	// MSPSecret (Optional) is secret used to store msp crypto
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	MSPSecret string `json:"mspSecret,omitempty"`

	// Secret is object for msp crypto
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Secret *SecretSpec `json:"secret,omitempty"`

	/* proxy ip passed if not OCP, domain for OCP */
	// Domain is the sub-domain used for peer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Domain string `json:"domain,omitempty"`

	// Ingress (Optional) is ingress object for ingress overrides
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Ingress Ingress `json:"ingress,omitempty"`

	// PeerExternalEndpoint (Optional) is used to override peer external endpoint
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PeerExternalEndpoint string `json:"peerExternalEndpoint,omitempty"`

	/* cluster related configs */
	// Arch (Optional) is the architecture of the nodes where peer should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Arch []string `json:"arch,omitempty"`

	// Region (Optional) is the region of the nodes where the peer should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Region string `json:"region,omitempty"`

	// Zone (Optional) is the zone of the nodes where the peer should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Zone string `json:"zone,omitempty"`

	/* advanced configs */
	// DindArgs (Optional) is used to override args passed to dind container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DindArgs []string `json:"dindArgs,omitempty"`

	// Action (Optional) is object for peer actions
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Action PeerAction `json:"action,omitempty"`

	// ChaincodeBuilderConfig (Optional) is a k/v map providing a scope for template
	// substitutions defined in chaincode-as-a-service package metadata files.
	// The map will be serialized as JSON and set in the peer deployment
	// CHAINCODE_AS_A_SERVICE_BUILDER_CONFIG env variable.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ChaincodeBuilderConfig ChaincodeBuilderConfig `json:"chaincodeBuilderConfig,omitempty"`
}

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
// IBPPeerStatus defines the observed state of IBPPeer
// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
type IBPPeerStatus struct {
	CRStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:storageversion
// +k8s:deepcopy-gen=true
// +kubebuilder:subresource:status
// IBPPeer is the Schema for the ibppeers API.
// Warning: Peer deployment using this tile is not supported. Please use the IBP Console to deploy a Peer.
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="IBP Peer"
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
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`clusterversions,v1,""`
type IBPPeer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IBPPeerSpec `json:"spec"`
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Status IBPPeerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// IBPPeerList contains a list of IBPPeer
type IBPPeerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IBPPeer `json:"items"`
}

// +k8s:deepcopy-gen=true
// PeerResources is the overrides to the resources of the peer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerResources struct {
	// Init (Optional) is the resources provided to the init container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Init *corev1.ResourceRequirements `json:"init,omitempty"`

	/// Peer (Optional) is the resources provided to the peer container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Peer *corev1.ResourceRequirements `json:"peer,omitempty"`

	// GRPCProxy (Optional) is the resources provided to the proxy container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	GRPCProxy *corev1.ResourceRequirements `json:"proxy,omitempty"`

	// DinD (Optional) is the resources provided to the dind container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DinD *corev1.ResourceRequirements `json:"dind,omitempty"`

	// CouchDB (Optional) is the resources provided to the couchdb container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CouchDB *corev1.ResourceRequirements `json:"couchdb,omitempty"`

	// CCLauncher (Optional) is the resources provided to the cclauncher container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CCLauncher *corev1.ResourceRequirements `json:"chaincodelauncher,omitempty"`

	// Enroller (Optional) is the resources provided to the enroller container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Enroller *corev1.ResourceRequirements `json:"enroller,omitempty"`

	// HSMDaemon (Optional) is the resources provided to the HSM Daemon container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HSMDaemon *corev1.ResourceRequirements `json:"hsmdaemon,omitempty"`
}

// +k8s:deepcopy-gen=true
// PeerStorages is the overrides to the storage of the peer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerStorages struct {
	// StateDB (Optional) is the configuration of the storage of the statedb
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	StateDB *StorageSpec `json:"statedb,omitempty"`

	// Peer (Optional) is the configuration of the storage of the peer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Peer *StorageSpec `json:"peer,omitempty"`
}

// PeerImages is the list of images to be used in peer deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerImages struct {
	// PeerInitImage is the name of the peer init image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PeerInitImage string `json:"peerInitImage,omitempty"`

	// PeerInitTag is the tag of the peer init image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PeerInitTag string `json:"peerInitTag,omitempty"`

	// PeerImage is the name of the peer image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PeerImage string `json:"peerImage,omitempty"`

	// PeerTag is the tag of the peer image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PeerTag string `json:"peerTag,omitempty"`

	// DindImage is the name of the dind image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DindImage string `json:"dindImage,omitempty"`

	// DindTag is the tag of the dind image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DindTag string `json:"dindTag,omitempty"`

	// GRPCWebImage is the name of the grpc web proxy image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	GRPCWebImage string `json:"grpcwebImage,omitempty"`

	// GRPCWebTag is the tag of the grpc web proxy image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	GRPCWebTag string `json:"grpcwebTag,omitempty"`

	// CouchDBImage is the name of the couchdb image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CouchDBImage string `json:"couchdbImage,omitempty"`

	// CouchDBTag is the tag of the couchdb image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CouchDBTag string `json:"couchdbTag,omitempty"`

	// CCLauncherImage is the name of the chaincode launcher image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CCLauncherImage string `json:"chaincodeLauncherImage,omitempty"`

	// CCLauncherTag is the tag of the chaincode launcher image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CCLauncherTag string `json:"chaincodeLauncherTag,omitempty"`

	// FileTransferImage is the name of the file transfer image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	FileTransferImage string `json:"fileTransferImage,omitempty"`

	// FileTransferTag is the tag of the file transfer image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	FileTransferTag string `json:"fileTransferTag,omitempty"`

	// BuilderImage is the name of the builder image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	BuilderImage string `json:"builderImage,omitempty"`

	// BuilderTag is the tag of the builder image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	BuilderTag string `json:"builderTag,omitempty"`

	// GoEnvImage is the name of the goenv image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	GoEnvImage string `json:"goEnvImage,omitempty"`

	// GoEnvTag is the tag of the goenv image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	GoEnvTag string `json:"goEnvTag,omitempty"`

	// JavaEnvImage is the name of the javaenv image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	JavaEnvImage string `json:"javaEnvImage,omitempty"`

	// JavaEnvTag is the tag of the javaenv image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	JavaEnvTag string `json:"javaEnvTag,omitempty"`

	// NodeEnvImage is the name of the nodeenv image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NodeEnvImage string `json:"nodeEnvImage,omitempty"`

	// NodeEnvTag is the tag of the nodeenv image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NodeEnvTag string `json:"nodeEnvTag,omitempty"`

	// HSMImage is the name of the hsm image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HSMImage string `json:"hsmImage,omitempty"`

	// HSMTag is the tag of the hsm image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HSMTag string `json:"hsmTag,omitempty"`

	// EnrollerImage is the name of the init image for crypto generation
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	EnrollerImage string `json:"enrollerImage,omitempty"`

	// EnrollerTag is the tag of the init image for crypto generation
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	EnrollerTag string `json:"enrollerTag,omitempty"`
}

// +k8s:deepcopy-gen=true
// PeerConnectionProfile provides necessary information to connect to the peer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerConnectionProfile struct {
	// Endpoints is list of endpoints to communicate with the peer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Endpoints PeerEndpoints `json:"endpoints"`

	// TLS is object with tls crypto material for peer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLS *MSP `json:"tls"`

	// Component is object with ecert crypto material for peer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Component *MSP `json:"component"`
}

// PeerEndpoints is the list of endpoints to communicate with the peer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerEndpoints struct {
	// API is the endpoint to communicate with peer's API
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	API string `json:"api"`

	// Operations is the endpoint to communicate with peer's Operations API
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Operations string `json:"operations"`

	// Grpcweb is the endpoint to communicate with peers's grpcweb proxy API
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Grpcweb string `json:"grpcweb"`
}

// +k8s:deepcopy-gen=true
// PeerCustomNames is the list of preconfigured objects to be used for peer's deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerCustomNames struct {
	// PVC is the list of PVC Names to be used for peer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PVC PeerPVCNames `json:"pvc,omitempty"`
}

// PeerPVCNames is the list of PVC Names to be used for peer's deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerPVCNames struct {
	// Peer is the pvc to be used as peer's storage
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Peer string `json:"peer,omitempty"`

	// StateDB is the pvc to be used as statedb's storage
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	StateDB string `json:"statedb,omitempty"`
}

// Action contains actions that can be performed on peer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerAction struct {
	// Restart action is used to restart peer deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Restart bool `json:"restart,omitempty"`

	// Reenroll contains actions for triggering crypto reenroll
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Reenroll PeerReenrollAction `json:"reenroll,omitempty"`

	// Enroll contains actions for triggering crypto enroll
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Enroll PeerEnrollAction `json:"enroll,omitempty"`

	// UpgradeDBs action is used to trigger peer node upgrade-dbs command
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	UpgradeDBs bool `json:"upgradedbs,omitempty"`
}

// PeerReenrollAction contains actions for reenrolling crypto
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerReenrollAction struct {
	// Ecert is used to trigger reenroll for ecert
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Ecert bool `json:"ecert,omitempty"`

	// EcertNewKey is used to trigger reenroll for ecert and also generating
	// a new private key
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	EcertNewKey bool `json:"ecertNewKey,omitempty"`

	// TLSCert is used to trigger reenroll for tlscert
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLSCert bool `json:"tlscert,omitempty"`

	// TLSCertNewKey is used to trigger reenroll for tlscert and also generating
	// a new private key
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLSCertNewKey bool `json:"tlscertNewKey,omitempty"`
}

// PeerReenrollAction contains actions for enrolling crypto
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type PeerEnrollAction struct {
	// Ecert is used to trigger enroll for ecert
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Ecert bool `json:"ecert,omitempty"`

	// TLSCert is used to trigger enroll for tlscert
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLSCert bool `json:"tlscert,omitempty"`
}

// ChaincodeBuilderConfig defines a k/v mapping scope for template substitutions
// referenced within a chaincode package archive.  The mapping is serialized as
// JSON and appended to the peer env as CHAINCODE_AS_A_SERVICE_BUILDER_CONFIG.
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type ChaincodeBuilderConfig map[string]string
