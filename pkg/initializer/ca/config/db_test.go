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

var _ = Describe("DB config", func() {
	const (
		homeDir = "homedir"
	)

	BeforeEach(func() {
		os.Mkdir(homeDir, 0777)
	})

	AfterEach(func() {
		err := os.RemoveAll(homeDir)
		Expect(err).NotTo(HaveOccurred())
	})

	var cfg *config.Config

	Context("parses DB configuration", func() {
		BeforeEach(func() {
			cfg = &config.Config{
				ServerConfig: &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						DB: &v1.CAConfigDB{
							Type:       string(config.Postgres),
							Datasource: "host=0.0.0.0 port=8080 user=db password=db dbname=fabric sslmode=true",
							TLS: v1.ClientTLSConfig{
								Enabled:   pointer.True(),
								CertFiles: []string{"../../../../testdata/tls/tls.crt"},
								Client: v1.KeyCertFiles{
									CertFile: certFile,
									KeyFile:  keyFile,
								},
							},
						},
					},
				},
				HomeDir:    homeDir,
				SqlitePath: "/tmp/ca.db",
			}
		})

		It("returns an error if invalid database type specified", func() {
			cfg.ServerConfig.CAConfig.DB.Type = "couchdb"
			_, err := cfg.ParseDBBlock()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("database type 'couchdb' is not supported"))
		})

		It("returns an error if mysql database type specified", func() {
			cfg.ServerConfig.CAConfig.DB.Type = string(config.MySQL)
			_, err := cfg.ParseDBBlock()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("MySQL is not supported"))
		})

		It("returns no error and an empty map if TLS disabled", func() {
			cfg.ServerConfig.CAConfig.DB.TLS.Enabled = pointer.False()
			crypto, err := cfg.ParseDBBlock()
			Expect(err).NotTo(HaveOccurred())
			Expect(crypto).To(BeNil())
		})

		It("returns an error if missing datasource", func() {
			cfg.ServerConfig.CAConfig.DB.Datasource = ""
			_, err := cfg.ParseDBBlock()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no datasource string specified for postgres"))
		})

		It("returns an error if datasource is unexpected format", func() {
			cfg.ServerConfig.CAConfig.DB.Datasource = "dbname=testdb"
			_, err := cfg.ParseDBBlock()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("datasource for postgres is not valid"))
		})

		It("parses config and returns a map containing all db crypto and updated paths to crypto material", func() {
			crypto, err := cfg.ParseDBBlock()
			Expect(err).NotTo(HaveOccurred())

			certData, certKeyExists := crypto["db-cert.pem"]
			Expect(certKeyExists).To(Equal(true))
			Expect(certData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.DB.TLS.Client.CertFile).To(Equal(filepath.Join(cfg.HomeDir, "db-cert.pem")))

			keyData, keyKeyExists := crypto["db-key.pem"]
			Expect(keyKeyExists).To(Equal(true))
			Expect(keyData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.DB.TLS.Client.KeyFile).To(Equal(filepath.Join(cfg.HomeDir, "db-key.pem")))

			clientAuthData, clientAuthCertKeyExists := crypto["db-certfile0.pem"]
			Expect(clientAuthCertKeyExists).To(Equal(true))
			Expect(clientAuthData).NotTo(BeNil())
			Expect(cfg.ServerConfig.CAConfig.DB.TLS.CertFiles[0]).To(Equal(filepath.Join(cfg.HomeDir, "db-certfile0.pem")))
		})

		It("creates SQLLite database and returns empty crypto map", func() {
			cfg.ServerConfig.CAConfig.DB.Type = string(config.SQLLite)
			crypto, err := cfg.ParseDBBlock()
			Expect(err).NotTo(HaveOccurred())
			Expect(crypto).To(BeNil())

			os.RemoveAll("dbconfigtest")
		})
	})
})
