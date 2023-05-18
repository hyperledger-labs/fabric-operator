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

package initializer

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"

	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/merge"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
	"github.com/hyperledger/fabric-ca/lib"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"
)

//go:generate counterfeiter -o mocks/config.go -fake-name CAConfig . CAConfig

type CAConfig interface {
	GetServerConfig() *v1.ServerConfig
	ParseCABlock() (map[string][]byte, error)
	ParseDBBlock() (map[string][]byte, error)
	ParseTLSBlock() (map[string][]byte, error)
	ParseOperationsBlock() (map[string][]byte, error)
	ParseIntermediateBlock() (map[string][]byte, error)
	SetServerConfig(*v1.ServerConfig)
	SetMountPaths(config.Type)
	GetHomeDir() string
	SetUpdate(bool)
	UsingPKCS11() bool
}

type CA struct {
	CN            string
	Config        CAConfig
	Viper         *viper.Viper
	Type          config.Type
	SqliteDir     string
	UsingHSMProxy bool

	configFile string
}

func LoadConfigFromFile(file string) (*v1.ServerConfig, error) {
	serverConfig := &v1.ServerConfig{}
	bytes, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, serverConfig)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, &serverConfig.CAConfig)
	if err != nil {
		return nil, err
	}

	return serverConfig, nil
}

func NewCA(config CAConfig, caType config.Type, sqliteDir string, hsmProxy bool, cn string) *CA {
	return &CA{
		CN:            cn,
		Config:        config,
		Viper:         viper.New(),
		Type:          caType,
		configFile:    fmt.Sprintf("%s/fabric-ca-server-config.yaml", config.GetHomeDir()),
		SqliteDir:     sqliteDir,
		UsingHSMProxy: hsmProxy,
	}
}

func (ca *CA) OverrideServerConfig(newConfig *v1.ServerConfig) (err error) {
	serverConfig := ca.Config.GetServerConfig()

	log.Info("Overriding config values from ca initializer")
	// If newConfig isn't passed, we want to make sure serverConfig.CAConfig.CSR.Cn is set
	// to ca.CN by default; if newConfig is passed for an intermediate CA, the logic below
	// will handle setting CN to blank if ParentServer.URL is set
	serverConfig.CAConfig.CSR.CN = ca.CN

	if newConfig != nil {
		log.Info("Overriding config values from spec")
		err = merge.WithOverwrite(ca.Config.GetServerConfig(), newConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to merge override configuration")
		}

		if ca.Config.UsingPKCS11() {
			ca.SetPKCS11Defaults(serverConfig)
		}

		// Passing in CN when enrolling an intermediate CA will cause the fabric-ca
		// server to error out, a CN cannot be passed for intermediate CA. Setting
		// CN to blank if ParentServer.URL is set
		if serverConfig.CAConfig.Intermediate.ParentServer.URL != "" {
			serverConfig.CAConfig.CSR.CN = ""
		}
	}

	ca.setDefaults(serverConfig)

	return nil
}

