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

package v1

import (
	"fmt"

	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

type DeliveryClient struct {
	v1.DeliveryClient
}

type AddressOverride struct {
	v1.AddressOverride
	certBytes []byte
}

func (a *AddressOverride) CACertsFileToBytes() ([]byte, error) {
	data, err := util.Base64ToBytes(a.CACertsFile)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (a *AddressOverride) GetCertBytes() []byte {
	return a.certBytes
}

func (d *DeliveryClient) HandleCAcertsFiles() ([]AddressOverride, error) {
	addrOverrides := []AddressOverride{}

	for i, addr := range d.AddressOverrides {
		addrOverride := AddressOverride{AddressOverride: addr}
		certBytes, err := addrOverride.CACertsFileToBytes()
		if err != nil {
			return nil, err
		}
		addrOverride.certBytes = certBytes
		addrOverrides = append(addrOverrides, addrOverride)

		d.AddressOverrides[i].CACertsFile = fmt.Sprintf("/orderer/certs/cert%d.pem", i)
	}

	return addrOverrides, nil
}
