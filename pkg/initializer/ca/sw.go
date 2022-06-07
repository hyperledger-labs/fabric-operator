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
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

type SW struct{}

func (sw *SW) Create(instance *current.IBPCA, overrides *v1.ServerConfig, ca IBPCA) (*Response, error) {
	var err error

	err = ca.RemoveHomeDir()
	if err != nil {
		return nil, err
	}

	err = ca.OverrideServerConfig(overrides)
	if err != nil {
		return nil, err
	}

	crypto, err := ca.ParseCrypto()
	if err != nil {
		return nil, err
	}

	err = ca.WriteConfig()
	if err != nil {
		return nil, err
	}

	err = ca.Init()
	if err != nil {
		return nil, err
	}

	caBlock, err := ca.ParseCABlock()
	if err != nil {
		return nil, err
	}
	crypto = util.JoinMaps(crypto, caBlock)

	ca.SetMountPaths()

	err = ca.RemoveHomeDir()
	if err != nil {
		return nil, err
	}

	return &Response{
		Config:    ca.GetServerConfig(),
		CryptoMap: crypto,
	}, nil
}
