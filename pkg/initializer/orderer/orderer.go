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
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/pkg/errors"
)

type OrdererConfig interface {
	MergeWith(interface{}, bool) error
	ToBytes() ([]byte, error)
	UsingPKCS11() bool
	SetPKCS11Defaults(bool)
	GetBCCSPSection() *commonapi.BCCSP
	SetDefaultKeyStore()
	SetBCCSPLibrary(string)
}

type Orderer struct {
	Config        OrdererConfig
	Cryptos       *commonconfig.Cryptos
	UsingHSMProxy bool
}

func (o *Orderer) OverrideConfig(newConfig OrdererConfig) (err error) {
	if newConfig == nil {
		return nil
	}

	log.Info("Overriding orderer config values from spec")
	err = o.Config.MergeWith(newConfig, o.UsingHSMProxy)
	if err != nil {
		return errors.Wrapf(err, "failed to merge override configuration")
	}

	return nil
}

func (o *Orderer) GenerateCrypto() (*commonconfig.CryptoResponse, error) {
	log.Info("Generating orderer's crypto material")
	if o.Cryptos != nil {
		response, err := o.Cryptos.GenerateCryptoResponse()
		if err != nil {
			return nil, err
		}
		return response, nil
	}

	return &config.CryptoResponse{}, nil
}

func (o *Orderer) GetConfig() OrdererConfig {
	return o.Config
}
