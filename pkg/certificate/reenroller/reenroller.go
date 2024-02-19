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
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/hyperledger/fabric-ca/api"
	"github.com/hyperledger/fabric-ca/lib"
	"github.com/hyperledger/fabric-ca/lib/client/credential"
	fabricx509 "github.com/hyperledger/fabric-ca/lib/client/credential/x509"
	"github.com/hyperledger/fabric-ca/lib/tls"
	utils "github.com/hyperledger/fabric-ca/util"
	"github.com/hyperledger/fabric-lib-go/bccsp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("reenroller")

//go:generate counterfeiter -o mocks/identity.go -fake-name Identity . Identity
type Identity interface {
	Reenroll(req *api.ReenrollmentRequest) (*lib.EnrollmentResponse, error)
	GetECert() *fabricx509.Signer
	GetClient() *lib.Client
}

type Reenroller struct {
	Client   *lib.Client
	Identity Identity

	HomeDir string
	Config  *current.Enrollment
	BCCSP   bool
	Timeout time.Duration
	NewKey  bool
}

func New(cfg *current.Enrollment, homeDir string, bccsp *commonapi.BCCSP, timeoutstring string, newKey bool) (*Reenroller, error) {
	if cfg == nil {
		return nil, errors.New("unable to reenroll, Enrollment config must be passed")
	}

	err := EnrollmentConfigValidation(cfg)
	if err != nil {
		return nil, err
	}

	client := &lib.Client{
		HomeDir: homeDir,
		Config: &lib.ClientConfig{
			TLS: tls.ClientTLSConfig{
				Enabled:   true,
				CertFiles: []string{"tlsCert.pem"},
			},
			URL: fmt.Sprintf("https://%s:%s", cfg.CAHost, cfg.CAPort),
		},
	}

	client = GetClient(client, bccsp)

	timeout, err := time.ParseDuration(timeoutstring)
	if err != nil || timeoutstring == "" {
		timeout = time.Duration(60 * time.Second)
	}

	r := &Reenroller{
		Client:  client,
		HomeDir: homeDir,
		Config:  cfg.DeepCopy(),
		Timeout: timeout,
		NewKey:  newKey,
	}

	if bccsp != nil {
		r.BCCSP = true
	}

	return r, nil
}

func (r *Reenroller) InitClient() error {
	if !r.IsCAReachable() {
		return errors.New("unable to init client for re-enroll, CA is not reachable")
	}

	tlsCertBytes, err := util.Base64ToBytes(r.Config.CATLS.CACert)
	if err != nil {
		return err
	}
	err = os.MkdirAll(r.HomeDir, 0750)
	if err != nil {
		return err
	}

	err = util.WriteFile(filepath.Join(r.HomeDir, "tlsCert.pem"), tlsCertBytes, 0755)
	if err != nil {
		return err
	}

	err = r.Client.Init()
	if err != nil {
		return errors.Wrap(err, "failed to initialize CA client")
	}
	return nil
}

func (r *Reenroller) loadHSMIdentity(certPemBytes []byte) error {
	log.Info("Loading HSM based identity...")

	csp := r.Client.GetCSP()
	certPubK, err := r.Client.GetCSP().KeyImport(certPemBytes, &bccsp.X509PublicKeyImportOpts{Temporary: true})
	if err != nil {
		return err
	}

	// Get the key given the SKI value
	ski := certPubK.SKI()
	privateKey, err := csp.GetKey(ski)
	if err != nil {
		return errors.WithMessage(err, "could not find matching private key for SKI")
	}

	// BCCSP returns a public key if the private key for the SKI wasn't found, so
	// we need to return an error in that case.
	if !privateKey.Private() {
		return errors.Errorf("The private key associated with the certificate with SKI '%s' was not found", hex.EncodeToString(ski))
	}

	signer, err := fabricx509.NewSigner(privateKey, certPemBytes)
	if err != nil {
		return err
	}

	cred := fabricx509.NewCredential("", "", r.Client)
	err = cred.SetVal(signer)
	if err != nil {
		return err
	}

	r.Identity = lib.NewIdentity(r.Client, r.Config.EnrollID, []credential.Credential{cred})

	return nil
}

func (r *Reenroller) loadIdentity(certPemBytes []byte, keyPemBytes []byte) error {
	log.Info("Loading software based identity...")

	client := r.Client
	enrollmentID := r.Config.EnrollID

	// NOTE: Utilized code from https://github.com/hyperledger/fabric-ca/blob/v2.0.0-alpha/util/csp.go#L220
	// but modified to use pem bytes instead of file since we store the key in a secret, not in filesystem
	var bccspKey bccsp.Key
	temporary := true
	key, err := utils.PEMtoPrivateKey(keyPemBytes, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get private key from pem bytes")
	}
	switch key.(type) {
	case *ecdsa.PrivateKey:
		priv, err := utils.PrivateKeyToDER(key.(*ecdsa.PrivateKey))
		if err != nil {
			return errors.Wrap(err, "failed to marshal ECDSA private key to der")
		}
		bccspKey, err = client.GetCSP().KeyImport(priv, &bccsp.ECDSAPrivateKeyImportOpts{Temporary: temporary})
		if err != nil {
			return errors.Wrap(err, "failed to import ECDSA private key")
		}
	default:
		return errors.New("failed to import key, invalid secret key type")
	}

	signer, err := fabricx509.NewSigner(bccspKey, certPemBytes)
	if err != nil {
		return err
	}

	cred := fabricx509.NewCredential("", "", client)
	err = cred.SetVal(signer)
	if err != nil {
		return err
	}

	r.Identity = lib.NewIdentity(client, enrollmentID, []credential.Credential{cred})

	return nil
}

