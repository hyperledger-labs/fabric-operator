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

func (c *Config) ParseIntermediateBlock() (map[string][]byte, error) {
	if c.intermediateCrypto == nil {
		c.intermediateCrypto = map[string][]byte{}
	}

	log.Info("Parsing Intermediate block")
	certFiles := c.ServerConfig.CAConfig.Intermediate.TLS.CertFiles
	for index, certFile := range certFiles {
		err := c.HandleCertInput(certFile, fmt.Sprintf("parent-certfile%d.pem", index), c.intermediateCrypto)
		if err != nil {
			return nil, err
		}
		certFiles[index] = filepath.Join(c.HomeDir, fmt.Sprintf("parent-certfile%d.pem", index))
	}
	c.ServerConfig.CAConfig.Intermediate.TLS.CertFiles = certFiles

	certFile := c.ServerConfig.CAConfig.Intermediate.TLS.Client.CertFile
	keyFile := c.ServerConfig.CAConfig.Intermediate.TLS.Client.KeyFile
	if certFile != "" && keyFile != "" {
		log.Info("Client authentication information provided for intermediate CA connection")
		err := c.HandleCertInput(certFile, "parent-cert.pem", c.intermediateCrypto)
		if err != nil {
			return nil, err
		}
		c.ServerConfig.CAConfig.Intermediate.TLS.Client.CertFile = filepath.Join(c.HomeDir, "parent-cert.pem")

		err = c.HandleKeyInput(keyFile, "parent-key.pem", c.intermediateCrypto)
		if err != nil {
			return nil, err
		}
		c.ServerConfig.CAConfig.Intermediate.TLS.Client.KeyFile = filepath.Join(c.HomeDir, "parent-key.pem")
	}

	return c.intermediateCrypto, nil
}

func (c *Config) IntermediateMountPath() {
	certFile := c.ServerConfig.CAConfig.Intermediate.TLS.Client.CertFile
	keyFile := c.ServerConfig.CAConfig.Intermediate.TLS.Client.KeyFile

	if certFile != "" && keyFile != "" {
		c.ServerConfig.CAConfig.Intermediate.TLS.Client.CertFile = filepath.Join(c.MountPath, "parent-cert.pem")
		c.ServerConfig.CAConfig.Intermediate.TLS.Client.KeyFile = filepath.Join(c.MountPath, "parent-key.pem")
	}

	certFiles := c.ServerConfig.CAConfig.Intermediate.TLS.CertFiles
	for index := range certFiles {
		certFiles[index] = filepath.Join(c.MountPath, fmt.Sprintf("parent-certfile%d.pem", index))
	}
	c.ServerConfig.CAConfig.Intermediate.TLS.CertFiles = certFiles
}
