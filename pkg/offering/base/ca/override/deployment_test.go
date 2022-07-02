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

package override_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	dep "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Deployment Overrides", func() {
	var (
		overrider      *override.Override
		instance       *current.IBPCA
		deployment     *appsv1.Deployment
		mockKubeClient *mocks.Client
	)

	BeforeEach(func() {
		var err error

		mockKubeClient = &mocks.Client{}
		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *corev1.ConfigMap:
				hsmConfig := &config.HSMConfig{
					Type:    "hsm",
					Version: "v1",
					MountPaths: []config.MountPath{
						config.MountPath{
							Name:      "hsmcrypto",
							Secret:    "hsmcrypto",
							MountPath: "/hsm",
							Paths: []config.Path{
								{
									Key:  "cafile.pem",
									Path: "cafile.pem",
								},
								{
									Key:  "cert.pem",
									Path: "cert.pem",
								},
								{
									Key:  "key.pem",
									Path: "key.pem",
								},
								{
									Key:  "server.pem",
									Path: "server.pem",
								},
							},
						},
						config.MountPath{
							Name:      "hsmconfig",
							Secret:    "hsmcrypto",
							MountPath: "/etc/Chrystoki.conf",
							SubPath:   "Chrystoki.conf",
							Paths: []config.Path{
								{
									Key:  "Chrystoki.conf",
									Path: "Chrystoki.conf",
								},
							},
						},
					},
					Envs: []corev1.EnvVar{
						{
							Name:  "env1",
							Value: "env1value",
						},
					},
				}

				configBytes, err := yaml.Marshal(hsmConfig)
				if err != nil {
					return err
				}
				o := obj.(*corev1.ConfigMap)
				o.Data = map[string]string{"ibp-hsm-config.yaml": string(configBytes)}
			}
			return nil
		}

		overrider = &override.Override{
			Client: mockKubeClient,
		}
		deployment, err = util.GetDeploymentFromFile("../../../../../definitions/ca/deployment.yaml")
		Expect(err).NotTo(HaveOccurred())
		deployment.Spec.Template.Spec.InitContainers[0].Image = "fake-init-image:1234"
		deployment.Spec.Template.Spec.Containers[0].Image = "fake-ca-image:1234"

		instance = &current.IBPCA{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "override1",
				Namespace: "namespace1",
			},
			Spec: current.IBPCASpec{
				License: current.License{
					Accept: true,
				},
				Storage: &current.CAStorages{},
				Service: &current.Service{},
				Images: &current.CAImages{
					CAImage:     "ca-image",
					CAInitImage: "init-image",
				},
				Arch:             []string{"test-arch"},
				Zone:             "dal",
				Region:           "us-south",
				ImagePullSecrets: []string{"pullsecret"},
				Resources: &current.CAResources{
					CA: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:              resource.MustParse("0.6m"),
							corev1.ResourceMemory:           resource.MustParse("0.4m"),
							corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:              resource.MustParse("0.7m"),
							corev1.ResourceMemory:           resource.MustParse("0.5m"),
							corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
						},
					},
				},
			},
		}
	})

	When("creating a new deployment", func() {
		It("returns an error if license is not accepted", func() {
			instance.Spec.License.Accept = false
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("user must accept license before continuing"))
		})

		It("overrides values in deployment based on CA's instance spec", func() {
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			By("setting service account name to be name of CA instance", func() {
				Expect(deployment.Spec.Template.Spec.ServiceAccountName).To(Equal(instance.Name))
			})

			By("setting image pull secret", func() {
				Expect(deployment.Spec.Template.Spec.ImagePullSecrets[0].Name).To(Equal(instance.Spec.ImagePullSecrets[0]))
			})

			By("setting resources", func() {
				updated, err := util.GetResourcePatch(&corev1.ResourceRequirements{}, instance.Spec.Resources.CA)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Template.Spec.Containers[0].Resources).To(Equal(*updated))
			})

			By("setting affinity", func() {
				affinity := corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								corev1.NodeSelectorTerm{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										corev1.NodeSelectorRequirement{
											Key:      "kubernetes.io/arch",
											Operator: corev1.NodeSelectorOpIn,
											Values:   instance.Spec.Arch,
										},
										corev1.NodeSelectorRequirement{
											Key:      "topology.kubernetes.io/zone",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{instance.Spec.Zone},
										},
										corev1.NodeSelectorRequirement{
											Key:      "topology.kubernetes.io/region",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{instance.Spec.Region},
										},
									},
								},
								corev1.NodeSelectorTerm{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										corev1.NodeSelectorRequirement{
											Key:      "failure-domain.beta.kubernetes.io/zone",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{instance.Spec.Zone},
										},
										corev1.NodeSelectorRequirement{
											Key:      "failure-domain.beta.kubernetes.io/region",
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{instance.Spec.Region},
										},
									},
								},
							},
						},
					},
				}
				affinity.PodAntiAffinity = &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						corev1.WeightedPodAffinityTerm{
							Weight: 100,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{
										metav1.LabelSelectorRequirement{
											Key:      "app",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{instance.Name},
										},
									},
								},
								TopologyKey: "topology.kubernetes.io/zone",
							},
						},
						corev1.WeightedPodAffinityTerm{
							Weight: 100,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{
										metav1.LabelSelectorRequirement{
											Key:      "app",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{instance.Name},
										},
									},
								},
								TopologyKey: "failure-domain.beta.kubernetes.io/zone",
							},
						},
					},
				}
				Expect(*deployment.Spec.Template.Spec.Affinity).To(Equal(affinity))
			})

			By("volumes creating a ca crypto volume", func() {
				volume := corev1.Volume{
					Name: "ca-crypto",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: instance.Name + "-ca-crypto",
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(volume))
			})

			By("volumes creating a tlsca crypto volume", func() {
				volume := corev1.Volume{
					Name: "tlsca-crypto",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: instance.Name + "-tlsca-crypto",
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(volume))
			})

			By("volumes creating a ca config volume", func() {
				volume := corev1.Volume{
					Name: "ca-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: instance.Name + "-ca-config",
							},
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(volume))
			})

			By("volumes creating a tlsca config volume", func() {
				volume := corev1.Volume{
					Name: "tlsca-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: instance.Name + "-tlsca-config",
							},
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(volume))
			})
		})

		Context("images", func() {
			When("no tag is passed", func() {
				It("uses 'latest' for image tags", func() {
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("fake-init-image:1234"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("fake-ca-image:1234"))

					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("ca-image:latest"))
				})
			})

			When("tag is passed", func() {
				It("uses the passed in tag for image tags", func() {
					instance.Spec.Images.CAInitTag = "2.0.0"
					instance.Spec.Images.CATag = "1.0.0"

					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:2.0.0"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("ca-image:1.0.0"))
				})
			})
		})

		Context("database overrides", func() {
			When("not using postgres", func() {
				It("performs overrides", func() {
					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())

					By("creating a PVC volume", func() {
						volume := corev1.Volume{
							Name: "fabric-ca",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: instance.Name + "-pvc",
								},
							},
						}
						Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(volume))
					})

					By("creating a volume mount for both init and ca containers", func() {
						volumeMount := corev1.VolumeMount{
							Name:      "fabric-ca",
							MountPath: "/data",
							SubPath:   "fabric-ca-server",
						}
						Expect(deployment.Spec.Template.Spec.InitContainers[0].VolumeMounts).To(ContainElement(volumeMount))
						Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(volumeMount))
					})
				})
			})

			When("using postgres", func() {
				BeforeEach(func() {
					instance.Spec.ConfigOverride = &current.ConfigOverride{
						CA:    &runtime.RawExtension{},
						TLSCA: &runtime.RawExtension{},
					}

					caConfig := &v1.ServerConfig{
						CAConfig: v1.CAConfig{
							DB: &v1.CAConfigDB{
								Type: "postgres",
							},
						},
					}

					caConfigJson, err := util.ConvertToJsonMessage(caConfig)
					Expect(err).NotTo(HaveOccurred())
					instance.Spec.ConfigOverride.CA = &runtime.RawExtension{Raw: *caConfigJson}
				})

				It("performs overrides", func() {
					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())

					By("creating a volume mount for both init and ca containers", func() {
						volumeMount := corev1.VolumeMount{
							Name:      "shared",
							MountPath: "/data",
						}
						Expect(deployment.Spec.Template.Spec.InitContainers[0].VolumeMounts).To(ContainElement(volumeMount))
						Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(volumeMount))
					})

					By("setting strategy to rolling update", func() {
						Expect(deployment.Spec.Strategy.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
					})
				})
			})
		})

		Context("replicas is greater than 1", func() {
			BeforeEach(func() {
				replicas := int32(2)
				instance.Spec.Replicas = &replicas
			})

			It("returns an error if db is not set in CA override", func() {
				err := overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Failed to provide override configuration to support greater than 1 replicas"))

			})

			It("returns an error if db is set to not equal postgres in CA override", func() {
				ca := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type: "mysql",
						},
					},
				}
				caJson, err := util.ConvertToJsonMessage(ca)
				Expect(err).NotTo(HaveOccurred())

				instance.Spec.ConfigOverride = &current.ConfigOverride{
					CA: &runtime.RawExtension{Raw: *caJson},
				}
				err = overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("DB Type in CA config override should be `postgres` to allow replicas > 1"))
			})

			It("returns an error if datasource is empty in CA override", func() {
				ca := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type: "postgres",
						},
					},
				}
				caJson, err := util.ConvertToJsonMessage(ca)
				Expect(err).NotTo(HaveOccurred())

				instance.Spec.ConfigOverride = &current.ConfigOverride{
					CA:    &runtime.RawExtension{Raw: *caJson},
					TLSCA: &runtime.RawExtension{Raw: *caJson},
				}

				err = overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Datasource in CA config override should not be empty to allow replicas > 1"))
			})

			It("returns an error if db is not set in TLSCA override", func() {
				ca := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type:       "postgres",
							Datasource: "datasource",
						},
					},
				}
				caJson, err := util.ConvertToJsonMessage(ca)
				Expect(err).NotTo(HaveOccurred())

				instance.Spec.ConfigOverride = &current.ConfigOverride{
					CA: &runtime.RawExtension{Raw: *caJson},
				}

				err = overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Failed to provide database configuration for TLSCA to support greater than 1 replicas"))
			})

			It("returns an error if db is set to not equal postgres in TLSCA override", func() {
				ca := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type:       "postgres",
							Datasource: "fake",
						},
					},
				}
				caJson, err := util.ConvertToJsonMessage(ca)
				Expect(err).NotTo(HaveOccurred())

				tlsca := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type: "mysql",
						},
					},
				}
				tlscaJson, err := util.ConvertToJsonMessage(tlsca)
				Expect(err).NotTo(HaveOccurred())

				instance.Spec.ConfigOverride = &current.ConfigOverride{
					CA:    &runtime.RawExtension{Raw: *caJson},
					TLSCA: &runtime.RawExtension{Raw: *tlscaJson},
				}
				err = overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("DB Type in TLSCA config override should be `postgres` to allow replicas > 1"))
			})

			It("returns an error if datasource is empty in TLSCA override", func() {
				ca := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type:       "postgres",
							Datasource: "fake",
						},
					},
				}
				caJson, err := util.ConvertToJsonMessage(ca)
				Expect(err).NotTo(HaveOccurred())

				tlsca := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type: "postgres",
						},
					},
				}
				tlscaJson, err := util.ConvertToJsonMessage(tlsca)
				Expect(err).NotTo(HaveOccurred())

				instance.Spec.ConfigOverride = &current.ConfigOverride{
					CA:    &runtime.RawExtension{Raw: *caJson},
					TLSCA: &runtime.RawExtension{Raw: *tlscaJson},
				}

				err = overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Datasource in TLSCA config override should not be empty to allow replicas > 1"))
			})

			It("returns no error if db is set to postgres", func() {
				ca := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type:       "postgres",
							Datasource: "fake",
						},
					},
				}
				caBytes, err := json.Marshal(ca)
				Expect(err).NotTo(HaveOccurred())
				caJson := json.RawMessage(caBytes)

				tlsca := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type:       "postgres",
							Datasource: "fake",
						},
					},
				}
				tlscaBytes, err := json.Marshal(tlsca)
				Expect(err).NotTo(HaveOccurred())
				tlscaJson := json.RawMessage(tlscaBytes)

				instance.Spec.ConfigOverride = &current.ConfigOverride{
					CA:    &runtime.RawExtension{Raw: caJson},
					TLSCA: &runtime.RawExtension{Raw: tlscaJson},
				}

				err = overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Strategy.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
			})
		})

		Context("Replicas is nil", func() {
			It("returns success", func() {
				instance.Spec.Replicas = nil
				err := overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	When("updating a deployment", func() {
		Context("images", func() {
			var image *current.CAImages

			BeforeEach(func() {
				image = &current.CAImages{
					CAImage:     "ca-image",
					CAInitImage: "init-image",
				}
				instance.Spec.Images = image
			})

			When("no tag is passed", func() {
				It("uses 'latest' for image tags", func() {
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("fake-init-image:1234"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("fake-ca-image:1234"))

					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("ca-image:latest"))
				})
			})

			When("tag is passed", func() {
				It("uses the passed in tag for image tags", func() {
					image.CATag = "1.0.0"
					image.CAInitTag = "2.0.0"

					err := overrider.Deployment(instance, deployment, resources.Update)
					Expect(err).NotTo(HaveOccurred())
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:2.0.0"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("ca-image:1.0.0"))
				})
			})
		})
	})

	Context("replicas is greater than 1", func() {
		BeforeEach(func() {
			replicas := int32(2)
			instance.Spec.Replicas = &replicas
		})

		It("returns an error if db is not set in CA override", func() {
			err := overrider.Deployment(instance, deployment, resources.Update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Failed to provide override configuration to support greater than 1 replicas"))

		})

		It("returns an error if db is set to not equal postgres in CA override", func() {
			ca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type: "mysql",
					},
				},
			}
			caBytes, err := json.Marshal(ca)
			Expect(err).NotTo(HaveOccurred())
			caJson := json.RawMessage(caBytes)

			instance.Spec.ConfigOverride = &current.ConfigOverride{
				CA: &runtime.RawExtension{Raw: caJson},
			}
			err = overrider.Deployment(instance, deployment, resources.Update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("DB Type in CA config override should be `postgres` to allow replicas > 1"))
		})

		It("returns an error if datasource is empty in CA override", func() {
			ca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type: "postgres",
					},
				},
			}
			caBytes, err := json.Marshal(ca)
			Expect(err).NotTo(HaveOccurred())

			tlsca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type: "postgres",
					},
				},
			}
			tlscaBytes, err := json.Marshal(tlsca)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.ConfigOverride = &current.ConfigOverride{
				CA:    &runtime.RawExtension{Raw: caBytes},
				TLSCA: &runtime.RawExtension{Raw: tlscaBytes},
			}

			err = overrider.Deployment(instance, deployment, resources.Update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Datasource in CA config override should not be empty to allow replicas > 1"))
		})

		It("returns an error if db is not set in TLSCA override", func() {
			ca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type:       "postgres",
						Datasource: "datasource",
					},
				},
			}
			caBytes, err := json.Marshal(ca)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.ConfigOverride = &current.ConfigOverride{
				CA: &runtime.RawExtension{Raw: caBytes},
			}

			err = overrider.Deployment(instance, deployment, resources.Update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Failed to provide database configuration for TLSCA to support greater than 1 replicas"))
		})

		It("returns an error if db is set to not equal postgres in TLSCA override", func() {
			ca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type:       "postgres",
						Datasource: "fake",
					},
				},
			}
			caBytes, err := json.Marshal(ca)
			Expect(err).NotTo(HaveOccurred())

			tlsca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type: "mysql",
					},
				},
			}
			tlscaBytes, err := json.Marshal(tlsca)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.ConfigOverride = &current.ConfigOverride{
				CA:    &runtime.RawExtension{Raw: caBytes},
				TLSCA: &runtime.RawExtension{Raw: tlscaBytes},
			}

			err = overrider.Deployment(instance, deployment, resources.Update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("DB Type in TLSCA config override should be `postgres` to allow replicas > 1"))
		})

		It("returns an error if datasource is empty in TLSCA override", func() {
			ca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type:       "postgres",
						Datasource: "fake",
					},
				},
			}
			caBytes, err := json.Marshal(ca)
			Expect(err).NotTo(HaveOccurred())

			tlsca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type: "postgres",
					},
				},
			}
			tlscaBytes, err := json.Marshal(tlsca)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.ConfigOverride = &current.ConfigOverride{
				CA:    &runtime.RawExtension{Raw: caBytes},
				TLSCA: &runtime.RawExtension{Raw: tlscaBytes},
			}

			err = overrider.Deployment(instance, deployment, resources.Update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Datasource in TLSCA config override should not be empty to allow replicas > 1"))
		})

		It("returns no error if db is set to postgres", func() {
			ca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type:       "postgres",
						Datasource: "fake",
					},
				},
			}
			caJson, err := util.ConvertToJsonMessage(ca)
			Expect(err).NotTo(HaveOccurred())

			tlsca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type:       "postgres",
						Datasource: "fake",
					},
				},
			}
			tlscaJson, err := util.ConvertToJsonMessage(tlsca)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.ConfigOverride = &current.ConfigOverride{
				CA:    &runtime.RawExtension{Raw: *caJson},
				TLSCA: &runtime.RawExtension{Raw: *tlscaJson},
			}

			err = overrider.Deployment(instance, deployment, resources.Update)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment.Spec.Strategy.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
		})
	})

	Context("HSM", func() {
		BeforeEach(func() {
			ca := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					CSP: &v1.BCCSP{
						ProviderName: "PKCS11",
						PKCS11: &v1.PKCS11Opts{
							Label: "partition1",
							Pin:   "B6T9Q7mGNG",
						},
					},
				},
			}
			caJson, err := util.ConvertToJsonMessage(ca)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.ConfigOverride = &current.ConfigOverride{
				CA: &runtime.RawExtension{Raw: *caJson},
			}
		})

		It("sets proxy env on ca container", func() {
			instance.Spec.HSM = &current.HSM{PKCS11Endpoint: "1.2.3.4"}
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			d := dep.New(deployment)
			Expect(d.MustGetContainer(override.CA).Env).To(ContainElement(corev1.EnvVar{
				Name:  "PKCS11_PROXY_SOCKET",
				Value: "1.2.3.4",
			}))
		})

		It("configures deployment to use HSM init image", func() {
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			d := dep.New(deployment)
			By("setting volume mounts", func() {
				Expect(d.MustGetContainer(override.CA).VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      "shared",
					MountPath: "/hsm/lib",
					SubPath:   "hsm",
				}))

				Expect(d.MustGetContainer(override.CA).VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      "hsmconfig",
					MountPath: "/etc/Chrystoki.conf",
					SubPath:   "Chrystoki.conf",
				}))
			})

			By("setting env vars", func() {
				Expect(d.MustGetContainer(override.CA).Env).To(ContainElement(corev1.EnvVar{
					Name:  "env1",
					Value: "env1value",
				}))
			})

			By("creating HSM init container", func() {
				Expect(d.ContainerExists("hsm-client")).To(Equal(true))
			})
		})
	})
})
