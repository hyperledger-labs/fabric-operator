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

package initializer

import (
	"fmt"

	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
	"github.com/pkg/errors"
)

type CoreConfig interface {
	MergeWith(interface{}, bool) error
	GetAddressOverrides() []v1.AddressOverride
	ToBytes() ([]byte, error)
	UsingPKCS11() bool
	SetPKCS11Defaults(bool)
	GetBCCSPSection() *commonapi.BCCSP
	SetBCCSPLibrary(string)
}

type Peer struct {
	Config        CoreConfig
	Cryptos       *commonconfig.Cryptos
	UsingHSMProxy bool
}

func (p *Peer) OverrideConfig(newConfig CoreConfig) (err error) {
	log.Info("Overriding peer config values from spec")
	err = p.Config.MergeWith(newConfig, p.UsingHSMProxy)
	if err != nil {
		return errors.Wrapf(err, "failed to merge override configuration")
	}

	return nil
}

func (p *Peer) GenerateCrypto() (*commonconfig.CryptoResponse, error) {
	log.Info("Generating peer's crypto material")
	if p.Cryptos != nil {
		response, err := p.Cryptos.GenerateCryptoResponse()
		if err != nil {
			return nil, err
		}
		return response, nil
	}

	return &config.CryptoResponse{}, nil
}

func (p *Peer) GetConfig() CoreConfig {
	return p.Config
}

func (p *Peer) DeliveryClientCrypto() map[string][]byte {
	data := map[string][]byte{}

	if p.Config != nil {
		for i, addr := range p.Config.GetAddressOverrides() {
			data[fmt.Sprintf("cert%d.pem", i)] = addr.GetCertBytes()
		}
	}

	return data
}
