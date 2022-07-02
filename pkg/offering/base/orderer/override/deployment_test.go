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
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v2orderer "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v2"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	v2ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v2"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	dep "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("Base Orderer Deployment Overrides", func() {
	var (
		overrider      *override.Override
		instance       *current.IBPOrderer
		deployment     *appsv1.Deployment
		mockKubeClient *mocks.Client
	)

	BeforeEach(func() {
		var err error

		deployment, err = util.GetDeploymentFromFile("../../../../../definitions/orderer/deployment.yaml")
		Expect(err).NotTo(HaveOccurred())

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

		replicas := int32(1)
		instance = &current.IBPOrderer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ordereroverride",
				Namespace: "namespace1",
			},
			Spec: current.IBPOrdererSpec{
				License: current.License{
					Accept: true,
				},
				OrgName:         "orderermsp",
				MSPID:           "orderermsp",
				OrdererType:     "solo",
				ExternalAddress: "0.0.0.0",
				GenesisProfile:  "Initial",
				Storage:         &current.OrdererStorages{},
				Service:         &current.Service{},
				Images: &current.OrdererImages{
					OrdererInitImage: "fake-init-image",
					OrdererInitTag:   "1234",
					OrdererImage:     "fake-orderer-image",
					OrdererTag:       "1234",
					GRPCWebImage:     "fake-grpcweb-image",
					GRPCWebTag:       "1234",
				},
				SystemChannelName: "testchainid",
				Arch:              []string{"test-arch"},
				Zone:              "dal",
				Region:            "us-south",
				ImagePullSecrets:  []string{"pullsecret1"},
				Replicas:          &replicas,
				Resources: &current.OrdererResources{
					Orderer: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:              resource.MustParse("0.6m"),
							corev1.ResourceMemory:           resource.MustParse("0.4m"),
							corev1.ResourceEphemeralStorage: resource.MustParse("0.1m"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:              resource.MustParse("0.7m"),
							corev1.ResourceMemory:           resource.MustParse("0.5m"),
							corev1.ResourceEphemeralStorage: resource.MustParse("0.5m"),
						},
					},
					GRPCProxy: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:              resource.MustParse("0.1m"),
							corev1.ResourceMemory:           resource.MustParse("0.2m"),
							corev1.ResourceEphemeralStorage: resource.MustParse("0.1m"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:              resource.MustParse("0.3m"),
							corev1.ResourceMemory:           resource.MustParse("0.4m"),
							corev1.ResourceEphemeralStorage: resource.MustParse("0.5m"),
						},
					},
				},
			},
		}
	})

	Context("create", func() {
		It("returns an error if license is not accepted", func() {
			instance.Spec.License.Accept = false
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("user must accept license before continuing"))
		})

		It("returns an error if value for Orderer Type not provided", func() {
			instance.Spec.OrdererType = ""
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Orderer Type not provided"))
		})

		It("returns an error if value for System Channel Name not provided", func() {
			instance.Spec.SystemChannelName = ""
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("System Channel Name not provided"))
		})

		It("returns an error if value for Org Name not provided", func() {
			instance.Spec.OrgName = ""
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Orderer Org Name not provided"))
		})

		It("returns an error if value for External Address not provided", func() {
			instance.Spec.ExternalAddress = ""
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("External Address not set"))
		})

		It("overrides values based on spec", func() {
			mockKubeClient.GetReturnsOnCall(1, errors.New("no inter cert found"))
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			By("setting pull secret", func() {
				Expect(deployment.Spec.Template.Spec.ImagePullSecrets).To(Equal([]corev1.LocalObjectReference{corev1.LocalObjectReference{
					Name: instance.Spec.ImagePullSecrets[0],
				}}))
			})

			By("setting env from", func() {
				envFrom := corev1.EnvFromSource{
					ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: instance.Name + "-env",
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Containers[0].EnvFrom).To(ContainElement(envFrom))
			})

			By("setting orderer-data volume", func() {
				volume := corev1.Volume{
					Name: "orderer-data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: instance.Name + "-pvc",
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(volume))
			})

			By("setting EXTERNAL_ADDRESS env var on grpcweb container", func() {
				ev := corev1.EnvVar{
					Name:  "EXTERNAL_ADDRESS",
					Value: instance.Spec.ExternalAddress,
				}
				Expect(deployment.Spec.Template.Spec.Containers[1].Env).To(ContainElement(ev))
			})

			By("setting ecert admincerts volume and volume mount", func() {
				v := corev1.Volume{
					Name: "ecert-admincerts",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: fmt.Sprintf("ecert-%s-admincerts", instance.Name),
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))

				vm := corev1.VolumeMount{
					Name:      "ecert-admincerts",
					MountPath: "/certs/msp/admincerts",
				}
				Expect(deployment.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(vm))
			})

			By("setting ecert cacerts volume", func() {
				v := corev1.Volume{
					Name: "ecert-cacerts",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: fmt.Sprintf("ecert-%s-cacerts", instance.Name),
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting ecert keystore volume", func() {
				v := corev1.Volume{
					Name: "ecert-keystore",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: fmt.Sprintf("ecert-%s-keystore", instance.Name),
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting ecert signcert volume", func() {
				v := corev1.Volume{
					Name: "ecert-signcert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: fmt.Sprintf("ecert-%s-signcert", instance.Name),
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting tls cacerts volume", func() {
				v := corev1.Volume{
					Name: "tls-cacerts",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: fmt.Sprintf("tls-%s-cacerts", instance.Name),
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting tls keystore volume", func() {
				v := corev1.Volume{
					Name: "tls-keystore",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: fmt.Sprintf("tls-%s-keystore", instance.Name),
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting tls signcert volume", func() {
				v := corev1.Volume{
					Name: "tls-signcert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: fmt.Sprintf("tls-%s-signcert", instance.Name),
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting orderer-genesis volume", func() {
				v := corev1.Volume{
					Name: "orderer-genesis",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: fmt.Sprintf("%s-genesis", instance.Name),
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting orderer-config volume", func() {
				v := corev1.Volume{
					Name: "orderer-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: instance.Name + "-config",
							},
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting affinity", func() {
				expectedAffinity := overrider.GetAffinity(instance)
				Expect(deployment.Spec.Template.Spec.Affinity).To(Equal(expectedAffinity))
			})

			OrdererDeploymentCommonOverrides(instance, deployment)
		})

		It("overrides values based on whether disableProbes is set to true", func() {
			overrider.Config = &operatorconfig.Config{
				Operator: operatorconfig.Operator{
					Orderer: operatorconfig.Orderer{
						DisableProbes: "true",
					},
				},
			}
			mockKubeClient.GetReturnsOnCall(1, errors.New("no inter cert found"))
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			By("setting probe values to nil when IBPOPERATOR_ORDERER_DISABLE_PROBES is set to true", func() {
				d := dep.New(deployment)
				Expect(d.MustGetContainer(override.ORDERER).ReadinessProbe).To(BeNil())
				Expect(d.MustGetContainer(override.ORDERER).LivenessProbe).To(BeNil())
				Expect(d.MustGetContainer(override.ORDERER).StartupProbe).To(BeNil())
			})

		})
	})

	Context("update", func() {
		It("overrides values based on spec", func() {
			err := overrider.Deployment(instance, deployment, resources.Update)
			Expect(err).NotTo(HaveOccurred())

			OrdererDeploymentCommonOverrides(instance, deployment)
		})
	})

	Context("Replicas", func() {
		When("Replicas is greater than 1", func() {
			It("returns an error", func() {
				replicas := int32(2)
				instance.Spec.Replicas = &replicas
				err := overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("replicas > 1 not allowed in IBPOrderer"))
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

	Context("images", func() {
		var image *current.OrdererImages

		BeforeEach(func() {
			image = &current.OrdererImages{
				OrdererInitImage: "init-image",
				OrdererImage:     "orderer-image",
				GRPCWebImage:     "grpcweb-image",
			}
			instance.Spec.Images = image
		})

		When("no tag is passed", func() {
			It("uses 'latest' for image tags", func() {
				err := overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:latest"))
				Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("orderer-image:latest"))
				Expect(deployment.Spec.Template.Spec.Containers[1].Image).To(Equal("grpcweb-image:latest"))
			})
		})

		When("tag is passed", func() {
			It("uses the passed in tag for image tags", func() {
				image.OrdererInitTag = "1.0.0"
				image.OrdererTag = "2.0.0"
				image.GRPCWebTag = "3.0.0"

				err := overrider.Deployment(instance, deployment, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:1.0.0"))
				Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("orderer-image:2.0.0"))
				Expect(deployment.Spec.Template.Spec.Containers[1].Image).To(Equal("grpcweb-image:3.0.0"))
			})
		})
	})

	Context("HSM", func() {
		BeforeEach(func() {
			configOverride := v2ordererconfig.Orderer{
				Orderer: v2orderer.Orderer{
					General: v2orderer.General{
						BCCSP: &common.BCCSP{
							ProviderName: "PKCS11",
							PKCS11: &common.PKCS11Opts{
								Label: "partition1",
								Pin:   "B6T9Q7mGNG",
							},
						},
					},
				},
			}

			configBytes, err := json.Marshal(configOverride)
			Expect(err).NotTo(HaveOccurred())
			configRaw := json.RawMessage(configBytes)

			instance.Spec.ConfigOverride = &runtime.RawExtension{Raw: configRaw}
		})

		It("sets proxy env on orderer container", func() {
			instance.Spec.HSM = &current.HSM{PKCS11Endpoint: "1.2.3.4"}
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			d := dep.New(deployment)
			Expect(d.MustGetContainer(override.ORDERER).Env).To(ContainElement(corev1.EnvVar{
				Name:  "PKCS11_PROXY_SOCKET",
				Value: "1.2.3.4",
			}))
		})

		It("configures deployment to use HSM init image", func() {
			err := overrider.Deployment(instance, deployment, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			d := dep.New(deployment)
			By("setting volume mounts", func() {
				Expect(d.MustGetContainer(override.ORDERER).VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      "shared",
					MountPath: "/hsm/lib",
					SubPath:   "hsm",
				}))

				Expect(d.MustGetContainer(override.ORDERER).VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      "hsmconfig",
					MountPath: "/etc/Chrystoki.conf",
					SubPath:   "Chrystoki.conf",
				}))
			})

			By("setting env vars", func() {
				Expect(d.MustGetContainer(override.ORDERER).Env).To(ContainElement(corev1.EnvVar{
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

func OrdererDeploymentCommonOverrides(instance *current.IBPOrderer, dep *appsv1.Deployment) {
	By("setting orderer resources", func() {
		r, err := util.GetResourcePatch(&corev1.ResourceRequirements{}, instance.Spec.Resources.Orderer)
		Expect(err).NotTo(HaveOccurred())
		Expect(dep.Spec.Template.Spec.Containers[0].Resources).To(Equal(*r))
	})

	By("setting grpcweb resources", func() {
		r, err := util.GetResourcePatch(&corev1.ResourceRequirements{}, instance.Spec.Resources.GRPCProxy)
		Expect(err).NotTo(HaveOccurred())
		Expect(dep.Spec.Template.Spec.Containers[1].Resources).To(Equal(*r))
	})

	By("setting init image", func() {
		Expect(dep.Spec.Template.Spec.InitContainers[0].Image).To(Equal(fmt.Sprintf("%s:%s", instance.Spec.Images.OrdererInitImage, instance.Spec.Images.OrdererInitTag)))
	})

	By("setting orderer image", func() {
		Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal(fmt.Sprintf("%s:%s", instance.Spec.Images.OrdererImage, instance.Spec.Images.OrdererTag)))
	})

	By("setting grpcweb image", func() {
		Expect(dep.Spec.Template.Spec.Containers[1].Image).To(Equal(fmt.Sprintf("%s:%s", instance.Spec.Images.GRPCWebImage, instance.Spec.Images.GRPCWebTag)))
	})

	By("setting replicas", func() {
		Expect(dep.Spec.Replicas).To(Equal(instance.Spec.Replicas))
	})
}