func (ca *CA) WriteConfig() (err error) {
	dir := ca.Config.GetHomeDir()
	log.Info(fmt.Sprintf("Writing config to file: '%s'", dir))

	bytes, err := ca.ConfigToBytes()
	if err != nil {
		return err
	}

	err = util.EnsureDir(dir)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Clean(ca.configFile), bytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (ca *CA) Init() (err error) {
	if ca.Config.UsingPKCS11() && ca.UsingHSMProxy {
		env := os.Getenv("PKCS11_PROXY_SOCKET")
		if env == "" {
			return errors.New("ca configured to use PKCS11, but no PKCS11 proxy endpoint set")
		}
		if !util.IsTCPReachable(env) {
			return errors.New(fmt.Sprintf("Unable to reach PKCS11 proxy: %s", env))
		}
	}

	cfg, err := ca.ViperUnmarshal(ca.configFile)
	if err != nil {
		return errors.Wrap(err, "viper unmarshal failed")
	}

	dir := filepath.Dir(ca.configFile)
	// TODO check if this is required!!
	cfg.Metrics.Provider = "disabled"

	if cfg.CAcfg.DB.Type == "postgres" {
		if !ca.IsPostgresReachable(cfg.CAcfg.DB) {
			return errors.New("Cannot initialize CA. Postgres is not reachable")
		}
	}

	parentURL := cfg.CAcfg.Intermediate.ParentServer.URL
	if parentURL != "" {
		log.Info(fmt.Sprintf("Request received to enroll with parent server: %s", parentURL))

		err = ca.HealthCheck(parentURL, cfg.CAcfg.Intermediate.TLS.CertFiles[0])
		if err != nil {
			return errors.Wrap(err, "could not connect to parent CA")
		}
	}

	caserver := &lib.Server{
		HomeDir: dir,
		Config:  cfg,
		CA: lib.CA{
			Config: &cfg.CAcfg,
		},
	}

	err = caserver.Init(false)
	if err != nil {
		return err
	}
	serverConfig := ca.Config.GetServerConfig()
	serverConfig.CA.Certfile = caserver.CA.Config.CA.Certfile
	serverConfig.CA.Keyfile = caserver.CA.Config.CA.Keyfile
	serverConfig.CA.Chainfile = caserver.CA.Config.CA.Chainfile

	if ca.Type.Is(config.EnrollmentCA) {
		serverConfig.CAfiles = []string{"/data/tlsca/fabric-ca-server-config.yaml"}
	}

	return nil
}

func (ca *CA) IsPostgresReachable(db lib.CAConfigDB) bool {

	datasource := db.Datasource
	if db.TLS.CertFiles != nil && len(db.TLS.CertFiles) > 0 {
		// The first cert because that is what hyperledger/fabric-ca uses
		datasource = fmt.Sprintf("%s sslrootcert=%s", datasource, db.TLS.CertFiles[0])
	}

	if db.TLS.Client.CertFile != "" {
		datasource = fmt.Sprintf("%s sslcert=%s", datasource, db.TLS.Client.CertFile)
	}

	if db.TLS.Client.KeyFile != "" {
		datasource = fmt.Sprintf("%s sslkey=%s", datasource, db.TLS.Client.KeyFile)
	}

	sqldb, err := sql.Open(db.Type, datasource)
	if err != nil {
		return false
	}
	defer sqldb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = sqldb.PingContext(ctx)

	return err == nil
}

// ViperUnmarshal as this is what fabric-ca uses when it reads it's configuration
// file
func (ca *CA) ViperUnmarshal(configFile string) (*lib.ServerConfig, error) {
	ca.Viper.SetConfigFile(configFile)
	err := ca.Viper.ReadInConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "viper unable to read in config: %s", configFile)
	}

	config := &lib.ServerConfig{}
	err = ca.Viper.Unmarshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "viper unable to unmarshal into server level config")
	}

	err = ca.Viper.Unmarshal(&config.CAcfg)
	if err != nil {
		return nil, errors.Wrap(err, "viper unable to unmarshal into CA level config")
	}

	return config, nil
}

func (ca *CA) ParseCrypto() (map[string][]byte, error) {
	switch ca.Type {
	case config.EnrollmentCA:
		return ca.ParseEnrollmentCACrypto()
	case config.TLSCA:
		return ca.ParseTLSCACrypto()
	}

	return nil, fmt.Errorf("unsupported ca type '%s'", ca.Type)
}

func (ca *CA) ParseEnrollmentCACrypto() (map[string][]byte, error) {
	serverConfig := ca.Config.GetServerConfig()
	if serverConfig.TLS.IsEnabled() {
		// TLS cert and key file must always be set. Operator should auto generate
		// TLS cert and key if none are provided.
		if serverConfig.TLS.CertFile == "" && serverConfig.TLS.KeyFile == "" {
			return nil, errors.New("no TLS cert and key file provided")
		}
	}

	if serverConfig.Operations.TLS.IsEnabled() {
		// Same set of TLS certificate that are used for CA endpoint is also used for operations endpoint
		serverConfig.Operations.TLS.CertFile = serverConfig.TLS.CertFile
		serverConfig.Operations.TLS.KeyFile = serverConfig.TLS.KeyFile
	}

	crypto, err := ca.Config.ParseCABlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ca block")
	}

	tlsCrypto, err := ca.Config.ParseTLSBlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse tls block")
	}
	crypto = util.JoinMaps(crypto, tlsCrypto)

	dbCrypto, err := ca.Config.ParseDBBlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse db block")
	}
	crypto = util.JoinMaps(crypto, dbCrypto)

	opsCrypto, err := ca.Config.ParseOperationsBlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse operations block")
	}
	crypto = util.JoinMaps(crypto, opsCrypto)

	intCrypto, err := ca.Config.ParseIntermediateBlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse intermediate block")
	}
	crypto = util.JoinMaps(crypto, intCrypto)

	return crypto, nil
}

