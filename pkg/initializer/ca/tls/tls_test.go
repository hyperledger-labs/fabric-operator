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

package tls_test

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/tls"
	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/hyperledger/fabric-ca/api"
	"github.com/hyperledger/fabric/bccsp/factory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("generating TLS crypto", func() {
	var (
		tlsGen *tls.TLS
		csr    *api.CSRInfo
	)

	BeforeEach(func() {
		csp := &factory.FactoryOpts{
			ProviderName: "SW",
		}
		tlsGen = &tls.TLS{
			CAHomeDir: "crypto",
			CSP:       csp,
		}

		csr = &api.CSRInfo{
			CN: "tls-ca",
			Names: []cfcsr.Name{
				cfcsr.Name{
					C:  "United States",
					ST: "North Carolina",
					L:  "Raleigh",
					O:  "IBM",
					OU: "Blockchain",
				},
			},
			Hosts: []string{"localhost", "127.0.0.1"},
		}
	})

	AfterEach(func() {
		err := os.RemoveAll("crypto")
		Expect(err).NotTo(HaveOccurred())
	})

	It("generates key and self-signed TLS certificate", func() {
		certBytes, err := tlsGen.GenerateSelfSignedTLSCrypto(csr)
		Expect(err).NotTo(HaveOccurred())

		By("returning a properly populated certificate", func() {
			cert, err := x509.ParseCertificate(certBytes)
			Expect(err).NotTo(HaveOccurred())

			Expect(cert.Subject.Country).To(Equal([]string{"United States"}))
			Expect(cert.Subject.Province).To(Equal([]string{"North Carolina"}))
			Expect(cert.Subject.Locality).To(Equal([]string{"Raleigh"}))
			Expect(cert.Subject.Organization).To(Equal([]string{"IBM"}))
			Expect(cert.Subject.OrganizationalUnit).To(Equal([]string{"Blockchain"}))

			Expect(cert.DNSNames[0]).To(Equal("localhost"))
			Expect(fmt.Sprintf("%s", cert.IPAddresses[0])).To(Equal("127.0.0.1"))

			Expect(cert.Subject).To(Equal(cert.Issuer))
		})

		By("writing the private key to proper location", func() {
			keystorePath := filepath.Join(tlsGen.CAHomeDir, "tls/msp/keystore")
			Expect(keystorePath).Should(BeADirectory())

			files, err := ioutil.ReadDir(keystorePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(files)).NotTo(Equal(0))
			Expect(files[0].Name()).To(ContainSubstring("sk"))
		})
	})

	It("stores the certificate in the proper directory", func() {
		certBytes, err := tlsGen.GenerateSelfSignedTLSCrypto(csr)
		Expect(err).NotTo(HaveOccurred())

		err = tlsGen.WriteCryptoToFile(certBytes, "tls-cert.pem")
		Expect(err).NotTo(HaveOccurred())

		certPath := filepath.Join(tlsGen.CAHomeDir, "tls", "tls-cert.pem")
		Expect(certPath).Should(BeAnExistingFile())
	})
})