func (r *Reenroller) LoadIdentity(certPemBytes []byte, keyPemBytes []byte, hsmEnabled bool) error {
	if hsmEnabled {
		err := r.loadHSMIdentity(certPemBytes)
		if err != nil {
			return errors.Wrap(err, "failed to load HSM based identity")
		}

		return nil
	}

	err := r.loadIdentity(certPemBytes, keyPemBytes)
	if err != nil {
		return errors.Wrap(err, "failed to load identity")
	}

	return nil
}

func (r *Reenroller) IsCAReachable() bool {
	timeout := r.Timeout
	url := fmt.Sprintf("https://%s:%s/cainfo", r.Config.CAHost, r.Config.CAPort)

	// Convert TLS certificate from base64 to file
	tlsCertBytes, err := util.Base64ToBytes(r.Config.CATLS.CACert)
	if err != nil {
		log.Error(err, "Cannot convert TLS Certificate from base64")
		return false
	}

	err = wait.Poll(500*time.Millisecond, timeout, func() (bool, error) {
		err = util.HealthCheck(url, tlsCertBytes, timeout)
		if err == nil {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		log.Error(err, "Health check failed")
		return false
	}

	return true
}

func (r *Reenroller) Reenroll() (*config.Response, error) {
	reuseKey := true
	if r.NewKey {
		reuseKey = false
	}

	reenrollReq := &api.ReenrollmentRequest{
		CAName: r.Config.CAName,
		CSR: &api.CSRInfo{
			KeyRequest: &api.KeyRequest{
				ReuseKey: reuseKey,
			},
		},
	}

	if r.Config.CSR != nil && len(r.Config.CSR.Hosts) > 0 {
		reenrollReq.CSR.Hosts = r.Config.CSR.Hosts
	}

	log.Info(fmt.Sprintf("Re-enrolling with CA '%s' with request %+v, csr %+v", r.Config.CAHost, reenrollReq, reenrollReq.CSR))

	reenrollResp, err := r.Identity.Reenroll(reenrollReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to re-enroll with CA")
	}

	newIdentity := reenrollResp.Identity

	resp := &config.Response{}
	resp.SignCert = newIdentity.GetECert().Cert()

	// Only need to read key if a new key is being generated, which does not happen
	// if the reenroll request has "ReuseKey" set to true
	if !reuseKey {
		key, err := r.ReadKey()
		if err != nil {
			return nil, err
		}
		resp.Keystore = key
	}

	// NOTE: Added this logic because the keystore file wasn't getting
	// deleted, which impacts the next time the certificate is renewed, in that
	// when trying to ReadKey(), there would be more than 1 file present.
	err = r.DeleteKeystoreFile()
	if err != nil {
		return nil, err
	}

	// TODO: Currently not parsing reenroll response to get CACerts and
	// Intermediate Certs again (like we do when inintially enrolling with CA)
	// as those certs shouldn't need to be updated

	return resp, nil
}

func (r *Reenroller) ReadKey() ([]byte, error) {
	if r.BCCSP {
		return nil, nil
	}

	keystoreDir := filepath.Join(r.HomeDir, "msp", "keystore")

	files, err := ioutil.ReadDir(keystoreDir)
	if err != nil {
		return nil, err
	}

	if len(files) > 1 {
		return nil, errors.Errorf("expecting only one key file to present in keystore '%s', but found multiple", keystoreDir)
	}

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

	return nil, errors.Errorf("failed to read private key from dir '%s'", keystoreDir)
}

func (r *Reenroller) DeleteKeystoreFile() error {
	keystoreDir := filepath.Join(r.HomeDir, "msp", "keystore")

	files, err := ioutil.ReadDir(keystoreDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = os.Remove(filepath.Join(keystoreDir, file.Name()))
		if err != nil {
			return errors.Wrapf(err, "failed to delete keystore directory '%s'", keystoreDir)
		}
	}

	return nil
}

func EnrollmentConfigValidation(enrollConfig *current.Enrollment) error {
	if enrollConfig.CAHost == "" {
		return errors.New("unable to reenroll, CA host not specified")
	}

	if enrollConfig.CAPort == "" {
		return errors.New("unable to reenroll, CA port not specified")
	}

	if enrollConfig.EnrollID == "" {
		return errors.New("unable to reenroll, enrollment ID not specified")
	}

	if enrollConfig.CATLS.CACert == "" {
		return errors.New("unable to reenroll, CA TLS certificate not specified")
	}

	return nil
}
