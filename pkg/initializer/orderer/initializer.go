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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/secretmanager"
	ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v1"
	v2ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v2"
	v24ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v24"
	v25ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v25"
	"github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("orderer_initializer")

type Config struct {
	ConfigTxFile       string
	OrdererFile        string
	OrdererV2File      string
	OrdererV24File     string
	OrdererV25File     string
	OUFile             string
	InterOUFile        string
	DeploymentFile     string
	PVCFile            string
	ServiceFile        string
	CMFile             string
	RoleFile           string
	ServiceAccountFile string
	RoleBindingFile    string
	IngressFile        string
	Ingressv1beta1File string
	RouteFile          string
	StoragePath        string
}

type Response struct {
	Config OrdererConfig
	Crypto *config.CryptoResponse
}

//go:generate counterfeiter -o mocks/ibporderer.go -fake-name IBPOrderer . IBPOrderer

type IBPOrderer interface {
	OverrideConfig(newConfig OrdererConfig) error
	GenerateCrypto() (*config.CryptoResponse, error)
	GetConfig() OrdererConfig
}

type Initializer struct {
	Config   *Config
	Scheme   *runtime.Scheme
	Client   k8sclient.Client
	Name     string
	Timeouts enroller.HSMEnrollJobTimeouts

	Validator     common.CryptoValidator
	SecretManager *secretmanager.SecretManager
}

func New(client controllerclient.Client, scheme *runtime.Scheme, cfg *Config, name string, validator common.CryptoValidator) *Initializer {
	initializer := &Initializer{
		Client:    client,
		Scheme:    scheme,
		Config:    cfg,
		Name:      name,
		Validator: validator,
	}

	initializer.SecretManager = secretmanager.New(client, scheme, initializer.GetLabels)

	return initializer
}

func (i *Initializer) Create(overrides OrdererConfig, orderer IBPOrderer, storagePath string) (*Response, error) {
	var err error

	log.Info(fmt.Sprintf("Creating orderer %s's config and crypto...", i.Name))

	err = os.RemoveAll(storagePath)
	if err != nil {
		return nil, err
	}

	err = orderer.OverrideConfig(overrides)
	if err != nil {
		return nil, err
	}

	cresp, err := orderer.GenerateCrypto()
	if err != nil {
		return nil, err
	}

	err = os.RemoveAll(storagePath)
	if err != nil {
		return nil, err
	}

	return &Response{
		Config: orderer.GetConfig(),
		Crypto: cresp,
	}, nil
}

func (i *Initializer) Update(overrides OrdererConfig, orderer IBPOrderer) (*Response, error) {
	var err error

	log.Info(fmt.Sprintf("Updating orderer %s's config...", i.Name))

	err = orderer.OverrideConfig(overrides)
	if err != nil {
		return nil, err
	}

	return &Response{
		Config: orderer.GetConfig(),
	}, nil
}

func (i *Initializer) GetEnrollers(cryptos *config.Cryptos, instance *current.IBPOrderer, storagePath string) error {
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

	// err := common.GetSWEnrollers(cryptos, enrollmentSpec, storagePath)
	err := common.GetCommonEnrollers(cryptos, enrollmentSpec, storagePath)
	if err != nil {
		return err
	}

	return nil
}

