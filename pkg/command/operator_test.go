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

package command_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	oconfig "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/command"
	"github.com/IBM-Blockchain/fabric-operator/pkg/command/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
)

var _ = Describe("Operator command", func() {
	Context("config initialization", func() {
		var config *oconfig.Config

		BeforeEach(func() {
			os.Setenv("CLUSTERTYPE", "K8S")

			config = &oconfig.Config{
				Operator: oconfig.Operator{
					Versions: &deployer.Versions{
						CA:      map[string]deployer.VersionCA{},
						Peer:    map[string]deployer.VersionPeer{},
						Orderer: map[string]deployer.VersionOrderer{},
					},
				},
			}
		})

		Context("cluster type", func() {
			It("returns error for invalid cluster type value", func() {
				os.Setenv("CLUSTERTYPE", "")
				err := command.InitConfig("", config, &mocks.Reader{})
				Expect(err).To(HaveOccurred())
			})

			It("sets value", func() {
				os.Setenv("CLUSTERTYPE", "K8S")
				err := command.InitConfig("", config, &mocks.Reader{})
				Expect(err).NotTo(HaveOccurred())
				Expect(config.Offering).To(Equal(offering.K8S))
			})
		})

		Context("secret poll timeout", func() {
			It("returns default value inf invalid timeout value set", func() {
				os.Setenv("IBPOPERATOR_ORDERER_TIMEOUTS_SECRETPOLL", "45")
				err := command.InitConfig("", config, &mocks.Reader{})
				Expect(err).To(HaveOccurred())
			})

			It("sets value", func() {
				os.Setenv("IBPOPERATOR_ORDERER_TIMEOUTS_SECRETPOLL", "45s")
				err := command.InitConfig("", config, &mocks.Reader{})
				Expect(err).NotTo(HaveOccurred())
				Expect(config.Operator.Orderer.Timeouts.SecretPoll).To(Equal(common.MustParseDuration("45s")))
			})
		})
	})
})
