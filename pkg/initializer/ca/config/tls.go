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

package config

import (
	"fmt"
	"path/filepath"
)

func (c *Config) ParseTLSBlock() (map[string][]byte, error) {
	if !c.ServerConfig.TLS.IsEnabled() {
		log.Info("TLS disabled for Fabric CA server")
		return nil, nil
	}

	if c.tlsCrypto == nil {
		c.tlsCrypto = map[string][]byte{}
	}

	log.Info("Parsing TLS block")

	certFile := c.ServerConfig.TLS.CertFile
	keyFile := c.ServerConfig.TLS.KeyFile

	// Values for both TLS certfile and keyfile required for Operations configuration.
	// TLS key look up is not supported via BCCSP
	err := ValidCryptoInput(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	err = c.HandleCertInput(certFile, "tls-cert.pem", c.tlsCrypto)
	if err != nil {
		return nil, err
	}
	c.ServerConfig.TLS.CertFile = filepath.Join(c.HomeDir, "tls-cert.pem")

	err = c.HandleKeyInput(keyFile, "tls-key.pem", c.tlsCrypto)
	if err != nil {
		return nil, err
	}
	c.ServerConfig.TLS.KeyFile = filepath.Join(c.HomeDir, "tls-key.pem")

	certFiles := c.ServerConfig.TLS.ClientAuth.CertFiles
	for index, certFile := range certFiles {
		fileLocation := filepath.Join(c.HomeDir, fmt.Sprintf("tls-certfile%d.pem", index))
		err = c.HandleCertInput(certFile, fmt.Sprintf("tls-certfile%d.pem", index), c.tlsCrypto)
		if err != nil {
			return nil, err
		}
		certFiles[index] = fileLocation
	}
	c.ServerConfig.TLS.ClientAuth.CertFiles = certFiles

	return c.tlsCrypto, nil
}

func (c *Config) TLSMountPath() {
	c.ServerConfig.TLS.CertFile = filepath.Join(c.MountPath, "tls-cert.pem")
	c.ServerConfig.TLS.KeyFile = filepath.Join(c.MountPath, "tls-key.pem")

	certFiles := c.ServerConfig.TLS.ClientAuth.CertFiles
	for index := range certFiles {
		certFiles[index] = filepath.Join(c.MountPath, fmt.Sprintf("tls-certfile%d.pem", index))
	}
	c.ServerConfig.TLS.ClientAuth.CertFiles = certFiles
}
