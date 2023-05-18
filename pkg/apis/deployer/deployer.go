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

package deployer

import (
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"
	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	ClusterType      string        `json:"clusterType"`
	Domain           string        `json:"domain"`
	DashboardURL     string        `json:"dashboardurl"`
	Database         Database      `json:"db"`
	Loglevel         string        `json:"loglevel"`
	Port             int           `json:"port"`
	TLS              TLSConfig     `json:"tls"`
	Auth             BasicAuth     `json:"auth"`
	Namespace        string        `json:"namespace"`
	Defaults         *Defaults     `json:"defaults"`
	Versions         *Versions     `json:"versions"`
	ImagePullSecrets []string      `json:"imagePullSecrets"`
	ServiceConfig    ServiceConfig `json:"serviceConfig"`
	CRN              *current.CRN  `json:"crn"`
	Timeouts         *Timeouts     `json:"timeouts"`
	OtherImages      *OtherImages  `json:"otherImages"`
	ServiceAccount   string        `json:"serviceAccount"`
	UseTags          *bool         `json:"usetags"`
}

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

// CAImages is the list of images to be used in CA deployment
type CAImages struct {
	// CAImage is the name of the CA image
	CAImage string `json:"caImage,omitempty"`

	// CATag is the tag of the CA image
	CATag string `json:"caTag,omitempty"`

	// CADigest is the digest tag of the CA image
	CADigest string `json:"caDigest,omitempty"`

	// CAInitImage is the name of the Init image
	CAInitImage string `json:"caInitImage,omitempty"`

	// CAInitTag is the tag of the Init image
	CAInitTag string `json:"caInitTag,omitempty"`

	// CAInitDigest is the digest tag of the Init image
	CAInitDigest string `json:"caInitDigest,omitempty"`

	// HSMImage is the name of the HSM image
	HSMImage string `json:"hsmImage,omitempty"`

	// HSMTag is the tag of the HSM image
	HSMTag string `json:"hsmTag,omitempty"`

	// HSMDigest is the tag of the HSM image
	HSMDigest string `json:"hsmDigest,omitempty"`

	// EnrollerImage is the name of the init image for crypto generation
	EnrollerImage string `json:"enrollerImage,omitempty"`

	// EnrollerTag is the tag of the init image for crypto generation
	EnrollerTag string `json:"enrollerTag,omitempty"`

	// EnrollerDigest is the digest tag of the init image for crypto generation
	EnrollerDigest string `json:"enrollerDigest,omitempty"`
}

