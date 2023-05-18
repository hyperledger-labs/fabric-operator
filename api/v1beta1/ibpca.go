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
	"os"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:generate=false
type CAConfig interface {
	UsingPKCS11() bool
}

func (s *IBPCA) ResetRestart() {
	s.Spec.Action.Restart = false
}

func (s *IBPCA) ResetTLSRenew() {
	s.Spec.Action.Renew.TLSCert = false
}

func (s *IBPCA) UsingHSMProxy() bool {
	if s.Spec.HSM != nil && s.Spec.HSM.PKCS11Endpoint != "" {
		return true
	}
	return false
}

func (s *IBPCA) IsHSMEnabled() bool {
	return s.isCAHSMEnabled() || s.isTLSCAHSMEnabled()
}

func (s *IBPCA) IsHSMEnabledForType(caType config.Type) bool {
	switch caType {
	case config.EnrollmentCA:
		return s.isCAHSMEnabled()
	case config.TLSCA:
		return s.isTLSCAHSMEnabled()
	}
	return false
}

func (s *IBPCA) isCAHSMEnabled() bool {
	configOverride, err := s.Spec.GetCAConfigOverride()
	if err != nil {
		return false
	}

	return configOverride.UsingPKCS11()
}

func (s *IBPCA) isTLSCAHSMEnabled() bool {
	configOverride, err := s.GetTLSCAConfigOverride()
	if err != nil {
		return false
	}

	return configOverride.UsingPKCS11()
}

func (s *IBPCA) GetTLSCAConfigOverride() (CAConfig, error) {
	if s.Spec.ConfigOverride == nil || s.Spec.ConfigOverride.TLSCA == nil {
		return &config.Config{}, nil
	}

	configOverride, err := config.ReadFrom(&s.Spec.ConfigOverride.TLSCA.Raw)
	if err != nil {
		return nil, err
	}

	return configOverride, nil
}

func (s *IBPCA) GetNumSecondsWarningPeriod() int64 {
	if s.Spec.NumSecondsWarningPeriod == 0 {
		// Default to the equivalent of 30 days
		daysToSecondsConversion := int64(24 * 60 * 60)
		return 30 * daysToSecondsConversion
	}
	return s.Spec.NumSecondsWarningPeriod
}

func (s *IBPCA) GetPullSecrets() []corev1.LocalObjectReference {
	pullSecrets := []corev1.LocalObjectReference{}
	for _, ps := range s.Spec.ImagePullSecrets {
		pullSecrets = append(pullSecrets, corev1.LocalObjectReference{Name: ps})
	}
	return pullSecrets
}

func (s *IBPCA) GetRegistryURL() string {
	return s.Spec.RegistryURL
}

func (s *IBPCA) GetArch() []string {
	return s.Spec.Arch
}

func (s *IBPCA) GetLabels() map[string]string {
	label := os.Getenv("OPERATOR_LABEL_PREFIX")
	if label == "" {
		label = "fabric"
	}

	return map[string]string{
		"app":                          s.GetName(),
		"creator":                      label,
		"release":                      "operator",
		"helm.sh/chart":                "ibm-" + label,
		"app.kubernetes.io/name":       label,
		"app.kubernetes.io/instance":   label + "ca",
		"app.kubernetes.io/managed-by": label + "-operator",
	}
}

// GetFabricVersion returns fabric version from CR spec
func (s *IBPCA) GetFabricVersion() string {
	return s.Spec.FabricVersion
}

// SetFabricVersion sets fabric version on spec
func (s *IBPCA) SetFabricVersion(version string) {
	s.Spec.FabricVersion = version
}

// ImagesSet returns true if the spec has images defined
func (s *IBPCA) ImagesSet() bool {
	return s.Spec.Images != nil
}

// GetResource returns resources defined in spec for request component, if no resources
// defined returns blank but initialized instance of resources
func (s *IBPCA) GetResource(comp Component) corev1.ResourceRequirements {
	if s.Spec.Resources != nil {
		switch comp {
		case INIT:
			if s.Spec.Resources.Init != nil {
				return *s.Spec.Resources.Init
			}
		case CA:
			if s.Spec.Resources.CA != nil {
				return *s.Spec.Resources.CA
			}
		case ENROLLER:
			if s.Spec.Resources.EnrollJob != nil {
				return *s.Spec.Resources.EnrollJob
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
func (s *IBPCA) PVCName() string {
	name := s.Name + "-pvc"
	if s.Spec.CustomNames.PVC.CA != "" {
		name = s.Spec.CustomNames.PVC.CA
	}
	return name
}

// GetMSPID returns empty string as we don't currently store
// the orgname/MSPID of the CA in its spec
func (s *IBPCA) GetMSPID() string {
	// no-op
	return ""
}

func (s *IBPCASpec) HSMSet() bool {
	if s.HSM != nil && s.HSM.PKCS11Endpoint != "" {
		return true
	}

	return false
}

func (s *IBPCASpec) DomainSet() bool {

	return s.Domain != ""
}

func (s *IBPCASpec) CAResourcesSet() bool {
	if s.Resources != nil {
		if s.Resources.CA != nil {
			return true
		}
	}

	return false
}

func (s *IBPCASpec) InitResourcesSet() bool {
	if s.Resources != nil {
		if s.Resources.Init != nil {
			return true
		}
	}

	return false
}

func (s *IBPCASpec) GetCAConfigOverride() (CAConfig, error) {
	if s.ConfigOverride == nil || s.ConfigOverride.CA == nil {
		return &config.Config{}, nil
	}

	configOverride, err := config.ReadFrom(&s.ConfigOverride.CA.Raw)
	if err != nil {
		return nil, err
	}
	return configOverride, nil
}

func (c *IBPCAStatus) HasType() bool {

	return c.CRStatus.Type != ""
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
}

func init() {
	SchemeBuilder.Register(&IBPCA{}, &IBPCAList{})
}
