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
	"fmt"
	"os"
	"path/filepath"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/secretmanager"
	configv1 "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("peer_initializer")

type Config struct {
	OUFile                 string
	InterOUFile            string
	CorePeerFile           string
	CorePeerV2File         string
	CorePeerV25File        string
	DeploymentFile         string
	PVCFile                string
	CouchDBPVCFile         string
	ServiceFile            string
	RoleFile               string
	ServiceAccountFile     string
	RoleBindingFile        string
	FluentdConfigMapFile   string
	CouchContainerFile     string
	CouchInitContainerFile string
	IngressFile            string
	Ingressv1beta1File     string
	CCLauncherFile         string
	RouteFile              string
	StoragePath            string
}

//go:generate counterfeiter -o mocks/ibppeer.go -fake-name IBPPeer . IBPPeer

type IBPPeer interface {
	DeliveryClientCrypto() map[string][]byte
	OverrideConfig(CoreConfig) error
	GenerateCrypto() (*config.CryptoResponse, error)
	GetConfig() CoreConfig
}

type PeerConfig interface {
	MergeWith(interface{}, bool) error
	GetAddressOverrides() []configv1.AddressOverride
	ToBytes() ([]byte, error)
	UsingPKCS11() bool
	SetPKCS11Defaults(bool)
	GetBCCSPSection() *commonapi.BCCSP
	GetMaxNameLength() *int
	SetDefaultKeyStore()
}

type Initializer struct {
	Config        *Config
	Scheme        *runtime.Scheme
	GetLabels     func(instance metav1.Object) map[string]string
	coreConfigMap *CoreConfigMap
	Timeouts      enroller.HSMEnrollJobTimeouts

	Client        k8sclient.Client
	Validator     common.CryptoValidator
	SecretManager *secretmanager.SecretManager
}

func New(config *Config, scheme *runtime.Scheme, client k8sclient.Client, labels func(instance metav1.Object) map[string]string, validator common.CryptoValidator, timeouts enroller.HSMEnrollJobTimeouts) *Initializer {
	secretManager := secretmanager.New(client, scheme, labels)

	return &Initializer{
		Client:        client,
		Config:        config,
		Scheme:        scheme,
		GetLabels:     labels,
		Validator:     validator,
		SecretManager: secretManager,
		coreConfigMap: &CoreConfigMap{Config: config, Scheme: scheme, GetLabels: labels, Client: client},
		Timeouts:      timeouts,
	}
}

type Response struct {
	Config              CoreConfig
	Crypto              *config.CryptoResponse
	DeliveryClientCerts map[string][]byte
}

func (i *Initializer) Create(overrides CoreConfig, peer IBPPeer, storagePath string) (*Response, error) {
	var err error

	err = os.RemoveAll(storagePath)
	if err != nil {
		return nil, err
	}

	err = peer.OverrideConfig(overrides)
	if err != nil {
		return nil, err
	}

	cresp, err := peer.GenerateCrypto()
	if err != nil {
		return nil, err
	}

	err = os.RemoveAll(storagePath)
	if err != nil {
		return nil, err
	}

	return &Response{
		Config:              peer.GetConfig(),
		DeliveryClientCerts: peer.DeliveryClientCrypto(),
		Crypto:              cresp,
	}, nil
}

func (i *Initializer) CoreConfigMap() *CoreConfigMap {
	return i.coreConfigMap
}

func (i *Initializer) Update(overrides CoreConfig, peer IBPPeer) (*Response, error) {
	var err error

	err = peer.OverrideConfig(overrides)
	if err != nil {
		return nil, err
	}

	return &Response{
		Config:              peer.GetConfig(),
		DeliveryClientCerts: peer.DeliveryClientCrypto(),
	}, nil
}

func (i *Initializer) GetEnrollers(cryptos *config.Cryptos, instance *current.IBPPeer, storagePath string) error {
	// If no enrollment information provided, don't need to proceed further
	if instance.Spec.Secret == nil || instance.Spec.Secret.Enrollment == nil {
		return nil
	}

	enrollmentSpec := instance.Spec.Secret.Enrollment
	if enrollmentSpec.Component != nil && cryptos.Enrollment == nil {
		bytes, err := enrollmentSpec.Component.GetCATLSBytes()
		if err != nil {
			return err
		}

		// Factory will determine if HSM or non-HSM enroller needed and return back appropriate type
		cryptos.Enrollment, err = enroller.Factory(enrollmentSpec.Component, i.Client, instance,
			filepath.Join(storagePath, "ecert"),
			i.Scheme,
			bytes,
			i.Timeouts,
		)
		if err != nil {
			return err
		}
	}

	// Common enrollers get software based enrollers for TLS and clientauth crypto,
	// these types are not supported for HSM
	err := common.GetCommonEnrollers(cryptos, enrollmentSpec, storagePath)
	if err != nil {
		return err
	}

	return nil
}

