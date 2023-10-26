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

package v25

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	V25 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v25"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/merge"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

type Orderer struct {
	V25.Orderer `json:",inline"`
}

func (o *Orderer) ToBytes() ([]byte, error) {
	bytes, err := yaml.Marshal(o)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (o *Orderer) WriteToFile(path string) error {
	bytes, err := yaml.Marshal(o)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, bytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (o *Orderer) MergeWith(newConfig interface{}, usingHSMProxy bool) error {
	newOrderer := newConfig.(*Orderer)

	if newOrderer != nil {
		err := merge.WithOverwrite(o, newConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to merge orderer configuration overrides")
		}
	}

	if o.UsingPKCS11() {
		o.SetPKCS11Defaults(usingHSMProxy)
	}

	return nil
}

func (o *Orderer) DeepCopyInto(into *Orderer) {
	b, err := json.Marshal(o)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, into)
	if err != nil {
		return
	}
}

func (o *Orderer) DeepCopy() *Orderer {
	if o == nil {
		return nil
	}
	out := new(Orderer)
	o.DeepCopyInto(out)
	return out
}

func (o *Orderer) UsingPKCS11() bool {
	if o.General.BCCSP != nil {
		if strings.ToLower(o.General.BCCSP.ProviderName) == "pkcs11" {
			return true
		}
	}
	return false
}

func (o *Orderer) SetPKCS11Defaults(usingHSMProxy bool) {
	if o.General.BCCSP.PKCS11 == nil {
		o.General.BCCSP.PKCS11 = &commonapi.PKCS11Opts{}
	}

	if usingHSMProxy {
		o.General.BCCSP.PKCS11.Library = "/usr/local/lib/libpkcs11-proxy.so"
	}

	if o.General.BCCSP.PKCS11.HashFamily == "" {
		o.General.BCCSP.PKCS11.HashFamily = "SHA2"
	}

	if o.General.BCCSP.PKCS11.SecLevel == 0 {
		o.General.BCCSP.PKCS11.SecLevel = 256
	}
}

func (o *Orderer) SetBCCSPLibrary(library string) {
	if o.General.BCCSP.PKCS11 == nil {
		o.General.BCCSP.PKCS11 = &commonapi.PKCS11Opts{}
	}

	o.General.BCCSP.PKCS11.Library = library
}

func (o *Orderer) SetDefaultKeyStore() {
	// No-op
	return
}

func (o *Orderer) GetBCCSPSection() *commonapi.BCCSP {
	return o.General.BCCSP
}