// PeerImages is the list of images to be used in peer deployment
type PeerImages struct {
	// PeerInitImage is the name of the peer init image
	PeerInitImage string `json:"peerInitImage,omitempty"`

	// PeerInitTag is the tag of the peer init image
	PeerInitTag string `json:"peerInitTag,omitempty"`

	// PeerInitDigest is the digest tag of the peer init image
	PeerInitDigest string `json:"peerInitDigest,omitempty"`

	// PeerImage is the name of the peer image
	PeerImage string `json:"peerImage,omitempty"`

	// PeerTag is the tag of the peer image
	PeerTag string `json:"peerTag,omitempty"`

	// PeerDigest is the digest tag of the peer image
	PeerDigest string `json:"peerDigest,omitempty"`

	// DindImage is the name of the dind image
	DindImage string `json:"dindImage,omitempty"`

	// DindTag is the tag of the dind image
	DindTag string `json:"dindTag,omitempty"`

	// DindDigest is the digest tag of the dind image
	DindDigest string `json:"dindDigest,omitempty"`

	// GRPCWebImage is the name of the grpc web proxy image
	GRPCWebImage string `json:"grpcwebImage,omitempty"`

	// GRPCWebTag is the tag of the grpc web proxy image
	GRPCWebTag string `json:"grpcwebTag,omitempty"`

	// GRPCWebDigest is the digest tag of the grpc web proxy image
	GRPCWebDigest string `json:"grpcwebDigest,omitempty"`

	// FluentdImage is the name of the fluentd logger image
	FluentdImage string `json:"fluentdImage,omitempty"`

	// FluentdTag is the tag of the fluentd logger image
	FluentdTag string `json:"fluentdTag,omitempty"`

	// FluentdDigest is the digest tag of the fluentd logger image
	FluentdDigest string `json:"fluentdDigest,omitempty"`

	// CouchDBImage is the name of the couchdb image
	CouchDBImage string `json:"couchdbImage,omitempty"`

	// CouchDBTag is the tag of the couchdb image
	CouchDBTag string `json:"couchdbTag,omitempty"`

	// CouchDBDigest is the digest tag of the couchdb image
	CouchDBDigest string `json:"couchdbDigest,omitempty"`

	// CCLauncherImage is the name of the chaincode launcher image
	CCLauncherImage string `json:"chaincodeLauncherImage,omitempty"`

	// CCLauncherTag is the tag of the chaincode launcher image
	CCLauncherTag string `json:"chaincodeLauncherTag,omitempty"`

	// CCLauncherDigest is the digest tag of the chaincode launcher image
	CCLauncherDigest string `json:"chaincodeLauncherDigest,omitempty"`

	// FileTransferImage is the name of the file transfer image
	FileTransferImage string `json:"fileTransferImage,omitempty"`

	// FileTransferTag is the tag of the file transfer image
	FileTransferTag string `json:"fileTransferTag,omitempty"`

	// FileTransferDigest is the digest tag of the file transfer image
	FileTransferDigest string `json:"fileTransferDigest,omitempty"`

	// BuilderImage is the name of the builder image
	BuilderImage string `json:"builderImage,omitempty"`

	// BuilderTag is the tag of the builder image
	BuilderTag string `json:"builderTag,omitempty"`

	// BuilderDigest is the digest tag of the builder image
	BuilderDigest string `json:"builderDigest,omitempty"`

	// GoEnvImage is the name of the goenv image
	GoEnvImage string `json:"goEnvImage,omitempty"`

	// GoEnvTag is the tag of the goenv image
	GoEnvTag string `json:"goEnvTag,omitempty"`

	// GoEnvDigest is the digest tag of the goenv image
	GoEnvDigest string `json:"goEnvDigest,omitempty"`

	// JavaEnvImage is the name of the javaenv image
	JavaEnvImage string `json:"javaEnvImage,omitempty"`

	// JavaEnvTag is the tag of the javaenv image
	JavaEnvTag string `json:"javaEnvTag,omitempty"`

	// JavaEnvDigest is the digest tag of the javaenv image
	JavaEnvDigest string `json:"javaEnvDigest,omitempty"`

	// NodeEnvImage is the name of the nodeenv image
	NodeEnvImage string `json:"nodeEnvImage,omitempty"`

	// NodeEnvTag is the tag of the nodeenv image
	NodeEnvTag string `json:"nodeEnvTag,omitempty"`

	// NodeEnvDigest is the digest tag of the nodeenv image
	NodeEnvDigest string `json:"nodeEnvDigest,omitempty"`

	// HSMImage is the name of the hsm image
	HSMImage string `json:"hsmImage,omitempty"`

	// HSMTag is the tag of the hsm image
	HSMTag string `json:"hsmTag,omitempty"`

	// HSMDigest is the digest tag of the hsm image
	HSMDigest string `json:"hsmDigest,omitempty"`

	// EnrollerImage is the name of the init image for crypto generation
	EnrollerImage string `json:"enrollerImage,omitempty"`

	// EnrollerTag is the tag of the init image for crypto generation
	EnrollerTag string `json:"enrollerTag,omitempty"`

	// EnrollerDigest is the digest tag of the init image for crypto generation
	EnrollerDigest string `json:"enrollerDigest,omitempty"`
}