func (i *Initializer) GetMSPCrypto(cryptos *config.Cryptos, instance *current.IBPOrderer) error {
	mspSpec := instance.Spec.Secret.MSP
	if mspSpec != nil {
		err := common.GetMSPCrypto(cryptos, mspSpec)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Initializer) GetInitOrderer(instance *current.IBPOrderer, storagePath string) (*Orderer, error) {
	cryptos := &config.Cryptos{}

	if instance.Spec.Secret != nil {
		// Prioritize any crypto passed through MSP spec first
		err := i.GetMSPCrypto(cryptos, instance)
		if err != nil {
			return nil, errors.Wrap(err, "failed to populate init orderer with MSP spec")
		}

		err = i.GetEnrollers(cryptos, instance, storagePath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to populate init orderer with Enrollment spec")
		}
	}

	return &Orderer{
		Cryptos: cryptos,
	}, nil
}

func (i *Initializer) GetUpdatedOrderer(instance *current.IBPOrderer) (*Orderer, error) {
	cryptos := &config.Cryptos{}

	// Only check for any new certs passed through MSP spec
	err := i.GetMSPCrypto(cryptos, instance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to populate updated init orderer with MSP spec")
	}

	return &Orderer{
		Cryptos: cryptos,
	}, nil
}

func (i *Initializer) GenerateSecrets(prefix common.SecretType, instance *current.IBPOrderer, crypto *config.Response) error {
	if crypto == nil {
		return nil
	}
	return i.SecretManager.GenerateSecrets(prefix, instance, crypto)
}

func (i *Initializer) GenerateSecretsFromResponse(instance *current.IBPOrderer, cryptoResponse *config.CryptoResponse) error {
	return i.SecretManager.GenerateSecretsFromResponse(instance, cryptoResponse)
}

func (i *Initializer) UpdateSecrets(prefix common.SecretType, instance *current.IBPOrderer, crypto *config.Response) error {
	if crypto == nil {
		return nil
	}
	return i.SecretManager.UpdateSecrets(prefix, instance, crypto)
}

func (i *Initializer) UpdateSecretsFromResponse(instance *current.IBPOrderer, cryptoResponse *config.CryptoResponse) error {
	return i.SecretManager.UpdateSecretsFromResponse(instance, cryptoResponse)
}

func (i *Initializer) GetCrypto(instance *current.IBPOrderer) (*config.CryptoResponse, error) {
	return i.SecretManager.GetCryptoResponseFromSecrets(instance)
}

func (i *Initializer) Delete(instance *current.IBPOrderer) error {
	name := fmt.Sprintf("%s%s", instance.Name, i.Name)
	prefix := "ecert"
	err := i.SecretManager.DeleteSecrets(prefix, instance, name)
	if err != nil {
		return err
	}

	prefix = "tls"
	err = i.SecretManager.DeleteSecrets(prefix, instance, name)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{}
	cm.Name = instance.Name + "-" + i.Name + "-config"
	cm.Namespace = instance.Namespace

	err = i.Client.Delete(context.TODO(), cm)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete config map '%s'", cm.Name)
		}
	}

	return nil
}

