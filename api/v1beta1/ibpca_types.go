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

// IBPCASpec defines the desired state of IBP CA
// +k8s:deepcopy-gen=true
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type IBPCASpec struct {
	// License should be accepted by the user to be able to setup CA
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	License License `json:"license"`

	/* generic configs - images/resources/storage/servicetype/version/replicas */

	// Images (Optional) lists the images to be used for CA's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Images *CAImages `json:"images,omitempty"`

	// RegistryURL is registry url used to pull images
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	RegistryURL string `json:"registryURL,omitempty"`

	// ImagePullSecrets (Optional) is the list of ImagePullSecrets to be used for CA's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`

	// Replicas (Optional - default 1) is the number of CA replicas to be setup
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources (Optional) is the amount of resources to be provided to CA deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Resources *CAResources `json:"resources,omitempty"`

	// Service (Optional) is the override object for CA's service
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Service *Service `json:"service,omitempty"`

	// Storage (Optional - uses default storageclass if not provided) is the override object for CA's PVC config
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Storage *CAStorages `json:"storage,omitempty"`

	/* CA specific configs */

	// ConfigOverride (Optional) is the object to provide overrides to CA & TLSCA config
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConfigOverride *ConfigOverride `json:"configoverride,omitempty"`

	// HSM (Optional) is DEPRECATED
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HSM *HSM `json:"hsm,omitempty"`

	// CustomNames (Optional) is to use pre-configured resources for CA's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CustomNames CACustomNames `json:"customNames,omitempty"`

	// NumSecondsWarningPeriod (Optional - default 30 days) is used to define certificate expiry warning period.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NumSecondsWarningPeriod int64 `json:"numSecondsWarningPeriod,omitempty"`

	// FabricVersion (Optional) set the fabric version you want to use.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	FabricVersion string `json:"version"`

	// Domain is the sub-domain used for CA's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Domain string `json:"domain,omitempty"`

	// Ingress (Optional) is ingress object for ingress overrides
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Ingress Ingress `json:"ingress,omitempty"`

	/* cluster related configs */

	// Arch (Optional) is the architecture of the nodes where CA should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Arch []string `json:"arch,omitempty"`

	// Region (Optional) is the region of the nodes where the CA should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Region string `json:"region,omitempty"`

	// Zone (Optional) is the zone of the nodes where the CA should be deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Zone string `json:"zone,omitempty"`

	// Action (Optional) is action object for trigerring actions
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Action CAAction `json:"action,omitempty"`
}

// +k8s:deepcopy-gen=true
// ConfigOverride is the overrides to CA's & TLSCA's configuration
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type ConfigOverride struct {
	// CA (Optional) is the overrides to CA's configuration
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	CA *runtime.RawExtension `json:"ca,omitempty"`
	// TLSCA (Optional) is the overrides to TLSCA's configuration
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +kubebuilder:validation:Type=object
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	TLSCA *runtime.RawExtension `json:"tlsca,omitempty"`
	// MaxNameLength (Optional) is the maximum length of the name that the CA can have
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	MaxNameLength *int `json:"maxnamelength,omitempty"`
}

// +k8s:deepcopy-gen=true
// IBPCAStatus defines the observed state of IBPCA
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type IBPCAStatus struct {
	// CRStatus is the status of the CA resource
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	CRStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// +kubebuilder:storageversion
// Certificate Authorities issue certificates for all the identities to transact on the network.
// Warning: CA deployment using this tile is not supported. Please use the IBP Console to deploy a CA.
// +kubebuilder:subresource:status
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="IBP CA"
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
type IBPCA struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Spec IBPCASpec `json:"spec,omitempty"`

	// Status is the observed state of IBPCA
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Status IBPCAStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:deepcopy-gen=true
// IBPCAList contains a list of IBPCA
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type IBPCAList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IBPCA `json:"items"`
}

// +k8s:deepcopy-gen=true
// CAResources is the overrides to the resources of the CA
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CAResources struct {
	// Init is the resources provided to the init container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Init *corev1.ResourceRequirements `json:"init,omitempty"`

	// CA is the resources provided to the CA container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CA *corev1.ResourceRequirements `json:"ca,omitempty"`

	// EnrollJJob is the resources provided to the enroll job container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	EnrollJob *corev1.ResourceRequirements `json:"enrollJob,omitempty"`

	// HSMDaemon is the resources provided to the HSM daemon container
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HSMDaemon *corev1.ResourceRequirements `json:"hsmDaemon,omitempty"`
}

// +k8s:deepcopy-gen=true
// CAStorages is the overrides to the storage of the CA
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CAStorages struct {
	// CA is the configuration of the storage of the CA
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CA *StorageSpec `json:"ca,omitempty"`
}

// +k8s:deepcopy-gen=true
// CAConnectionProfile is the object for connection profile
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CAConnectionProfile struct {
	// Endpoints is the endpoints to talk to CA
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Endpoints CAEndpoints `json:"endpoints"`

	// TLS is the object with CA servers TLS information
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLS *ConnectionProfileTLS `json:"tls"`

	// CA is the object with CA crypto in connection profile
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CA *MSP `json:"ca"`

	// TLSCA is the object with tls CA crypto in connection profile
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLSCA *MSP `json:"tlsca"`
}

// ConnectionProfileTLS is the object with CA servers TLS information
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type ConnectionProfileTLS struct {

	// Cert is the base64 encoded tls cert of CA server
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Cert string `json:"cert"`
}

// CAEndpoints is the list of endpoints to communicate with the CA
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CAEndpoints struct {
	// API is the endpoint to communicate with CA's API
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	API string `json:"api"`
	// Operations is the endpoint to communicate with CA's Operations API
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Operations string `json:"operations"`
}

// CAImages is the list of images to be used in CA deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CAImages struct {
	// CAImage is the name of the CA image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CAImage string `json:"caImage,omitempty"`
	// CATag is the tag of the CA image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CATag string `json:"caTag,omitempty"`
	// CAInitImage is the name of the Init image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CAInitImage string `json:"caInitImage,omitempty"`
	// CAInitTag is the tag of the Init image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CAInitTag string `json:"caInitTag,omitempty"`
	// HSMImage is the name of the HSM image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HSMImage string `json:"hsmImage,omitempty"`
	// HSMTag is the tag of the HSM image
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
// CACustomNames is the list of preconfigured objects to be used for CA's deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CACustomNames struct {
	// PVC is the list of PVC Names to be used for CA's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PVC CAPVCNames `json:"pvc,omitempty"`
	// Sqlite is the sqlite path to be used for CA's deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Sqlite string `json:"sqlitepath,omitempty"`
}

// CAPVCNames is the list of PVC Names to be used for CA's deployment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CAPVCNames struct {
	// CA is the pvc to be used as CA's storage
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CA string `json:"ca,omitempty"`
}

// CAAction contains actions that can be performed on CA
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CAAction struct {
	// Restart action is used to restart the running CA
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Restart bool `json:"restart,omitempty"`

	// Renew action is object for certificate renewals
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Renew Renew `json:"renew,omitempty"`
}

// Renew is object for certificate renewals
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type Renew struct {
	// TLSCert action is used to renew TLS crypto for CA server
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLSCert bool `json:"tlscert,omitempty"`
}
