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

package tls

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/cloudflare/cfssl/csr"
	"github.com/hyperledger/fabric-ca/api"
	cautil "github.com/hyperledger/fabric-ca/util"
	"github.com/hyperledger/fabric-lib-go/bccsp/factory"
)

type TLS struct {
	CAHomeDir string
	CSP       *factory.FactoryOpts
}

func (t *TLS) GenerateSelfSignedTLSCrypto(csr *api.CSRInfo) ([]byte, error) {
	err := os.RemoveAll(filepath.Join(t.CAHomeDir, "tls"))
	if err != nil {
		return nil, err
	}

	csp, err := cautil.InitBCCSP(&t.CSP, "msp", filepath.Join(t.CAHomeDir, "tls"))
	if err != nil {
		return nil, err
	}

	cr := NewCertificateRequest(csr)
	privKey, signer, err := cautil.BCCSPKeyRequestGenerate(cr, csp)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notBefore.Add(-5 * time.Minute)
	notAfter := notBefore.Add(time.Hour * 24 * 365) // Valid for one year

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	subject := pkix.Name{
		CommonName:   csr.CN,
		SerialNumber: csr.SerialNumber,
	}
	if len(csr.Names) != 0 {
		for _, name := range csr.Names {
			subject.Country = append(subject.Country, name.C)
			subject.Province = append(subject.Province, name.ST)
			subject.Locality = append(subject.Locality, name.L)
			subject.Organization = append(subject.Organization, name.O)
			subject.OrganizationalUnit = append(subject.OrganizationalUnit, name.OU)
		}
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    notBefore,
		NotAfter:     notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range csr.Hosts {
		ip := net.ParseIP(h)
		if ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	pubKey, err := privKey.PublicKey()
	if err != nil {
		return nil, err
	}
	pubKeyBytes, err := pubKey.Bytes()
	if err != nil {
		return nil, err
	}
	pub, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, err
	}
	ecdsaPubKey := pub.(*ecdsa.PublicKey)

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, ecdsaPubKey, signer)
	if err != nil {
		return nil, err
	}

	return certBytes, nil
}

func (t *TLS) WriteCryptoToFile(cert []byte, certName string) error {
	certPath := filepath.Join(t.CAHomeDir, "tls", certName)
	err := util.EnsureDir(filepath.Dir(certPath))
	if err != nil {
		return err
	}

	certOut, err := os.Create(filepath.Clean(certPath))
	if err != nil {
		return err
	}
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert})
	if err != nil {
		return err
	}
	err = certOut.Close()
	if err != nil {
		return err
	}

	return nil
}

func NewCertificateRequest(req *api.CSRInfo) *csr.CertificateRequest {
	cr := csr.CertificateRequest{}
	if req != nil && req.Names != nil {
		cr.Names = req.Names
	}
	if req != nil && req.Hosts != nil {
		cr.Hosts = req.Hosts
	} else {
		// Default requested hosts are local hostname
		hostname, err := os.Hostname()
		if err == nil && hostname != "" {
			cr.Hosts = make([]string, 1)
			cr.Hosts[0] = hostname
		}
	}
	if req != nil && req.KeyRequest != nil {
		cr.KeyRequest = newCfsslKeyRequest(req.KeyRequest)
	}
	if req != nil {
		cr.CA = req.CA
		cr.SerialNumber = req.SerialNumber
	}
	return &cr
}

func newCfsslKeyRequest(bkr *api.KeyRequest) *csr.KeyRequest {
	return &csr.KeyRequest{A: bkr.Algo, S: bkr.Size}
}
