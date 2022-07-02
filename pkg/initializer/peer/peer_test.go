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

package initializer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v1"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	configmocks "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config/mocks"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
)

var _ = Describe("peer", func() {
	var (
		peer *initializer.Peer

		mockCrypto *configmocks.Crypto
	)

	BeforeEach(func() {
		mockCrypto = &configmocks.Crypto{}

		peer = &initializer.Peer{
			Config: &config.Core{},
			Cryptos: &commonconfig.Cryptos{
				Enrollment: mockCrypto,
			},
		}
	})

	Context("config override", func() {
		When("using hsm proxy", func() {
			BeforeEach(func() {
				peer.UsingHSMProxy = true
			})

			It("overrides peer's config", func() {
				core := &config.Core{
					Core: v1.Core{
						Peer: v1.Peer{
							BCCSP: &commonapi.BCCSP{
								ProviderName: "PKCS11",
								PKCS11:       &commonapi.PKCS11Opts{},
							},
						},
					},
				}

				err := peer.OverrideConfig(core)
				Expect(err).NotTo(HaveOccurred())

				Expect(core.Peer.BCCSP.PKCS11.Library).To(Equal("/usr/local/lib/libpkcs11-proxy.so"))
			})
		})
	})

	Context("generate crypto", func() {
		It("returns error if unable to get crypto", func() {
			mockCrypto.GetCryptoReturns(nil, errors.New("get crypto error"))
			_, err := peer.GenerateCrypto()
			Expect(err).To(HaveOccurred())
		})

		It("gets crypto", func() {
			resp, err := peer.GenerateCrypto()
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})
	})
})
