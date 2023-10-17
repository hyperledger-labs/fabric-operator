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
package commoncore_test

import (
	"io/ioutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"

	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	peerv1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v1"
	peerv2 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v2"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/commoncore"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
)

var _ = Describe("Common", func() {

	Context("convert bootstrap from string to string array", func() {
		Context("file", func() {
			It("converts core file", func() {
				bytes, err := ioutil.ReadFile("testdata/test_core.yaml")
				Expect(err).NotTo(HaveOccurred())

				newBytes, err := commoncore.ConvertBootstrapToArray(bytes)
				Expect(err).NotTo(HaveOccurred())

				coreStruct := bytesToCore(newBytes)

				By("converting bootstrap into a string array", func() {
					Expect(coreStruct.Peer.Gossip.Bootstrap).To(Equal([]string{"127.0.0.1:7051"}))
				})

				By("persisting the remainder of the struct", func() {
					Expect(coreStruct.Chaincode).NotTo(Equal(peerv2.Chaincode{}))
					Expect(coreStruct.VM).NotTo(Equal(peerv1.VM{}))
					Expect(coreStruct.Ledger).NotTo(Equal(peerv2.Ledger{}))
					// Sanity check some of the core components
					Expect(coreStruct.Operations).To(Equal(peerv1.Operations{
						ListenAddress: "127.0.0.1:9443",
						TLS: peerv1.OperationsTLS{
							Enabled: pointer.False(),
							Certificate: peerv1.File{
								File: "cert.pem",
							},
							PrivateKey: peerv1.File{
								File: "key.pem",
							},
							ClientAuthRequired: pointer.False(),
							ClientRootCAs: peerv1.Files{
								Files: []string{"rootcert.pem"},
							},
						},
					}))
					Expect(coreStruct.Metrics).To(Equal(peerv1.Metrics{
						Provider: "prometheus",
						Statsd: peerv1.Statsd{
							Network:       "udp",
							Address:       "127.0.0.1:8125",
							WriteInterval: common.MustParseDuration("10s"),
							Prefix:        "",
						},
					}))
				})
			})

			It("returns config if bootstrap is already []string", func() {
				bytes, err := ioutil.ReadFile("testdata/test_core_no_change.yaml")
				Expect(err).NotTo(HaveOccurred())

				newBytes, err := commoncore.ConvertBootstrapToArray(bytes)
				Expect(err).NotTo(HaveOccurred())

				coreStruct := bytesToCore(newBytes)
				Expect(coreStruct.Peer.Gossip.Bootstrap).To(Equal([]string{"1.2.3.4"}))
			})

			It("returns config if peer is not present in config", func() {
				bytes, err := ioutil.ReadFile("testdata/test_core_no_peer.yaml")
				Expect(err).NotTo(HaveOccurred())

				newBytes, err := commoncore.ConvertBootstrapToArray(bytes)
				Expect(err).NotTo(HaveOccurred())

				coreStruct := bytesToCore(newBytes)
				By("not setting anything in core.peer", func() {
					Expect(coreStruct.Peer).To(Equal(peerv2.Peer{}))
				})

				By("persisting existing config", func() {
					Expect(coreStruct.Chaincode).NotTo(Equal(peerv2.Chaincode{}))
					Expect(coreStruct.VM).NotTo(Equal(peerv1.VM{}))
					Expect(coreStruct.Ledger).NotTo(Equal(peerv2.Ledger{}))
					// Sanity check some of the core components
					Expect(coreStruct.Operations).To(Equal(peerv1.Operations{
						ListenAddress: "127.0.0.1:9443",
						TLS: peerv1.OperationsTLS{
							Enabled: pointer.False(),
							Certificate: peerv1.File{
								File: "cert.pem",
							},
							PrivateKey: peerv1.File{
								File: "key.pem",
							},
							ClientAuthRequired: pointer.False(),
							ClientRootCAs: peerv1.Files{
								Files: []string{"rootcert.pem"},
							},
						},
					}))
					Expect(coreStruct.Metrics).To(Equal(peerv1.Metrics{
						Provider: "prometheus",
						Statsd: peerv1.Statsd{
							Network:       "udp",
							Address:       "127.0.0.1:8125",
							WriteInterval: common.MustParseDuration("10s"),
							Prefix:        "",
						},
					}))
				})

			})
		})

		Context("bytes", func() {
			var (
				coreBytes []byte
				err       error
			)

			BeforeEach(func() {
				testCore := &TestCore{
					Peer: Peer{
						Gossip: Gossip{
							Bootstrap: "1.2.3.4",
						},
					},
				}
				coreBytes, err = yaml.Marshal(testCore)
				Expect(err).NotTo(HaveOccurred())
			})

			It("converts core bytes", func() {
				newBytes, err := commoncore.ConvertBootstrapToArray(coreBytes)
				Expect(err).NotTo(HaveOccurred())

				coreStruct := bytesToCore(newBytes)
				Expect(coreStruct.Peer.Gossip.Bootstrap).To(Equal([]string{"1.2.3.4"}))
			})

			It("returns same bytes if peer.gossip.bootstrap not found", func() {
				core := map[string]interface{}{
					"chaincode": map[string]interface{}{
						"pull": true,
					},
				}
				bytes, err := yaml.Marshal(core)
				Expect(err).NotTo(HaveOccurred())

				newBytes, err := commoncore.ConvertBootstrapToArray(bytes)
				Expect(err).NotTo(HaveOccurred())

				coreStruct := bytesToCore(newBytes)
				trueVal := true
				Expect(coreStruct.Peer).To(Equal(peerv2.Peer{}))
				Expect(coreStruct.Chaincode).To(Equal(peerv2.Chaincode{
					Pull: &trueVal,
				}))
			})
		})

		Context("interface", func() {
			var (
				intf interface{}
			)

			BeforeEach(func() {
				intf = &TestCore{
					Peer: Peer{
						Gossip: Gossip{
							Bootstrap: "1.2.3.4",
						},
					},
				}
			})

			It("converts interface", func() {
				newBytes, err := commoncore.ConvertBootstrapToArray(intf)
				Expect(err).NotTo(HaveOccurred())

				coreStruct := bytesToCore(newBytes)
				Expect(coreStruct.Peer.Gossip.Bootstrap).To(Equal([]string{"1.2.3.4"}))
			})

			It("returns config if no conversion required", func() {
				intf = &TestCore{
					Peer: Peer{
						Gossip: Gossip{},
					},
				}
				newBytes, err := commoncore.ConvertBootstrapToArray(intf)
				Expect(err).NotTo(HaveOccurred())

				coreStruct := bytesToCore(newBytes)
				Expect(coreStruct.Peer.Gossip).To(Equal(peerv2.Gossip{}))
			})

			It("converts json raw message", func() {
				rawMsg, err := util.ConvertToJsonMessage(intf)
				Expect(err).NotTo(HaveOccurred())
				newBytes, err := commoncore.ConvertBootstrapToArray(rawMsg)
				Expect(err).NotTo(HaveOccurred())

				coreStruct := bytesToCore(newBytes)
				Expect(coreStruct.Peer.Gossip.Bootstrap).To(Equal([]string{"1.2.3.4"}))
			})
		})

	})
})

func bytesToCore(bytes []byte) *peerv2.Core {
	coreStruct := &peerv2.Core{}
	err := yaml.Unmarshal(bytes, coreStruct)
	Expect(err).NotTo(HaveOccurred())
	return coreStruct
}

type TestCore struct {
	Peer Peer `json:"peer"`
}

type Peer struct {
	Gossip Gossip `json:"gossip"`
}

type Gossip struct {
	Bootstrap string `json:"bootstrap"`
}
