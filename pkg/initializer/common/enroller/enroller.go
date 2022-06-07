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

package enroller

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/hyperledger/fabric-ca/lib"
	"github.com/pkg/errors"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("init_enroller")

//go:generate counterfeiter -o mocks/cryptoenroller.go -fake-name CryptoEnroller . CryptoEnroller

type CryptoEnroller interface {
	GetEnrollmentRequest() *current.Enrollment
	Enroll() (*config.Response, error)
	PingCA(time.Duration) error
}

type Enroller struct {
	Enroller CryptoEnroller
	Timeout  time.Duration
}

func New(enroller CryptoEnroller) *Enroller {
	return &Enroller{
		Enroller: enroller,
		Timeout:  30 * time.Second,
	}
}

func (e *Enroller) GetCrypto() (*config.Response, error) {
	log.Info("Getting crypto...")
	resp, err := e.Enroller.Enroll()
	if err != nil {
		return nil, errors.Wrap(err, "failed to enroll with CA")
	}

	// Store crypto
	for _, adminCert := range e.Enroller.GetEnrollmentRequest().AdminCerts {
		bytes, err := util.Base64ToBytes(adminCert)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse admin cert")
		}
		resp.AdminCerts = append(resp.AdminCerts, bytes)
	}

	return resp, nil
}

func (e *Enroller) PingCA() error {
	log.Info("Check if CA is reachable before triggering enroll job")
	return e.Enroller.PingCA(e.Timeout)
}

func (e *Enroller) Validate() error {
	req := e.Enroller.GetEnrollmentRequest()

	if req.CAHost == "" {
		return errors.New("unable to enroll, CA host not specified")
	}

	if req.CAPort == "" {
		return errors.New("unable to enroll, CA port not specified")
	}

	if req.EnrollID == "" {
		return errors.New("unable to enroll, enrollment ID not specified")
	}

	if req.EnrollSecret == "" {
		return errors.New("unable to enroll, enrollment secret not specified")
	}

	if req.CATLS.CACert == "" {
		return errors.New("unable to enroll, CA TLS certificate not specified")
	}

	return nil
}

func ParseEnrollmentResponse(resp *config.Response, si *lib.GetCAInfoResponse) (*config.Response, error) {
	chain := si.CAChain
	for len(chain) > 0 {
		var block *pem.Block
		block, chain = pem.Decode(chain)
		if block == nil {
			break
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse certificate in the CA chain")
		}

		if !cert.IsCA {
			return nil, errors.New("A certificate in the CA chain is not a CA certificate")
		}

		// If authority key id is not present or if it is present and equal to subject key id,
		// then it is a root certificate
		if len(cert.AuthorityKeyId) == 0 || bytes.Equal(cert.AuthorityKeyId, cert.SubjectKeyId) {
			resp.CACerts = append(resp.CACerts, pem.EncodeToMemory(block))
		} else {
			resp.IntermediateCerts = append(resp.IntermediateCerts, pem.EncodeToMemory(block))
		}
	}

	// for intermediate cert, put the whole chain as is
	if len(resp.IntermediateCerts) > 0 {
		resp.IntermediateCerts = [][]byte{si.CAChain}
	}

	return resp, nil
}