// OrdererImages is the list of images to be used in orderer deployment
type OrdererImages struct {
	// OrdererInitImage is the name of the orderer init image
	OrdererInitImage string `json:"ordererInitImage,omitempty"`

	// OrdererInitTag is the tag of the orderer init image
	OrdererInitTag string `json:"ordererInitTag,omitempty"`

	// OrdererInitDigest is the digest tag of the orderer init image
	OrdererInitDigest string `json:"ordererInitDigest,omitempty"`

	// OrdererImage is the name of the orderer image
	OrdererImage string `json:"ordererImage,omitempty"`

	// OrdererTag is the tag of the orderer image
	OrdererTag string `json:"ordererTag,omitempty"`

	// OrdererDigest is the digest tag of the orderer image
	OrdererDigest string `json:"ordererDigest,omitempty"`

	// GRPCWebImage is the name of the grpc web proxy image
	GRPCWebImage string `json:"grpcwebImage,omitempty"`

	// GRPCWebTag is the tag of the grpc web proxy image
	GRPCWebTag string `json:"grpcwebTag,omitempty"`

	// GRPCWebDigest is the digest tag of the grpc web proxy image
	GRPCWebDigest string `json:"grpcwebDigest,omitempty"`

	// HSMImage is the name of the hsm image
	HSMImage string `json:"hsmImage,omitempty"`

	// HSMTag is the tag of the hsm image
	HSMTag string `json:"hsmTag,omitempty"`

	// HSMDigest is the digest tag of the hsm image
	HSMDigest string `json:"hsmDigest,omitempty"`

	// EnrollerImage is the name of the init image for crypto generation
	EnrollerImage string `json:"enrollerImage,omitempty"`

	// EnrollerTag is the tag of the init image for crypto generation
	EnrollerTag string `json:"enrollerTag,omitempty"`

	// EnrollerDigest is the digest tag of the init image for crypto generation
	EnrollerDigest string `json:"enrollerDigest,omitempty"`
}

type Defaults struct {
	Storage   *Storage   `json:"storage"`
	Resources *Resources `json:"resources"`
}

type Storage struct {
	Peer    *current.PeerStorages    `json:"peer"`
	CA      *current.CAStorages      `json:"ca"`
	Orderer *current.OrdererStorages `json:"orderer"`
}

type Resources struct {
	Peer    *current.PeerResources    `json:"peer"`
	CA      *current.CAResources      `json:"ca"`
	Orderer *current.OrdererResources `json:"orderer"`
}

type ServiceConfig struct {
	Type corev1.ServiceType `json:"type"`
}

// IndividualDatabase describes the initialization of databases
type IndividualDatabase struct {
	Name       string   `json:"name"`
	DesignDocs []string `json:"designdocs"`
}

// Database is connection details to connect to couchdb database
type Database struct {
	ConnectionURL string             `json:"connectionurl"`
	Components    IndividualDatabase `json:"components"`
	CreateDB      bool               `json:"createdb"`
}

// TLSConfig is to configure the tls server
type TLSConfig struct {
	Enabled       bool   `json:"enabled"`
	ListenAddress string `json:"listenaddress"`
	CertPath      string `json:"certpath"`
	KeyPath       string `json:"keypath"`
}

// BasicAuth provides implementation to store basic auth info
type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Timeouts struct {
	Deployment int `json:"componentDeploy"`
	APIServer  int `json:"apiServer"`
}

// OtherImages contains other images and tags required to run deployer.
type OtherImages struct {
	// MustgatherImage is the name of the mustgather image
	MustgatherImage string `json:"mustgatherImage,omitempty"`

	// MustgatherTag is the tag of the mustgatherTag image
	MustgatherTag string `json:"mustgatherTag,omitempty"`

	// MustgatherDigest is the tag of the mustgatherDigest image
	MustgatherDigest string `json:"mustgatherDigest,omitempty"`
}

