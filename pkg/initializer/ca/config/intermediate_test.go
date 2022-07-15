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
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Intermediate config", func() {
	var (
		cfg     *config.Config
		homeDir = "interconfigtest"
	)

	BeforeEach(func() {
		os.Mkdir(homeDir, 0777)
	})

	AfterEach(func() {
		err := os.RemoveAll(homeDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("parses intermediate configuration", func() {
		BeforeEach(func() {
			cfg = &config.Config{
				ServerConfig: &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						Intermediate: v1.IntermediateCA{
							TLS: v1.ClientTLSConfig{
								Enabled:   pointer.True(),
								CertFiles: []string{certFile},
								Client: v1.KeyCertFiles{
									CertFile: certFile,
									KeyFile:  keyFile,
								},
							},
						},
					},
				},
				HomeDir: homeDir,
			}
		})

		It("parses config and returns a map containing all crypto and updated paths to crypto material", func() {
			crypto, err := cfg.ParseIntermediateBlock()
			Expect(err).NotTo(HaveOccurred())

			certData, certKeyExists := crypto["parent-cert.pem"]
			Expect(certKeyExists).To(Equal(true))
			Expect(certData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.Intermediate.TLS.Client.CertFile).To(Equal(filepath.Join(cfg.HomeDir, "parent-cert.pem")))

			keyData, keyKeyExists := crypto["parent-key.pem"]
			Expect(keyKeyExists).To(Equal(true))
			Expect(keyData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.Intermediate.TLS.Client.KeyFile).To(Equal(filepath.Join(cfg.HomeDir, "parent-key.pem")))

			chainData, chainKeyExists := crypto["parent-certfile0.pem"]
			Expect(chainKeyExists).To(Equal(true))
			Expect(chainData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.Intermediate.TLS.CertFiles[0]).To(Equal(filepath.Join(cfg.HomeDir, "parent-certfile0.pem")))
		})
	})
})
