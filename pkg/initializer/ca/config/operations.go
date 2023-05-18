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

func (c *Config) ParseOperationsBlock() (map[string][]byte, error) {
	if !c.ServerConfig.Operations.TLS.IsEnabled() {
		log.Info("TLS disabled for Operations endpoint")
		return nil, nil
	}

	log.Info("Parsing Operations block")
	certFile := c.ServerConfig.Operations.TLS.CertFile
	keyFile := c.ServerConfig.Operations.TLS.KeyFile

	// Values for both TLS certfile and keyfile required for Operations configuration.
	// TLS key look up is not supported via BCCSP
	err := ValidCryptoInput(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	if c.operationsCrypto == nil {
		c.operationsCrypto = map[string][]byte{}
	}

	err = c.HandleCertInput(certFile, "operations-cert.pem", c.operationsCrypto)
	if err != nil {
		return nil, err
	}
	c.ServerConfig.Operations.TLS.CertFile = filepath.Join(c.HomeDir, "operations-cert.pem")

	err = c.HandleKeyInput(keyFile, "operations-key.pem", c.operationsCrypto)
	if err != nil {
		return nil, err
	}
	c.ServerConfig.Operations.TLS.KeyFile = filepath.Join(c.HomeDir, "operations-key.pem")

	certFiles := c.ServerConfig.Operations.TLS.ClientCACertFiles
	for index, certFile := range certFiles {
		err = c.HandleCertInput(certFile, fmt.Sprintf("operations-certfile%d.pem", index), c.operationsCrypto)
		if err != nil {
			return nil, err
		}
		certFiles[index] = filepath.Join(c.HomeDir, fmt.Sprintf("operations-certfile%d.pem", index))
	}
	c.ServerConfig.Operations.TLS.ClientCACertFiles = certFiles

	return c.operationsCrypto, nil
}

func (c *Config) OperationsMountPath() {
	c.ServerConfig.Operations.TLS.CertFile = filepath.Join(c.MountPath, "operations-cert.pem")
	c.ServerConfig.Operations.TLS.KeyFile = filepath.Join(c.MountPath, "operations-key.pem")

	certFiles := c.ServerConfig.Operations.TLS.ClientCACertFiles
	for index := range certFiles {
		certFiles[index] = filepath.Join(c.MountPath, fmt.Sprintf("operations-certfile%d.pem", index))
	}
	c.ServerConfig.Operations.TLS.ClientCACertFiles = certFiles
}