// ConsoleImages is the list of images to be used in console deployment
type ConsoleImages struct {
	// ConsoleInitImage is the name of the console init image
	ConsoleInitImage string `json:"consoleInitImage,omitempty"`
	// ConsoleInitTag is the tag of the console init image
	ConsoleInitTag string `json:"consoleInitTag,omitempty"`
	// ConsoleInitDigest is the digest of the console init image
	ConsoleInitDigest string `json:"consoleInitDigest,omitempty"`

	// ConsoleImage is the name of the console image
	ConsoleImage string `json:"consoleImage,omitempty"`
	// ConsoleTag is the tag of the console image
	ConsoleTag string `json:"consoleTag,omitempty"`
	// ConsoleDigest is the digest of the console image
	ConsoleDigest string `json:"consoleDigest,omitempty"`

	// ConfigtxlatorImage is the name of the configtxlator image
	ConfigtxlatorImage string `json:"configtxlatorImage,omitempty"`
	// ConfigtxlatorTag is the tag of the configtxlator image
	ConfigtxlatorTag string `json:"configtxlatorTag,omitempty"`
	// ConfigtxlatorDigest is the digest of the configtxlator image
	ConfigtxlatorDigest string `json:"configtxlatorDigest,omitempty"`

	// DeployerImage is the name of the deployer image
	DeployerImage string `json:"deployerImage,omitempty"`
	// DeployerTag is the tag of the deployer image
	DeployerTag string `json:"deployerTag,omitempty"`
	// DeployerDigest is the digest of the deployer image
	DeployerDigest string `json:"deployerDigest,omitempty"`

	// CouchDBImage is the name of the couchdb image
	CouchDBImage string `json:"couchdbImage,omitempty"`
	// CouchDBTag is the tag of the couchdb image
	CouchDBTag string `json:"couchdbTag,omitempty"`
	// CouchDBDigest is the digest of the couchdb image
	CouchDBDigest string `json:"couchdbDigest,omitempty"`

	// MustgatherImage is the name of the mustgather image
	MustgatherImage string `json:"mustgatherImage,omitempty"`
	// MustgatherTag is the tag of the mustgather image
	MustgatherTag string `json:"mustgatherTag,omitempty"`
	// MustgatherDigest is the digest of the mustgather image
	MustgatherDigest string `json:"mustgatherDigest,omitempty"`
}

func (v *Versions) Override(requestedVersions *Versions, registryURL string, arch string) {
	if requestedVersions == nil {
		return
	}

	if len(requestedVersions.CA) != 0 {
		CAVersions := map[string]VersionCA{}
		for key := range requestedVersions.CA {
			var caConfig VersionCA
			requestedCAVersion := requestedVersions.CA[key]
			caConfig.Image.Override(&requestedCAVersion.Image, registryURL, arch)
			caConfig.Default = requestedCAVersion.Default
			caConfig.Version = requestedCAVersion.Version
			CAVersions[key] = caConfig
		}
		v.CA = CAVersions
	}

	if len(requestedVersions.Peer) != 0 {
		PeerVersions := map[string]VersionPeer{}
		for key := range requestedVersions.Peer {
			var peerConfig VersionPeer
			requestedPeerVersion := requestedVersions.Peer[key]
			peerConfig.Image.Override(&requestedPeerVersion.Image, registryURL, arch)
			peerConfig.Default = requestedPeerVersion.Default
			peerConfig.Version = requestedPeerVersion.Version
			PeerVersions[key] = peerConfig
		}
		v.Peer = PeerVersions
	}

	if len(requestedVersions.Orderer) != 0 {
		OrdererVersions := map[string]VersionOrderer{}
		for key := range requestedVersions.Orderer {
			var ordererConfig VersionOrderer
			requestedOrdererVersion := requestedVersions.Orderer[key]
			ordererConfig.Image.Override(&requestedOrdererVersion.Image, registryURL, arch)
			ordererConfig.Default = requestedOrdererVersion.Default
			ordererConfig.Version = requestedOrdererVersion.Version
			OrdererVersions[key] = ordererConfig
		}
		v.Orderer = OrdererVersions
	}
}

