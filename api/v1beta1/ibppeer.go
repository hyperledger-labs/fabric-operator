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
	"strings"

	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
	v2config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v2"
	v25config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v25"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"
	"github.com/IBM-Blockchain/fabric-operator/version"
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:generate=false

type CoreConfig interface {
	UsingPKCS11() bool
}

func (s *IBPPeer) ResetRestart() {
	s.Spec.Action.Restart = false
}

func (s *IBPPeer) ResetEcertReenroll() {
	s.Spec.Action.Reenroll.Ecert = false
	s.Spec.Action.Reenroll.EcertNewKey = false
}

func (s *IBPPeer) ResetTLSReenroll() {
	s.Spec.Action.Reenroll.TLSCert = false
	s.Spec.Action.Reenroll.TLSCertNewKey = false
}

func (s *IBPPeer) ResetEcertEnroll() {
	s.Spec.Action.Enroll.Ecert = false
}

func (s *IBPPeer) ResetTLSEnroll() {
	s.Spec.Action.Enroll.TLSCert = false
}

func (s *IBPPeer) ResetUpgradeDBs() {
	s.Spec.Action.UpgradeDBs = false
}

func (p *IBPPeer) ClientAuthCryptoSet() bool {
	secret := p.Spec.Secret
	if secret != nil {
		if secret.MSP != nil && secret.MSP.ClientAuth != nil {
			return true
		}
		if secret.Enrollment != nil && secret.Enrollment.ClientAuth != nil {
			return true
		}
	}

	return false
}

func (p *IBPPeer) UsingHSMProxy() bool {
	if p.Spec.HSM != nil && p.Spec.HSM.PKCS11Endpoint != "" {
		return true
	}
	return false
}

func (p *IBPPeer) UsingHSMImage() bool {
	if p.Spec.Images != nil && p.Spec.Images.HSMImage != "" {
		return true
	}
	return false
}

func (p *IBPPeer) UsingCCLauncherImage() bool {
	if p.Spec.Images != nil && p.Spec.Images.CCLauncherImage != "" {
		return true
	}

	return false
}

func (p *IBPPeer) EnrollerImage() string {
	return image.Format(p.Spec.Images.EnrollerImage, p.Spec.Images.EnrollerTag)
}
func IsV25Peer(fabricVersion string) bool {
	currentVer := version.String(fabricVersion)
	if currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_5_1) {
		return true
	}
	return false
}

func (s *IBPPeer) GetConfigOverride() (interface{}, error) {
	switch version.GetMajorReleaseVersion(s.Spec.FabricVersion) {
	case version.V2:
		isv25Peer := IsV25Peer(s.Spec.FabricVersion)
		if s.Spec.ConfigOverride == nil {
			if isv25Peer {
				return &v25config.Core{}, nil
			} else {
				return &v2config.Core{}, nil
			}
		}

		var configOverride interface{}
		var err error
		if isv25Peer {
			configOverride, err = v25config.ReadFrom(&s.Spec.ConfigOverride.Raw)
		} else {
			configOverride, err = v2config.ReadFrom(&s.Spec.ConfigOverride.Raw)
		}
		if err != nil {
			return nil, err
		}
		return configOverride, nil
	case version.V1:
		fallthrough
	default:
		if s.Spec.ConfigOverride == nil {
			return &config.Core{}, nil
		}

		configOverride, err := config.ReadFrom(&s.Spec.ConfigOverride.Raw)
		if err != nil {
			return nil, err
		}
		return configOverride, nil
	}
}

func (s *IBPPeer) IsHSMEnabled() bool {
	configOverride, err := s.GetConfigOverride()
	if err != nil {
		return false
	}

	return configOverride.(CoreConfig).UsingPKCS11()
}

func (s *IBPPeer) UsingCouchDB() bool {
	if strings.ToLower(s.Spec.StateDb) == "couchdb" {
		return true
	}

	return false
}