func (i *Initializer) GetMSPCrypto(cryptos *config.Cryptos, instance *current.IBPPeer) error {
	if instance.Spec.Secret == nil || instance.Spec.Secret.MSP == nil {
		return nil
	}

	mspSpec := instance.Spec.Secret.MSP
	err := common.GetMSPCrypto(cryptos, mspSpec)
	if err != nil {
		return err
	}

	return nil
}

func (i *Initializer) GetInitPeer(instance *current.IBPPeer, storagePath string) (*Peer, error) {
	cryptos := &config.Cryptos{}

	if instance.Spec.Secret != nil {
		// Prioritize any crypto passed through MSP spec first
		err := i.GetMSPCrypto(cryptos, instance)
		if err != nil {
			return nil, errors.Wrap(err, "failed to populate init peer with MSP spec")
		}

		err = i.GetEnrollers(cryptos, instance, storagePath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to populate init peer with Enrollment spec")
		}
	}

	return &Peer{
		Cryptos: cryptos,
	}, nil
}

func (i *Initializer) GetUpdatedPeer(instance *current.IBPPeer) (*Peer, error) {
	cryptos := &config.Cryptos{}

	// Only check for any new certs passed through MSP spec
	err := i.GetMSPCrypto(cryptos, instance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to populate updated init peer with MSP spec")
	}

	return &Peer{
		Cryptos: cryptos,
	}, nil
}

func (i *Initializer) GenerateSecrets(prefix common.SecretType, instance metav1.Object, crypto *config.Response) error {
	if crypto == nil {
		return nil
	}
	return i.SecretManager.GenerateSecrets(prefix, instance, crypto)
}

func (i *Initializer) GenerateSecretsFromResponse(instance *current.IBPPeer, cryptoResponse *config.CryptoResponse) error {
	return i.SecretManager.GenerateSecretsFromResponse(instance, cryptoResponse)
}

func (i *Initializer) UpdateSecrets(prefix common.SecretType, instance *current.IBPPeer, crypto *config.Response) error {
	if crypto == nil {
		return nil
	}
	return i.SecretManager.UpdateSecrets(prefix, instance, crypto)
}

func (i *Initializer) UpdateSecretsFromResponse(instance *current.IBPPeer, cryptoResponse *config.CryptoResponse) error {
	return i.SecretManager.UpdateSecretsFromResponse(instance, cryptoResponse)
}

func (i *Initializer) GetCrypto(instance *current.IBPPeer) (*config.CryptoResponse, error) {
	return i.SecretManager.GetCryptoResponseFromSecrets(instance)
}

func (i *Initializer) GenerateOrdererCACertsSecret(instance *current.IBPPeer, certs map[string][]byte) error {
	secretName := fmt.Sprintf("%s-orderercacerts", instance.GetName())
	err := i.CreateOrUpdateSecret(instance, secretName, certs)
	if err != nil {
		return err
	}

	return nil
}

func (i *Initializer) MissingCrypto(instance *current.IBPPeer) bool {
	if instance.IsHSMEnabled() {
		i.Validator.SetHSMEnabled(true)
	}

	checkClientAuth := instance.ClientAuthCryptoSet()
	err := common.CheckCrypto(i.Validator, instance, checkClientAuth)
	if err != nil {
		log.Info(err.Error())
		return true
	}

	return false
}

func (i *Initializer) CheckIfAdminCertsUpdated(instance *current.IBPPeer) (bool, error) {
	current := common.GetAdminCertsFromSecret(i.Client, instance)
	updated := common.GetAdminCertsFromSpec(instance.Spec.Secret)

	return common.CheckIfCertsDifferent(current, updated)
}

func (i *Initializer) UpdateAdminSecret(instance *current.IBPPeer) error {
	return i.SecretManager.UpdateAdminCertSecret(instance, instance.Spec.Secret)
}

func (i *Initializer) CreateOrUpdateSecret(instance *current.IBPPeer, name string, data map[string][]byte) error {
	log.Info(fmt.Sprintf("Creating secret '%s'", name))

	secret := i.SecretManager.BuildSecret(instance, name, data, i.GetLabels(instance))
	err := i.Client.CreateOrUpdate(context.TODO(), secret, k8sclient.CreateOrUpdateOption{
		Owner:  instance,
		Scheme: i.Scheme,
	})
	if err != nil {
		return err
	}

	return nil
}
