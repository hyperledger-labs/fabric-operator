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

package config_test

import (
	"os"
	"path/filepath"

	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CA config", func() {
	var (
		cfg     *config.Config
		homeDir = "caconfigtest"
	)

	BeforeEach(func() {
		os.Mkdir(homeDir, 0777)
	})

	AfterEach(func() {
		err := os.RemoveAll(homeDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("parses CA configuration", func() {
		BeforeEach(func() {
			cfg = &config.Config{
				ServerConfig: &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						CA: v1.CAInfo{
							Certfile:  certFile,
							Keyfile:   keyFile,
							Chainfile: certFile,
						},
					},
				},
				HomeDir: homeDir,
			}
		})

		It("returns an error if unexpected type passed for keyfile and no key found in keystore", func() {
			cfg.HomeDir = "fake"
			cfg.ServerConfig.CAConfig.CA.Keyfile = "invalidType"
			_, err := cfg.ParseCABlock()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such file or directory"))
			os.RemoveAll(cfg.HomeDir)
		})

		It("if key is unexpected type look in keystore folder for key", func() {
			cfg.HomeDir = "../../../../testdata"
			cfg.ServerConfig.CAConfig.CA.Keyfile = "invalidType"
			crypto, err := cfg.ParseCABlock()
			Expect(err).NotTo(HaveOccurred())

			keyData, keyKeyExists := crypto["key.pem"]
			Expect(keyKeyExists).To(Equal(true))
			Expect(keyData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.CA.Keyfile).To(Equal(filepath.Join(cfg.HomeDir, "key.pem")))

			os.Remove(filepath.Join(cfg.HomeDir, "cert.pem"))
			os.Remove(filepath.Join(cfg.HomeDir, "key.pem"))
			os.Remove(filepath.Join(cfg.HomeDir, "chain.pem"))
		})

		It("returns if unexpected type passed for trusted root cert files", func() {
			cfg.ServerConfig.CAConfig.CA.Chainfile = "invalidType"
			c, err := cfg.ParseCABlock()
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
		})

		It("parses config and returns a map containing all crypto and updated paths to crypto material", func() {
			crypto, err := cfg.ParseCABlock()
			Expect(err).NotTo(HaveOccurred())

			certData, certKeyExists := crypto["cert.pem"]
			Expect(certKeyExists).To(Equal(true))
			Expect(certData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.CA.Certfile).To(Equal(filepath.Join(cfg.HomeDir, "cert.pem")))

			keyData, keyKeyExists := crypto["key.pem"]
			Expect(keyKeyExists).To(Equal(true))
			Expect(keyData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.CA.Keyfile).To(Equal(filepath.Join(cfg.HomeDir, "key.pem")))

			chainData, chainKeyExists := crypto["chain.pem"]
			Expect(chainKeyExists).To(Equal(true))
			Expect(chainData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.CA.Chainfile).To(Equal(filepath.Join(cfg.HomeDir, "chain.pem")))
		})
	})
})
