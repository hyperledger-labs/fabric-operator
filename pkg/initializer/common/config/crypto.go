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

package config

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

//go:generate counterfeiter -o mocks/crypto.go -fake-name Crypto . Crypto

type Crypto interface {
	GetCrypto() (*Response, error)
	PingCA() error
	Validate() error
}

// TODO: Next refactor should move this outside of config package into cryptogen package
// along with the Response struct, which is required to avoid cyclical dependencies
func GenerateCrypto(generator Crypto) (*Response, error) {
	if err := generator.PingCA(); err != nil {
		return nil, errors.Wrap(err, "ca is not reachable")
	}

	if err := generator.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid crypto")
	}

	return generator.GetCrypto()
}

type Cryptos struct {
	Enrollment Crypto
	TLS        Crypto
	ClientAuth Crypto
}

func (c *Cryptos) GenerateCryptoResponse() (*CryptoResponse, error) {
	response := &CryptoResponse{}

	if c.Enrollment != nil {
		resp, err := GenerateCrypto(c.Enrollment)
		if err != nil {
			return nil, err
		}

		response.Enrollment = resp
	}

	if c.TLS != nil {
		resp, err := GenerateCrypto(c.TLS)
		if err != nil {
			return nil, err
		}

		response.TLS = resp
	}

	if c.ClientAuth != nil {
		resp, err := GenerateCrypto(c.ClientAuth)
		if err != nil {
			return nil, err
		}

		response.ClientAuth = resp
	}

	return response, nil
}

type CryptoResponse struct {
	Enrollment *Response
	TLS        *Response
	ClientAuth *Response
}

func (c *CryptoResponse) VerifyCertOU(crType string) error {
	if c.Enrollment != nil {
		err := c.Enrollment.VerifyCertOU(crType)
		if err != nil {
			return errors.Wrapf(err, "invalid OU for %s identity", crType)
		}
	}
	return nil
}

type Response struct {
	CACerts           [][]byte
	IntermediateCerts [][]byte
	AdminCerts        [][]byte
	SignCert          []byte
	Keystore          []byte
}

func (r *Response) VerifyCertOU(crType string) error {
	if r.SignCert == nil || len(r.SignCert) == 0 {
		return nil
	}

	crType = strings.ToLower(crType)

	err := verifyCertOU(r.SignCert, crType)
	if err != nil {
		return errors.Wrap(err, "invalid OU for signcert")
	}

	if r.AdminCerts == nil {
		return nil
	}

	return nil
}

func verifyCertOU(pemBytes []byte, ou string) error {
	cert, err := util.GetCertificateFromPEMBytes(pemBytes)
	if err != nil {
		return err
	}

	if cert.Subject.OrganizationalUnit == nil {
		return errors.New("OU not defined")
	}

	if !util.FindStringInArray(ou, cert.Subject.OrganizationalUnit) {
		return errors.New(fmt.Sprintf("cert does not have right OU, expecting '%s'", ou))
	}

	return nil
}
