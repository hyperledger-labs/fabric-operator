//go:build pkcs11
// +build pkcs11

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

package bccsp

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
)

func GetBCCSPOpts(from config.BCCSP) *factory.FactoryOpts {
	factoryOpts := &factory.FactoryOpts{
		ProviderName: from.ProviderName,
	}

	if from.SW != nil {
		factoryOpts.SwOpts = &factory.SwOpts{
			SecLevel:   from.SW.SecLevel,
			HashFamily: from.SW.HashFamily,
			FileKeystore: &factory.FileKeystoreOpts{
				KeyStorePath: from.SW.FileKeyStore.KeyStorePath,
			},
		}
	}

	if from.PKCS11 != nil {
		factoryOpts.Pkcs11Opts = &pkcs11.PKCS11Opts{
			SecLevel:   from.PKCS11.SecLevel,
			HashFamily: from.PKCS11.HashFamily,
			Library:    from.PKCS11.Library,
			Label:      from.PKCS11.Label,
			Pin:        from.PKCS11.Pin,
			SoftVerify: from.PKCS11.SoftVerify,
		}

		if from.PKCS11.FileKeystore != nil {
			factoryOpts.Pkcs11Opts.FileKeystore = &pkcs11.FileKeystoreOpts{
				KeyStorePath: from.PKCS11.FileKeyStore.KeyStorePath,
			}
		}
	}

	return factoryOpts
}
