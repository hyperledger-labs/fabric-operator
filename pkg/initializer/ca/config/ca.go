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
	"path/filepath"
)

func (c *Config) ParseCABlock() (map[string][]byte, error) {
	log.Info("Parsing CA block")

	if c.caCrypto == nil {
		c.caCrypto = map[string][]byte{}
	}

	certFile := c.ServerConfig.CAConfig.CA.Certfile
	keyFile := c.ServerConfig.CAConfig.CA.Keyfile

	if certFile == "" && keyFile == "" {
		return nil, nil
	}

	err := c.HandleCertInput(certFile, "cert.pem", c.caCrypto)
	if err != nil {
		return nil, err
	}
	c.ServerConfig.CAConfig.CA.Certfile = filepath.Join(c.HomeDir, "cert.pem")

	err = c.HandleKeyInput(keyFile, "key.pem", c.caCrypto)
	if err != nil {
		return nil, err
	}
	c.ServerConfig.CAConfig.CA.Keyfile = filepath.Join(c.HomeDir, "key.pem")

	chainFile := c.ServerConfig.CAConfig.CA.Chainfile
	if chainFile != "" {
		err := c.HandleCertInput(chainFile, "chain.pem", c.caCrypto)
		if err != nil {
			return nil, err
		}
		c.ServerConfig.CAConfig.CA.Chainfile = filepath.Join(c.HomeDir, "chain.pem")
	}

	return c.caCrypto, nil
}

func (c *Config) CAMountPath() {
	c.ServerConfig.CAConfig.CA.Keyfile = filepath.Join(c.MountPath, "key.pem")
	c.ServerConfig.CAConfig.CA.Certfile = filepath.Join(c.MountPath, "cert.pem")

	chainFile := c.ServerConfig.CAConfig.CA.Chainfile
	if chainFile != "" {
		c.ServerConfig.CAConfig.CA.Chainfile = filepath.Join(c.MountPath, "chain.pem")
	}
}
