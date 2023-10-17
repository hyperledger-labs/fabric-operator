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

package images_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common/reconcilechecks/images"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common/reconcilechecks/images/mocks"
)

var _ = Describe("default images", func() {
	var (
		operatorCfg *config.Operator
	)

	BeforeEach(func() {
		operatorCfg = &config.Operator{
			Versions: &deployer.Versions{
				CA: map[string]deployer.VersionCA{
					"1.4.9-0": {
						Default: false,
						Version: "1.4.9-0",
						Image: deployer.CAImages{
							CAImage:       "caimage",
							CATag:         "catag",
							CAInitImage:   "cainitimage",
							CAInitTag:     "cainittag",
							EnrollerImage: "enrolleriamge",
							EnrollerTag:   "enrollertag",
						},
					},
					"1.4.9-1": {
						Default: true,
						Version: "1.4.9-1",
						Image: deployer.CAImages{
							CAImage:       "caimage",
							CATag:         "newcatag",
							CAInitImage:   "cainitimage",
							CAInitTag:     "cainittag",
							EnrollerImage: "enrolleriamge",
							EnrollerTag:   "enrollertag",
						},
					},
				},
				Peer: map[string]deployer.VersionPeer{
					"1.4.9-0": {
						Default: true,
						Version: "1.4.9-0",
						Image: deployer.PeerImages{
							PeerInitImage: "ibp-init",
							PeerInitTag:   "2.5.1-2511004-amd64",
							PeerImage:     "ibp-peer",
							PeerTag:       "1.4.9-2511004-amd64",
							DindImage:     "ibp-dind",
							DindTag:       "noimages-amd64",
							CouchDBImage:  "ibp-couchdb",
							CouchDBTag:    "2.3.1-2511004-amd64",
							GRPCWebImage:  "ibp-grpcweb",
							GRPCWebTag:    "2.5.1-2511004-amd64",
							FluentdImage:  "ibp-fluentd",
							FluentdTag:    "2.5.1-2511004-amd64",
							EnrollerImage: "ibp-enroller",
							EnrollerTag:   "1.0.0-amd64",
						},
					},
					"2.2.1-0": {
						Default: false,
						Version: "2.2.1-0",
						Image: deployer.PeerImages{
							PeerInitImage: "ibp-init",
							PeerInitTag:   "2.5.1-2511004-amd64",
							PeerImage:     "ibp-peer",
							PeerTag:       "2.2.1-2511004-amd64",
							DindImage:     "ibp-dind",
							DindTag:       "noimages-amd64",
							CouchDBImage:  "ibp-couchdb",
							CouchDBTag:    "2.3.1-2511004-amd64",
							GRPCWebImage:  "ibp-grpcweb",
							GRPCWebTag:    "2.5.1-2511004-amd64",
							FluentdImage:  "ibp-fluentd",
							FluentdTag:    "2.5.1-2511004-amd64",
							EnrollerImage: "ibp-enroller",
							EnrollerTag:   "1.0.0-amd64",
						},
					},
					"2.2.1-1": {
						Default: true,
						Version: "2.2.1-1",
						Image: deployer.PeerImages{
							PeerInitImage: "ibp-init",
							PeerInitTag:   "2.5.1-2511004-amd64",
							PeerImage:     "ibp-peer",
							PeerTag:       "2.2.1-2511204-amd64",
							DindImage:     "ibp-dind",
							DindTag:       "noimages-amd64",
							CouchDBImage:  "ibp-couchdb",
							CouchDBTag:    "2.3.1-2511004-amd64",
							GRPCWebImage:  "ibp-grpcweb",
							GRPCWebTag:    "2.5.1-2511004-amd64",
							FluentdImage:  "ibp-fluentd",
							FluentdTag:    "2.5.1-2511004-amd64",
							EnrollerImage: "ibp-enroller",
							EnrollerTag:   "1.0.0-amd64",
						},
					},
				},
				Orderer: map[string]deployer.VersionOrderer{
					"1.4-9-0": {
						Default: true,
						Version: "1.4.9-0",
						Image: deployer.OrdererImages{
							OrdererInitImage: "ibp-init",
							OrdererInitTag:   "2.5.1-2511004-amd64",
							OrdererImage:     "ibp-orderer",
							OrdererTag:       "1.4.9-2511004-amd64",
							GRPCWebImage:     "ibp-grpcweb",
							GRPCWebTag:       "2.5.1-2511004-amd64",
							EnrollerImage:    "ibp-enroller",
							EnrollerTag:      "1.0.0-amd64",
						},
					},
					"2.2.1-0": {
						Default: false,
						Version: "2.2.1-0",
						Image: deployer.OrdererImages{
							OrdererInitImage: "ibp-init",
							OrdererInitTag:   "2.5.1-2511004-amd64",
							OrdererImage:     "ibp-orderer",
							OrdererTag:       "2.2.1-2511004-amd64",
							GRPCWebImage:     "ibp-grpcweb",
							GRPCWebTag:       "2.5.1-2511004-amd64",
							EnrollerImage:    "ibp-enroller",
							EnrollerTag:      "1.0.0-amd64",
						},
					},
					"2.2.1-1": {
						Default: true,
						Version: "2.2.1-0",
						Image: deployer.OrdererImages{
							OrdererInitImage: "ibp-init",
							OrdererInitTag:   "2.5.1-2511004-amd64",
							OrdererImage:     "ibp-orderer",
							OrdererTag:       "2.2.1-2511204-amd64",
							GRPCWebImage:     "ibp-grpcweb",
							GRPCWebTag:       "2.5.1-2511004-amd64",
							EnrollerImage:    "ibp-enroller",
							EnrollerTag:      "1.0.0-amd64",
						},
					},
				},
			},
		}
	})

	Context("images", func() {
		var image *images.Image

		BeforeEach(func() {
			image = &images.Image{
				Versions:    operatorCfg.Versions,
				DefaultArch: "amd64",
				// DefaultRegistryURL: "",
			}
		})

		It("returns an error if fabric version is not in correct format", func() {
			instance := &mocks.Instance{}
			instance.GetFabricVersionReturns("1.4.9")
			err := image.SetDefaults(instance)
			Expect(err).To(MatchError("fabric version format '1.4.9' is not valid, must pass hyphenated version (e.g. 2.2.1-1)"))
		})

		Context("update required", func() {
			var update *mocks.Update

			BeforeEach(func() {
				update = &mocks.Update{}
			})

			It("returns false if images updated", func() {
				update.ImagesUpdatedReturns(true)
				required := image.UpdateRequired(update)
				Expect(required).To(Equal(false))
			})

			It("returns false if neither images nor fabric version updated", func() {
				required := image.UpdateRequired(update)
				Expect(required).To(Equal(false))
			})

			It("returns true if fabric version updated and images not updated", func() {
				update.FabricVersionUpdatedReturns(true)
				required := image.UpdateRequired(update)
				Expect(required).To(Equal(true))
			})
		})

		Context("ca", func() {
			var (
				instance *current.IBPCA
			)

			BeforeEach(func() {
				instance = &current.IBPCA{
					Spec: current.IBPCASpec{
						RegistryURL:   "ghcr.io/ibm-blockchain/",
						FabricVersion: "1.4.9-1",
					},
				}
			})

			Context("registry url", func() {
				When("is not set", func() {
					BeforeEach(func() {
						instance.Spec.RegistryURL = ""
					})

					It("sets default images based on operator's config with registry of blank", func() {
						err := image.SetDefaults(instance)
						Expect(err).NotTo(HaveOccurred())
						caImages := deployer.CAImages{
							CAImage:       "caimage",
							CATag:         "newcatag",
							CAInitImage:   "cainitimage",
							CAInitTag:     "cainittag",
							EnrollerImage: "enrolleriamge",
							EnrollerTag:   "enrollertag",
						}
						Expect(instance.Spec.Images.CAImage).To(Equal(caImages.CAImage))
						Expect(instance.Spec.Images.CATag).To(Equal(caImages.CATag))
						Expect(instance.Spec.Images.CAInitImage).To(Equal(caImages.CAInitImage))
						Expect(instance.Spec.Images.CAInitTag).To(Equal(caImages.CAInitTag))
						Expect(instance.Spec.Images.EnrollerImage).To(Equal(caImages.EnrollerImage))
						Expect(instance.Spec.Images.EnrollerTag).To(Equal(caImages.EnrollerTag))
					})
				})

				When("is set", func() {
					It("sets default images based on operator's config", func() {
						err := image.SetDefaults(instance)
						Expect(err).NotTo(HaveOccurred())
						caImages := deployer.CAImages{
							CAImage:       "ghcr.io/ibm-blockchain/caimage",
							CATag:         "newcatag",
							CAInitImage:   "ghcr.io/ibm-blockchain/cainitimage",
							CAInitTag:     "cainittag",
							EnrollerImage: "ghcr.io/ibm-blockchain/enrolleriamge",
							EnrollerTag:   "enrollertag",
						}

						Expect(instance.Spec.Images.CAImage).To(Equal(caImages.CAImage))
						Expect(instance.Spec.Images.CATag).To(Equal(caImages.CATag))
						Expect(instance.Spec.Images.CAInitImage).To(Equal(caImages.CAInitImage))
						Expect(instance.Spec.Images.CAInitTag).To(Equal(caImages.CAInitTag))
						Expect(instance.Spec.Images.EnrollerImage).To(Equal(caImages.EnrollerImage))
						Expect(instance.Spec.Images.EnrollerTag).To(Equal(caImages.EnrollerTag))
					})
				})
			})

			When("using normalized fabric version", func() {
				BeforeEach(func() {
					instance.Spec.FabricVersion = "1.4.9-0"
				})

				It("returns default images for the base fabric version", func() {
					err := image.SetDefaults(instance)
					Expect(err).NotTo(HaveOccurred())

					caImages := deployer.CAImages{
						CAImage:       "ghcr.io/ibm-blockchain/caimage",
						CATag:         "catag",
						CAInitImage:   "ghcr.io/ibm-blockchain/cainitimage",
						CAInitTag:     "cainittag",
						EnrollerImage: "ghcr.io/ibm-blockchain/enrolleriamge",
						EnrollerTag:   "enrollertag",
					}

					Expect(instance.Spec.Images.CAImage).To(Equal(caImages.CAImage))
					Expect(instance.Spec.Images.CATag).To(Equal(caImages.CATag))
					Expect(instance.Spec.Images.CAInitImage).To(Equal(caImages.CAInitImage))
					Expect(instance.Spec.Images.CAInitTag).To(Equal(caImages.CAInitTag))
					Expect(instance.Spec.Images.EnrollerImage).To(Equal(caImages.EnrollerImage))
					Expect(instance.Spec.Images.EnrollerTag).To(Equal(caImages.EnrollerTag))
				})
			})

			It("returns error if requested version not found", func() {
				instance.Spec.FabricVersion = "5.1.0-1"
				err := image.SetDefaults(instance)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("peer", func() {
			var (
				instance *current.IBPPeer
			)

			BeforeEach(func() {
				instance = &current.IBPPeer{
					Spec: current.IBPPeerSpec{
						RegistryURL:   "ghcr.io/ibm-blockchain/",
						FabricVersion: "2.2.1-1",
					},
				}
			})

			Context("registy URL", func() {
				When("is not set", func() {
					BeforeEach(func() {
						instance.Spec.RegistryURL = ""
					})

					It("sets registry URL to blank", func() {
						err := image.SetDefaults(instance)
						Expect(err).NotTo(HaveOccurred())
						peerImages := deployer.PeerImages{
							PeerInitImage: "ibp-init",
							PeerInitTag:   "2.5.1-2511004-amd64",
							PeerImage:     "ibp-peer",
							PeerTag:       "2.2.1-2511204-amd64",
							DindImage:     "ibp-dind",
							DindTag:       "noimages-amd64",
							CouchDBImage:  "ibp-couchdb",
							CouchDBTag:    "2.3.1-2511004-amd64",
							GRPCWebImage:  "ibp-grpcweb",
							GRPCWebTag:    "2.5.1-2511004-amd64",
							FluentdImage:  "ibp-fluentd",
							FluentdTag:    "2.5.1-2511004-amd64",
							EnrollerImage: "ibp-enroller",
							EnrollerTag:   "1.0.0-amd64",
						}

						Expect(instance.Spec.Images.PeerInitImage).To(Equal(peerImages.PeerInitImage))
						Expect(instance.Spec.Images.PeerInitTag).To(Equal(peerImages.PeerInitTag))
						Expect(instance.Spec.Images.PeerImage).To(Equal(peerImages.PeerImage))
						Expect(instance.Spec.Images.PeerTag).To(Equal(peerImages.PeerTag))
						Expect(instance.Spec.Images.DindImage).To(Equal(peerImages.DindImage))
						Expect(instance.Spec.Images.DindTag).To(Equal(peerImages.DindTag))
						Expect(instance.Spec.Images.CouchDBImage).To(Equal(peerImages.CouchDBImage))
						Expect(instance.Spec.Images.CouchDBTag).To(Equal(peerImages.CouchDBTag))
						Expect(instance.Spec.Images.GRPCWebImage).To(Equal(peerImages.GRPCWebImage))
						Expect(instance.Spec.Images.GRPCWebTag).To(Equal(peerImages.GRPCWebTag))
						Expect(instance.Spec.Images.FluentdImage).To(Equal(peerImages.FluentdImage))
						Expect(instance.Spec.Images.FluentdTag).To(Equal(peerImages.FluentdTag))
						Expect(instance.Spec.Images.EnrollerImage).To(Equal(peerImages.EnrollerImage))
						Expect(instance.Spec.Images.EnrollerTag).To(Equal(peerImages.EnrollerTag))
					})
				})

				When("is set", func() {
					It("sets the requested registry URL", func() {
						err := image.SetDefaults(instance)
						Expect(err).NotTo(HaveOccurred())
						peerImages := deployer.PeerImages{
							PeerInitImage: "ghcr.io/ibm-blockchain/ibp-init",
							PeerInitTag:   "2.5.1-2511004-amd64",
							PeerImage:     "ghcr.io/ibm-blockchain/ibp-peer",
							PeerTag:       "2.2.1-2511204-amd64",
							DindImage:     "ghcr.io/ibm-blockchain/ibp-dind",
							DindTag:       "noimages-amd64",
							CouchDBImage:  "ghcr.io/ibm-blockchain/ibp-couchdb",
							CouchDBTag:    "2.3.1-2511004-amd64",
							GRPCWebImage:  "ghcr.io/ibm-blockchain/ibp-grpcweb",
							GRPCWebTag:    "2.5.1-2511004-amd64",
							FluentdImage:  "ghcr.io/ibm-blockchain/ibp-fluentd",
							FluentdTag:    "2.5.1-2511004-amd64",
							EnrollerImage: "ghcr.io/ibm-blockchain/ibp-enroller",
							EnrollerTag:   "1.0.0-amd64",
						}

						Expect(instance.Spec.Images.PeerInitImage).To(Equal(peerImages.PeerInitImage))
						Expect(instance.Spec.Images.PeerInitTag).To(Equal(peerImages.PeerInitTag))
						Expect(instance.Spec.Images.PeerImage).To(Equal(peerImages.PeerImage))
						Expect(instance.Spec.Images.PeerTag).To(Equal(peerImages.PeerTag))
						Expect(instance.Spec.Images.DindImage).To(Equal(peerImages.DindImage))
						Expect(instance.Spec.Images.DindTag).To(Equal(peerImages.DindTag))
						Expect(instance.Spec.Images.CouchDBImage).To(Equal(peerImages.CouchDBImage))
						Expect(instance.Spec.Images.CouchDBTag).To(Equal(peerImages.CouchDBTag))
						Expect(instance.Spec.Images.GRPCWebImage).To(Equal(peerImages.GRPCWebImage))
						Expect(instance.Spec.Images.GRPCWebTag).To(Equal(peerImages.GRPCWebTag))
						Expect(instance.Spec.Images.FluentdImage).To(Equal(peerImages.FluentdImage))
						Expect(instance.Spec.Images.FluentdTag).To(Equal(peerImages.FluentdTag))
						Expect(instance.Spec.Images.EnrollerImage).To(Equal(peerImages.EnrollerImage))
						Expect(instance.Spec.Images.EnrollerTag).To(Equal(peerImages.EnrollerTag))
					})
				})

			})

			When("using normalized fabric version", func() {
				BeforeEach(func() {
					instance.Spec.FabricVersion = "2.2.1-0"
				})

				It("returns images for the requested fabric version", func() {
					err := image.SetDefaults(instance)
					Expect(err).NotTo(HaveOccurred())
					peerImages := deployer.PeerImages{
						PeerInitImage: "ghcr.io/ibm-blockchain/ibp-init",
						PeerInitTag:   "2.5.1-2511004-amd64",
						PeerImage:     "ghcr.io/ibm-blockchain/ibp-peer",
						PeerTag:       "2.2.1-2511004-amd64",
						DindImage:     "ghcr.io/ibm-blockchain/ibp-dind",
						DindTag:       "noimages-amd64",
						CouchDBImage:  "ghcr.io/ibm-blockchain/ibp-couchdb",
						CouchDBTag:    "2.3.1-2511004-amd64",
						GRPCWebImage:  "ghcr.io/ibm-blockchain/ibp-grpcweb",
						GRPCWebTag:    "2.5.1-2511004-amd64",
						FluentdImage:  "ghcr.io/ibm-blockchain/ibp-fluentd",
						FluentdTag:    "2.5.1-2511004-amd64",
						EnrollerImage: "ghcr.io/ibm-blockchain/ibp-enroller",
						EnrollerTag:   "1.0.0-amd64",
					}

					Expect(instance.Spec.Images.PeerInitImage).To(Equal(peerImages.PeerInitImage))
					Expect(instance.Spec.Images.PeerInitTag).To(Equal(peerImages.PeerInitTag))
					Expect(instance.Spec.Images.PeerImage).To(Equal(peerImages.PeerImage))
					Expect(instance.Spec.Images.PeerTag).To(Equal(peerImages.PeerTag))
					Expect(instance.Spec.Images.DindImage).To(Equal(peerImages.DindImage))
					Expect(instance.Spec.Images.DindTag).To(Equal(peerImages.DindTag))
					Expect(instance.Spec.Images.CouchDBImage).To(Equal(peerImages.CouchDBImage))
					Expect(instance.Spec.Images.CouchDBTag).To(Equal(peerImages.CouchDBTag))
					Expect(instance.Spec.Images.GRPCWebImage).To(Equal(peerImages.GRPCWebImage))
					Expect(instance.Spec.Images.GRPCWebTag).To(Equal(peerImages.GRPCWebTag))
					Expect(instance.Spec.Images.FluentdImage).To(Equal(peerImages.FluentdImage))
					Expect(instance.Spec.Images.FluentdTag).To(Equal(peerImages.FluentdTag))
					Expect(instance.Spec.Images.EnrollerImage).To(Equal(peerImages.EnrollerImage))
					Expect(instance.Spec.Images.EnrollerTag).To(Equal(peerImages.EnrollerTag))
				})
			})

			It("returns error if requested version not found", func() {
				instance.Spec.FabricVersion = "5.1.0-1"
				err := image.SetDefaults(instance)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("orderer", func() {
			var (
				instance *current.IBPOrderer
			)

			BeforeEach(func() {
				instance = &current.IBPOrderer{
					Spec: current.IBPOrdererSpec{
						RegistryURL:   "ghcr.io/ibm-blockchain/",
						FabricVersion: "2.2.1-1",
					},
				}
			})

			Context("registry URL", func() {
				When("is not set", func() {
					BeforeEach(func() {
						instance.Spec.RegistryURL = ""
					})

					It("sets default images based on operator's config with registry of blank", func() {
						err := image.SetDefaults(instance)
						Expect(err).NotTo(HaveOccurred())
						ordererImages := deployer.OrdererImages{
							OrdererInitImage: "ibp-init",
							OrdererInitTag:   "2.5.1-2511004-amd64",
							OrdererImage:     "ibp-orderer",
							OrdererTag:       "2.2.1-2511204-amd64",
							GRPCWebImage:     "ibp-grpcweb",
							GRPCWebTag:       "2.5.1-2511004-amd64",
							EnrollerImage:    "ibp-enroller",
							EnrollerTag:      "1.0.0-amd64",
						}

						Expect(instance.Spec.Images.OrdererInitImage).To(Equal(ordererImages.OrdererInitImage))
						Expect(instance.Spec.Images.OrdererInitTag).To(Equal(ordererImages.OrdererInitTag))
						Expect(instance.Spec.Images.OrdererImage).To(Equal(ordererImages.OrdererImage))
						Expect(instance.Spec.Images.OrdererTag).To(Equal(ordererImages.OrdererTag))
						Expect(instance.Spec.Images.GRPCWebImage).To(Equal(ordererImages.GRPCWebImage))
						Expect(instance.Spec.Images.GRPCWebTag).To(Equal(ordererImages.GRPCWebTag))
						Expect(instance.Spec.Images.EnrollerImage).To(Equal(ordererImages.EnrollerImage))
						Expect(instance.Spec.Images.EnrollerTag).To(Equal(ordererImages.EnrollerTag))
					})
				})

				When("is set", func() {
					It("sets default images based on operator's config", func() {
						err := image.SetDefaults(instance)
						Expect(err).NotTo(HaveOccurred())
						ordererImages := deployer.OrdererImages{
							OrdererInitImage: "ghcr.io/ibm-blockchain/ibp-init",
							OrdererInitTag:   "2.5.1-2511004-amd64",
							OrdererImage:     "ghcr.io/ibm-blockchain/ibp-orderer",
							OrdererTag:       "2.2.1-2511204-amd64",
							GRPCWebImage:     "ghcr.io/ibm-blockchain/ibp-grpcweb",
							GRPCWebTag:       "2.5.1-2511004-amd64",
							EnrollerImage:    "ghcr.io/ibm-blockchain/ibp-enroller",
							EnrollerTag:      "1.0.0-amd64",
						}

						Expect(instance.Spec.Images.OrdererInitImage).To(Equal(ordererImages.OrdererInitImage))
						Expect(instance.Spec.Images.OrdererInitTag).To(Equal(ordererImages.OrdererInitTag))
						Expect(instance.Spec.Images.OrdererImage).To(Equal(ordererImages.OrdererImage))
						Expect(instance.Spec.Images.OrdererTag).To(Equal(ordererImages.OrdererTag))
						Expect(instance.Spec.Images.GRPCWebImage).To(Equal(ordererImages.GRPCWebImage))
						Expect(instance.Spec.Images.GRPCWebTag).To(Equal(ordererImages.GRPCWebTag))
						Expect(instance.Spec.Images.EnrollerImage).To(Equal(ordererImages.EnrollerImage))
						Expect(instance.Spec.Images.EnrollerTag).To(Equal(ordererImages.EnrollerTag))
					})
				})
			})

			When("using normalized fabric version", func() {
				BeforeEach(func() {
					instance.Spec.FabricVersion = "2.2.1-0"
				})

				It("returns default images for the base fabric version", func() {
					err := image.SetDefaults(instance)
					Expect(err).NotTo(HaveOccurred())
					ordererImages := deployer.OrdererImages{
						OrdererInitImage: "ghcr.io/ibm-blockchain/ibp-init",
						OrdererInitTag:   "2.5.1-2511004-amd64",
						OrdererImage:     "ghcr.io/ibm-blockchain/ibp-orderer",
						OrdererTag:       "2.2.1-2511004-amd64",
						GRPCWebImage:     "ghcr.io/ibm-blockchain/ibp-grpcweb",
						GRPCWebTag:       "2.5.1-2511004-amd64",
						EnrollerImage:    "ghcr.io/ibm-blockchain/ibp-enroller",
						EnrollerTag:      "1.0.0-amd64",
					}

					Expect(instance.Spec.Images.OrdererInitImage).To(Equal(ordererImages.OrdererInitImage))
					Expect(instance.Spec.Images.OrdererInitTag).To(Equal(ordererImages.OrdererInitTag))
					Expect(instance.Spec.Images.OrdererImage).To(Equal(ordererImages.OrdererImage))
					Expect(instance.Spec.Images.OrdererTag).To(Equal(ordererImages.OrdererTag))
					Expect(instance.Spec.Images.GRPCWebImage).To(Equal(ordererImages.GRPCWebImage))
					Expect(instance.Spec.Images.GRPCWebTag).To(Equal(ordererImages.GRPCWebTag))
					Expect(instance.Spec.Images.EnrollerImage).To(Equal(ordererImages.EnrollerImage))
					Expect(instance.Spec.Images.EnrollerTag).To(Equal(ordererImages.EnrollerTag))
				})
			})

			It("returns error if requested version not found", func() {
				instance.Spec.FabricVersion = "5.1.0-1"
				err := image.SetDefaults(instance)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
