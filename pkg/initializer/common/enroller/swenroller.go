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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/hyperledger/fabric-ca/api"
	"github.com/hyperledger/fabric-ca/lib"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o mocks/caclient.go -fake-name CAClient . CAClient

type CAClient interface {
	Init() error
	Enroll(*api.EnrollmentRequest) (*lib.EnrollmentResponse, error)
	GetEnrollmentRequest() *current.Enrollment
	GetHomeDir() string
	GetTLSCert() []byte
	PingCA(time.Duration) error
}

type SWEnroller struct {
	Client CAClient
}

func NewSWEnroller(caClient CAClient) *SWEnroller {
	return &SWEnroller{
		Client: caClient,
	}
}

func (e *SWEnroller) GetEnrollmentRequest() *current.Enrollment {
	return e.Client.GetEnrollmentRequest()
}

func (e *SWEnroller) PingCA(timeout time.Duration) error {
	return e.Client.PingCA(timeout)
}

func (e *SWEnroller) Enroll() (*config.Response, error) {
	resp, err := enroll(e.Client)
	if err != nil {
		return nil, err
	}

	key, err := e.ReadKey()
	if err != nil {
		return nil, err
	}
	resp.Keystore = key

	return resp, nil
}

func (e *SWEnroller) ReadKey() ([]byte, error) {
	keystoreDir := filepath.Join(e.Client.GetHomeDir(), "msp", "keystore")
	files, err := ioutil.ReadDir(keystoreDir)
	if err != nil {
		return nil, err
	}

	if len(files) > 1 {
		return nil, errors.Errorf("expecting only one key file to present in keystore '%s', but found multiple", keystoreDir)
	}

	for _, file := range files {
		fileBytes, err := ioutil.ReadFile(filepath.Clean(filepath.Join(keystoreDir, file.Name())))
		if err != nil {
			return nil, err
		}

		block, _ := pem.Decode(fileBytes)
		if block == nil {
			continue
		}

		_, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err == nil {
			return fileBytes, nil
		}
	}

	return nil, errors.Errorf("failed to read private key")
}

func enroll(client CAClient) (*config.Response, error) {
	req := client.GetEnrollmentRequest()
	log.Info(fmt.Sprintf("Enrolling with CA '%s'", req.CAHost))

	err := os.MkdirAll(client.GetHomeDir(), 0750)
	if err != nil {
		return nil, err
	}

	err = util.WriteFile(filepath.Join(client.GetHomeDir(), "tlsCert.pem"), client.GetTLSCert(), 0755)
	if err != nil {
		return nil, err
	}

	err = client.Init()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize CA client")
	}

	// Enroll with CA
	enrollReq := &api.EnrollmentRequest{
		Type:   "x509",
		Name:   req.EnrollID,
		Secret: req.EnrollSecret,
		CAName: req.CAName,
	}
	if req.CSR != nil && len(req.CSR.Hosts) > 0 {
		enrollReq.CSR = &api.CSRInfo{
			Hosts: req.CSR.Hosts,
		}
	}

	enrollResp, err := client.Enroll(enrollReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to enroll with CA")
	}

	resp := &config.Response{}
	resp, err = ParseEnrollmentResponse(resp, &enrollResp.CAInfo)
	if err != nil {
		return nil, err
	}

	id := enrollResp.Identity
	if id.GetECert() != nil {
		resp.SignCert = id.GetECert().Cert()
	}

	return resp, nil
}
