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
// IBPOrdererSpec defines the desired state of IBPOrderer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type IBPOrdererSpec struct {
	// License should be accepted by the user to be able to setup orderer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	License License `json:"license"`

	// Images (Optional) lists the images to be used for orderer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Images *OrdererImages `json:"images,omitempty"`

	// RegistryURL is registry url used to pull images
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	RegistryURL string `json:"registryURL,omitempty"`

	// ImagePullSecrets (Optional) is the list of ImagePullSecrets to be used for orderer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`

	// Replicas (Optional - default 1) is the number of orderer replicas to be setup
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources (Optional) is the amount of resources to be provided to orderer deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Resources *OrdererResources `json:"resources,omitempty"`

	// Service (Optional) is the override object for orderer's service
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Service *Service `json:"service,omitempty"`

	// Storage (Optional - uses default storageclass if not provided) is the override object for CA's PVC config
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Storage *OrdererStorages `json:"storage,omitempty"`

	// GenesisBlock (Optional) is genesis block to start the orderer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	GenesisBlock   string `json:"genesisBlock,omitempty"`
	GenesisProfile string `json:"genesisProfile,omitempty"`
	UseChannelLess *bool  `json:"useChannelLess,omitempty"`

	// MSPID is the msp id of the orderer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	MSPID string `json:"mspID,omitempty"`

	// OrdererType is type of orderer you want to start
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	OrdererType string `json:"ordererType,omitempty"`

	// OrgName is the organization name of the orderer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	OrgName string `json:"orgName,omitempty"`

	// SystemChannelName is the name of systemchannel
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	SystemChannelName string `json:"systemChannelName,omitempty"`

	// Secret is object for msp crypto
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Secret *SecretSpec `json:"secret,omitempty"`

	// ConfigOverride (Optional) is the object to provide overrides to core yaml config
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	ConfigOverride *runtime.RawExtension `json:"configoverride,omitempty"`

	// HSM (Optional) is DEPRECATED
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HSM *HSM `json:"hsm,omitempty"`

	// IsPrecreate (Optional) defines if orderer is in precreate state
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	IsPrecreate *bool `json:"isprecreate,omitempty"`

	// FabricVersion (Optional) is fabric version for the orderer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	FabricVersion string `json:"version"`

	// NumSecondsWarningPeriod (Optional - default 30 days) is used to define certificate expiry warning period.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NumSecondsWarningPeriod int64 `json:"numSecondsWarningPeriod,omitempty"`

	// ClusterSize (Optional) number of orderers if a cluster
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ClusterSize int `json:"clusterSize,omitempty"`

	// ClusterLocation (Optional) is array of cluster location settings for cluster
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ClusterLocation []IBPOrdererClusterLocation `json:"location,omitempty"`

	// ClusterConfigOverride (Optional) is array of config overrides for cluster
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +kubebuilder:pruning:PreserveUnknownFields
	ClusterConfigOverride []*runtime.RawExtension `json:"clusterconfigoverride,omitempty"`

	// ClusterSecret (Optional) is array of msp crypto for cluster
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ClusterSecret []*SecretSpec `json:"clustersecret,omitempty"`

	// NodeNumber (Optional) is the number of this node in cluster - used internally
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NodeNumber *int `json:"number,omitempty"`

	// Ingress (Optional) is ingress object for ingress overrides
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Ingress Ingress `json:"ingress,omitempty"`

	// Domain is the sub-domain used for orderer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Domain string `json:"domain,omitempty"`

	// Arch (Optional) is the architecture of the nodes where orderer should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Arch []string `json:"arch,omitempty"`

	// Zone (Optional) is the zone of the nodes where the orderer should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Zone string `json:"zone,omitempty"`

	// Region (Optional) is the region of the nodes where the orderer should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Region string `json:"region,omitempty"`

	// DisableNodeOU (Optional) is used to switch nodeou on and off
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	DisableNodeOU *bool `json:"disablenodeou,omitempty"`

	// CustomNames (Optional) is to use pre-configured resources for orderer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CustomNames OrdererCustomNames `json:"customNames,omitempty"`

	// Action (Optional) is object for orderer actions
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Action OrdererAction `json:"action,omitempty"`

	// ExternalAddress (Optional) is used internally
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ExternalAddress string `json:"externalAddress,omitempty"`
}

