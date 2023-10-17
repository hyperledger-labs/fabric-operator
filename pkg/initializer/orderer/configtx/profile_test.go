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

package configtx_test

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/configtx"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/orderer/etcdraft"
	"github.com/hyperledger/fabric/common/channelconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("profile", func() {
	var (
		err       error
		profile   *configtx.Profile
		mspConfig map[string]*msp.MSPConfig
	)

	BeforeEach(func() {
		configTx := configtx.New()
		profile, err = configTx.GetProfile("Initial")
		Expect(err).NotTo(HaveOccurred())

		mspConfig = map[string]*msp.MSPConfig{
			"testorg3": &msp.MSPConfig{},
		}
	})

	Context("profile configuration updates", func() {
		It("adds orderer address to profile", func() {
			blockBytes, err := profile.GenerateBlock("channel1", mspConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(blockBytes)).NotTo(ContainSubstring("127.0.0.1:7051"))

			profile.AddOrdererAddress("127.0.0.1:7051")
			blockBytes, err = profile.GenerateBlock("channel1", mspConfig)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(blockBytes)).To(ContainSubstring("127.0.0.1:7051"))
		})

		It("sets orderer type", func() {
			profile.SetOrdererType("etcdraft")
			blockBytes, err := profile.GenerateBlock("channel1", mspConfig)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(blockBytes)).To(ContainSubstring("etcdraft"))
		})

		It("adds raft consenting node", func() {
			consenter := &etcdraft.Consenter{
				Host:          "testrafthost",
				Port:          7050,
				ClientTlsCert: []byte("../../../../testdata/tls/tls.crt"),
				ServerTlsCert: []byte("../../../../testdata/tls/tls.crt"),
			}

			profile.SetOrdererType("etcdraft")
			err := profile.AddRaftConsentingNode(consenter)
			Expect(err).NotTo(HaveOccurred())

			blockBytes, err := profile.GenerateBlock("channel1", mspConfig)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(blockBytes)).To(ContainSubstring("testrafthost"))
		})

		It("adds consortium", func() {
			profile.Policies = map[string]*configtx.Policy{
				channelconfig.AdminsPolicyKey: &configtx.Policy{
					Type: configtx.ImplicitMetaPolicyType,
					Rule: "ALL bar",
				},
				channelconfig.ReadersPolicyKey: &configtx.Policy{
					Type: configtx.ImplicitMetaPolicyType,
					Rule: "ALL bar",
				},
				channelconfig.WritersPolicyKey: &configtx.Policy{
					Type: configtx.ImplicitMetaPolicyType,
					Rule: "ALL bar",
				},
			}

			org := &configtx.Organization{
				Name:           "testorg",
				ID:             "testorg",
				MSPType:        "bccsp",
				MSPDir:         "../../../../testdata/init/orderer/msp",
				AdminPrincipal: "Role.MEMBER",
				Policies:       profile.Policies,
			}

			consortium := &configtx.Consortium{
				Organizations: []*configtx.Organization{org},
			}

			profile.SetOrdererType("etcdraft")
			err := profile.AddConsortium("testconsortium", consortium)
			Expect(err).NotTo(HaveOccurred())

			blockBytes, err := profile.GenerateBlock("channel1", mspConfig)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(blockBytes)).To(ContainSubstring("testconsortium"))
			Expect(string(blockBytes)).To(ContainSubstring("testorg"))
		})

		It("adds org to consortium", func() {
			profile.Policies = map[string]*configtx.Policy{
				channelconfig.AdminsPolicyKey: &configtx.Policy{
					Type: configtx.ImplicitMetaPolicyType,
					Rule: "ALL bar",
				},
				channelconfig.ReadersPolicyKey: &configtx.Policy{
					Type: configtx.ImplicitMetaPolicyType,
					Rule: "ALL bar",
				},
				channelconfig.WritersPolicyKey: &configtx.Policy{
					Type: configtx.ImplicitMetaPolicyType,
					Rule: "ALL bar",
				},
			}

			org := &configtx.Organization{
				Name:           "testorg",
				ID:             "testorg",
				MSPType:        "bccsp",
				MSPDir:         "../../../../testdata/init/orderer/msp",
				AdminPrincipal: "Role.MEMBER",
				Policies:       profile.Policies,
			}

			consortium := &configtx.Consortium{
				Organizations: []*configtx.Organization{org},
			}

			profile.SetOrdererType("etcdraft")
			profile.AddConsortium("testconsortium", consortium)

			org.Name = "testorg2"
			err := profile.AddOrgToConsortium("testconsortium", org)
			Expect(err).NotTo(HaveOccurred())

			blockBytes, err := profile.GenerateBlock("channel1", mspConfig)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(blockBytes)).To(ContainSubstring("testconsortium"))
			Expect(string(blockBytes)).To(ContainSubstring("testorg2"))
		})
	})

	It("adds org to orderer", func() {
		org := &configtx.Organization{
			Name:           "testorg3",
			ID:             "testorg3",
			MSPType:        "bccsp",
			MSPDir:         "../../../../testdata/init/orderer/msp",
			AdminPrincipal: "Role.MEMBER",
		}

		profile.SetOrdererType("etcdraft")
		err := profile.AddOrgToOrderer(org)
		Expect(err).NotTo(HaveOccurred())

		blockBytes, err := profile.GenerateBlock("channel1", mspConfig)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(blockBytes)).To(ContainSubstring("testorg3"))
	})
})
