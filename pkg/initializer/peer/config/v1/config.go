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
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/merge"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

type Core struct {
	v1.Core       `json:",inline"`
	addrOverrides []AddressOverride
}

func (c *Core) ToBytes() ([]byte, error) {
	bytes, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (c *Core) WriteToFile(path string) error {
	bytes, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Clean(path), bytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (c *Core) MergeWith(newConfig interface{}, UsingHSMProxy bool) error {
	newCore := newConfig.(*Core)

	if newCore != nil {
		err := merge.WithOverwrite(c, newCore)
		if err != nil {
			return errors.Wrapf(err, "failed to merge peer configuration overrides")
		}
	}

	if c.UsingPKCS11() {
		c.SetPKCS11Defaults(UsingHSMProxy)
	}

	dc := DeliveryClient{DeliveryClient: c.Peer.DeliveryClient}
	addrOverrides, err := dc.HandleCAcertsFiles()
	if err != nil {
		return errors.Wrapf(err, "failed to convert base64 certs to filepath")
	}
	c.Peer.DeliveryClient = dc.DeliveryClient
	c.addrOverrides = addrOverrides

	return nil
}

func (c *Core) DeepCopyInto(into *Core) {
	b, err := json.Marshal(c)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, into)
	if err != nil {
		return
	}
}

func (c *Core) DeepCopy() *Core {
	if c == nil {
		return nil
	}
	out := new(Core)
	c.DeepCopyInto(out)
	return out
}

func (c *Core) UsingPKCS11() bool {
	if c.Peer.BCCSP != nil {
		if strings.ToLower(c.Peer.BCCSP.Default) == "pkcs11" {
			return true
		}
	}
	return false
}

func (c *Core) SetPKCS11Defaults(usingHSMProxy bool) {
	if c.Peer.BCCSP.PKCS11 == nil {
		c.Peer.BCCSP.PKCS11 = &common.PKCS11Opts{}
	}

	if usingHSMProxy {
		c.Peer.BCCSP.PKCS11.Library = "/usr/local/lib/libpkcs11-proxy.so"
	}

	if c.Peer.BCCSP.PKCS11.Hash == "" {
		c.Peer.BCCSP.PKCS11.Hash = "SHA2"
	}

	if c.Peer.BCCSP.PKCS11.Security == 0 {
		c.Peer.BCCSP.PKCS11.Security = 256
	}

	c.Peer.BCCSP.PKCS11.SoftwareVerify = true
}

func (c *Core) SetDefaultKeyStore() {
	if c.Peer.BCCSP.PKCS11 != nil {
		c.Peer.BCCSP.PKCS11.FileKeyStore = &common.FileKeyStoreOpts{
			KeyStorePath: "msp/keystore",
		}
	}
}

func (c *Core) GetAddressOverrides() []AddressOverride {
	return c.addrOverrides
}

func (c *Core) GetBCCSPSection() *common.BCCSP {
	return c.Peer.BCCSP
}

func (c *Core) GetMaxNameLength() *int {
	return c.MaxNameLength
}

func (c *Core) SetBCCSPLibrary(library string) {
	if c.Peer.BCCSP.PKCS11 == nil {
		c.Peer.BCCSP.PKCS11 = &common.PKCS11Opts{}
	}

	c.Peer.BCCSP.PKCS11.Library = library
}