// Override will look at requested images and use those to override default image
// values. Override also format the image tag to include arch for non-sha based
// tags.
func (i *CAImages) Override(requested *CAImages, registryURL string, arch string) {
	// If requested is nil, we are only interested in properly prepending registry
	// URL to the image and with overriding default values so a empty struct is initialized.
	if requested == nil {
		requested = &CAImages{}
	}

	// Images
	i.CAInitImage = image.GetImage(registryURL, i.CAInitImage, requested.CAInitImage)
	i.CAImage = image.GetImage(registryURL, i.CAImage, requested.CAImage)
	i.HSMImage = image.GetImage(registryURL, i.HSMImage, requested.HSMImage)
	i.EnrollerImage = image.GetImage(registryURL, i.EnrollerImage, requested.EnrollerImage)

	// Tags
	i.CAInitTag = image.GetTag(arch, i.CAInitTag, requested.CAInitTag)
	i.CATag = image.GetTag(arch, i.CATag, requested.CATag)
	i.HSMTag = image.GetTag(arch, i.HSMTag, requested.HSMTag)
	i.EnrollerTag = image.GetTag(arch, i.EnrollerTag, requested.EnrollerTag)

	// Digests
	i.CAInitDigest = image.GetTag(arch, i.CAInitDigest, requested.CAInitDigest)
	i.CADigest = image.GetTag(arch, i.CADigest, requested.CADigest)
	i.HSMDigest = image.GetTag(arch, i.HSMDigest, requested.HSMDigest)
	i.EnrollerDigest = image.GetTag(arch, i.EnrollerDigest, requested.EnrollerDigest)
}

func (i *PeerImages) Override(requested *PeerImages, registryURL string, arch string) {
	if requested == nil {
		requested = &PeerImages{}
	}

	// Images
	i.PeerInitImage = image.GetImage(registryURL, i.PeerInitImage, requested.PeerInitImage)
	i.PeerImage = image.GetImage(registryURL, i.PeerImage, requested.PeerImage)
	i.CouchDBImage = image.GetImage(registryURL, i.CouchDBImage, requested.CouchDBImage)
	i.DindImage = image.GetImage(registryURL, i.DindImage, requested.DindImage)
	i.GRPCWebImage = image.GetImage(registryURL, i.GRPCWebImage, requested.GRPCWebImage)
	i.FluentdImage = image.GetImage(registryURL, i.FluentdImage, requested.FluentdImage)
	i.CCLauncherImage = image.GetImage(registryURL, i.CCLauncherImage, requested.CCLauncherImage)
	i.FileTransferImage = image.GetImage(registryURL, i.FileTransferImage, requested.FileTransferImage)
	i.BuilderImage = image.GetImage(registryURL, i.BuilderImage, requested.BuilderImage)
	i.GoEnvImage = image.GetImage(registryURL, i.GoEnvImage, requested.GoEnvImage)
	i.JavaEnvImage = image.GetImage(registryURL, i.JavaEnvImage, requested.JavaEnvImage)
	i.NodeEnvImage = image.GetImage(registryURL, i.NodeEnvImage, requested.NodeEnvImage)
	i.HSMImage = image.GetImage(registryURL, i.HSMImage, requested.HSMImage)
	i.EnrollerImage = image.GetImage(registryURL, i.EnrollerImage, requested.EnrollerImage)

	// Tags
	i.PeerInitTag = image.GetTag(arch, i.PeerInitTag, requested.PeerInitTag)
	i.PeerTag = image.GetTag(arch, i.PeerTag, requested.PeerTag)
	i.CouchDBTag = image.GetTag(arch, i.CouchDBTag, requested.CouchDBTag)
	i.DindTag = image.GetTag(arch, i.DindTag, requested.DindTag)
	i.GRPCWebTag = image.GetTag(arch, i.GRPCWebTag, requested.GRPCWebTag)
	i.FluentdTag = image.GetTag(arch, i.FluentdTag, requested.FluentdTag)
	i.CCLauncherTag = image.GetTag(arch, i.CCLauncherTag, requested.CCLauncherTag)
	i.FileTransferTag = image.GetTag(arch, i.FileTransferTag, requested.FileTransferTag)
	i.BuilderTag = image.GetTag(arch, i.BuilderTag, requested.BuilderTag)
	i.GoEnvTag = image.GetTag(arch, i.GoEnvTag, requested.GoEnvTag)
	i.JavaEnvTag = image.GetTag(arch, i.JavaEnvTag, requested.JavaEnvTag)
	i.NodeEnvTag = image.GetTag(arch, i.NodeEnvTag, requested.NodeEnvTag)
	i.HSMTag = image.GetTag(arch, i.HSMTag, requested.HSMTag)
	i.EnrollerTag = image.GetTag(arch, i.EnrollerTag, requested.EnrollerTag)

	// Digests
	i.PeerInitDigest = image.GetTag(arch, i.PeerInitDigest, requested.PeerInitDigest)
	i.PeerDigest = image.GetTag(arch, i.PeerDigest, requested.PeerDigest)
	i.CouchDBDigest = image.GetTag(arch, i.CouchDBDigest, requested.CouchDBDigest)
	i.DindDigest = image.GetTag(arch, i.DindDigest, requested.DindDigest)
	i.GRPCWebDigest = image.GetTag(arch, i.GRPCWebDigest, requested.GRPCWebDigest)
	i.FluentdDigest = image.GetTag(arch, i.FluentdDigest, requested.FluentdDigest)
	i.CCLauncherDigest = image.GetTag(arch, i.CCLauncherDigest, requested.CCLauncherDigest)
	i.FileTransferDigest = image.GetTag(arch, i.FileTransferDigest, requested.FileTransferDigest)
	i.BuilderDigest = image.GetTag(arch, i.BuilderDigest, requested.BuilderDigest)
	i.GoEnvDigest = image.GetTag(arch, i.GoEnvDigest, requested.GoEnvDigest)
	i.JavaEnvDigest = image.GetTag(arch, i.JavaEnvDigest, requested.JavaEnvDigest)
	i.NodeEnvDigest = image.GetTag(arch, i.NodeEnvDigest, requested.NodeEnvDigest)
	i.HSMDigest = image.GetTag(arch, i.HSMDigest, requested.HSMDigest)
	i.EnrollerDigest = image.GetTag(arch, i.EnrollerDigest, requested.EnrollerDigest)
}

