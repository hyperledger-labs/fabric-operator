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
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v2peer "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v2"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	v2peerconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v2"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	dep "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var testMatrix [][]resource.Quantity

var _ = Describe("Base Peer Deployment Overrides", func() {
	const (
		definitionsDir = "../../../../../definitions/peer"
	)

	var (
		overrider      *override.Override
		instance       *current.IBPPeer
		deployment     *dep.Deployment
		k8sDep         *appsv1.Deployment
		mockKubeClient *mocks.Client
	)

	BeforeEach(func() {
		var err error

		k8sDep, err = util.GetDeploymentFromFile("../../../../../definitions/peer/deployment.yaml")
		Expect(err).NotTo(HaveOccurred())
		deployment = dep.New(k8sDep)

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
			Client:                        mockKubeClient,
			DefaultCouchContainerFile:     filepath.Join(definitionsDir, "couchdb.yaml"),
			DefaultCouchInitContainerFile: filepath.Join(definitionsDir, "couchdb-init.yaml"),
			DefaultCCLauncherFile:         filepath.Join(definitionsDir, "chaincode-launcher.yaml"),
			CouchdbUser:                   "dbuser",
			CouchdbPassword:               "dbpassword",
		}
		testMatrix = [][]resource.Quantity{
			{resource.MustParse("10m"), resource.MustParse("15m"), resource.MustParse("11m"), resource.MustParse("16m"), resource.MustParse("1G"), resource.MustParse("2G")},
			{resource.MustParse("20m"), resource.MustParse("25m"), resource.MustParse("21m"), resource.MustParse("26m"), resource.MustParse("1G"), resource.MustParse("4G")},
			{resource.MustParse("30m"), resource.MustParse("35m"), resource.MustParse("31m"), resource.MustParse("36m"), resource.MustParse("3G"), resource.MustParse("6G")},
			{resource.MustParse("40m"), resource.MustParse("45m"), resource.MustParse("41m"), resource.MustParse("46m"), resource.MustParse("4G"), resource.MustParse("8G")},
			{resource.MustParse("50m"), resource.MustParse("55m"), resource.MustParse("51m"), resource.MustParse("56m"), resource.MustParse("5G"), resource.MustParse("10G")},
			{resource.MustParse("60m"), resource.MustParse("65m"), resource.MustParse("61m"), resource.MustParse("66m"), resource.MustParse("6G"), resource.MustParse("12G")},
		}

		instance = &current.IBPPeer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "peeroverride",
				Namespace: "namespace1",
			},
			Spec: current.IBPPeerSpec{
				License: current.License{
					Accept: true,
				},
				MSPID:    "peer-msp-id",
				Storage:  &current.PeerStorages{},
				Service:  &current.Service{},
				Images:   &current.PeerImages{},
				Arch:     []string{"test-arch"},
				DindArgs: []string{"--log-driver", "fluentd", "--mtu", "1480"},
				Ingress: current.Ingress{
					TlsSecretName: "tlssecret",
				},
				Zone:             "dal",
				Region:           "us-south",
				StateDb:          "couchdb",
				ImagePullSecrets: []string{"pullsecret1"},
				Resources: &current.PeerResources{
					DinD: &corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[0][0],
							corev1.ResourceMemory:           testMatrix[0][1],
							corev1.ResourceEphemeralStorage: testMatrix[0][4],
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[0][2],
							corev1.ResourceMemory:           testMatrix[0][3],
							corev1.ResourceEphemeralStorage: testMatrix[0][5],
						},
					},
					Peer: &corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[1][0],
							corev1.ResourceMemory:           testMatrix[1][1],
							corev1.ResourceEphemeralStorage: testMatrix[1][4],
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[1][2],
							corev1.ResourceMemory:           testMatrix[1][3],
							corev1.ResourceEphemeralStorage: testMatrix[1][5],
						},
					},
					GRPCProxy: &corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[2][0],
							corev1.ResourceMemory:           testMatrix[2][1],
							corev1.ResourceEphemeralStorage: testMatrix[2][4],
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[2][2],
							corev1.ResourceMemory:           testMatrix[2][3],
							corev1.ResourceEphemeralStorage: testMatrix[2][5],
						},
					},
					FluentD: &corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[3][0],
							corev1.ResourceMemory:           testMatrix[3][1],
							corev1.ResourceEphemeralStorage: testMatrix[3][4],
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[3][2],
							corev1.ResourceMemory:           testMatrix[3][3],
							corev1.ResourceEphemeralStorage: testMatrix[3][5],
						},
					},
					CouchDB: &corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[4][0],
							corev1.ResourceMemory:           testMatrix[4][1],
							corev1.ResourceEphemeralStorage: testMatrix[4][4],
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[4][2],
							corev1.ResourceMemory:           testMatrix[4][3],
							corev1.ResourceEphemeralStorage: testMatrix[4][5],
						},
					},
					CCLauncher: &corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[5][0],
							corev1.ResourceMemory:           testMatrix[5][1],
							corev1.ResourceEphemeralStorage: testMatrix[5][4],
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:              testMatrix[5][2],
							corev1.ResourceMemory:           testMatrix[5][3],
							corev1.ResourceEphemeralStorage: testMatrix[5][5],
						},
					},
				},
			},
		}
	})

	Context("create", func() {
		It("returns an error if license is not accepted", func() {
			instance.Spec.License.Accept = false
			err := overrider.Deployment(instance, k8sDep, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("user must accept license before continuing"))
		})

		It("returns an error if MSP ID not provided", func() {
			instance.Spec.MSPID = ""
			err := overrider.Deployment(instance, k8sDep, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to provide MSP ID for peer"))
		})

		It("sets default dind args if none provided", func() {
			instance.Spec.DindArgs = nil
			err := overrider.Deployment(instance, k8sDep, resources.Create)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment.Spec.Template.Spec.Containers[0].Args).To(Equal([]string{"--log-driver", "fluentd", "--log-opt", "fluentd-address=localhost:9880", "--mtu", "1400"}))
		})

		It("overrides value based on spec", func() {
			mockKubeClient.GetReturnsOnCall(1, errors.New("no inter cert found"))
			err := overrider.Deployment(instance, k8sDep, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			By("setting dind args", func() {
				Expect(len(deployment.Spec.Template.Spec.Containers[0].Args)).To(Equal(4))
			})

			By("setting service account", func() {
				Expect(deployment.Spec.Template.Spec.ServiceAccountName).To(Equal(instance.Name))
			})

			By("setting CORE_PEER_ID env var", func() {
				ev := corev1.EnvVar{
					Name:  "CORE_PEER_ID",
					Value: instance.Name,
				}
				Expect(deployment.Spec.Template.Spec.Containers[1].Env).To(ContainElement(ev))
			})

			By("setting CORE_PEER_LOCALMSPID env var", func() {
				ev := corev1.EnvVar{
					Name:  "CORE_PEER_LOCALMSPID",
					Value: instance.Spec.MSPID,
				}
				Expect(deployment.Spec.Template.Spec.Containers[1].Env).To(ContainElement(ev))
			})

			By("setting db-data volume", func() {
				v := corev1.Volume{
					Name: "db-data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: instance.Name + "-statedb-pvc",
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting CORE_LEDGER_STATE_STATEDATABASE env var", func() {
				ev := corev1.EnvVar{
					Name:  "CORE_LEDGER_STATE_STATEDATABASE",
					Value: "CouchDB",
				}
				Expect(deployment.Spec.Template.Spec.Containers[1].Env).To(ContainElement(ev))
			})

			By("setting CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME env var", func() {
				ev := corev1.EnvVar{
					Name:  "CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME",
					Value: overrider.CouchdbUser,
				}
				Expect(deployment.Spec.Template.Spec.Containers[1].Env).To(ContainElement(ev))
			})

			By("setting CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD env var", func() {
				ev := corev1.EnvVar{
					Name:  "CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD",
					Value: overrider.CouchdbPassword,
				}
				Expect(deployment.Spec.Template.Spec.Containers[1].Env).To(ContainElement(ev))
			})

			By("setting COUCHDB_USER env var", func() {
				ev := corev1.EnvVar{
					Name:  "COUCHDB_USER",
					Value: overrider.CouchdbUser,
				}
				Expect(deployment.Spec.Template.Spec.Containers[4].Env).To(ContainElement(ev))
			})

			By("setting COUCHDB_PASSWORD env var", func() {
				ev := corev1.EnvVar{
					Name:  "COUCHDB_PASSWORD",
					Value: overrider.CouchdbPassword,
				}
				Expect(deployment.Spec.Template.Spec.Containers[4].Env).To(ContainElement(ev))
			})

			By("setting SKIP_PERMISSIONS_UPDATE env var", func() {
				ev := corev1.EnvVar{
					Name:  "SKIP_PERMISSIONS_UPDATE",
					Value: "true",
				}
				Expect(deployment.Spec.Template.Spec.Containers[4].Env).To(ContainElement(ev))
			})

			By("setting image pull secret", func() {
				Expect(deployment.Spec.Template.Spec.ImagePullSecrets).To(ContainElement(corev1.LocalObjectReference{
					Name: instance.Spec.ImagePullSecrets[0],
				}))
			})

			By("setting fabric-peer-0 volume", func() {
				v := corev1.Volume{
					Name: "fabric-peer-0",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: instance.Name + "-pvc",
						},
					},
				}
				Expect(deployment.Spec.Template.Spec.Volumes).To(ContainElement(v))
			})

			By("setting fluentd-config volume", func() {
				v := corev1.Volume{
					Name: "fluentd-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: instance.Name + "-fluentd",
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
				Expect(deployment.Spec.Template.Spec.Containers[1].VolumeMounts).To(ContainElement(vm))
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

			By("setting peer-config volume", func() {
				v := corev1.Volume{
					Name: "peer-config",
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

			CommonPeerDeploymentOverrides(instance, k8sDep)
		})

		Context("images", func() {
			var (
				image *current.PeerImages
			)

			BeforeEach(func() {
				image = &current.PeerImages{
					PeerInitImage: "init-image",
					DindImage:     "dind-image",
					CouchDBImage:  "couchdb-image",
					PeerImage:     "peer-image",
					GRPCWebImage:  "proxy-image",
					FluentdImage:  "fluentd-image",
				}
				instance.Spec.Images = image
			})

			When("no tag is passed", func() {
				It("uses 'latest' for image tags", func() {
					err := overrider.Deployment(instance, k8sDep, resources.Create)
					Expect(err).NotTo(HaveOccurred())
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("dind-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[1].Image).To(Equal("peer-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[2].Image).To(Equal("proxy-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[3].Image).To(Equal("fluentd-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[4].Image).To(Equal("couchdb-image:latest"))
				})
			})

			When("tag is passed", func() {
				It("uses the passed in tag for image tags", func() {
					instance.Spec.Images = image
					image.DindTag = "1.0.1"
					image.CouchDBTag = "1.0.2"
					image.PeerTag = "1.0.3"
					image.GRPCWebTag = "1.0.4"
					image.PeerInitTag = "2.0.0"
					image.FluentdTag = "1.0.5"

					err := overrider.Deployment(instance, k8sDep, resources.Create)
					Expect(err).NotTo(HaveOccurred())
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:2.0.0"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("dind-image:1.0.1"))
					Expect(deployment.Spec.Template.Spec.Containers[1].Image).To(Equal("peer-image:1.0.3"))
					Expect(deployment.Spec.Template.Spec.Containers[2].Image).To(Equal("proxy-image:1.0.4"))
					Expect(deployment.Spec.Template.Spec.Containers[3].Image).To(Equal("fluentd-image:1.0.5"))
					Expect(deployment.Spec.Template.Spec.Containers[4].Image).To(Equal("couchdb-image:1.0.2"))
				})
			})

			Context("chaincode container", func() {
				BeforeEach(func() {
					instance.Spec.Images = &current.PeerImages{
						PeerInitImage:   "ibp-init",
						PeerInitTag:     "latest",
						CCLauncherImage: "chaincode-builder",
						CCLauncherTag:   "cclauncher-amd64",
						BuilderImage:    "ibp-ccenv",
						BuilderTag:      "builder-tag",
						GoEnvImage:      "ibp-goenv",
						GoEnvTag:        "goenv-tag",
						JavaEnvImage:    "ibp-javaenv",
						JavaEnvTag:      "javaenv-tag",
						NodeEnvImage:    "ibp-nodeenv",
						NodeEnvTag:      "nodeenv-tag",
					}
				})

				It("creates chaincode launcher container", func() {
					err := overrider.CreateCCLauncherContainer(instance, deployment)
					Expect(err).NotTo(HaveOccurred())

					ccLauncher := deployment.MustGetContainer("chaincode-launcher")

					By("setting resources from spec", func() {
						Expect(ccLauncher.Resources.Requests[corev1.ResourceCPU]).To(Equal(testMatrix[5][0]))
						Expect(ccLauncher.Resources.Requests[corev1.ResourceMemory]).To(Equal(testMatrix[5][1]))
						Expect(ccLauncher.Resources.Requests[corev1.ResourceEphemeralStorage]).To(Equal(testMatrix[5][4]))

						Expect(ccLauncher.Resources.Limits[corev1.ResourceCPU]).To(Equal(testMatrix[5][2]))
						Expect(ccLauncher.Resources.Limits[corev1.ResourceMemory]).To(Equal(testMatrix[5][3]))
						Expect(ccLauncher.Resources.Limits[corev1.ResourceEphemeralStorage]).To(Equal(testMatrix[5][5]))
					})

					By("setting envs with the requestes images/spec", func() {
						Expect(ccLauncher.Env).To(ContainElement(corev1.EnvVar{
							Name:  "FILETRANSFERIMAGE",
							Value: fmt.Sprintf("%s:%s", instance.Spec.Images.PeerInitImage, instance.Spec.Images.PeerInitTag),
						}))

						Expect(ccLauncher.Env).To(ContainElement(corev1.EnvVar{
							Name:  "BUILDERIMAGE",
							Value: fmt.Sprintf("%s:%s", instance.Spec.Images.BuilderImage, instance.Spec.Images.BuilderTag),
						}))

						Expect(ccLauncher.Env).To(ContainElement(corev1.EnvVar{
							Name:  "GOENVIMAGE",
							Value: fmt.Sprintf("%s:%s", instance.Spec.Images.GoEnvImage, instance.Spec.Images.GoEnvTag),
						}))

						Expect(ccLauncher.Env).To(ContainElement(corev1.EnvVar{
							Name:  "JAVAENVIMAGE",
							Value: fmt.Sprintf("%s:%s", instance.Spec.Images.JavaEnvImage, instance.Spec.Images.JavaEnvTag),
						}))

						Expect(ccLauncher.Env).To(ContainElement(corev1.EnvVar{
							Name:  "NODEENVIMAGE",
							Value: fmt.Sprintf("%s:%s", instance.Spec.Images.NodeEnvImage, instance.Spec.Images.NodeEnvTag),
						}))
					})
				})
			})
		})

		Context("leveldb", func() {
			BeforeEach(func() {
				instance.Spec.StateDb = "leveldb"
			})

			It("overrides value based on spec", func() {
				err := overrider.Deployment(instance, k8sDep, resources.Create)
				Expect(err).NotTo(HaveOccurred())

				By("setting volume mount env var", func() {
					vm := corev1.VolumeMount{
						Name:      "db-data",
						MountPath: "/data/peer/ledgersData/stateLeveldb/",
						SubPath:   "data",
					}
					Expect(deployment.Spec.Template.Spec.InitContainers[0].VolumeMounts).To(ContainElement(vm))
					Expect(deployment.Spec.Template.Spec.Containers[1].VolumeMounts).To(ContainElement(vm))
				})

				By("setting CORE_LEDGER_STATE_STATEDATABASE env var", func() {
					ev := corev1.EnvVar{
						Name:  "CORE_LEDGER_STATE_STATEDATABASE",
						Value: "goleveldb",
					}
					Expect(deployment.Spec.Template.Spec.Containers[1].Env).To(ContainElement(ev))
				})
			})
		})
	})

	Context("update", func() {
		BeforeEach(func() {
			var err error

			err = overrider.CreateCouchDBContainers(instance, deployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("overrides value based on spec", func() {
			err := overrider.Deployment(instance, k8sDep, resources.Update)
			Expect(err).NotTo(HaveOccurred())

			CommonPeerDeploymentOverrides(instance, k8sDep)
		})

		It("sets init container command", func() {
			err := overrider.Deployment(instance, k8sDep, resources.Update)
			Expect(err).NotTo(HaveOccurred())

			init, err := deployment.GetContainer(override.INIT)
			Expect(err).NotTo(HaveOccurred())
			cmd := "DEFAULT_PERM=775 && DEFAULT_USER=7051 && DEFAULT_GROUP=1000 "
			cmd += `&& PERM=$(stat -c "%a" /data/) && USER=$(stat -c "%u" /data/) && GROUP=$(stat -c "%g" /data/) `
			cmd += `&& if [ ${PERM} != ${DEFAULT_PERM} ] || [ ${USER} != ${DEFAULT_USER} ] || [ ${GROUP} != ${DEFAULT_GROUP} ]; `
			cmd += `then chmod -R ${DEFAULT_PERM} /data/ && chown -R -H ${DEFAULT_USER}:${DEFAULT_GROUP} /data/; fi`
			Expect(init.Command).To(Equal([]string{"bash", "-c", cmd}))
		})

		Context("images", func() {
			var (
				image *current.PeerImages
			)

			BeforeEach(func() {
				image = &current.PeerImages{
					PeerInitImage: "init-image",
					DindImage:     "dind-image",
					CouchDBImage:  "couchdb-image",
					PeerImage:     "peer-image",
					GRPCWebImage:  "proxy-image",
					FluentdImage:  "fluentd-image",
				}
				instance.Spec.Images = image
			})

			When("no tag is passed", func() {
				It("uses 'latest' for image tags", func() {
					err := overrider.Deployment(instance, k8sDep, resources.Update)
					Expect(err).NotTo(HaveOccurred())
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("dind-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[1].Image).To(Equal("peer-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[2].Image).To(Equal("proxy-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[3].Image).To(Equal("fluentd-image:latest"))
					Expect(deployment.Spec.Template.Spec.Containers[4].Image).To(Equal("couchdb-image:latest"))
				})
			})

			When("tag is passed", func() {
				It("uses the passed in tag for image tags", func() {
					image.DindTag = "1.0.1"
					image.CouchDBTag = "1.0.2"
					image.PeerTag = "1.0.3"
					image.GRPCWebTag = "1.0.4"
					image.PeerInitTag = "2.0.0"
					image.FluentdTag = "1.0.5"

					err := overrider.Deployment(instance, k8sDep, resources.Update)
					Expect(err).NotTo(HaveOccurred())
					Expect(deployment.Spec.Template.Spec.InitContainers[0].Image).To(Equal("init-image:2.0.0"))
					Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("dind-image:1.0.1"))
					Expect(deployment.Spec.Template.Spec.Containers[1].Image).To(Equal("peer-image:1.0.3"))
					Expect(deployment.Spec.Template.Spec.Containers[2].Image).To(Equal("proxy-image:1.0.4"))
					Expect(deployment.Spec.Template.Spec.Containers[3].Image).To(Equal("fluentd-image:1.0.5"))
					Expect(deployment.Spec.Template.Spec.Containers[4].Image).To(Equal("couchdb-image:1.0.2"))
				})
			})
		})

		Context("v2", func() {
			BeforeEach(func() {
				instance.Spec.FabricVersion = "2.4.1"
			})

			Context("chaincode launcher", func() {
				BeforeEach(func() {
					instance.Spec.Images = &current.PeerImages{
						CCLauncherImage: "new-cclauncher",
						CCLauncherTag:   "v2",
						PeerInitImage:   "new-peerinit",
						PeerInitTag:     "v2",
						BuilderImage:    "new-builder",
						BuilderTag:      "v2",
						GoEnvImage:      "new-goenv",
						GoEnvTag:        "v2",
						JavaEnvImage:    "new-javaenv",
						JavaEnvTag:      "v2",
						NodeEnvImage:    "new-nodeenv",
						NodeEnvTag:      "v2",
					}
				})

				It("updates", func() {
					err := overrider.Deployment(instance, k8sDep, resources.Update)
					Expect(err).NotTo(HaveOccurred())

					By("setting chaincode launcher from spec to deployment", func() {
						Expect(deployment.MustGetContainer(override.CCLAUNCHER).Image).To(Equal("new-cclauncher:v2"))
					})

					By("having a non-null cc launcher tag", func() {
						_, err = deployment.GetContainer(override.CCLAUNCHER)
						Expect(err).ToNot(HaveOccurred())
					})

					By("setting env vars with new image values", func() {
						Expect(deployment.MustGetContainer(override.CCLAUNCHER).Env).To(ContainElements(
							corev1.EnvVar{
								Name:  "FILETRANSFERIMAGE",
								Value: "new-peerinit:v2",
							},
							corev1.EnvVar{
								Name:  "BUILDERIMAGE",
								Value: "new-builder:v2",
							},
							corev1.EnvVar{
								Name:  "GOENVIMAGE",
								Value: "new-goenv:v2",
							},
							corev1.EnvVar{
								Name:  "JAVAENVIMAGE",
								Value: "new-javaenv:v2",
							},
							corev1.EnvVar{
								Name:  "NODEENVIMAGE",
								Value: "new-nodeenv:v2",
							},
						))
					})

					By("changing permissions on the /cclauncher volume", func() {
						init, err := deployment.GetContainer(override.INIT)
						Expect(err).NotTo(HaveOccurred())
						Expect(init.Command).To(HaveLen(3))
						Expect(init.Command[0]).To(Equal("bash"))
						Expect(init.Command[1]).To(Equal("-c"))
						Expect(init.Command[2]).To(Equal("DEFAULT_PERM=775 && DEFAULT_USER=7051 && DEFAULT_GROUP=1000 && PERM=$(stat -c \"%a\" /data/) && USER=$(stat -c \"%u\" /data/) && GROUP=$(stat -c \"%g\" /data/) && if [ ${PERM} != ${DEFAULT_PERM} ] || [ ${USER} != ${DEFAULT_USER} ] || [ ${GROUP} != ${DEFAULT_GROUP} ]; then chmod -R ${DEFAULT_PERM} /{data/,cclauncher/} && chown -R -H ${DEFAULT_USER}:${DEFAULT_GROUP} /{data/,cclauncher/}; fi"))
					})
				})
			})

			Context("chaincode launcher with leveldb", func() {
				BeforeEach(func() {
					instance.Spec.Images = &current.PeerImages{
						CCLauncherImage: "new-cclauncher",
						CCLauncherTag:   "v2",
					}
					instance.Spec.StateDb = "leveldb"
				})

				It("updates", func() {
					err := overrider.Deployment(instance, k8sDep, resources.Update)
					Expect(err).NotTo(HaveOccurred())

					By("having a non-null cc launcher tag", func() {
						_, err = deployment.GetContainer(override.CCLAUNCHER)
						Expect(err).ToNot(HaveOccurred())
					})

					By("setting chaincode launcher from spec to deployment", func() {
						Expect(deployment.MustGetContainer(override.CCLAUNCHER).Image).To(Equal("new-cclauncher:v2"))
					})

					By("changing permissions on the cclauncher and stateLeveldb volumes", func() {
						init, err := deployment.GetContainer(override.INIT)
						Expect(err).NotTo(HaveOccurred())
						Expect(init.Command).To(HaveLen(3))
						Expect(init.Command[0]).To(Equal("bash"))
						Expect(init.Command[1]).To(Equal("-c"))
						Expect(init.Command[2]).To(Equal("DEFAULT_PERM=775 && DEFAULT_USER=7051 && DEFAULT_GROUP=1000 && PERM=$(stat -c \"%a\" /data/) && USER=$(stat -c \"%u\" /data/) && GROUP=$(stat -c \"%g\" /data/) && if [ ${PERM} != ${DEFAULT_PERM} ] || [ ${USER} != ${DEFAULT_USER} ] || [ ${GROUP} != ${DEFAULT_GROUP} ]; then chmod -R ${DEFAULT_PERM} /{data/,data/peer/ledgersData/stateLeveldb,cclauncher/} && chown -R -H ${DEFAULT_USER}:${DEFAULT_GROUP} /{data/,data/peer/ledgersData/stateLeveldb,cclauncher/}; fi"))
					})
				})
			})

			Context("external chaincode builder", func() {
				BeforeEach(func() {
					instance.Spec.Images = &current.PeerImages{
						PeerInitImage: "new-peer-init",
						PeerInitTag:   "latest",
						PeerImage:     "hyperledger/fabric-peer",
						PeerTag:       "2.4.1",
					}

					err := overrider.Deployment(instance, k8sDep, resources.Update)
					Expect(err).NotTo(HaveOccurred())
				})

				When("a nil launcher is specified", func() {
					It("emits a deployment without a launcher sidecar", func() {
						_, err := deployment.GetContainer(override.CCLAUNCHER)
						Expect(err).To(HaveOccurred())
					})

					It("does not change permissions on the /cclauncher volume", func() {
						init, err := deployment.GetContainer(override.INIT)
						Expect(err).NotTo(HaveOccurred())
						Expect(init.Command).To(HaveLen(3))
						Expect(init.Command[0]).To(Equal("bash"))
						Expect(init.Command[1]).To(Equal("-c"))
						Expect(init.Command[2]).To(Equal("DEFAULT_PERM=775 && DEFAULT_USER=7051 && DEFAULT_GROUP=1000 && PERM=$(stat -c \"%a\" /data/) && USER=$(stat -c \"%u\" /data/) && GROUP=$(stat -c \"%g\" /data/) && if [ ${PERM} != ${DEFAULT_PERM} ] || [ ${USER} != ${DEFAULT_USER} ] || [ ${GROUP} != ${DEFAULT_GROUP} ]; then chmod -R ${DEFAULT_PERM} /data/ && chown -R -H ${DEFAULT_USER}:${DEFAULT_GROUP} /data/; fi"))
					})
				})
			})

			Context("chaincode builder config map as env", func() {
				BeforeEach(func() {
					instance.Spec.ChaincodeBuilderConfig = map[string]string{
						"peername": "org1peer1",
					}

					err := overrider.Deployment(instance, k8sDep, resources.Update)
					Expect(err).NotTo(HaveOccurred())
				})

				When("A chaincode builder config is present", func() {
					It("Sets a JSON env map in the peer deployment", func() {
						peer, err := deployment.GetContainer(override.PEER)
						Expect(err).NotTo(HaveOccurred())
						Expect(peer.Env).NotTo(BeNil())

						Expect(deployment.MustGetContainer(override.PEER).Env).To(ContainElement(corev1.EnvVar{
							Name:  "CHAINCODE_AS_A_SERVICE_BUILDER_CONFIG",
							Value: "{\"peername\":\"org1peer1\"}",
						}))
					})
				})
			})

			Context("couchbase and external builder images: regression test for Issue #3269", func() {
				BeforeEach(func() {
					instance.Spec.Images = &current.PeerImages{
						PeerInitImage: "new-peer-init",
						PeerInitTag:   "latest",
						PeerImage:     "hyperledger/fabric-peer",
						PeerTag:       "2.4.1",
					}

					instance.Spec.StateDb = "couchdb"

					err := overrider.Deployment(instance, k8sDep, resources.Update)
					Expect(err).NotTo(HaveOccurred())
				})

				When("a nil launcher is specified with a couchdb state table", func() {
					It("does not specify a bash set for the init container permission command", func() {
						init, err := deployment.GetContainer(override.INIT)
						Expect(err).NotTo(HaveOccurred())
						Expect(init.Command).To(HaveLen(3))
						Expect(init.Command[0]).To(Equal("bash"))
						Expect(init.Command[1]).To(Equal("-c"))
						Expect(init.Command[2]).To(Equal("DEFAULT_PERM=775 && DEFAULT_USER=7051 && DEFAULT_GROUP=1000 && PERM=$(stat -c \"%a\" /data/) && USER=$(stat -c \"%u\" /data/) && GROUP=$(stat -c \"%g\" /data/) && if [ ${PERM} != ${DEFAULT_PERM} ] || [ ${USER} != ${DEFAULT_USER} ] || [ ${GROUP} != ${DEFAULT_GROUP} ]; then chmod -R ${DEFAULT_PERM} /data/ && chown -R -H ${DEFAULT_USER}:${DEFAULT_GROUP} /data/; fi"))
					})
				})
			})

			Context("leveldb and external builder images: regression test for Issue #3269", func() {
				BeforeEach(func() {
					instance.Spec.Images = &current.PeerImages{
						PeerInitImage: "new-peer-init",
						PeerInitTag:   "latest",
						PeerImage:     "hyperledger/fabric-peer",
						PeerTag:       "2.4.1",
					}

					instance.Spec.StateDb = "leveldb"

					err := overrider.Deployment(instance, k8sDep, resources.Update)
					Expect(err).NotTo(HaveOccurred())
				})

				When("a nil launcher is specified with a leveldb state table", func() {
					It("specifies a bash set for the init container permission command", func() {
						init, err := deployment.GetContainer(override.INIT)
						Expect(err).NotTo(HaveOccurred())
						Expect(init.Command).To(HaveLen(3))
						Expect(init.Command[0]).To(Equal("bash"))
						Expect(init.Command[1]).To(Equal("-c"))
						Expect(init.Command[2]).To(Equal("DEFAULT_PERM=775 && DEFAULT_USER=7051 && DEFAULT_GROUP=1000 && PERM=$(stat -c \"%a\" /data/) && USER=$(stat -c \"%u\" /data/) && GROUP=$(stat -c \"%g\" /data/) && if [ ${PERM} != ${DEFAULT_PERM} ] || [ ${USER} != ${DEFAULT_USER} ] || [ ${GROUP} != ${DEFAULT_GROUP} ]; then chmod -R ${DEFAULT_PERM} /{data/,data/peer/ledgersData/stateLeveldb} && chown -R -H ${DEFAULT_USER}:${DEFAULT_GROUP} /{data/,data/peer/ledgersData/stateLeveldb}; fi"))
					})
				})
			})
		})

		Context("v24", func() {
			BeforeEach(func() {
				instance.Spec.FabricVersion = "2.4.3"
				instance.Spec.Images = &current.PeerImages{
					CCLauncherImage: "new-cclauncher",
					CCLauncherTag:   "v2",
					PeerInitImage:   "new-peerinit",
					PeerInitTag:     "v2",
					BuilderImage:    "new-builder",
					BuilderTag:      "v2",
					GoEnvImage:      "new-goenv",
					GoEnvTag:        "v2",
					JavaEnvImage:    "new-javaenv",
					JavaEnvTag:      "v2",
					NodeEnvImage:    "new-nodeenv",
					NodeEnvTag:      "v2",
				}
			})

			Context("chaincode launcher", func() {
				It("updates", func() {
					err := overrider.Deployment(instance, k8sDep, resources.Update)
					Expect(err).NotTo(HaveOccurred())

					By("setting liveliness probe to https", func() {
						Expect(deployment.MustGetContainer(override.CCLAUNCHER).LivenessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTPS))
					})

					By("setting readiness probe to https", func() {
						Expect(deployment.MustGetContainer(override.CCLAUNCHER).ReadinessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTPS))
					})
				})
			})
		})
	})

	Context("Replicas", func() {
		When("Replicas is greater than 1", func() {
			It("returns an error", func() {
				replicas := int32(2)
				instance.Spec.Replicas = &replicas
				err := overrider.Deployment(instance, k8sDep, resources.Create)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("replicas > 1 not allowed in IBPPeer"))
			})
		})
		When("Replicas is equal to 1", func() {
			It("returns success", func() {
				replicas := int32(1)
				instance.Spec.Replicas = &replicas
				err := overrider.Deployment(instance, k8sDep, resources.Create)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("Replicas is equal to 0", func() {
			It("returns success", func() {
				replicas := int32(0)
				instance.Spec.Replicas = &replicas
				err := overrider.Deployment(instance, k8sDep, resources.Create)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("Replicas is nil", func() {
			It("returns success", func() {
				instance.Spec.Replicas = nil
				err := overrider.Deployment(instance, k8sDep, resources.Create)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("HSM", func() {
		BeforeEach(func() {
			configOverride := v2peerconfig.Core{
				Core: v2peer.Core{
					Peer: v2peer.Peer{
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

			instance.Spec.ConfigOverride = &runtime.RawExtension{Raw: configBytes}
		})

		It("sets proxy env on peer container", func() {
			instance.Spec.HSM = &current.HSM{PKCS11Endpoint: "1.2.3.4"}
			err := overrider.Deployment(instance, k8sDep, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			d := dep.New(k8sDep)
			Expect(d.MustGetContainer(override.PEER).Env).To(ContainElement(corev1.EnvVar{
				Name:  "PKCS11_PROXY_SOCKET",
				Value: "1.2.3.4",
			}))
		})

		It("configures deployment to use HSM init image", func() {
			err := overrider.Deployment(instance, k8sDep, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			d := dep.New(k8sDep)
			By("setting volume mounts", func() {
				Expect(d.MustGetContainer(override.PEER).VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      "shared",
					MountPath: "/hsm/lib",
					SubPath:   "hsm",
				}))

				Expect(d.MustGetContainer(override.PEER).VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      "hsmconfig",
					MountPath: "/etc/Chrystoki.conf",
					SubPath:   "Chrystoki.conf",
				}))
			})

			By("setting env vars", func() {
				Expect(d.MustGetContainer(override.PEER).Env).To(ContainElement(corev1.EnvVar{
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

func CommonPeerDeploymentOverrides(instance *current.IBPPeer, deployment *appsv1.Deployment) {
	// Perform check after override to make sure new values are in place
	for i, c := range deployment.Spec.Template.Spec.Containers {
		By(fmt.Sprintf("setting resources for container '%s'", c.Name), func() {
			Expect(c.Resources.Requests[corev1.ResourceCPU]).To(Equal(testMatrix[i][0]))
			Expect(c.Resources.Requests[corev1.ResourceMemory]).To(Equal(testMatrix[i][1]))
			Expect(c.Resources.Requests[corev1.ResourceEphemeralStorage]).To(Equal(testMatrix[i][4]))
			Expect(c.Resources.Limits[corev1.ResourceCPU]).To(Equal(testMatrix[i][2]))
			Expect(c.Resources.Limits[corev1.ResourceMemory]).To(Equal(testMatrix[i][3]))
			Expect(c.Resources.Limits[corev1.ResourceEphemeralStorage]).To(Equal(testMatrix[i][5]))
		})
		if version.GetMajorReleaseVersion(instance.Spec.FabricVersion) == version.V2 {
			if c.Name == "peer" {
				By("string PEER_NAME in peer container", func() {
					Expect(util.EnvExists(c.Env, "PEER_NAME")).To(Equal(true))
					Expect(util.GetEnvValue(c.Env, "PEER_NAME")).To(Equal(instance.GetName()))
				})
			}
			if c.Name == "chaincode-launcher" {
				By("string CORE_PEER_LOCALMSPID in v2 chaincode-launcher container", func() {
					Expect(util.EnvExists(c.Env, "CORE_PEER_LOCALMSPID")).To(Equal(true))
					Expect(util.GetEnvValue(c.Env, "CORE_PEER_LOCALMSPID")).To(Equal(instance.Spec.MSPID))
				})
			}
		}
	}
}
