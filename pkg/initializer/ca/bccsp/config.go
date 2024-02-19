//go:build !pkcs11
// +build !pkcs11

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
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/hyperledger/fabric-lib-go/bccsp/factory"
)

func GetBCCSPOpts(from v1.BCCSP) *factory.FactoryOpts {
	factoryOpts := &factory.FactoryOpts{
		Default: from.Default,
	}

	if from.SW != nil {
		factoryOpts.SW = &factory.SwOpts{
			Security: from.SW.Security,
			Hash:     from.SW.Hash,
			FileKeystore: &factory.FileKeystoreOpts{
				KeyStorePath: from.SW.FileKeyStore.KeyStorePath,
			},
		}
	}

	return factoryOpts
}
