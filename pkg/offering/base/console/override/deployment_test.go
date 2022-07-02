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
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Base Console Deployment Overrides", func() {
	Context("Deployment", func() {
		var (
			overrider                  *override.Override
			instance, instanceWithTags *current.IBPConsole
			deployment                 *appsv1.Deployment
			err                        error
			usetagsFlag                bool
		)

		BeforeEach(func() {
			overrider = &override.Override{}

			instance = &current.IBPConsole{
				Spec: current.IBPConsoleSpec{
					License: current.License{
						Accept: true,
					},
					ServiceAccountName:   "test",
					AuthScheme:           "couchdb",
					DeployerTimeout:      30000,
					Components:           "athena-components",
					Sessions:             "athena-sessions",
					System:               "athena-system",
					ConnectionString:     "test.com",
					Service:              &current.Service{},
					Email:                "xyz@ibm.com",
					PasswordSecretName:   "secret",
					Password:             "cGFzc3dvcmQ=",
					KubeconfigSecretName: "kubeconfig-secret",
					SystemChannel:        "testchainid",
					ImagePullSecrets:     []string{"testsecret"},
					Images: &current.ConsoleImages{
						ConsoleInitImage:   "fake-init-image",
						ConsoleInitTag:     "1234",
						CouchDBImage:       "fake-couchdb-image",
						CouchDBTag:         "1234",
						ConsoleImage:       "fake-console-image",
						ConsoleTag:         "1234",
						ConfigtxlatorImage: "fake-configtxlator-image",
						ConfigtxlatorTag:   "1234",
						DeployerImage:      "fake-deployer-image",
						DeployerTag:        "1234",
					},
					RegistryURL: "ghcr.io/ibm-blockchain/",
					NetworkInfo: &current.NetworkInfo{
						Domain:      "test.domain",
						ConsolePort: 31010,
						ProxyPort:   31011,
					},
					TLSSecretName: "secret",
					Resources:     &current.ConsoleResources{},
					Storage: &current.ConsoleStorage{
						Console: &current.StorageSpec{
							Size:  "100m",
							Class: "manual",
						},
					},
				},
			}
			deployment, err = util.GetDeploymentFromFile("../../../../../definitions/console/deployment.yaml")
			Expect(err).NotTo(HaveOccurred())
			usetagsFlag = true
			instanceWithTags = &current.IBPConsole{
				Spec: current.IBPConsoleSpec{
					License: current.License{
						Accept: true,
					},
					ServiceAccountName:   "test",
					AuthScheme:           "couchdb",
					DeployerTimeout:      30000,
					Components:           "athena-components",
					Sessions:             "athena-sessions",
					System:               "athena-system",
					ConnectionString:     "test.com",
					Service:              &current.Service{},
					Email:                "xyz@ibm.com",
					PasswordSecretName:   "secret",
					Password:             "cGFzc3dvcmQ=",
					KubeconfigSecretName: "kubeconfig-secret",
					SystemChannel:        "testchainid",
					ImagePullSecrets:     []string{"testsecret"},
					Images: &current.ConsoleImages{
						ConsoleInitImage:   "fake-init-image",
						ConsoleInitTag:     "1234",
						CouchDBImage:       "fake-couchdb-image",
						CouchDBTag:         "1234",
						ConsoleImage:       "fake-console-image",
						ConsoleTag:         "1234",
						ConfigtxlatorImage: "fake-configtxlator-image",
						ConfigtxlatorTag:   "1234",
						DeployerImage:      "fake-deployer-image",
						DeployerTag:        "1234",
						MustgatherImage:    "fake-mustgather-image",
						MustgatherTag:      "1234",
					},
					RegistryURL: "ghcr.io/ibm-blockchain/",
					NetworkInfo: &current.NetworkInfo{
						Domain:      "test.domain",
						ConsolePort: 31010,
						ProxyPort:   31011,
					},
					TLSSecretName: "secret",
					Resources:     &current.ConsoleResources{},
					Storage: &current.ConsoleStorage{
						Console: &current.StorageSpec{
							Size:  "100m",
							Class: "manual",
						},
					},
					UseTags: &usetagsFlag,
				},
			}
		})

		Context("create", func() {
			It("overrides values based on spec", func() {
				err := overrider.Deployment(instanceWithTags, deployment, resources.Create)
				Expect(err).NotTo(HaveOccurred())

				By("setting service account name", func() {
					Expect(deployment.Spec.Template.Spec.ServiceAccountName).To(Equal(instanceWithTags.Name))
				})

				By("image pull secret", func() {
					Expect(deployment.Spec.Template.Spec.ImagePullSecrets).To(Equal([]corev1.LocalObjectReference{
						corev1.LocalObjectReference{
							Name: instanceWithTags.Spec.ImagePullSecrets[0],
						},
					}))
				})

				By("setting DEFAULT_USER_PASSWORD_INITIAL env var", func() {
					envVar := corev1.EnvVar{
						Name: "DEFAULT_USER_PASSWORD_INITIAL",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: instanceWithTags.Spec.PasswordSecretName,
								},
								Key: "password",
							},
						},
					}
					Expect(deployment.Spec.Template.Spec.Containers[0].Env).To(ContainElement(envVar))
				})

				By("setting TLS volume and volume mount if TLS secret name provided in spec", func() {
					vm := corev1.VolumeMount{
						Name:      "tls-certs",
						MountPath: "/certs/tls",
					}
					Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(vm))

					v := corev1.Volume{
						Name: "tls-certs",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: instanceWithTags.Spec.TLSSecretName,
							},
						},
					}
					Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
				})

				By("setting deployer volume", func() {
					v := corev1.Volume{
						Name: "deployer-template",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: instanceWithTags.Name + "-deployer",
								},
							},
						},
					}
					Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
				})

				By("setting console volume", func() {
					v := corev1.Volume{
						Name: "template",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: instanceWithTags.Name + "-console",
								},
							},
						},
					}
					Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
				})

				By("setting affinity", func() {
					expectedAffinity := overrider.GetAffinity(instanceWithTags)
					Expect(deployment.Spec.Template.Spec.Affinity).To(Equal(expectedAffinity))
				})

				ConsoleDeploymentCommonOverrides(instanceWithTags, deployment)
			})

			Context("using couchdb", func() {
				BeforeEach(func() {
					instance.Spec.ConnectionString = "localhost"
				})

				It("overrides values based on spec", func() {
					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())

					By("setting couchdb TLS volume and volume mount", func() {
						vm := corev1.VolumeMount{
							Name:      "couchdb",
							MountPath: "/opt/couchdb/data",
							SubPath:   "data",
						}
						Expect(deployment.Spec.Template.Spec.Containers[3].VolumeMounts).To(ContainElement(vm))
						Expect(deployment.Spec.Template.Spec.InitContainers[0].VolumeMounts).To(ContainElement(vm))
					})

					By("setting cert volume and volume mount", func() {
						vm := corev1.VolumeMount{
							Name:      "couchdb",
							MountPath: "/certs/",
							SubPath:   "tls",
						}
						Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(vm))
						Expect(deployment.Spec.Template.Spec.InitContainers[0].VolumeMounts).To(ContainElement(vm))
					})
				})
			})

			Context("not using TLS secret name", func() {
				BeforeEach(func() {
					instance.Spec.TLSSecretName = ""
				})

				It("overrides values based on spec", func() {
					vm := corev1.VolumeMount{
						Name:      "tls-certs",
						MountPath: "/certs/tls",
					}
					Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts).NotTo(ContainElement(vm))

					v := corev1.Volume{
						Name: "tls-certs",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: instance.Spec.TLSSecretName,
							},
						},
					}
					Expect(deployment.Spec.Template.Spec.Volumes).NotTo(ContainElement(v))
				})
			})
		})

		Context("enabling activity tracker", func() {
			It("overrides mounts based on spec overrides", func() {
				consoleOverride := &current.ConsoleOverridesConsole{
					ActivityTrackerConsolePath: "fake/path",
					ActivityTrackerHostPath:    "host/path",
				}
				consoleBytes, err := json.Marshal(consoleOverride)
				Expect(err).NotTo(HaveOccurred())
				instance.Spec.ConfigOverride = &current.ConsoleOverrides{
					Console: &runtime.RawExtension{Raw: consoleBytes},
				}

				err = overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				vm := corev1.VolumeMount{
					Name:      "activity",
					MountPath: "fake/path",
					SubPath:   "",
				}
				Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(vm))
				hostPathType := corev1.HostPathDirectoryOrCreate
				v := corev1.Volume{
					Name: "activity",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "host/path",
							Type: &hostPathType,
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))

				By("adding to init container command", func() {
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Command).To(Equal([]string{
						"sh",
						"-c",
						"chmod -R 775 fake/path && chown -R -H 1000:1000 fake/path",
					}))
				})
			})

			It("overrides mounts based on spec overrides when only console path provided", func() {
				consoleOverride := &current.ConsoleOverridesConsole{
					ActivityTrackerConsolePath: "fake/path",
				}
				consoleBytes, err := json.Marshal(consoleOverride)
				Expect(err).NotTo(HaveOccurred())
				instance.Spec.ConfigOverride = &current.ConsoleOverrides{
					Console: &runtime.RawExtension{Raw: consoleBytes},
				}

				err = overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				vm := corev1.VolumeMount{
					Name:      "activity",
					MountPath: "fake/path",
					SubPath:   "",
				}
				Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(vm))
				hostPathType := corev1.HostPathDirectoryOrCreate
				v := corev1.Volume{
					Name: "activity",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/var/log/at",
							Type: &hostPathType,
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			It("adds command to init container command correctly when not using remote DB", func() {
				consoleOverride := &current.ConsoleOverridesConsole{
					ActivityTrackerConsolePath: "fake/path",
					ActivityTrackerHostPath:    "host/path",
				}
				consoleBytes, err := json.Marshal(consoleOverride)
				Expect(err).NotTo(HaveOccurred())
				instance.Spec.ConfigOverride = &current.ConsoleOverrides{
					Console: &runtime.RawExtension{Raw: consoleBytes},
				}
				instance.Spec.ConnectionString = "localhost"

				err = overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).NotTo(HaveOccurred())

				By("appending to init container command", func() {
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Command).To(Equal([]string{
						"sh",
						"-c",
						"chmod -R 775 /opt/couchdb/data/ && chown -R -H 5984:5984 /opt/couchdb/data/ && chmod -R 775 /certs/ && chown -R -H 1000:1000 /certs/ && chmod -R 775 fake/path && chown -R -H 1000:1000 fake/path",
					}))
				})
			})
		})

		// TODO:OSS
		// as both the console and deployer defaults are blank
		// Context("update", func() {
		// 	It("doesn't overrides images and tags values, when usetags flag is not set", func() {
		// 		err := overrider.Deployment(instance, deployment, resources.Update)
		// 		Expect(err).NotTo(HaveOccurred())
		// 		Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(ContainSubstring("ghcr.io/ibm-blockchain/fake-console-image@sha256"))
		// 		Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(ContainSubstring("ghcr.io/ibm-blockchain/fake-init-image@sha256"))
		// 		Expect(deployment.Spec.Template.Spec.Containers[1].Image).To(ContainSubstring("ghcr.io/ibm-blockchain/fake-deployer-image@sha256"))
		// 		Expect(deployment.Spec.Template.Spec.Containers[2].Image).To(ContainSubstring("ghcr.io/ibm-blockchain/fake-configtxlator-image@sha256"))
		// 	})
		// })

		Context("update when usetags set", func() {
			It("overrides values based on spec, when usetags flag is set", func() {
				err := overrider.Deployment(instanceWithTags, deployment, resources.Update)
				Expect(err).NotTo(HaveOccurred())
				ConsoleDeploymentCommonOverrides(instanceWithTags, deployment)
			})
		})

		Context("Replicas", func() {
			When("using remote db", func() {
				It("using replica value from spec", func() {
					replicas := int32(2)
					instance.Spec.Replicas = &replicas
					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())
					Expect(*deployment.Spec.Replicas).To(Equal(replicas))
				})
			})

			When("Replicas is greater than 1", func() {
				BeforeEach(func() {
					instance.Spec.ConnectionString = "localhost"
				})

				It("returns an error", func() {
					replicas := int32(2)
					instance.Spec.Replicas = &replicas
					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("replicas > 1 not allowed in IBPConsole"))
				})
			})

			When("Replicas is equal to 1", func() {
				It("returns success", func() {
					replicas := int32(1)
					instance.Spec.Replicas = &replicas
					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())
				})
			})
			When("Replicas is equal to 0", func() {
				It("returns success", func() {
					replicas := int32(0)
					instance.Spec.Replicas = &replicas
					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())
				})
			})
			When("Replicas is nil", func() {
				It("returns success", func() {
					instance.Spec.Replicas = nil
					err := overrider.Deployment(instance, deployment, resources.Create)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})
})

