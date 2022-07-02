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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	common "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Peer configuration", func() {
	Context("verify cert OU", func() {
		var (
			resp         *common.Response
			certtemplate *x509.Certificate
		)

		BeforeEach(func() {
			certtemplate = &x509.Certificate{
				SerialNumber: big.NewInt(1),
				Subject: pkix.Name{
					OrganizationalUnit: []string{"peer", "orderer", "admin"},
				},
			}
			certBytes := createCertBytes(certtemplate)

			resp = &common.Response{
				SignCert:   certBytes,
				AdminCerts: [][]byte{certBytes},
			}
		})

		It("returns error if peer signcert doesn't have OU type 'peer'", func() {
			certtemplate.Subject.OrganizationalUnit = []string{"invalidou"}
			certbytes := createCertBytes(certtemplate)
			resp.SignCert = certbytes

			err := resp.VerifyCertOU("peer")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid OU for signcert: cert does not have right OU, expecting 'peer'"))
		})

		It("return error if sign cert has no OU defined", func() {
			certtemplate.Subject.OrganizationalUnit = nil
			certbytes := createCertBytes(certtemplate)
			resp.SignCert = certbytes

			err := resp.VerifyCertOU("peer")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invalid OU for signcert: OU not defined"))
		})

		It("verifies that peer signcert has correct OU", func() {
			err := resp.VerifyCertOU("peer")
			Expect(err).NotTo(HaveOccurred())
		})

		It("verifies that orderer signcert and admincerts have correct OU", func() {
			err := resp.VerifyCertOU("orderer")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("generate crypto response", func() {
		var (
			cryptos          *config.Cryptos
			enrollmentCrypto *mocks.Crypto
			tlsCrypto        *mocks.Crypto
			clientAuthCrypto *mocks.Crypto
		)

		BeforeEach(func() {
			enrollmentCrypto = &mocks.Crypto{}
			tlsCrypto = &mocks.Crypto{}
			clientAuthCrypto = &mocks.Crypto{}

			cryptos = &config.Cryptos{
				Enrollment: enrollmentCrypto,
				TLS:        tlsCrypto,
				ClientAuth: clientAuthCrypto,
			}
		})

		Context("enrollment", func() {
			It("returns an error on failure", func() {
				msg := "could not enrollment get crypto"
				enrollmentCrypto.GetCryptoReturns(nil, errors.New(msg))
				_, err := cryptos.GenerateCryptoResponse()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(msg))
			})
		})

		Context("tls", func() {
			It("returns an error on failure", func() {
				msg := "could not tls get crypto"
				tlsCrypto.GetCryptoReturns(nil, errors.New(msg))
				_, err := cryptos.GenerateCryptoResponse()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(msg))
			})
		})

		Context("client auth", func() {
			It("returns an error on failure", func() {
				msg := "could not client auth get crypto"
				clientAuthCrypto.GetCryptoReturns(nil, errors.New(msg))
				_, err := cryptos.GenerateCryptoResponse()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(msg))
			})
		})

		It("gets crypto", func() {
			resp, err := cryptos.GenerateCryptoResponse()
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})
	})
})

func createCertBytes(certTemplate *x509.Certificate) []byte {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())

	cert, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &priv.PublicKey, priv)
	Expect(err).NotTo(HaveOccurred())

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}

	return pem.EncodeToMemory(block)
}
