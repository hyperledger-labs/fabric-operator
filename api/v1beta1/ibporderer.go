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
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v1"
	v2config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v2"
	v24config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v24"
	v25config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v25"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"
	"github.com/IBM-Blockchain/fabric-operator/version"
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:generate=false
type OrdererConfig interface {
	UsingPKCS11() bool
}

func (s *IBPOrderer) ResetRestart() {
	s.Spec.Action.Restart = false
}

func (s *IBPOrderer) ResetEcertReenroll() {
	s.Spec.Action.Reenroll.Ecert = false
	s.Spec.Action.Reenroll.EcertNewKey = false
}

func (s *IBPOrderer) ResetTLSReenroll() {
	s.Spec.Action.Reenroll.TLSCert = false
	s.Spec.Action.Reenroll.TLSCertNewKey = false
}

func (s *IBPOrderer) ResetEcertEnroll() {
	s.Spec.Action.Enroll.Ecert = false
}

func (s *IBPOrderer) ResetTLSEnroll() {
	s.Spec.Action.Enroll.TLSCert = false
}

func (o *IBPOrderer) IsHSMEnabled() bool {
	ordererConfig, err := o.GetConfigOverride()
	if err != nil {
		return false
	}

	return ordererConfig.(OrdererConfig).UsingPKCS11()
}

func (o *IBPOrderer) ClientAuthCryptoSet() bool {
	secret := o.Spec.Secret
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

func (o *IBPOrderer) UsingHSMProxy() bool {
	if o.Spec.HSM != nil && o.Spec.HSM.PKCS11Endpoint != "" {
		return true
	}
	return false
}

func (o *IBPOrderer) GetConfigOverride() (interface{}, error) {
	switch version.GetMajorReleaseVersion(o.Spec.FabricVersion) {
	case version.V2:
		currentVer := version.String(o.Spec.FabricVersion)
		if currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_5_1) {
			if o.Spec.ConfigOverride == nil {
				return &v25config.Orderer{}, nil
			}

			configOverride, err := v25config.ReadFrom(&o.Spec.ConfigOverride.Raw)
			if err != nil {
				return nil, err
			}
			return configOverride, nil
		} else if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.GreaterThan(version.V2_4_1) {
			if o.Spec.ConfigOverride == nil {
				return &v24config.Orderer{}, nil
			}

			configOverride, err := v24config.ReadFrom(&o.Spec.ConfigOverride.Raw)
			if err != nil {
				return nil, err
			}
			return configOverride, nil
		} else {
			if o.Spec.ConfigOverride == nil {
				return &v2config.Orderer{}, nil
			}

			configOverride, err := v2config.ReadFrom(&o.Spec.ConfigOverride.Raw)
			if err != nil {
				return nil, err
			}
			return configOverride, nil
		}

	case version.V1:
		fallthrough
	default:
		if o.Spec.ConfigOverride == nil {
			return &config.Orderer{}, nil
		}

		configOverride, err := config.ReadFrom(&o.Spec.ConfigOverride.Raw)
		if err != nil {
			return nil, err
		}
		return configOverride, nil
	}
}

func (o *IBPOrderer) UsingHSMImage() bool {
	if o.Spec.Images != nil && o.Spec.Images.HSMImage != "" {
		return true
	}
	return false
}

func (o *IBPOrderer) EnrollerImage() string {
	return image.Format(o.Spec.Images.EnrollerImage, o.Spec.Images.EnrollerTag)
}

func (s *IBPOrderer) GetPullSecrets() []corev1.LocalObjectReference {
	pullSecrets := []corev1.LocalObjectReference{}
	for _, ps := range s.Spec.ImagePullSecrets {
		pullSecrets = append(pullSecrets, corev1.LocalObjectReference{Name: ps})
	}
	return pullSecrets
}

func (s *IBPOrderer) GetRegistryURL() string {
	return s.Spec.RegistryURL
}

func (s *IBPOrderer) GetArch() []string {
	return s.Spec.Arch
}

// GetFabricVersion returns fabric version from CR spec
func (s *IBPOrderer) GetFabricVersion() string {
	return s.Spec.FabricVersion
}

// SetFabricVersion sets fabric version on spec
func (s *IBPOrderer) SetFabricVersion(version string) {
	s.Spec.FabricVersion = version
}

// ImagesSet returns true if the spec has images defined
func (s *IBPOrderer) ImagesSet() bool {
	return s.Spec.Images != nil
}

// GetResource returns resources defined in spec for request component, if no resources
// defined returns blank but initialized instance of resources
func (s *IBPOrderer) GetResource(comp Component) corev1.ResourceRequirements {
	if s.Spec.Resources != nil {
		switch comp {
		case INIT:
			if s.Spec.Resources.Init != nil {
				return *s.Spec.Resources.Init
			}
		case ORDERER:
			if s.Spec.Resources.Orderer != nil {
				return *s.Spec.Resources.Orderer
			}
		case GRPCPROXY:
			if s.Spec.Resources.GRPCProxy != nil {
				return *s.Spec.Resources.GRPCProxy
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
func (s *IBPOrderer) PVCName() string {
	name := s.Name + "-pvc"
	if s.Spec.CustomNames.PVC.Orderer != "" {
		name = s.Spec.CustomNames.PVC.Orderer
	}
	return name
}

func (s *IBPOrderer) GetMSPID() string {
	return s.Spec.MSPID
}

func (s *IBPOrdererSpec) NodeOUDisabled() bool {
	if s.DisableNodeOU != nil {
		return *s.DisableNodeOU
	}

	return false
}

func (s *IBPOrdererSpec) HSMSet() bool {
	if s.HSM != nil && s.HSM.PKCS11Endpoint != "" {
		return true
	}

	return false
}

func (s *IBPOrdererSpec) DomainSet() bool {
	if s.Domain != "" {
		return true
	}

	return false
}

func (s *IBPOrdererSpec) IsPrecreateOrderer() bool {
	return s.IsPrecreate != nil && *s.IsPrecreate
}

func (s *IBPOrdererSpec) IsUsingChannelLess() bool {
	return s.UseChannelLess != nil && *s.UseChannelLess
}

func (s *IBPOrdererSpec) GetNumSecondsWarningPeriod() int64 {
	daysToSecondsConversion := int64(24 * 60 * 60)
	if s.NumSecondsWarningPeriod == 0 {
		// Default to the equivalent of 30 days
		return 30 * daysToSecondsConversion
	}
	return s.NumSecondsWarningPeriod
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
}

func init() {
	SchemeBuilder.Register(&IBPOrderer{}, &IBPOrdererList{})
}

func (o *IBPOrdererStatus) HasType() bool {
	if o.CRStatus.Type != "" {
		return true
	}
	return false
}
