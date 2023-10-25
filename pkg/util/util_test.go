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

package util_test

import (
	"errors"

	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Util", func() {

	Context("Convert yaml file to json", func() {
		It("returns an error if file does not exist", func() {
			_, err := util.ConvertYamlFileToJson("fake.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such file"))
		})

		It("returns an error if yaml file is not properly formatted", func() {
			_, err := util.ConvertYamlFileToJson("testdata/bad.yaml")
			Expect(err).To(HaveOccurred())
		})

		It("return a byte arrary if the file exists and is a valid yaml file", func() {
			bytes, err := util.ConvertYamlFileToJson("../../definitions/peer/pvc.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(bytes)).NotTo(Equal(0))
		})
	})

	Context("GetPVCFromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetPVCFromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})

		It("reads file with PVC configuration and unmarshals into a struct", func() {
			pvc, err := util.GetPVCFromFile("../../definitions/peer/pvc.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc).NotTo(BeNil())
		})
	})

	Context("GetDeploymentFromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetDeploymentFromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})

		It("reads file with Deployment configuration and unmarshals into a struct", func() {
			dep, err := util.GetDeploymentFromFile("../../definitions/peer/deployment.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(dep).NotTo(BeNil())
		})
	})

	Context("GetServiceFromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetServiceFromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})

		It("reads file with Service configuration and unmarshals into a struct", func() {
			srvc, err := util.GetServiceFromFile("../../definitions/peer/service.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(srvc).NotTo(BeNil())
		})
	})

	Context("GetSecretFromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetSecretFromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})
		It("reads file with Service configuration and unmarshals into a struct", func() {
			srvc, err := util.GetSecretFromFile("../../testdata/secret.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(srvc).NotTo(BeNil())
		})
	})

	Context("GetIngressFromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetIngressFromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})
	})

	Context("GetIngressv1beta1FromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetIngressv1beta1FromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})
	})

	Context("GetRoleFromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetRoleFromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})

		It("reads file with Role configuration and unmarshals into a struct", func() {
			srvc, err := util.GetRoleFromFile("../../definitions/peer/role.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(srvc).NotTo(BeNil())
		})
	})

	Context("GetRoleBindingFromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetRoleBindingFromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})

		It("reads file with RoleBinding configuration and unmarshals into a struct", func() {
			srvc, err := util.GetRoleBindingFromFile("../../definitions/peer/rolebinding.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(srvc).NotTo(BeNil())
		})
	})

	Context("GetServiceAccountFromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetServiceAccountFromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})

		It("reads file with SA configuration and unmarshals into a struct", func() {
			srvc, err := util.GetServiceAccountFromFile("../../definitions/peer/serviceaccount.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(srvc).NotTo(BeNil())
		})
	})

	Context("GetCRDFromFile", func() {
		It("returns an error if config is incorrectly defined", func() {
			_, err := util.GetCRDFromFile("testdata/invalid_kind.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot unmarshal"))
		})

		It("reads file with CRD configuration and unmarshals into a struct", func() {
			srvc, err := util.GetCRDFromFile("../../config/crd/bases/ibp.com_ibpcas.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(srvc).NotTo(BeNil())
		})
	})

	Context("GetResourcePatch", func() {
		It("return resource struct that retains values that have not been modified and contains new values for if values are updated", func() {

			resourceList := make(corev1.ResourceList)
			resourceList[corev1.ResourceCPU] = resource.MustParse("0.5")
			resourceList[corev1.ResourceMemory] = resource.MustParse("5Gi")
			resourceList[corev1.ResourceEphemeralStorage] = resource.MustParse("1Gi")

			current := &corev1.ResourceRequirements{
				Requests: resourceList,
			}

			resourceList[corev1.ResourceCPU] = resource.MustParse("0.7")
			new := &corev1.ResourceRequirements{
				Requests: resourceList,
			}

			patched, err := util.GetResourcePatch(current, new)
			Expect(err).NotTo(HaveOccurred())

			cpu := patched.Requests[corev1.ResourceCPU]
			Expect(cpu.String()).To(Equal("700m"))
			memory := patched.Requests[corev1.ResourceMemory]
			Expect(memory.String()).To(Equal("5Gi"))
			ephermalStorage := patched.Requests[corev1.ResourceEphemeralStorage]
			Expect(ephermalStorage.String()).To(Equal("1Gi"))
		})
	})

	Context("already exists error", func() {
		It("returns error if it is not an already exists error", func() {
			err := util.IgnoreAlreadyExistError(errors.New("failed to create resource"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to create resource"))
		})

		It("does not return error if an already exists error", func() {
			err := util.IgnoreAlreadyExistError(errors.New("resource already exists"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("update existing env var", func() {
		var envs []corev1.EnvVar

		BeforeEach(func() {
			env := corev1.EnvVar{
				Name:  "GENERATE_GENESIS",
				Value: "false",
			}
			envs = append(envs, env)
		})

		It("updates env var if found in slice", func() {
			newEnvs := util.UpdateEnvVar("GENERATE_GENESIS", "true", envs)
			Expect(newEnvs[0].Value).To(Equal("true"))
		})
	})

	Context("env exists", func() {
		var envs []corev1.EnvVar

		BeforeEach(func() {
			env := corev1.EnvVar{
				Name:  "GENERATE_GENESIS",
				Value: "false",
			}
			envs = append(envs, env)

			env = corev1.EnvVar{
				Name:  "TEST_NAME",
				Value: "false",
			}
			envs = append(envs, env)
		})

		It("returns true if found in slice", func() {
			exists := util.EnvExists(envs, "TEST_NAME")
			Expect(exists).To(Equal(true))
		})

		It("returns false if not found in slice", func() {
			exists := util.EnvExists(envs, "FAKE_NAME")
			Expect(exists).To(Equal(false))
		})
	})

	Context("replaces (updates) env if diff", func() {
		var envs []corev1.EnvVar

		BeforeEach(func() {
			env := corev1.EnvVar{
				Name:  "GENERATE_GENESIS",
				Value: "false",
			}
			envs = append(envs, env)

			env = corev1.EnvVar{
				Name:  "TEST_NAME",
				Value: "false",
			}
			envs = append(envs, env)
		})

		It("returns env with updated replaced value", func() {
			key := "TEST_NAME"
			replace := "true"
			newEnvs, _ := util.ReplaceEnvIfDiff(envs, key, replace)
			Expect(newEnvs[1].Value).To(Equal("true"))
		})
	})

	Context("Resource Validation", func() {
		It("returns an error if controller handling the request is reconciling a resource of different type", func() {
			typemeta := metav1.TypeMeta{
				Kind: "NOTIBPCA",
			}
			maxNameLength := 50
			err := util.ValidationChecks(typemeta, metav1.ObjectMeta{}, "IBPCA", &maxNameLength)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not an IBPCA kind resource, please check to make sure there are no name collisions across resources"))
		})

		It("returns an error if the instance name is greater than maxNameLength", func() {
			typemeta := metav1.TypeMeta{
				Kind: "IBPCA",
			}
			objectmeta := metav1.ObjectMeta{
				Name: "012345678901234567890123456789",
			}
			maxNameLength := 25
			err := util.ValidationChecks(typemeta, objectmeta, "IBPCA", &maxNameLength)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is too long, the name must be less than or equal to "))
		})

		It("returns an error if the instance name is greater than default name length", func() {
			typemeta := metav1.TypeMeta{
				Kind: "IBPCA",
			}
			objectmeta := metav1.ObjectMeta{
				Name: "0123456789012345678901234567890123",
			}
			err := util.ValidationChecks(typemeta, objectmeta, "IBPCA", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is too long, the name must be less than or equal to"))
		})
	})

	Context("HSM proxy endpoint validation", func() {
		It("returns no error for a valid endpoint", func() {
			err := util.ValidateHSMProxyURL("tcp://0.0.0.0:2348")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns no error for a valid TLS endpoint", func() {
			err := util.ValidateHSMProxyURL("tls://0.0.0.0:2348")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error for incomplete endpoint", func() {
			err := util.ValidateHSMProxyURL("tcp://0.0.0.0")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("must specify both IP address and port"))
		})

		It("returns an error for missing port", func() {
			err := util.ValidateHSMProxyURL("tcp://0.0.0.0:")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("missing port"))
		})

		It("returns an error for missing IP address", func() {
			err := util.ValidateHSMProxyURL("tcp://:2348")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("missing IP address"))
		})

		It("returns an error for invalid scheme", func() {
			err := util.ValidateHSMProxyURL("http://0.0.0.0:8888")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unsupported scheme 'http', only tcp and tls are supported"))
		})
	})

	Context("append image pull secret if missing", func() {
		var (
			pullSecrets []corev1.LocalObjectReference
		)

		BeforeEach(func() {
			pullSecrets = []corev1.LocalObjectReference{
				corev1.LocalObjectReference{
					Name: "pullsecret1",
				},
			}
		})

		It("appends new image pull secret", func() {
			new := corev1.LocalObjectReference{Name: "pullsecret2"}
			pullSecrets := util.AppendImagePullSecretIfMissing(pullSecrets, new)
			Expect(len(pullSecrets)).To(Equal(2))
			Expect(pullSecrets[1]).To(Equal(new))
		})

		It("does not append existing image pull secret", func() {
			new := corev1.LocalObjectReference{Name: "pullsecret1"}
			pullSecrets := util.AppendImagePullSecretIfMissing(pullSecrets, new)
			Expect(len(pullSecrets)).To(Equal(1))
			Expect(pullSecrets[0].Name).To(Equal("pullsecret1"))
		})

		It("does not appen blank image pull secret", func() {
			new := corev1.LocalObjectReference{}
			pullSecrets := util.AppendImagePullSecretIfMissing(pullSecrets, new)
			Expect(len(pullSecrets)).To(Equal(1))
			Expect(pullSecrets[0].Name).To(Equal("pullsecret1"))
		})
	})

	Context("Image format verification", func() {
		var (
			img         string
			registryURL string
			defaultImg  string
		)

		BeforeEach(func() {
			registryURL = "ghcr.io/hyperledger-labs/"
			img = "fabric-operator"
			defaultImg = "ghcr.io/hyperledger-labs/fabric-peer"
		})

		It("Use Registry URL and image tag when default image tag", func() {
			resultImg := image.GetImage(registryURL, img, "")
			Expect(resultImg).To(Equal(registryURL + img))
		})

		It("Use Default Image tag when RegistryURL", func() {
			resultImg := image.GetImage("", "", defaultImg)
			Expect(resultImg).To(Equal(defaultImg))
		})

		It("Use Default Image when everything is passed", func() {
			resultImg := image.GetImage(registryURL, img, defaultImg)
			Expect(resultImg).To(Equal(defaultImg))
		})
		It("Use default Image with registry URL when image is missing", func() {
			defaultImg = "fabric-peer"
			resultImg := image.GetImage(registryURL, "", defaultImg)
			Expect(resultImg).To(Equal(registryURL + defaultImg))
		})
	})
})
