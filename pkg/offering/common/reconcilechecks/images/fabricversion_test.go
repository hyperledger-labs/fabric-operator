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
)

var _ = Describe("fabric version", func() {
	var (
		fv          *images.FabricVersion
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

		fv = &images.FabricVersion{
			Versions: operatorCfg.Versions,
		}
	})

	Context("ca", func() {
		var (
			instance *current.IBPCA
		)

		Context("normalize version", func() {
			When("using non-hyphenated fabric version", func() {
				BeforeEach(func() {
					instance = &current.IBPCA{
						Spec: current.IBPCASpec{
							FabricVersion: "1.4.9",
						},
					}
				})

				It("returns default images for the base fabric version", func() {
					version := fv.Normalize(instance)
					Expect(version).To(Equal("1.4.9-1"))
				})
			})
		})

		Context("validate version", func() {
			BeforeEach(func() {
				instance = &current.IBPCA{
					Spec: current.IBPCASpec{
						FabricVersion: "1.8.9-1",
					},
				}
			})

			It("returns error if unsupported version", func() {
				err := fv.Validate(instance)
				Expect(err).To(MatchError(ContainSubstring("is not supported for CA")))
			})
		})
	})

	Context("peer", func() {
		var (
			instance *current.IBPPeer
		)

		Context("normalize version", func() {
			When("using non-hyphenated fabric version", func() {
				BeforeEach(func() {
					instance = &current.IBPPeer{
						Spec: current.IBPPeerSpec{
							FabricVersion: "2.2.1",
						},
					}
				})

				It("returns default images for the base fabric version", func() {
					version := fv.Normalize(instance)
					Expect(version).To(Equal("2.2.1-1"))
				})
			})
		})

		Context("validate version", func() {
			BeforeEach(func() {
				instance = &current.IBPPeer{
					Spec: current.IBPPeerSpec{
						FabricVersion: "1.8.9-1",
					},
				}
			})

			It("returns error if unsupported version", func() {
				err := fv.Validate(instance)
				Expect(err).To(MatchError(ContainSubstring("is not supported for Peer")))
			})
		})
	})

	Context("orderer", func() {
		var (
			instance *current.IBPOrderer
		)

		Context("normalize version", func() {
			When("using non-hyphenated fabric version", func() {
				BeforeEach(func() {
					instance = &current.IBPOrderer{
						Spec: current.IBPOrdererSpec{
							FabricVersion: "2.2.1",
						},
					}
				})

				It("returns default images for the base fabric version", func() {
					version := fv.Normalize(instance)
					Expect(version).To(Equal("2.2.1-1"))
				})
			})
		})

		Context("validate version", func() {
			BeforeEach(func() {
				instance = &current.IBPOrderer{
					Spec: current.IBPOrdererSpec{
						FabricVersion: "1.8.9-1",
					},
				}
			})

			It("returns error if unsupported version", func() {
				err := fv.Validate(instance)
				Expect(err).To(MatchError(ContainSubstring("is not supported for Orderer")))
			})
		})
	})
})