func (s *IBPPeer) GetPullSecrets() []corev1.LocalObjectReference {
	pullSecrets := []corev1.LocalObjectReference{}
	for _, ps := range s.Spec.ImagePullSecrets {
		pullSecrets = append(pullSecrets, corev1.LocalObjectReference{Name: ps})
	}
	return pullSecrets
}

func (s *IBPPeer) GetRegistryURL() string {
	return s.Spec.RegistryURL
}

func (s *IBPPeer) GetArch() []string {
	return s.Spec.Arch
}

// GetFabricVersion returns fabric version from CR spec
func (s *IBPPeer) GetFabricVersion() string {
	return s.Spec.FabricVersion
}

// SetFabricVersion sets fabric version on spec
func (s *IBPPeer) SetFabricVersion(version string) {
	s.Spec.FabricVersion = version
}

// ImagesSet returns true if the spec has images defined
func (s *IBPPeer) ImagesSet() bool {
	return s.Spec.Images != nil
}

// GetResource returns resources defined in spec for request component, if no resources
// defined returns blank but initialized instance of resources
func (s *IBPPeer) GetResource(comp Component) corev1.ResourceRequirements {
	if s.Spec.Resources != nil {
		switch comp {
		case INIT:
			if s.Spec.Resources.Init != nil {
				return *s.Spec.Resources.Init
			}
		case PEER:
			if s.Spec.Resources.Peer != nil {
				return *s.Spec.Resources.Peer
			}
		case GRPCPROXY:
			if s.Spec.Resources.GRPCProxy != nil {
				return *s.Spec.Resources.GRPCProxy
			}
		case FLUENTD:
			if s.Spec.Resources.FluentD != nil {
				return *s.Spec.Resources.FluentD
			}
		case DIND:
			if s.Spec.Resources.DinD != nil {
				return *s.Spec.Resources.DinD
			}
		case COUCHDB:
			if s.Spec.Resources.CouchDB != nil {
				return *s.Spec.Resources.CouchDB
			}
		case CCLAUNCHER:
			if s.Spec.Resources.CCLauncher != nil {
				return *s.Spec.Resources.CCLauncher
			}
		case ENROLLER:
			if s.Spec.Resources.Enroller != nil {
				return *s.Spec.Resources.Enroller
			}
		case HSMDAEMON:
			if s.Spec.Resources.HSMDaemon != nil {
				return *s.Spec.Resources.HSMDaemon
			}
		}
	}

	return corev1.ResourceRequirements{}
}

// PVCName returns pvc name associated with instance
func (s *IBPPeer) PVCName() string {
	name := s.Name + "-pvc"
	if s.Spec.CustomNames.PVC.Peer != "" {
		name = s.Spec.CustomNames.PVC.Peer
	}
	return name
}

func (s *IBPPeer) GetMSPID() string {
	return s.Spec.MSPID
}

func (s *IBPPeerSpec) NodeOUDisabled() bool {
	if s.DisableNodeOU != nil {
		return *s.DisableNodeOU
	}

	return false
}

func (s *IBPPeerSpec) HSMSet() bool {
	if s.HSM != nil && s.HSM.PKCS11Endpoint != "" {
		return true
	}

	return false
}

func (s *IBPPeerSpec) DomainSet() bool {
	if s.Domain != "" {
		return true
	}

	return false
}

func (s *IBPPeerSpec) UsingLevelDB() bool {
	if strings.ToLower(s.StateDb) == "leveldb" {
		return true
	}

	return false
}

func (s *IBPPeerSpec) GetNumSecondsWarningPeriod() int64 {
	daysToSecondsConversion := int64(24 * 60 * 60)
	if s.NumSecondsWarningPeriod == 0 {
		// Default to the equivalent of 30 days
		return 30 * daysToSecondsConversion
	}
	return s.NumSecondsWarningPeriod
}

func (p *IBPPeerStatus) HasType() bool {
	if p.CRStatus.Type != "" {
		return true
	}
	return false
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
}

func init() {
	SchemeBuilder.Register(&IBPPeer{}, &IBPPeerList{})
}