func ConsoleDeploymentCommonOverrides(instance *current.IBPConsole, dep *appsv1.Deployment) {
	By("setting init image", func() {
		Expect(dep.Spec.Template.Spec.InitContainers[0].Image).To(Equal(fmt.Sprintf("%s%s:%s", instance.Spec.RegistryURL, instance.Spec.Images.ConsoleInitImage, instance.Spec.Images.ConsoleInitTag)))
	})

	By("setting console image", func() {
		Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal(fmt.Sprintf("%s%s:%s", instance.Spec.RegistryURL, instance.Spec.Images.ConsoleImage, instance.Spec.Images.ConsoleTag)))
	})

	By("setting deployer image", func() {
		Expect(dep.Spec.Template.Spec.Containers[1].Image).To(Equal(fmt.Sprintf("%s%s:%s", instance.Spec.RegistryURL, instance.Spec.Images.DeployerImage, instance.Spec.Images.DeployerTag)))
	})

	By("setting configtxlator image", func() {
		Expect(dep.Spec.Template.Spec.Containers[2].Image).To(Equal(fmt.Sprintf("%s%s:%s", instance.Spec.RegistryURL, instance.Spec.Images.ConfigtxlatorImage, instance.Spec.Images.ConfigtxlatorTag)))
	})

	By("setting replicas", func() {
		Expect(dep.Spec.Replicas).To(Equal(instance.Spec.Replicas))
	})

	By("setting KUBECONFIGPATH env var", func() {
		envVar := corev1.EnvVar{
			Name:  "KUBECONFIGPATH",
			Value: "/kubeconfig/kubeconfig.yaml",
		}
		Expect(dep.Spec.Template.Spec.Containers[1].Env).To(ContainElement(envVar))
	})
}
