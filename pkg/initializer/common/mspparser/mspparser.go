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

package mspparser

import (
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/pkg/errors"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("peer_init_msp_parser")

type MSPParser struct {
	Config *current.MSP
}

func New(cfg *current.MSP) *MSPParser {
	return &MSPParser{
		Config: cfg,
	}
}

func (m *MSPParser) GetCrypto() (*config.Response, error) {
	return m.Parse()
}

func (m *MSPParser) Parse() (*config.Response, error) {
	resp := &config.Response{}

	certBytes, err := util.Base64ToBytes(m.Config.SignCerts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse signcert")
	}
	resp.SignCert = certBytes

	keyBytes, err := util.Base64ToBytes(m.Config.KeyStore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse keystore")
	}
	resp.Keystore = keyBytes

	for _, adminCert := range m.Config.AdminCerts {
		bytes, err := util.Base64ToBytes(adminCert)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse admin cert")
		}
		resp.AdminCerts = append(resp.AdminCerts, bytes)
	}

	for _, interCert := range m.Config.IntermediateCerts {
		bytes, err := util.Base64ToBytes(interCert)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse intermediate cert")
		}
		resp.IntermediateCerts = append(resp.IntermediateCerts, bytes)
	}

	for _, caCert := range m.Config.CACerts {
		bytes, err := util.Base64ToBytes(caCert)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse ca cert")
		}
		resp.CACerts = append(resp.CACerts, bytes)
	}

	return resp, nil
}

// MSP parser requires no interaction with CA, ping CA is a no-op
func (m *MSPParser) PingCA() error {
	// no-op
	return nil
}

func (m *MSPParser) Validate() error {
	cfg := m.Config

	if cfg.KeyStore == "" {
		return errors.New("unable to parse MSP, keystore not specified")
	}

	if cfg.SignCerts == "" {
		return errors.New("unable to parse MSP, signcert not specified")
	}

	if len(cfg.CACerts) == 0 {
		return errors.New("unable to parse MSP, ca certs not specified")
	}

	return nil
}