func (i *OrdererImages) Override(requested *OrdererImages, registryURL string, arch string) {
	if requested == nil {
		requested = &OrdererImages{}
	}
	// Images
	i.GRPCWebImage = image.GetImage(registryURL, i.GRPCWebImage, requested.GRPCWebImage)
	i.OrdererInitImage = image.GetImage(registryURL, i.OrdererInitImage, requested.OrdererInitImage)
	i.OrdererImage = image.GetImage(registryURL, i.OrdererImage, requested.OrdererImage)
	i.HSMImage = image.GetImage(registryURL, i.HSMImage, requested.HSMImage)
	i.EnrollerImage = image.GetImage(registryURL, i.EnrollerImage, requested.EnrollerImage)

	// Tags
	i.GRPCWebTag = image.GetTag(arch, i.GRPCWebTag, requested.GRPCWebTag)
	i.OrdererInitTag = image.GetTag(arch, i.OrdererInitTag, requested.OrdererInitTag)
	i.OrdererTag = image.GetTag(arch, i.OrdererTag, requested.OrdererTag)
	i.HSMTag = image.GetTag(arch, i.HSMTag, requested.HSMTag)
	i.EnrollerTag = image.GetTag(arch, i.EnrollerTag, requested.EnrollerTag)

	// Digests
	i.GRPCWebDigest = image.GetTag(arch, i.GRPCWebDigest, requested.GRPCWebDigest)
	i.OrdererInitDigest = image.GetTag(arch, i.OrdererInitDigest, requested.OrdererInitDigest)
	i.OrdererDigest = image.GetTag(arch, i.OrdererDigest, requested.OrdererDigest)
	i.HSMDigest = image.GetTag(arch, i.HSMDigest, requested.HSMDigest)
	i.EnrollerDigest = image.GetTag(arch, i.EnrollerDigest, requested.EnrollerDigest)
}
