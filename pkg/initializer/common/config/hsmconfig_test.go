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

package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("HSM Config", func() {
	var hsmConfig *config.HSMConfig

	BeforeEach(func() {
		hsmConfig = &config.HSMConfig{
			Type:    "hsm",
			Version: "v1",
			MountPaths: []config.MountPath{
				config.MountPath{
					Name:      "hsmcrypto",
					Secret:    "hsmcrypto",
					MountPath: "/hsm",
					Paths: []config.Path{
						{
							Key:  "cert.pem",
							Path: "cert.pem",
						},
						{
							Key:  "key.pem",
							Path: "key.pem",
						},
					},
				},
				config.MountPath{
					Name:      "hsmconfig",
					Secret:    "hsmcrypto",
					MountPath: "/etc/Chrystoki.conf",
					SubPath:   "Chrystoki.conf",
				},
			},
			Envs: []corev1.EnvVar{
				{
					Name:  "env1",
					Value: "env1value",
				},
			},
		}
	})

	Context("volume mounts", func() {
		It("builds volume mounts from config", func() {
			vms := hsmConfig.GetVolumeMounts()
			Expect(vms).To(ContainElements(
				corev1.VolumeMount{
					Name:      "hsmcrypto",
					MountPath: "/hsm",
				},
				corev1.VolumeMount{
					Name:      "hsmconfig",
					MountPath: "/etc/Chrystoki.conf",
					SubPath:   "Chrystoki.conf",
				},
			))
		})
	})

	Context("volumes", func() {
		It("builds volumes from config", func() {
			v := hsmConfig.GetVolumes()
			Expect(v).To(ContainElements(
				corev1.Volume{
					Name: "hsmcrypto",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "hsmcrypto",
							Items: []corev1.KeyToPath{
								{
									Key:  "cert.pem",
									Path: "cert.pem",
								},
								{
									Key:  "key.pem",
									Path: "key.pem",
								},
							},
						},
					},
				},
				corev1.Volume{
					Name: "hsmconfig",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "hsmcrypto",
						},
					},
				},
			))
		})
	})

	Context("env vars", func() {
		It("builds env vars from config", func() {
			envs := hsmConfig.GetEnvs()
			Expect(envs).To(ContainElements(
				corev1.EnvVar{
					Name:  "env1",
					Value: "env1value",
				},
			))
		})
	})

	Context("build pull secret", func() {
		It("returns empty LocalObjectReference obj if pull secret not passed in config", func() {
			ps := hsmConfig.BuildPullSecret()
			Expect(ps).To(Equal(corev1.LocalObjectReference{}))
		})

		It("returns LocalObjectReference with pull secret from config", func() {
			hsmConfig.Library.Auth = &config.Auth{
				ImagePullSecret: "pullsecret",
			}
			ps := hsmConfig.BuildPullSecret()
			Expect(ps.Name).To(Equal("pullsecret"))
		})
	})
})