func (i *Initializer) MissingCrypto(instance *current.IBPOrderer) bool {
	isHSMEnabled := instance.IsHSMEnabled()
	if isHSMEnabled {
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

func (i *Initializer) CreateOrUpdateConfigMap(instance *current.IBPOrderer, orderer OrdererConfig) error {
	name := fmt.Sprintf("%s-config", instance.GetName())
	log.Info(fmt.Sprintf("Creating/Updating config map '%s'...", name))

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.GetNamespace(),
			Labels:    i.GetLabels(instance),
		},
		BinaryData: map[string][]byte{},
	}

	existing, err := i.GetConfigFromConfigMap(instance)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	if existing != nil {
		cm.BinaryData = existing.BinaryData
	}

	if orderer != nil {
		err := i.addOrdererConfigToCM(instance, cm, orderer)
		if err != nil {
			return err
		}
	}

	err = i.addNodeOUToCM(instance, cm)
	if err != nil {
		return err
	}

	err = i.Client.CreateOrUpdate(context.TODO(), cm, k8sclient.CreateOrUpdateOption{
		Owner:  instance,
		Scheme: i.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create Orderer config map")
	}

	return nil
}

func (i *Initializer) addOrdererConfigToCM(instance *current.IBPOrderer, cm *corev1.ConfigMap, orderer OrdererConfig) error {
	ordererBytes, err := orderer.ToBytes()
	if err != nil {
		return err
	}
	cm.BinaryData["orderer.yaml"] = ordererBytes

	return nil
}

func (i *Initializer) addNodeOUToCM(instance *current.IBPOrderer, cm *corev1.ConfigMap) error {
	if !instance.Spec.NodeOUDisabled() {
		configFilePath := i.Config.OUFile
		// Check if both intermediate ecerts and tlscerts secrets exists
		if util.IntermediateSecretExists(i.Client, instance.Namespace, fmt.Sprintf("ecert-%s-intercerts", instance.Name)) &&
			util.IntermediateSecretExists(i.Client, instance.Namespace, fmt.Sprintf("tls-%s-intercerts", instance.Name)) {
			configFilePath = i.Config.InterOUFile
		}
		ouBytes, err := ioutil.ReadFile(filepath.Clean(configFilePath))
		if err != nil {
			return err
		}
		cm.BinaryData["config.yaml"] = ouBytes
	} else {
		// Set enabled to false in config
		nodeOUConfig, err := config.NodeOUConfigFromBytes(cm.BinaryData["config.yaml"])
		if err != nil {
			return err
		}

		nodeOUConfig.NodeOUs.Enable = false
		ouBytes, err := config.NodeOUConfigToBytes(nodeOUConfig)
		if err != nil {
			return err
		}

		cm.BinaryData["config.yaml"] = ouBytes
	}

	return nil
}

func (i *Initializer) GetConfigFromConfigMap(instance *current.IBPOrderer) (*corev1.ConfigMap, error) {
	return common.GetConfigFromConfigMap(i.Client, instance)
}

func GetDomain(address string) string {
	u := strings.Split(address, ":")
	return u[0]
}

func (i *Initializer) GetLabels(instance metav1.Object) map[string]string {
	label := os.Getenv("OPERATOR_LABEL_PREFIX")
	if label == "" {
		label = "fabric"
	}

	return map[string]string{
		"app":                          instance.GetName(),
		"app.kubernetes.io/name":       label,
		"app.kubernetes.io/instance":   label + "orderer",
		"app.kubernetes.io/managed-by": label + "-operator",
	}
}

func (i *Initializer) CheckIfAdminCertsUpdated(instance *current.IBPOrderer) (bool, error) {
	log.Info("Checking if admin certs updated")
	current := common.GetAdminCertsFromSecret(i.Client, instance)
	updated := common.GetAdminCertsFromSpec(instance.Spec.Secret)

	return common.CheckIfCertsDifferent(current, updated)
}

func (i *Initializer) UpdateAdminSecret(instance *current.IBPOrderer) error {
	return i.SecretManager.UpdateAdminCertSecret(instance, instance.Spec.Secret)
}

func (i *Initializer) GetCoreConfigFromFile(instance *current.IBPOrderer, file string) (OrdererConfig, error) {
	switch version.GetMajorReleaseVersion(instance.Spec.FabricVersion) {
	case version.V2:
		currentVer := version.String(instance.Spec.FabricVersion)
		if currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_5_1) {
			log.Info("v2.5.x Fabric Orderer requested")
			v25config, err := v25ordererconfig.ReadOrdererFile(file)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read v2.5.x default config file")
			}
			return v25config, nil
		} else if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.GreaterThan(version.V2_4_1) {
			log.Info("v2.4.x Fabric Orderer requested")
			v24config, err := v24ordererconfig.ReadOrdererFile(file)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read v2.4.x default config file")
			}
			return v24config, nil
		} else {
			log.Info("v2.2.x Fabric Orderer requested")
			v2config, err := v2ordererconfig.ReadOrdererFile(file)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read v2.2.x default config file")
			}
			return v2config, nil
		}
	case version.V1:
		fallthrough
	default:
		// Choosing to default to v1.4 to not break backwards comptability, if coming
		// from a previous version of operator the 'FabricVersion' field would not be set and would
		// result in an error. // TODO: Determine if we want to throw error or handle setting
		// FabricVersion as part of migration logic.
		log.Info("v1.4 Fabric Orderer requested")
		oconfig, err := ordererconfig.ReadOrdererFile(file)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read v1.4 default config file")
		}
		return oconfig, nil
	}
}

func (i *Initializer) GetCoreConfigFromBytes(instance *current.IBPOrderer, bytes []byte) (OrdererConfig, error) {
	switch version.GetMajorReleaseVersion(instance.Spec.FabricVersion) {
	case version.V2:
		currentVer := version.String(instance.Spec.FabricVersion)
		if currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_5_1) {
			log.Info("v2.5.x Fabric Orderer requested")
			v25config, err := v25ordererconfig.ReadOrdererFromBytes(bytes)
			if err != nil {
				return nil, err
			}
			return v25config, nil
		} else if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.GreaterThan(version.V2_4_1) {
			log.Info("v2.4.x Fabric Orderer requested")
			v24config, err := v24ordererconfig.ReadOrdererFromBytes(bytes)
			if err != nil {
				return nil, err
			}
			return v24config, nil
		} else {
			log.Info("v2.2.x Fabric Orderer requested")
			v2config, err := v2ordererconfig.ReadOrdererFromBytes(bytes)
			if err != nil {
				return nil, err
			}
			return v2config, nil
		}
	case version.V1:
		fallthrough
	default:
		// Choosing to default to v1.4 to not break backwards comptability, if coming
		// from a previous version of operator the 'FabricVersion' field would not be set and would
		// result in an error.
		log.Info("v1.4 Fabric Orderer requested")
		oconfig, err := ordererconfig.ReadOrdererFromBytes(bytes)
		if err != nil {
			return nil, err
		}
		return oconfig, nil
	}
}
