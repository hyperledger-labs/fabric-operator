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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"

	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("Base Console Deployer Config Map Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPConsole
		cm        *corev1.ConfigMap
	)

	BeforeEach(func() {
		var err error
		overrider = &override.Override{}
		instance = &current.IBPConsole{
			Spec: current.IBPConsoleSpec{
				ImagePullSecrets: []string{"pullsecret"},
				ConnectionString: "connectionString1",
				Storage: &current.ConsoleStorage{
					Console: &current.StorageSpec{
						Class: "sc1",
					},
				},
				NetworkInfo: &current.NetworkInfo{
					Domain: "domain1",
				},
				Versions: &current.Versions{
					CA: map[string]current.VersionCA{
						"v1-0": current.VersionCA{
							Default: true,
							Version: "v1-0",
							Image: current.CAImages{
								CAInitImage: "ca-init-image",
								CAInitTag:   "1.0.0",
								CAImage:     "ca-image",
								CATag:       "1.0.0",
							},
						},
						"v2-0": current.VersionCA{
							Default: false,
							Version: "v2-0",
							Image: current.CAImages{
								CAInitImage: "ca-init-image",
								CAInitTag:   "2.0.0",
								CAImage:     "ca-image",
								CATag:       "2.0.0",
							},
						},
					},
					Peer: map[string]current.VersionPeer{
						"v1-0": current.VersionPeer{
							Default: true,
							Version: "v1-0",
							Image: current.PeerImages{
								PeerInitImage: "peer-init-image",
								PeerInitTag:   "1.0.0",
								PeerImage:     "peer-image",
								PeerTag:       "1.0.0",
								DindImage:     "dind-iamge",
								DindTag:       "1.0.0",
								GRPCWebImage:  "grpcweb-image",
								GRPCWebTag:    "1.0.0",
								FluentdImage:  "fluentd-image",
								FluentdTag:    "1.0.0",
								CouchDBImage:  "couchdb-image",
								CouchDBTag:    "1.0.0",
							},
						},
						"v2-0": current.VersionPeer{
							Default: false,
							Version: "v2-0",
							Image: current.PeerImages{
								PeerInitImage:   "peer-init-image",
								PeerInitTag:     "2.0.0",
								PeerImage:       "peer-image",
								PeerTag:         "2.0.0",
								DindImage:       "dind-iamge",
								DindTag:         "2.0.0",
								GRPCWebImage:    "grpcweb-image",
								GRPCWebTag:      "2.0.0",
								FluentdImage:    "fluentd-image",
								FluentdTag:      "2.0.0",
								CouchDBImage:    "couchdb-image",
								CouchDBTag:      "2.0.0",
								CCLauncherImage: "cclauncher-image",
								CCLauncherTag:   "2.0.0",
							},
						},
					},
					Orderer: map[string]current.VersionOrderer{
						"v1-0": current.VersionOrderer{
							Default: true,
							Version: "v1-0",
							Image: current.OrdererImages{
								OrdererInitImage: "orderer-init-image",
								OrdererInitTag:   "1.0.0",
								OrdererImage:     "orderer-image",
								OrdererTag:       "1.0.0",
								GRPCWebImage:     "grpcweb-image",
								GRPCWebTag:       "1.0.0",
							},
						},
						"v2-0": current.VersionOrderer{
							Default: false,
							Version: "v2-0",
							Image: current.OrdererImages{
								OrdererInitImage: "orderer-init-image",
								OrdererInitTag:   "2.0.0",
								OrdererImage:     "orderer-image",
								OrdererTag:       "2.0.0",
								GRPCWebImage:     "grpcweb-image",
								GRPCWebTag:       "2.0.0",
							},
						},
					},
				},
				CRN: &current.CRN{
					CName:       "cname",
					CType:       "ctype",
					Location:    "location1",
					Servicename: "Servicename1",
					Version:     "version1",
					AccountID:   "id123",
				},
				Deployer: &current.Deployer{
					ConnectionString: "connectionstring2",
				},
				Images: &current.ConsoleImages{
					MustgatherImage: "test-image",
					MustgatherTag:   "test-tag",
				},
			},
		}
		cm, err = util.GetConfigMapFromFile("../../../../../testdata/deployercm/deployer-configmap.yaml")
		Expect(err).NotTo(HaveOccurred())
	})

	Context("create", func() {
		It("returns an error if base create function called", func() {
			err := overrider.DeployerCM(instance, cm, resources.Create, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no create deployer cm defined, this needs to implemented"))
		})
	})

	Context("update", func() {
		It("return an error if no image pull secret provided", func() {
			instance.Spec.ImagePullSecrets = []string{}
			err := overrider.DeployerCM(instance, cm, resources.Update, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no image pull secret provided"))
		})

		It("return an error if no domain provided", func() {
			instance.Spec.NetworkInfo.Domain = ""
			err := overrider.DeployerCM(instance, cm, resources.Update, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no domain provided"))
		})

		It("overrides values based on spec", func() {
			err := overrider.DeployerCM(instance, cm, resources.Update, nil)
			Expect(err).NotTo(HaveOccurred())

			config := &deployer.Config{}

			err = yaml.Unmarshal([]byte(cm.Data["settings.yaml"]), config)
			Expect(err).NotTo(HaveOccurred())

			By("setting image pull secret", func() {
				Expect(config.ImagePullSecrets).To(Equal(instance.Spec.ImagePullSecrets))
			})

			By("setting connection string", func() {
				Expect(config.Database.ConnectionURL).To(Equal(instance.Spec.Deployer.ConnectionString))
			})

			By("setting versions", func() {
				expectedVersions := &current.Versions{
					CA: map[string]current.VersionCA{
						"v1-0": current.VersionCA{
							Default: true,
							Version: "v1-0",
							Image: current.CAImages{
								CAInitImage: "ca-init-image",
								CAInitTag:   "1.0.0",
								CAImage:     "ca-image",
								CATag:       "1.0.0",
							},
						},
						"v2-0": current.VersionCA{
							Default: false,
							Version: "v2-0",
							Image: current.CAImages{
								CAInitImage: "ca-init-image",
								CAInitTag:   "2.0.0",
								CAImage:     "ca-image",
								CATag:       "2.0.0",
							},
						},
					},
					Peer: map[string]current.VersionPeer{
						"v1-0": current.VersionPeer{
							Default: true,
							Version: "v1-0",
							Image: current.PeerImages{
								PeerInitImage: "peer-init-image",
								PeerInitTag:   "1.0.0",
								PeerImage:     "peer-image",
								PeerTag:       "1.0.0",
								DindImage:     "dind-iamge",
								DindTag:       "1.0.0",
								GRPCWebImage:  "grpcweb-image",
								GRPCWebTag:    "1.0.0",
								FluentdImage:  "fluentd-image",
								FluentdTag:    "1.0.0",
								CouchDBImage:  "couchdb-image",
								CouchDBTag:    "1.0.0",
							},
						},
						"v2-0": current.VersionPeer{
							Default: false,
							Version: "v2-0",
							Image: current.PeerImages{
								PeerInitImage:   "peer-init-image",
								PeerInitTag:     "2.0.0",
								PeerImage:       "peer-image",
								PeerTag:         "2.0.0",
								DindImage:       "dind-iamge",
								DindTag:         "2.0.0",
								GRPCWebImage:    "grpcweb-image",
								GRPCWebTag:      "2.0.0",
								FluentdImage:    "fluentd-image",
								FluentdTag:      "2.0.0",
								CouchDBImage:    "couchdb-image",
								CouchDBTag:      "2.0.0",
								CCLauncherImage: "cclauncher-image",
								CCLauncherTag:   "2.0.0",
							},
						},
					},
					Orderer: map[string]current.VersionOrderer{
						"v1-0": current.VersionOrderer{
							Default: true,
							Version: "v1-0",
							Image: current.OrdererImages{
								OrdererInitImage: "orderer-init-image",
								OrdererInitTag:   "1.0.0",
								OrdererImage:     "orderer-image",
								OrdererTag:       "1.0.0",
								GRPCWebImage:     "grpcweb-image",
								GRPCWebTag:       "1.0.0",
							},
						},
						"v2-0": current.VersionOrderer{
							Default: false,
							Version: "v2-0",
							Image: current.OrdererImages{
								OrdererInitImage: "orderer-init-image",
								OrdererInitTag:   "2.0.0",
								OrdererImage:     "orderer-image",
								OrdererTag:       "2.0.0",
								GRPCWebImage:     "grpcweb-image",
								GRPCWebTag:       "2.0.0",
							},
						},
					},
				}

				typeConvertedVersions := &current.Versions{}
				util.ConvertSpec(config.Versions, typeConvertedVersions)
				Expect(typeConvertedVersions).To(Equal(expectedVersions))
			})

			By("setting storage class name", func() {
				Expect(config.Defaults.Storage.CA.CA.Class).To(Equal(instance.Spec.Storage.Console.Class))
				Expect(config.Defaults.Storage.Peer.Peer.Class).To(Equal(instance.Spec.Storage.Console.Class))
				Expect(config.Defaults.Storage.Peer.StateDB.Class).To(Equal(instance.Spec.Storage.Console.Class))
				Expect(config.Defaults.Storage.Orderer.Orderer.Class).To(Equal(instance.Spec.Storage.Console.Class))
			})

			By("setting CRN", func() {
				crn := &current.CRN{
					CName:       instance.Spec.CRN.CName,
					CType:       instance.Spec.CRN.CType,
					Location:    instance.Spec.CRN.Location,
					Servicename: instance.Spec.CRN.Servicename,
					Version:     instance.Spec.CRN.Version,
					AccountID:   instance.Spec.CRN.AccountID,
				}
				Expect(config.CRN).To(Equal(crn))
			})

			By("setting domain", func() {
				Expect(config.Domain).To(Equal(instance.Spec.NetworkInfo.Domain))
			})

			By("setting must gather images", func() {
				Expect(config.OtherImages.MustgatherImage).To(Equal("test-image"))
				Expect(config.OtherImages.MustgatherTag).To(Equal("test-tag"))
			})
		})

		It("should get default versions if overrides are not passed", func() {
			instance.Spec.Versions = nil
			err := overrider.DeployerCM(instance, cm, resources.Update, nil)
			Expect(err).NotTo(HaveOccurred())

			config := &deployer.Config{}

			err = yaml.Unmarshal([]byte(cm.Data["settings.yaml"]), config)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("common overrides", func() {
		Context("version overrides", func() {
			When("registry url is not set", func() {
				BeforeEach(func() {
					instance = &current.IBPConsole{
						Spec: current.IBPConsoleSpec{
							ImagePullSecrets: []string{"pullsecret"},
							NetworkInfo: &current.NetworkInfo{
								Domain: "domain1",
							},
							Versions: &current.Versions{
								CA: map[string]current.VersionCA{
									"v1-0": current.VersionCA{
										Image: current.CAImages{
											CAInitImage: "ghcr.io/ibm-blockchain/ca-init-image",
											CAInitTag:   "1.0.0",
											CAImage:     "ghcr.io/ibm-blockchain/ca-image",
											CATag:       "1.0.0",
										},
									},
								},
								Peer: map[string]current.VersionPeer{
									"v1-0": current.VersionPeer{
										Image: current.PeerImages{
											PeerInitImage: "ghcr.io/ibm-blockchain/peer-init-image",
											PeerInitTag:   "1.0.0",
											PeerImage:     "ghcr.io/ibm-blockchain/peer-image",
											PeerTag:       "1.0.0",
											DindImage:     "ghcr.io/ibm-blockchain/dind-iamge",
											DindTag:       "1.0.0",
											GRPCWebImage:  "ghcr.io/ibm-blockchain/grpcweb-image",
											GRPCWebTag:    "1.0.0",
											FluentdImage:  "ghcr.io/ibm-blockchain/fluentd-image",
											FluentdTag:    "1.0.0",
											CouchDBImage:  "ghcr.io/ibm-blockchain/couchdb-image",
											CouchDBTag:    "1.0.0",
										},
									},
								},
								Orderer: map[string]current.VersionOrderer{
									"v1-0": current.VersionOrderer{
										Image: current.OrdererImages{
											OrdererInitImage: "ghcr.io/ibm-blockchain/orderer-init-image",
											OrdererInitTag:   "1.0.0",
											OrdererImage:     "ghcr.io/ibm-blockchain/orderer-image",
											OrdererTag:       "1.0.0",
											GRPCWebImage:     "ghcr.io/ibm-blockchain/grpcweb-image",
											GRPCWebTag:       "1.0.0",
										},
									},
								},
							},
						},
					}
				})

				It("keeps images as passed", func() {
					expectedVersions := &current.Versions{
						CA: map[string]current.VersionCA{
							"v1-0": current.VersionCA{
								Image: current.CAImages{
									CAInitImage: "ghcr.io/ibm-blockchain/ca-init-image",
									CAInitTag:   "1.0.0-amd64",
									CAImage:     "ghcr.io/ibm-blockchain/ca-image",
									CATag:       "1.0.0-amd64",
								},
							},
						},
						Peer: map[string]current.VersionPeer{
							"v1-0": current.VersionPeer{
								Image: current.PeerImages{
									PeerInitImage: "ghcr.io/ibm-blockchain/peer-init-image",
									PeerInitTag:   "1.0.0-amd64",
									PeerImage:     "ghcr.io/ibm-blockchain/peer-image",
									PeerTag:       "1.0.0-amd64",
									DindImage:     "ghcr.io/ibm-blockchain/dind-iamge",
									DindTag:       "1.0.0-amd64",
									GRPCWebImage:  "ghcr.io/ibm-blockchain/grpcweb-image",
									GRPCWebTag:    "1.0.0-amd64",
									FluentdImage:  "ghcr.io/ibm-blockchain/fluentd-image",
									FluentdTag:    "1.0.0-amd64",
									CouchDBImage:  "ghcr.io/ibm-blockchain/couchdb-image",
									CouchDBTag:    "1.0.0-amd64",
								},
							},
						},
						Orderer: map[string]current.VersionOrderer{
							"v1-0": current.VersionOrderer{
								Image: current.OrdererImages{
									OrdererInitImage: "ghcr.io/ibm-blockchain/orderer-init-image",
									OrdererInitTag:   "1.0.0-amd64",
									OrdererImage:     "ghcr.io/ibm-blockchain/orderer-image",
									OrdererTag:       "1.0.0-amd64",
									GRPCWebImage:     "ghcr.io/ibm-blockchain/grpcweb-image",
									GRPCWebTag:       "1.0.0-amd64",
								},
							},
						},
					}
					versions := &deployer.Versions{
						CA: map[string]deployer.VersionCA{
							"1.4": deployer.VersionCA{
								Image: deployer.CAImages{
									CAInitImage:  "ca-init-image",
									CAInitTag:    "1.0.0",
									CAInitDigest: "",
									CAImage:      "ca-image",
									CATag:        "1.0.0",
									CADigest:     "",
								},
							},
						},
						Peer: map[string]deployer.VersionPeer{
							"1.4": deployer.VersionPeer{
								Image: deployer.PeerImages{
									PeerInitImage:  "peer-init-image",
									PeerInitTag:    "1.0.0",
									PeerInitDigest: "",
									PeerImage:      "peer-image",
									PeerTag:        "1.0.0",
									PeerDigest:     "",
									DindImage:      "dind-iamge",
									DindTag:        "1.0.0",
									DindDigest:     "",
									GRPCWebImage:   "grpcweb-image",
									GRPCWebTag:     "1.0.0",
									GRPCWebDigest:  "",
									FluentdImage:   "fluentd-image",
									FluentdTag:     "1.0.0",
									FluentdDigest:  "",
									CouchDBImage:   "couchdb-image",
									CouchDBTag:     "1.0.0",
									CouchDBDigest:  "",
								},
							},
						},
						Orderer: map[string]deployer.VersionOrderer{
							"1.4": deployer.VersionOrderer{
								Image: deployer.OrdererImages{
									OrdererInitImage:  "orderer-init-image",
									OrdererInitTag:    "1.0.0",
									OrdererInitDigest: "",
									OrdererImage:      "orderer-image",
									OrdererTag:        "1.0.0",
									OrdererDigest:     "",
									GRPCWebImage:      "grpcweb-image",
									GRPCWebTag:        "1.0.0",
									GRPCWebDigest:     "",
								},
							},
						},
					}
					config := &deployer.Config{
						Versions: versions,
						Defaults: &deployer.Defaults{
							Storage: &deployer.Storage{
								Peer: &current.PeerStorages{
									Peer:    &current.StorageSpec{},
									StateDB: &current.StorageSpec{},
								},
								CA: &current.CAStorages{
									CA: &current.StorageSpec{},
								},
								Orderer: &current.OrdererStorages{
									Orderer: &current.StorageSpec{},
								},
							},
							Resources: &deployer.Resources{},
						},
					}
					err := override.CommonDeployerCM(instance, config, nil)
					Expect(err).NotTo(HaveOccurred())
					// verify CA images and tags
					Expect(config.Versions.CA["1.4"].Image.CAImage).To(Equal(expectedVersions.CA["1.4"].Image.CAImage))
					Expect(config.Versions.CA["1.4"].Image.CATag).To(Equal(expectedVersions.CA["1.4"].Image.CATag))
					Expect(config.Versions.CA["1.4"].Image.CAInitImage).To(Equal(expectedVersions.CA["1.4"].Image.CAInitImage))
					Expect(config.Versions.CA["1.4"].Image.CAInitTag).To(Equal(expectedVersions.CA["1.4"].Image.CAInitTag))
					// verify Peer images and tags
					Expect(config.Versions.Peer["1.4"].Image.PeerInitImage).To(Equal(expectedVersions.Peer["1.4"].Image.PeerInitImage))
					Expect(config.Versions.Peer["1.4"].Image.PeerInitTag).To(Equal(expectedVersions.Peer["1.4"].Image.PeerInitTag))
					Expect(config.Versions.Peer["1.4"].Image.PeerImage).To(Equal(expectedVersions.Peer["1.4"].Image.PeerImage))
					Expect(config.Versions.Peer["1.4"].Image.PeerTag).To(Equal(expectedVersions.Peer["1.4"].Image.PeerTag))
					Expect(config.Versions.Peer["1.4"].Image.DindImage).To(Equal(expectedVersions.Peer["1.4"].Image.DindImage))
					Expect(config.Versions.Peer["1.4"].Image.DindTag).To(Equal(expectedVersions.Peer["1.4"].Image.DindTag))
					Expect(config.Versions.Peer["1.4"].Image.FluentdImage).To(Equal(expectedVersions.Peer["1.4"].Image.FluentdImage))
					Expect(config.Versions.Peer["1.4"].Image.FluentdTag).To(Equal(expectedVersions.Peer["1.4"].Image.FluentdTag))
					Expect(config.Versions.Peer["1.4"].Image.CouchDBImage).To(Equal(expectedVersions.Peer["1.4"].Image.CouchDBImage))
					Expect(config.Versions.Peer["1.4"].Image.CouchDBTag).To(Equal(expectedVersions.Peer["1.4"].Image.CouchDBTag))
					Expect(config.Versions.Peer["1.4"].Image.GRPCWebImage).To(Equal(expectedVersions.Peer["1.4"].Image.GRPCWebImage))
					Expect(config.Versions.Peer["1.4"].Image.GRPCWebTag).To(Equal(expectedVersions.Peer["1.4"].Image.GRPCWebTag))
					// verify Orderer images and tags
					Expect(config.Versions.Orderer["1.4"].Image.OrdererImage).To(Equal(expectedVersions.Orderer["1.4"].Image.OrdererImage))
					Expect(config.Versions.Orderer["1.4"].Image.OrdererTag).To(Equal(expectedVersions.Orderer["1.4"].Image.OrdererTag))
					Expect(config.Versions.Orderer["1.4"].Image.OrdererInitImage).To(Equal(expectedVersions.Orderer["1.4"].Image.OrdererInitImage))
					Expect(config.Versions.Orderer["1.4"].Image.OrdererInitTag).To(Equal(expectedVersions.Orderer["1.4"].Image.OrdererInitTag))
					Expect(config.Versions.Orderer["1.4"].Image.GRPCWebImage).To(Equal(expectedVersions.Orderer["1.4"].Image.GRPCWebImage))
					Expect(config.Versions.Orderer["1.4"].Image.GRPCWebTag).To(Equal(expectedVersions.Orderer["1.4"].Image.GRPCWebTag))
				})
			})

			When("registry url is set", func() {
				BeforeEach(func() {
					instance = &current.IBPConsole{
						Spec: current.IBPConsoleSpec{
							ImagePullSecrets: []string{"pullsecret"},
							NetworkInfo: &current.NetworkInfo{
								Domain: "domain1",
							},
							RegistryURL: "ghcr.io/ibm-blockchain/",
							Versions: &current.Versions{
								CA: map[string]current.VersionCA{
									"v1-0": current.VersionCA{
										Image: current.CAImages{
											CAInitImage: "ca-init-image",
											CAInitTag:   "1.0.0",
											CAImage:     "ca-image",
											CATag:       "1.0.0",
										},
									},
								},
								Peer: map[string]current.VersionPeer{
									"v1-0": current.VersionPeer{
										Image: current.PeerImages{
											PeerInitImage: "peer-init-image",
											PeerInitTag:   "1.0.0",
											PeerImage:     "peer-image",
											PeerTag:       "1.0.0",
											DindImage:     "dind-iamge",
											DindTag:       "1.0.0",
											GRPCWebImage:  "grpcweb-image",
											GRPCWebTag:    "1.0.0",
											FluentdImage:  "fluentd-image",
											FluentdTag:    "1.0.0",
											CouchDBImage:  "couchdb-image",
											CouchDBTag:    "1.0.0",
										},
									},
								},
								Orderer: map[string]current.VersionOrderer{
									"v1-0": current.VersionOrderer{
										Image: current.OrdererImages{
											OrdererInitImage: "orderer-init-image",
											OrdererInitTag:   "1.0.0",
											OrdererImage:     "orderer-image",
											OrdererTag:       "1.0.0",
											GRPCWebImage:     "grpcweb-image",
											GRPCWebTag:       "1.0.0",
										},
									},
								},
							},
						},
					}
				})

				It("prepends registry url to images", func() {
					expectedVersions := &current.Versions{
						CA: map[string]current.VersionCA{
							"v1-0": current.VersionCA{
								Image: current.CAImages{
									CAInitImage: "ghcr.io/ibm-blockchain/ca-init-image",
									CAInitTag:   "1.0.0-amd64",
									CAImage:     "ghcr.io/ibm-blockchain/ca-image",
									CATag:       "1.0.0-amd64",
								},
							},
						},
						Peer: map[string]current.VersionPeer{
							"v1-0": current.VersionPeer{
								Image: current.PeerImages{
									PeerInitImage: "ghcr.io/ibm-blockchain/peer-init-image",
									PeerInitTag:   "1.0.0-amd64",
									PeerImage:     "ghcr.io/ibm-blockchain/peer-image",
									PeerTag:       "1.0.0-amd64",
									DindImage:     "ghcr.io/ibm-blockchain/dind-iamge",
									DindTag:       "1.0.0-amd64",
									GRPCWebImage:  "ghcr.io/ibm-blockchain/grpcweb-image",
									GRPCWebTag:    "1.0.0-amd64",
									FluentdImage:  "ghcr.io/ibm-blockchain/fluentd-image",
									FluentdTag:    "1.0.0-amd64",
									CouchDBImage:  "ghcr.io/ibm-blockchain/couchdb-image",
									CouchDBTag:    "1.0.0-amd64",
								},
							},
						},
						Orderer: map[string]current.VersionOrderer{
							"v1-0": current.VersionOrderer{
								Image: current.OrdererImages{
									OrdererInitImage: "ghcr.io/ibm-blockchain/orderer-init-image",
									OrdererInitTag:   "1.0.0-amd64",
									OrdererImage:     "ghcr.io/ibm-blockchain/orderer-image",
									OrdererTag:       "1.0.0-amd64",
									GRPCWebImage:     "ghcr.io/ibm-blockchain/grpcweb-image",
									GRPCWebTag:       "1.0.0-amd64",
								},
							},
						},
					}
					versions := &deployer.Versions{
						CA: map[string]deployer.VersionCA{
							"1.4": deployer.VersionCA{
								Image: deployer.CAImages{
									CAInitImage:  "ca-init-image",
									CAInitTag:    "1.0.0",
									CAInitDigest: "",
									CAImage:      "ca-image",
									CATag:        "1.0.0",
									CADigest:     "",
								},
							},
						},
						Peer: map[string]deployer.VersionPeer{
							"1.4": deployer.VersionPeer{
								Image: deployer.PeerImages{
									PeerInitImage:  "peer-init-image",
									PeerInitTag:    "1.0.0",
									PeerInitDigest: "",
									PeerImage:      "peer-image",
									PeerTag:        "1.0.0",
									PeerDigest:     "",
									DindImage:      "dind-iamge",
									DindTag:        "1.0.0",
									DindDigest:     "",
									GRPCWebImage:   "grpcweb-image",
									GRPCWebTag:     "1.0.0",
									GRPCWebDigest:  "",
									FluentdImage:   "fluentd-image",
									FluentdTag:     "1.0.0",
									FluentdDigest:  "",
									CouchDBImage:   "couchdb-image",
									CouchDBTag:     "1.0.0",
									CouchDBDigest:  "",
								},
							},
						},
						Orderer: map[string]deployer.VersionOrderer{
							"1.4": deployer.VersionOrderer{
								Image: deployer.OrdererImages{
									OrdererInitImage:  "orderer-init-image",
									OrdererInitTag:    "1.0.0",
									OrdererInitDigest: "",
									OrdererImage:      "orderer-image",
									OrdererTag:        "1.0.0",
									OrdererDigest:     "",
									GRPCWebImage:      "grpcweb-image",
									GRPCWebTag:        "1.0.0",
									GRPCWebDigest:     "",
								},
							},
						},
					}
					config := &deployer.Config{
						Versions: versions,
						Defaults: &deployer.Defaults{
							Storage: &deployer.Storage{
								Peer: &current.PeerStorages{
									Peer:    &current.StorageSpec{},
									StateDB: &current.StorageSpec{},
								},
								CA: &current.CAStorages{
									CA: &current.StorageSpec{},
								},
								Orderer: &current.OrdererStorages{
									Orderer: &current.StorageSpec{},
								},
							},
							Resources: &deployer.Resources{},
						},
					}
					err := override.CommonDeployerCM(instance, config, nil)
					Expect(err).NotTo(HaveOccurred())
					// verify CA images and tags
					Expect(config.Versions.CA["1.4"].Image.CAImage).To(Equal(expectedVersions.CA["1.4"].Image.CAImage))
					Expect(config.Versions.CA["1.4"].Image.CATag).To(Equal(expectedVersions.CA["1.4"].Image.CATag))
					Expect(config.Versions.CA["1.4"].Image.CAInitImage).To(Equal(expectedVersions.CA["1.4"].Image.CAInitImage))
					Expect(config.Versions.CA["1.4"].Image.CAInitTag).To(Equal(expectedVersions.CA["1.4"].Image.CAInitTag))
					// verify Peer images and tags
					Expect(config.Versions.Peer["1.4"].Image.PeerInitImage).To(Equal(expectedVersions.Peer["1.4"].Image.PeerInitImage))
					Expect(config.Versions.Peer["1.4"].Image.PeerInitTag).To(Equal(expectedVersions.Peer["1.4"].Image.PeerInitTag))
					Expect(config.Versions.Peer["1.4"].Image.PeerImage).To(Equal(expectedVersions.Peer["1.4"].Image.PeerImage))
					Expect(config.Versions.Peer["1.4"].Image.PeerTag).To(Equal(expectedVersions.Peer["1.4"].Image.PeerTag))
					Expect(config.Versions.Peer["1.4"].Image.DindImage).To(Equal(expectedVersions.Peer["1.4"].Image.DindImage))
					Expect(config.Versions.Peer["1.4"].Image.DindTag).To(Equal(expectedVersions.Peer["1.4"].Image.DindTag))
					Expect(config.Versions.Peer["1.4"].Image.FluentdImage).To(Equal(expectedVersions.Peer["1.4"].Image.FluentdImage))
					Expect(config.Versions.Peer["1.4"].Image.FluentdTag).To(Equal(expectedVersions.Peer["1.4"].Image.FluentdTag))
					Expect(config.Versions.Peer["1.4"].Image.CouchDBImage).To(Equal(expectedVersions.Peer["1.4"].Image.CouchDBImage))
					Expect(config.Versions.Peer["1.4"].Image.CouchDBTag).To(Equal(expectedVersions.Peer["1.4"].Image.CouchDBTag))
					Expect(config.Versions.Peer["1.4"].Image.GRPCWebImage).To(Equal(expectedVersions.Peer["1.4"].Image.GRPCWebImage))
					Expect(config.Versions.Peer["1.4"].Image.GRPCWebTag).To(Equal(expectedVersions.Peer["1.4"].Image.GRPCWebTag))
					// verify Orderer images and tags
					Expect(config.Versions.Orderer["1.4"].Image.OrdererImage).To(Equal(expectedVersions.Orderer["1.4"].Image.OrdererImage))
					Expect(config.Versions.Orderer["1.4"].Image.OrdererTag).To(Equal(expectedVersions.Orderer["1.4"].Image.OrdererTag))
					Expect(config.Versions.Orderer["1.4"].Image.OrdererInitImage).To(Equal(expectedVersions.Orderer["1.4"].Image.OrdererInitImage))
					Expect(config.Versions.Orderer["1.4"].Image.OrdererInitTag).To(Equal(expectedVersions.Orderer["1.4"].Image.OrdererInitTag))
					Expect(config.Versions.Orderer["1.4"].Image.GRPCWebImage).To(Equal(expectedVersions.Orderer["1.4"].Image.GRPCWebImage))
					Expect(config.Versions.Orderer["1.4"].Image.GRPCWebTag).To(Equal(expectedVersions.Orderer["1.4"].Image.GRPCWebTag))
				})
			})
		})
	})
})