func (ca *CA) ParseTLSCACrypto() (map[string][]byte, error) {
	crypto, err := ca.ParseCABlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ca block")
	}

	tlsCrypto, err := ca.Config.ParseTLSBlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse tls block")
	}
	crypto = util.JoinMaps(crypto, tlsCrypto)

	dbCrypto, err := ca.Config.ParseDBBlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse db block")
	}
	crypto = util.JoinMaps(crypto, dbCrypto)

	intCrypto, err := ca.Config.ParseIntermediateBlock()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse intermediate block")
	}
	crypto = util.JoinMaps(crypto, intCrypto)

	return crypto, nil
}

func (ca *CA) ParseCABlock() (map[string][]byte, error) {
	crypto, err := ca.Config.ParseCABlock()
	if err != nil {
		return nil, err
	}

	return crypto, nil
}

func (ca *CA) ConfigToBytes() ([]byte, error) {

	bytes, err := yaml.Marshal(ca.Config.GetServerConfig())
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (ca *CA) SetMountPaths() {
	ca.Config.SetMountPaths(ca.Type)
}

func (ca *CA) SetPKCS11Defaults(serverConfig *v1.ServerConfig) {
	if serverConfig.CAConfig.CSP.PKCS11 == nil {
		serverConfig.CAConfig.CSP.PKCS11 = &v1.PKCS11Opts{}
	}

	if ca.UsingHSMProxy {
		serverConfig.CAConfig.CSP.PKCS11.Library = "/usr/local/lib/libpkcs11-proxy.so"
	}

	serverConfig.CAConfig.CSP.PKCS11.FileKeyStore.KeyStorePath = "msp/keystore"

	if serverConfig.CAConfig.CSP.PKCS11.HashFamily == "" {
		serverConfig.CAConfig.CSP.PKCS11.HashFamily = "SHA2"
	}

	if serverConfig.CAConfig.CSP.PKCS11.SecLevel == 0 {
		serverConfig.CAConfig.CSP.PKCS11.SecLevel = 256
	}
}

func (ca *CA) GetHomeDir() string {
	return ca.Config.GetHomeDir()
}

func (ca *CA) GetServerConfig() *v1.ServerConfig {
	return ca.Config.GetServerConfig()
}

func (ca *CA) RemoveHomeDir() error {
	err := os.RemoveAll(ca.GetHomeDir())
	if err != nil {
		return err
	}
	return nil
}

func (ca *CA) IsBeingUpdated() {
	ca.Config.SetUpdate(true)
}

func (ca *CA) IsHSMEnabled() bool {

	return ca.Config.UsingPKCS11()
}

func (ca *CA) HealthCheck(parentURL, certPath string) error {
	parsedURL, err := url.Parse(parentURL)
	if err != nil {
		return errors.Wrapf(err, "invalid CA url")
	}

	healthURL := getHealthCheckEndpoint(parsedURL)
	log.Info(fmt.Sprintf("Health checking parent server, pinging %s", healthURL))

	// Make sure that parent server is running before trying to enroll
	// intermediate CA. Retry 5 times for a total of 5 seconds to make
	// sure parent server is up. If parent server is found, bail early
	// and continue with enrollment
	cert, err := ioutil.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return errors.Wrap(err, "failed to read TLS cert for intermediate enrollment")
	}

	for i := 0; i < 5; i++ {
		err = util.HealthCheck(healthURL, cert, 30*time.Second)
		if err != nil {
			log.Info(fmt.Sprintf("Health check error: %s", err))
			time.Sleep(1 * time.Second)
			log.Info("Health check failed, retrying")
			continue
		}
		log.Info("Health check successfull")
		break
	}

	return nil
}

func (ca *CA) GetType() config.Type {
	return ca.Type
}

func getHealthCheckEndpoint(u *url.URL) string {
	return fmt.Sprintf("%s://%s/cainfo", u.Scheme, u.Host)
}

func (ca *CA) setDefaults(serverConfig *v1.ServerConfig) {
	serverConfig.CAConfig.Cfg.Identities.AllowRemove = pointer.True()
	serverConfig.CAConfig.Cfg.Affiliations.AllowRemove = pointer.True()
	// Ignore Certificate Expiry for re-enroll
	serverConfig.CA.ReenrollIgnoreCertExpiry = pointer.True()
}
