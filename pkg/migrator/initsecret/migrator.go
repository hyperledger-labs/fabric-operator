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

package initsecret

import (
	"errors"

	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

type Secret struct {
	Component *MSP `json:"component,omitempty"`
	TLS       *MSP `json:"tls,omitempty"`
}

type MSP struct {
	Keystore          []string `json:"keystore,omitempty"`
	SignCerts         []string `json:"signcerts,omitempty"`
	CACerts           []string `json:"cacerts,omitempty"`
	IntermediateCerts []string `json:"intermediatecerts,omitempty"`
	AdminCerts        []string `json:"admincerts,omitempty"`
}

type Migrator struct {
	Secret *Secret
}

func (m *Migrator) ParseComponentCrypto() (*commonconfig.Response, error) {
	crypto := m.Secret.Component
	if crypto == nil {
		return nil, errors.New("init secret missing component crypto")
	}
	return m.ParseCrypto(crypto)
}

func (m *Migrator) ParseTLSCrypto() (*commonconfig.Response, error) {
	crypto := m.Secret.TLS
	if crypto == nil {
		return nil, errors.New("init secret missing TLS crypto")
	}
	return m.ParseCrypto(crypto)
}

func (m *Migrator) ParseCrypto(crypto *MSP) (*commonconfig.Response, error) {
	signcert := crypto.SignCerts[0] // When would there ever be more then 1 signed cert? Assuming only one as of right now. However, the MSP secret json has this defined as an array
	keystore := crypto.Keystore[0]

	signcertBytes, err := util.Base64ToBytes(signcert)
	if err != nil {
		return nil, err
	}

	keystoreBytes, err := util.Base64ToBytes(keystore)
	if err != nil {
		return nil, err
	}

	adminCerts := [][]byte{}
	for _, cert := range crypto.AdminCerts {
		certBytes, err := util.Base64ToBytes(cert)
		if err != nil {
			return nil, err
		}

		adminCerts = append(adminCerts, certBytes)
	}

	caCerts := [][]byte{}
	for _, cert := range crypto.CACerts {
		certBytes, err := util.Base64ToBytes(cert)
		if err != nil {
			return nil, err
		}

		caCerts = append(caCerts, certBytes)
	}

	interCerts := [][]byte{}
	for _, cert := range crypto.IntermediateCerts {
		certBytes, err := util.Base64ToBytes(cert)
		if err != nil {
			return nil, err
		}

		interCerts = append(interCerts, certBytes)
	}

	return &commonconfig.Response{
		SignCert:          signcertBytes,
		Keystore:          keystoreBytes,
		CACerts:           caCerts,
		AdminCerts:        adminCerts,
		IntermediateCerts: interCerts,
	}, nil

}
