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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

type Type string

const (
	EnrollmentCA Type = "enrollment"
	TLSCA        Type = "tls"
)

func (t Type) Is(typ Type) bool {
	return t == typ
}

type InputType string

var (
	File   InputType = "File"
	Pem    InputType = "Pem"
	Base64 InputType = "Base64"
	Bccsp  InputType = "Bccsp"
)

var log = logf.Log.WithName("initializer_config")

type Config struct {
	ServerConfig *v1.ServerConfig
	HomeDir      string
	MountPath    string
	Update       bool
	SqlitePath   string

	tlsCrypto          map[string][]byte
	dbCrypto           map[string][]byte
	caCrypto           map[string][]byte
	operationsCrypto   map[string][]byte
	intermediateCrypto map[string][]byte
}

func (c *Config) GetServerConfig() *v1.ServerConfig {
	return c.ServerConfig
}

func (c *Config) GetHomeDir() string {
	return c.HomeDir
}

func (c *Config) GetTLSCrypto() map[string][]byte {
	return c.tlsCrypto
}

func (c *Config) HandleCertInput(input, location string, store map[string][]byte) error {
	var err error
	inputType := GetInputType(input)

	log.Info(fmt.Sprintf("Handling input of cert type '%s', to be stored at '%s'", inputType, location))

	data := []byte{}
	switch inputType {
	case Pem:
		data = util.PemStringToBytes(input)
		err = c.StoreInMap(data, location, store)
		if err != nil {
			return err
		}
	case File:
		// On an update of config overrides, file is not a valid override value as the operator
		// won't have access to it. Cert can only be passed as base64.
		if !c.Update {
			data, err = util.FileToBytes(input)
			if err != nil {
				return err
			}
			err = c.StoreInMap(data, location, store)
			if err != nil {
				return err
			}
		}
	case Base64:
		data, err = util.Base64ToBytes(input)
		if err != nil {
			return err
		}
		err = c.StoreInMap(data, location, store)
		if err != nil {
			return err
		}
	case Bccsp:
		return nil
	default:
		return errors.Errorf("invalid input type: %s", input)
	}

	if len(data) != 0 {
		err := c.EnsureDirAndWriteFile(location, data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) EnsureDirAndWriteFile(location string, data []byte) error {
	path := filepath.Join(c.HomeDir, location)
	err := util.EnsureDir(filepath.Dir(path))
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Clean(path), data, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) HandleKeyInput(input, location string, store map[string][]byte) error {
	var err error

	inputType := GetInputType(input)

	log.Info(fmt.Sprintf("Handling input of key type '%s', to be stored at '%s'", inputType, location))

	data := []byte{}
	switch inputType {
	case Pem:
		data = util.PemStringToBytes(input)
		err = c.StoreInMap(data, location, store)
		if err != nil {
			return err
		}
	case File:
		// On an update of config overrides, file is not a valid override value as the operator
		// won't have access to it. Key can only be passed as base64.
		if !c.Update {
			data, err = util.FileToBytes(input)
			if err != nil {
				return err
			}
			err = c.StoreInMap(data, location, store)
			if err != nil {
				return err
			}
		}
	case Base64:
		data, err = util.Base64ToBytes(input)
		if err != nil {
			return err
		}
		err = c.StoreInMap(data, location, store)
		if err != nil {
			return err
		}
	case Bccsp:
		// If HSM enabled, don't try to read key from file system
		if c.UsingPKCS11() {
			return nil
		}
		// On an update of config overrides, reading from keystore is not valid. After init create
		// the key stored in a kubernetes secret and operator won't have access to it.
		if !c.Update {
			data, err = c.GetSigningKey(c.HomeDir)
			if err != nil {
				return err
			}
			err = c.StoreInMap(data, location, store)
			if err != nil {
				return err
			}
		}
	default:
		return errors.Errorf("invalid input type: %s", input)
	}

	if len(data) != 0 {
		err := c.EnsureDirAndWriteFile(location, data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) StoreInMap(data []byte, location string, store map[string][]byte) error {
	if len(data) == 0 {
		return nil
	}

	key := ConvertStringForSecrets(location, true)
	store[key] = data
	return nil
}

// GetSigningKey applies to non-hsm use cases where the key exists on the filesystem.
// The filesystem is read and then key is then stored in a kubernetes secret.
func (c *Config) GetSigningKey(path string) ([]byte, error) {

	keystoreDir := filepath.Join(path, "msp", "keystore")
	files, err := ioutil.ReadDir(keystoreDir)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no keys found in keystore directory: %s", keystoreDir)
	}

	// Need this loop to find appropriate key. Three files are generated
	// by default by the CA: IssuerRevocationPrivateKey, IssuerSecretKey, and *_sk
	// We are only interested in file ending with 'sk' which the is Private Key
	// associated with the x509 certificate
	for _, file := range files {
		fileBytes, err := ioutil.ReadFile(filepath.Clean(filepath.Join(keystoreDir, file.Name())))
		if err != nil {
			return nil, err
		}

		block, _ := pem.Decode(fileBytes)
		if block == nil {
			continue
		}

		_, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err == nil {
			return fileBytes, nil
		}
	}

	return nil, errors.Errorf("failed to parse CA's private key")
}

func (c *Config) SetUpdate(update bool) {
	c.Update = update
}

func (c *Config) SetServerConfig(cfg *v1.ServerConfig) {
	c.ServerConfig = cfg
}

func (c *Config) SetMountPaths(caType Type) {
	switch caType {
	case EnrollmentCA:
		c.CAMountPath()
		c.DBMountPath()
		c.IntermediateMountPath()
		c.OperationsMountPath()
		c.TLSMountPath()
	case TLSCA:
		c.CAMountPath()
		c.DBMountPath()
	}
}

func (c *Config) UsingPKCS11() bool {
	if c.ServerConfig != nil && c.ServerConfig.CAConfig.CSP != nil {
		if strings.ToLower(c.ServerConfig.CAConfig.CSP.Default) == "pkcs11" {
			return true
		}
	}
	return false
}

func GetInputType(input string) InputType {
	data := []byte(input)
	block, _ := pem.Decode(data)
	if block != nil {
		return Pem
	}

	data, err := util.Base64ToBytes(input)
	if err == nil && data != nil {
		return Base64
	}

	// If input string is found as an already exisiting file, return CertFile type
	_, err = os.Stat(input)
	if err == nil {
		return File
	}

	return Bccsp
}

func ConvertStringForSecrets(filepath string, forward bool) string {
	// shared//tlsca//db/certs/certfile0.pem
	if forward {
		return strings.Replace(filepath, "/", "_", -1)
	}
	// data[shared__tlsca__db_certs_certfile0.pem
	return strings.Replace(filepath, "_", "/", -1)
}

func IsValidPostgressDatasource(datasourceStr string) bool {
	regexpssions := []string{`host=\S+`, `port=\d+`, `user=\S+`, `password=\S+`, `dbname=\S+`, `sslmode=\S+`}
	for _, regexpression := range regexpssions {
		re := regexp.MustCompile(regexpression)
		matches := len(re.FindStringSubmatch(datasourceStr))
		if matches == 0 {
			return false
		}
	}
	return true
}

func ValidCryptoInput(certFile, keyFile string) error {
	if certFile == "" && keyFile != "" {
		return errors.New("Key file specified but no corresponding certificate file specified, both must be passed")
	}
	if certFile != "" && keyFile == "" {
		return errors.New("Certificate file specified but no corresponding key file specified, both must be passed")
	}
	return nil
}

func ReadFrom(from *[]byte) (*Config, error) {
	config := &v1.ServerConfig{}
	err := yaml.Unmarshal(*from, config)
	if err != nil {
		return nil, err
	}

	return &Config{
		ServerConfig: config,
	}, nil
}
