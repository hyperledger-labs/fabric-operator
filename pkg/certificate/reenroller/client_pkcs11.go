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

package reenroller

import (
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/hyperledger/fabric-ca/lib"
	"github.com/hyperledger/fabric-lib-go/bccsp/factory"
	"github.com/hyperledger/fabric-lib-go/bccsp/pkcs11"
)

func GetClient(client *lib.Client, bccsp *commonapi.BCCSP) *lib.Client {
	if bccsp != nil {
		if bccsp.PKCS11 != nil {
			client.Config.CSP = &factory.FactoryOpts{
				Default: bccsp.Default,
				PKCS11: &pkcs11.PKCS11Opts{
					Security:       bccsp.PKCS11.Security,
					Hash:           bccsp.PKCS11.Hash,
					Library:        bccsp.PKCS11.Library,
					Label:          bccsp.PKCS11.Label,
					Pin:            bccsp.PKCS11.Pin,
					SoftwareVerify: bccsp.PKCS11.SoftwareVerify,
					Immutable:      bccsp.PKCS11.Immutable,
				},
			}
		}
	}

	return client
}
