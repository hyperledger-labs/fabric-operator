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
	"path/filepath"

	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("IBPCA", func() {
	Context("reading file", func() {
		It("fails load configuration file that doesn't exist", func() {
			_, err := initializer.LoadConfigFromFile("notexist.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such file or directory"))
		})

		It("loads from ca configuration file", func() {
			cfg, err := initializer.LoadConfigFromFile("../../../defaultconfig/ca/ca.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
		})
	})

	Context("ca initializion", func() {
		var (
			ca            *initializer.CA
			defaultConfig *mocks.CAConfig
		)

		BeforeEach(func() {
			cfg := &config.Config{
				HomeDir:   "ca_test",
				MountPath: "/mount",
				ServerConfig: &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						CSP: &v1.BCCSP{
							ProviderName: "PKCS11",
							PKCS11: &v1.PKCS11Opts{
								Pin:   "1234",
								Label: "root",
							},
						},
						CSR: v1.CSRInfo{
							CN: "ca",
						},
					},
				},
			}

			defaultConfig = &mocks.CAConfig{}
			defaultConfig.GetServerConfigReturns(cfg.ServerConfig)
			defaultConfig.GetHomeDirReturns(cfg.HomeDir)

			ca = initializer.NewCA(defaultConfig, config.EnrollmentCA, "/tmp", true, "ca_test")
		})

		It("sets default values", func() {
			err := ca.OverrideServerConfig(nil)
			Expect(err).NotTo(HaveOccurred())

			By("enabling removal of identities and affiliations", func() {
				Expect(*defaultConfig.GetServerConfig().CAConfig.Cfg.Identities.AllowRemove).To(Equal(true))
				Expect(*defaultConfig.GetServerConfig().CAConfig.Cfg.Affiliations.AllowRemove).To(Equal(true))
			})
			By("enabling ignore cert expiry for re-enroll", func() {
				Expect(*defaultConfig.GetServerConfig().CAConfig.CA.ReenrollIgnoreCertExpiry).To(Equal(true))
			})
		})

		It("does not crash if CSP is nil in override config", func() {
			override := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					CSP: nil,
				},
			}

			err := ca.OverrideServerConfig(override)
			Expect(err).NotTo(HaveOccurred())

			By("setting in defaults when using pkcs11", func() {
				Expect(defaultConfig.GetServerConfig().CAConfig.CSP).To(Equal(ca.GetServerConfig().CAConfig.CSP))
				Expect(defaultConfig.GetServerConfig().CAConfig.CSR.CN).To(Equal("ca_test"))
			})
		})

		It("overrides config", func() {
			override := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					CSP: &v1.BCCSP{
						ProviderName: "PKCS11",
					},
				},
			}
			defaultConfig.UsingPKCS11Returns(true)

			err := ca.OverrideServerConfig(override)
			Expect(err).NotTo(HaveOccurred())

			By("setting in defaults when using pkcs11", func() {
				Expect(defaultConfig.GetServerConfig().CAConfig.CSP.PKCS11.Library).To(Equal("/usr/local/lib/libpkcs11-proxy.so"))
				Expect(defaultConfig.GetServerConfig().CAConfig.CSP.PKCS11.FileKeyStore.KeyStorePath).To(Equal("msp/keystore"))
				Expect(defaultConfig.GetServerConfig().CAConfig.CSP.PKCS11.HashFamily).To(Equal("SHA2"))
				Expect(defaultConfig.GetServerConfig().CAConfig.CSP.PKCS11.SecLevel).To(Equal(256))
				Expect(defaultConfig.GetServerConfig().CAConfig.CSR.CN).To(Equal("ca_test"))
			})
		})

		It("successfully completes initializing intermediate ca", func() {
			override := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					Intermediate: v1.IntermediateCA{
						ParentServer: v1.ParentServer{
							URL: "127.0.0.1",
						},
					},
				},
			}

			err := ca.OverrideServerConfig(override)
			Expect(err).NotTo(HaveOccurred())

			By("setting cn in csr to be empty", func() {
				Expect(defaultConfig.GetServerConfig().CAConfig.CSR.CN).To(Equal(""))
			})
		})

		It("writes configuration to file", func() {
			err := ca.WriteConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(defaultConfig.GetHomeDir(), "fabric-ca-server-config.yaml")).Should(BeAnExistingFile())

			ca.RemoveHomeDir()
		})

		Context("run fabric-ca init", func() {
			BeforeEach(func() {
				cfg, err := initializer.LoadConfigFromFile("../../../defaultconfig/ca/ca.yaml")
				Expect(err).NotTo(HaveOccurred())
				defaultConfig.GetServerConfigReturns(cfg)

				err = ca.WriteConfig()
				Expect(err).NotTo(HaveOccurred())
			})

			It("successfully completes Initializing ca", func() {
				err := ca.Init()
				Expect(err).NotTo(HaveOccurred())

				By("setting ca files property to point to tls ca config file", func() {
					Expect(defaultConfig.GetServerConfig().CAfiles).To(Equal([]string{"/data/tlsca/fabric-ca-server-config.yaml"}))
				})

				By("setting cert/key file to generate location", func() {
					Expect(defaultConfig.GetServerConfig().CA.Certfile).To(ContainSubstring(filepath.Join(ca.Config.GetHomeDir(), "ca-cert.pem")))
					Expect(defaultConfig.GetServerConfig().CA.Keyfile).To(ContainSubstring(filepath.Join(ca.Config.GetHomeDir(), "ca-key.pem")))
				})

			})

			AfterEach(func() {
				err := ca.RemoveHomeDir()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("viper unmarshal", func() {
			It("returns an error if fails to find file", func() {
				_, err := ca.ViperUnmarshal("../../../defaultconfig/ca/foo.yaml")
				Expect(err).To(HaveOccurred())
			})

			It("successfully unmarshals", func() {
				cfg, err := ca.ViperUnmarshal("../../../defaultconfig/ca/tlsca.yaml")
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
			})
		})

		Context("parse enrollment ca crypto", func() {
			BeforeEach(func() {
				defaultConfig.GetServerConfig().TLS = v1.ServerTLSConfig{
					Enabled:  pointer.True(),
					CertFile: "../../../testdata/tls/tls.crt",
					KeyFile:  "../../../testdata/tls/tls.key",
				}

				defaultConfig.GetServerConfig().Operations = v1.Options{
					TLS: v1.TLS{
						Enabled: pointer.True(),
					},
				}
			})

			It("returns an error if TLS cert and key not provided", func() {
				defaultConfig.GetServerConfig().TLS.CertFile = ""
				defaultConfig.GetServerConfig().TLS.KeyFile = ""

				_, err := ca.ParseEnrollmentCACrypto()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("no TLS cert and key file provided"))
			})

			It("returns an error is parsing ca blocks fails", func() {
				msg := "failed ca parse"
				defaultConfig.ParseCABlockReturns(nil, errors.New(msg))

				_, err := ca.ParseEnrollmentCACrypto()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to parse ca block: " + msg))
			})

			It("returns an error is parsing TLS blocks fails", func() {
				msg := "failed tls parse"
				defaultConfig.ParseTLSBlockReturns(nil, errors.New(msg))

				_, err := ca.ParseEnrollmentCACrypto()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to parse tls block: " + msg))
			})

			It("returns an error is parsing DB blocks fails", func() {
				msg := "failed db parse"
				defaultConfig.ParseDBBlockReturns(nil, errors.New(msg))

				_, err := ca.ParseEnrollmentCACrypto()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to parse db block: " + msg))
			})

			It("returns an error is parsing operations blocks fails", func() {
				msg := "failed operations parse"
				defaultConfig.ParseOperationsBlockReturns(nil, errors.New(msg))

				_, err := ca.ParseEnrollmentCACrypto()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to parse operations block: " + msg))
			})

			It("returns an error is parsing intermediate blocks fails", func() {
				msg := "failed operations parse"
				defaultConfig.ParseIntermediateBlockReturns(nil, errors.New(msg))

				_, err := ca.ParseEnrollmentCACrypto()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to parse intermediate block: " + msg))
			})

			It("sets the operations TLS path to be equal to server's TLS path", func() {
				_, err := ca.ParseEnrollmentCACrypto()
				Expect(err).NotTo(HaveOccurred())

				Expect(defaultConfig.GetServerConfig().Operations.TLS.CertFile).To(ContainSubstring("tls/tls.crt"))
				Expect(defaultConfig.GetServerConfig().Operations.TLS.KeyFile).To(ContainSubstring("tls/tls.key"))
			})
		})

		Context("parse TLS ca crypto", func() {
			It("returns an error is parsing ca blocks fails", func() {
				msg := "failed ca parse"
				defaultConfig.ParseCABlockReturns(nil, errors.New(msg))

				_, err := ca.ParseTLSCACrypto()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to parse ca block: " + msg))
			})

			It("parses ca blocks fails", func() {
				_, err := ca.ParseTLSCACrypto()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
