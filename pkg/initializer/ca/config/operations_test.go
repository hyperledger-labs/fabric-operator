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

var _ = Describe("Operations config", func() {
	var (
		cfg     *config.Config
		homeDir = "operationsconfigtest"
	)

	BeforeEach(func() {
		os.Mkdir(homeDir, 0777)
	})

	AfterEach(func() {
		err := os.RemoveAll(homeDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("parses Operations configuration", func() {
		BeforeEach(func() {
			cfg = &config.Config{
				ServerConfig: &v1.ServerConfig{
					Operations: v1.Options{
						TLS: v1.TLS{
							Enabled:           pointer.True(),
							CertFile:          certFile,
							KeyFile:           keyFile,
							ClientCACertFiles: []string{certFile},
						},
					},
				},
				HomeDir: homeDir,
			}
		})

		It("returns no error and an empty map if TLS disabled", func() {
			cfg.ServerConfig.Operations.TLS.Enabled = pointer.False()
			crypto, err := cfg.ParseOperationsBlock()
			Expect(err).NotTo(HaveOccurred())
			Expect(crypto).To(BeNil())
		})

		It("parses config and returns a map containing all crypto and updated paths to crypto material", func() {
			crypto, err := cfg.ParseOperationsBlock()
			Expect(err).NotTo(HaveOccurred())

			certData, certKeyExists := crypto["operations-cert.pem"]
			Expect(certKeyExists).To(Equal(true))
			Expect(certData).NotTo(BeNil())
			Expect(cfg.ServerConfig.Operations.TLS.CertFile).To(Equal(filepath.Join(cfg.HomeDir, "operations-cert.pem")))

			keyData, keyKeyExists := crypto["operations-key.pem"]
			Expect(keyKeyExists).To(Equal(true))
			Expect(keyData).NotTo(BeNil())
			Expect(cfg.ServerConfig.Operations.TLS.KeyFile).To(Equal(filepath.Join(cfg.HomeDir, "operations-key.pem")))

			chainData, chainKeyExists := crypto["operations-certfile0.pem"]
			Expect(chainKeyExists).To(Equal(true))
			Expect(chainData).NotTo(BeNil())
			Expect(cfg.ServerConfig.Operations.TLS.ClientCACertFiles[0]).To(Equal(filepath.Join(cfg.HomeDir, "operations-certfile0.pem")))
		})
	})
})
