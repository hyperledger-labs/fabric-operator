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
)

// Service is the overrides to be used for Service of the component
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type Service struct {
	// The "type" of the service to be used
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Type corev1.ServiceType `json:"type,omitempty"`
}

// StorageSpec is the overrides to be used for storage of the component
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type StorageSpec struct {
	// Size of storage
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Size string `json:"size,omitempty"`

	// Class is the storage class
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Class string `json:"class,omitempty"`
}

// NetworkInfo is the overrides for the network of the component
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type NetworkInfo struct {
	// Domain for the components
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Domain string `json:"domain,omitempty"`

	// ConsolePort is the port to access the console
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConsolePort int32 `json:"consolePort,omitempty"`

	// ConfigtxlatorPort is the port to access configtxlator
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ConfigtxlatorPort int32 `json:"configtxlatorPort,omitempty"`

	// ProxyPort is the port to access console proxy
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ProxyPort int32 `json:"proxyPort,omitempty"`
}

// Ingress (Optional) is the list of overrides for ingress of the components
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type Ingress struct {
	// TlsSecretName (Optional) is the secret name to be used for tls certificates
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TlsSecretName string `json:"tlsSecretName,omitempty"`

	// Class (Optional) is the class to set for ingress
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Class string `json:"class,omitempty"`
}

// IBPCRStatus is the string that defines if status is set by the controller
// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
type IBPCRStatus string

const (
	// True means that the status is set by the controller successfully
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	True IBPCRStatus = "True"

	// False stands for the status which is not correctly set and should be ignored
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	False IBPCRStatus = "False"

	// Unknown stands for unknown status
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Unknown IBPCRStatus = "Unknown"
)

// IBPCRStatusType is the string that stores teh status
// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
type IBPCRStatusType string

const (
	// Deploying is the status when component is being deployed
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Deploying IBPCRStatusType = "Deploying"

	// Deployed is the status when the component's deployment is done successfully
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Deployed IBPCRStatusType = "Deployed"

	// Precreated is the status of the orderers when they are waiting for config block
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Precreated IBPCRStatusType = "Precreated"

	// Error is the status when a component's deployment has failed due to an error
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Error IBPCRStatusType = "Error"

	// Warning is the status when a component is running, but will fail in future
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Warning IBPCRStatusType = "Warning"

	// Initializing is the status when a component is initializing
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Initializing IBPCRStatusType = "Initializing"
)

// +k8s:deepcopy-gen=true
// CRStatus is the object that defines the status of a CR
// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
type CRStatus struct {
	// Type is true or false based on if status is valid
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Type IBPCRStatusType `json:"type,omitempty"`

	// Status is defined based on the current status of the component
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Status IBPCRStatus `json:"status,omitempty"`

	// Reason provides a reason for an error
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Reason string `json:"reason,omitempty"`

	// Message provides a message for the status to be shown to customer
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Message string `json:"message,omitempty"`

	// LastHeartbeatTime is when the controller reconciled this component
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	LastHeartbeatTime string `json:"lastHeartbeatTime,omitempty"`

	// Version is the product (IBP) version of the component
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Version string `json:"version,omitempty"`

	// ErrorCode is the code of classification of errors
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	ErrorCode int `json:"errorcode,omitempty"`

	// Versions is the operand version of the component
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Versions CRStatusVersion `json:"versions,omitempty"`
}

// CRStatusVersion provides the current reconciled version of the operand
// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
type CRStatusVersion struct {
	// Reconciled provides the reconciled version of the operand
	Reconciled string `json:"reconciled"`
}

// HSM struct is DEPRECATED
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type HSM struct {
	// PKCS11Endpoint is DEPRECATED
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	PKCS11Endpoint string `json:"pkcs11endpoint,omitempty"`
}

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

// License should be accepted to install custom resources
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type License struct {
	// Accept should be set to true to accept the license.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:checkbox"
	// +kubebuilder:validation:Enum=true
	Accept bool `json:"accept,omitempty"`
}

// +k8s:deepcopy-gen=true
// SecretSpec defines the crypto spec to pass to components
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type SecretSpec struct {
	// Enrollment defines enrollment part of secret spec
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Enrollment *EnrollmentSpec `json:"enrollment,omitempty"`

	// MSP defines msp part of secret spec
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	MSP *MSPSpec `json:"msp,omitempty"`
}

// CATLS contains the TLS CA certificate of the CA
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CATLS struct {
	// CACert is the base64 encoded certificate
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CACert string `json:"cacert,omitempty"`
}

// +k8s:deepcopy-gen=true
// Enrollment is the enrollment section of secret spec
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type Enrollment struct {
	// CAHost is host part of the CA to use
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CAHost string `json:"cahost,omitempty"`

	// CAPort is port of the CA to use
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CAPort string `json:"caport,omitempty"`

	// CAName is name of CA
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CAName string `json:"caname,omitempty"`

	// CATLS is tls details to talk to CA endpoint
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CATLS *CATLS `json:"catls,omitempty"`

	// EnrollID is the enrollment username
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	EnrollID string `json:"enrollid,omitempty"`

	// EnrollSecret is enrollment secret ( password )
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	EnrollSecret string `json:"enrollsecret,omitempty"`

	// AdminCerts is the base64 encoded admincerts
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	AdminCerts []string `json:"admincerts,omitempty"`

	// CSR is the CSR override object
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CSR *CSR `json:"csr,omitempty"`
}

// +k8s:deepcopy-gen=true
// EnrollmentSpec contains all the configurations that a component needs to enroll with
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type EnrollmentSpec struct {
	// Component contains ecert enrollment details
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Component *Enrollment `json:"component,omitempty"`

	// TLS contains tls enrollment details
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLS *Enrollment `json:"tls,omitempty"`

	// ClientAuth contains client uath enrollment details
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ClientAuth *Enrollment `json:"clientauth,omitempty"`
}

// +k8s:deepcopy-gen=true
// CSR has the Hosts for the CSR to be sent in the enrollment
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type CSR struct {
	// Hosts override for CSR
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Hosts []string `json:"hosts,omitempty"`
}

// +k8s:deepcopy-gen=true
// MSPSpec contains the configuration for the component to start with all the certificates
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type MSPSpec struct {
	// Component contains crypto for ecerts
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Component *MSP `json:"component,omitempty"`

	// TLS contains crypto for tls certs
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TLS *MSP `json:"tls,omitempty"`

	// ClientAuth contains crypto for client auth certs
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ClientAuth *MSP `json:"clientauth,omitempty"`
}

// +k8s:deepcopy-gen=true
// MSP contains the common definitions crypto material for the component
// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
type MSP struct {
	// KeyStore is base64 encoded private key
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	KeyStore string `json:"keystore,omitempty"`

	// SignCerts is base64 encoded sign cert
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	SignCerts string `json:"signcerts,omitempty"`

	// CACerts is base64 encoded cacerts array
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	CACerts []string `json:"cacerts,omitempty"`

	// IntermediateCerts is base64 encoded intermediate certs array
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	IntermediateCerts []string `json:"intermediatecerts,omitempty"`

	// AdminCerts is base64 encoded admincerts array
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	AdminCerts []string `json:"admincerts,omitempty"`
}
