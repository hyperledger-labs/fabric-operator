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

package v25_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v1"
	v24 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v24"
	v25 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v25"
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v25"
)

var _ = Describe("V2 Orderer Configuration", func() {
	Context("reading and writing orderer configuration file", func() {
		BeforeEach(func() {
			config := &config.Orderer{}

			err := config.WriteToFile("/tmp/orderer.yaml")
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates orderer.yaml", func() {
			Expect("/tmp/orderer.yaml").Should(BeAnExistingFile())
		})

		It("read orderer.yaml", func() {
			_, err := config.ReadOrdererFile("/tmp/orderer.yaml")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("merges current configuration with overrides values", func() {
		It("merges with defaults based on HSM proxy", func() {
			orderer, err := config.ReadOrdererFile("../../../../../testdata/init/orderer/orderer.yaml")
			Expect(err).NotTo(HaveOccurred())

			newConfig := &config.Orderer{
				Orderer: v25.Orderer{
					General: v24.General{
						BCCSP: &commonapi.BCCSP{
							ProviderName: "PKCS11",
							PKCS11: &commonapi.PKCS11Opts{
								Library:    "library2",
								Label:      "label2",
								Pin:        "2222",
								HashFamily: "SHA3",
								SecLevel:   512,
								FileKeyStore: &commonapi.FileKeyStoreOpts{
									KeyStorePath: "keystore3",
								},
							},
						},
					},
				},
			}

			err = orderer.MergeWith(newConfig, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(orderer.General.BCCSP.PKCS11.Library).To(Equal("/usr/local/lib/libpkcs11-proxy.so"))
			Expect(orderer.General.BCCSP.PKCS11.Label).To(Equal("label2"))
			Expect(orderer.General.BCCSP.PKCS11.Pin).To(Equal("2222"))
			Expect(orderer.General.BCCSP.PKCS11.HashFamily).To(Equal("SHA3"))
			Expect(orderer.General.BCCSP.PKCS11.SecLevel).To(Equal(512))
			Expect(orderer.General.BCCSP.PKCS11.FileKeyStore.KeyStorePath).To(Equal("keystore3"))
		})

		It("correctly merges boolean fields", func() {
			orderer, err := config.ReadOrdererFile("../../../../../testdata/init/orderer/orderer.yaml")
			Expect(err).NotTo(HaveOccurred())

			trueVal := true
			orderer.General.Authentication.NoExpirationChecks = &trueVal
			orderer.General.Profile.Enabled = &trueVal
			Expect(*orderer.General.Authentication.NoExpirationChecks).To(Equal(true))
			Expect(*orderer.General.Profile.Enabled).To(Equal(true))

			falseVal := false
			newConfig := &config.Orderer{
				Orderer: v25.Orderer{
					General: v24.General{
						Authentication: v1.Authentication{
							NoExpirationChecks: &falseVal,
						},
					},
				},
			}

			err = orderer.MergeWith(newConfig, false)
			Expect(err).NotTo(HaveOccurred())

			By("setting field from 'true' to 'false' if bool pointer set to 'false' in override config", func() {
				Expect(*orderer.General.Authentication.NoExpirationChecks).To(Equal(false))
			})

			By("persisting boolean fields set to 'true' when bool pointer not set to 'false' in override config", func() {
				Expect(*orderer.General.Profile.Enabled).To(Equal(true))
			})

		})
	})

	It("reads in orderer.yaml and unmarshal it to peer config", func() {
		orderer, err := config.ReadOrdererFile("../../../../../testdata/init/orderer/orderer.yaml")
		Expect(err).NotTo(HaveOccurred())

		// General
		general := orderer.General
		By("setting General.ListenAddress", func() {
			Expect(general.ListenAddress).To(Equal("127.0.0.1"))
		})

		By("setting General.ListenPort", func() {
			Expect(general.ListenPort).To(Equal(uint16(7050)))
		})

		By("setting General.TLS.Enabled", func() {
			Expect(*general.TLS.Enabled).To(Equal(true))
		})

		By("setting General.TLS.PrivateKey", func() {
			Expect(general.TLS.PrivateKey).To(Equal("tls/server.key"))
		})

		By("setting General.TLS.Certificate", func() {
			Expect(general.TLS.Certificate).To(Equal("tls/server.crt"))
		})

		By("setting General.TLS.RootCAs", func() {
			Expect(general.TLS.RootCAs).To(Equal([]string{"tls/ca.crt"}))
		})

		By("setting General.TLS.ClientAuthRequired", func() {
			Expect(*general.TLS.ClientAuthRequired).To(Equal(true))
		})

		By("setting General.TLS.ClientRootCAs", func() {
			Expect(general.TLS.ClientRootCAs).To(Equal([]string{"tls/client.crt"}))
		})

		By("setting General.BCCSP.ProviderName", func() {
			Expect(general.BCCSP.ProviderName).To(Equal("SW"))
		})

		By("setting General.BCCSP.SW.HashFamily", func() {
			Expect(general.BCCSP.SW.HashFamily).To(Equal("SHA2"))
		})

		By("setting General.BCCSP.SW.SecLevel", func() {
			Expect(general.BCCSP.SW.SecLevel).To(Equal(256))
		})

		By("setting General.BCCSP.SW.FileKeyStore.KeyStore", func() {
			Expect(general.BCCSP.SW.FileKeyStore.KeyStorePath).To(Equal("msp/keystore"))
		})

		By("setting BCCSP.PKCS11.Library", func() {
			Expect(general.BCCSP.PKCS11.Library).To(Equal("library1"))
		})

		By("setting BCCSP.PKCS11.Label", func() {
			Expect(general.BCCSP.PKCS11.Label).To(Equal("label1"))
		})

		By("setting BCCSP.PKCS11.Pin", func() {
			Expect(general.BCCSP.PKCS11.Pin).To(Equal("1234"))
		})

		By("setting BCCSP.PKCS11.HashFamily", func() {
			Expect(general.BCCSP.PKCS11.HashFamily).To(Equal("SHA2"))
		})

		By("setting BCCSP.PKCS11.Security", func() {
			Expect(general.BCCSP.PKCS11.SecLevel).To(Equal(256))
		})

		By("setting BCCSP.PKCS11.FileKeystore.KeystorePath", func() {
			Expect(general.BCCSP.PKCS11.FileKeyStore.KeyStorePath).To(Equal("keystore2"))
		})
	})
})