// IBPOrdererClusterLocation (Optional) is object of cluster location settings for cluster
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type IBPOrdererClusterLocation struct {
	// Zone (Optional) is the zone of the nodes where the orderer should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Zone string `json:"zone,omitempty"`

	// Region (Optional) is the region of the nodes where the orderer should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Region string `json:"region,omitempty"`
}

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=true
// IBPOrdererStatus defines the observed state of IBPOrderer
// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
type IBPOrdererStatus struct {
	CRStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen=true
// Ordering nodes create the blocks that form the ledger and send them to peers.
// Warning: Orderer deployment using this tile is not supported. Please use the IBP Console to deploy an orderer.
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="IBP Orderer"
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
type IBPOrderer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IBPOrdererSpec `json:"spec,omitempty"`
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Status IBPOrdererStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// IBPOrdererList contains a list of IBPOrderer
type IBPOrdererList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IBPOrderer `json:"items"`
}

// +k8s:deepcopy-gen=true
// OrdererResources is the overrides to the resources of the orderer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererResources struct {
	// Init (Optional) is the resources provided to the init container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Init *corev1.ResourceRequirements `json:"init,omitempty"`

	// Orderer (Optional) is the resources provided to the orderer container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Orderer *corev1.ResourceRequirements `json:"orderer,omitempty"`

	// GRPCProxy (Optional) is the resources provided to the proxy container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	GRPCProxy *corev1.ResourceRequirements `json:"proxy,omitempty"`

	// Enroller (Optional) is the resources provided to the enroller container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Enroller *corev1.ResourceRequirements `json:"enroller,omitempty"`

	// HSMDaemon (Optional) is the resources provided to the HSM Daemon container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HSMDaemon *corev1.ResourceRequirements `json:"hsmdaemon,omitempty"`
}

// OrdererImages is the list of images to be used in orderer deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererImages struct {
	// OrdererInitImage is the name of the orderer init image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	OrdererInitImage string `json:"ordererInitImage,omitempty"`

	// OrdererInitTag is the tag of the orderer init image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	OrdererInitTag string `json:"ordererInitTag,omitempty"`

	// OrdererImage is the name of the orderer image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	OrdererImage string `json:"ordererImage,omitempty"`

	// OrdererTag is the tag of the orderer image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	OrdererTag string `json:"ordererTag,omitempty"`

	// GRPCWebImage is the name of the grpc web proxy image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	GRPCWebImage string `json:"grpcwebImage,omitempty"`

	// GRPCWebTag is the tag of the grpc web proxy image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	GRPCWebTag string `json:"grpcwebTag,omitempty"`

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
// OrdererStorages is the overrides to the storage of the orderer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererStorages struct {
	// Orderer (Optional) is the configuration of the storage of the orderer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Orderer *StorageSpec `json:"orderer,omitempty"`
}

// +k8s:deepcopy-gen=true
// OrdererConnectionProfile provides necessary information to connect to the orderer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererConnectionProfile struct {
	// Endpoints is list of endpoints to communicate with the orderer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Endpoints OrdererEndpoints `json:"endpoints"`

	// TLS is object with tls crypto material for orderer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLS *MSP `json:"tls"`

	// Component is object with ecert crypto material for orderer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Component *MSP `json:"component"`
}

// OrdererEndpoints is the list of endpoints to communicate with the orderer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererEndpoints struct {
	// API is the endpoint to communicate with orderer's API
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	API string `json:"api"`

	// Operations is the endpoint to communicate with orderer's Operations API
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Operations string `json:"operations"`

	// Grpcweb is the endpoint to communicate with orderer's grpcweb proxy API
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Grpcweb string `json:"grpcweb"`

	// Admin is the endpoint to communicate with orderer's admin service API
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Admin string `json:"admin"`
}

// +k8s:deepcopy-gen=true
// OrdererCustomNames is the list of preconfigured objects to be used for orderer's deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererCustomNames struct {
	// PVC is the list of PVC Names to be used for orderer's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PVC OrdererPVCNames `json:"pvc,omitempty"`
}

// OrdererPVCNames is the list of PVC Names to be used for orderer's deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererPVCNames struct {
	// Orderer is the pvc to be used as orderer's storage
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Orderer string `json:"orderer,omitempty"`
}

// Action contains actions that can be performed on orderer
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererAction struct {
	// Restart action is used to restart orderer deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Restart bool `json:"restart,omitempty"`

	// Reenroll contains actions for triggering crypto reenroll
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Reenroll OrdererReenrollAction `json:"reenroll,omitempty"`

	// Enroll contains actions for triggering crypto enroll
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Enroll OrdererEnrollAction `json:"enroll,omitempty"`
}

// OrdererReenrollAction contains actions for reenrolling crypto
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererReenrollAction struct {
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

// OrdererEnrollAction contains actions for enrolling crypto
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type OrdererEnrollAction struct {
	// Ecert is used to trigger enroll for ecert
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Ecert bool `json:"ecert,omitempty"`

	// TLSCert is used to trigger enroll for tls certs
	TLSCert bool `json:"tlscert,omitempty"`
}
