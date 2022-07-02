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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
)

var _ = Describe("TLS Config", func() {
	const (
		homeDir = "configtest"
	)

	Context("parses TLS configuration", func() {
		var cfg *config.Config

		BeforeEach(func() {
			cfg = &config.Config{
				ServerConfig: &v1.ServerConfig{
					TLS: v1.ServerTLSConfig{
						Enabled:  pointer.True(),
						CertFile: certFile,
						KeyFile:  keyFile,
						ClientAuth: v1.ClientAuth{
							CertFiles: []string{"../../../../testdata/tls/tls.crt"},
						},
					},
				},
				HomeDir: homeDir,
			}

			os.Mkdir(homeDir, 0777)
		})

		AfterEach(func() {
			err := os.RemoveAll(homeDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns no error and an empty map if TLS disabled", func() {
			cfg.ServerConfig.TLS.Enabled = pointer.False()
			crypto, err := cfg.ParseTLSBlock()
			Expect(err).NotTo(HaveOccurred())
			Expect(crypto).To(BeNil())
		})

		It("parses config and returns a map containing all crypto and updated paths to crypto material", func() {
			crypto, err := cfg.ParseTLSBlock()
			Expect(err).NotTo(HaveOccurred())

			certData, certKeyExists := crypto["tls-cert.pem"]
			Expect(certKeyExists).To(Equal(true))
			Expect(certData).NotTo(BeNil())
			Expect(cfg.ServerConfig.TLS.CertFile).To(Equal(filepath.Join(cfg.HomeDir, "tls-cert.pem")))

			keyData, keyKeyExists := crypto["tls-key.pem"]
			Expect(keyKeyExists).To(Equal(true))
			Expect(keyData).NotTo(BeNil())
			Expect(cfg.ServerConfig.TLS.KeyFile).To(Equal(filepath.Join(cfg.HomeDir, "tls-key.pem")))

			clientAuthData, clientAuthCertKeyExists := crypto["tls-certfile0.pem"]
			Expect(clientAuthCertKeyExists).To(Equal(true))
			Expect(clientAuthData).NotTo(BeNil())
			Expect(cfg.ServerConfig.TLS.ClientAuth.CertFiles[0]).To(Equal(filepath.Join(cfg.HomeDir, "tls-certfile0.pem")))
		})
	})
})
