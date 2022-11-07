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

package baseca

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	cav1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	caconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/yaml"
)

//go:generate counterfeiter -o mocks/initializer.go -fake-name Initializer . Initializer

type Initializer interface {
	Create(*current.IBPCA, *cav1.ServerConfig, initializer.IBPCA) (*initializer.Response, error)
	Update(*current.IBPCA, *cav1.ServerConfig, initializer.IBPCA) (*initializer.Response, error)
}

type Initialize struct {
	Config *initializer.Config
	Scheme *runtime.Scheme
	Labels func(instance v1.Object) map[string]string

	Initializer Initializer
	Client      k8sclient.Client
}

func NewInitializer(config *initializer.Config, scheme *runtime.Scheme, client k8sclient.Client, labels func(instance v1.Object) map[string]string, timeouts initializer.HSMInitJobTimeouts) *Initialize {
	return &Initialize{
		Config:      config,
		Initializer: &initializer.Initializer{Client: client, Timeouts: timeouts},
		Scheme:      scheme,
		Client:      client,
		Labels:      labels,
	}
}

func (i *Initialize) HandleEnrollmentCAInit(instance *current.IBPCA, update Update) (*initializer.Response, error) {
	var err error
	var resp *initializer.Response

	log.Info(fmt.Sprintf("Checking if enrollment CA '%s' needs initialization", instance.GetName()))

	if i.SecretExists(instance, fmt.Sprintf("%s-ca-crypto", instance.GetName())) {
		if update.CAOverridesUpdated() {
			resp, err = i.UpdateEnrollmentCAConfig(instance)
			if err != nil {
				return nil, err
			}
		}
	} else {
		resp, err = i.CreateEnrollmentCAConfig(instance)
		if err != nil {
			return nil, err
		}

	}

	return resp, nil
}

func (i *Initialize) HandleTLSCAInit(instance *current.IBPCA, update Update) (*initializer.Response, error) {
	var err error
	var resp *initializer.Response

	log.Info(fmt.Sprintf("Checking if TLS CA '%s' needs initialization", instance.GetName()))

	if i.SecretExists(instance, fmt.Sprintf("%s-tlsca-crypto", instance.GetName())) {
		if update.TLSCAOverridesUpdated() {
			resp, err = i.UpdateTLSCAConfig(instance)
			if err != nil {
				return nil, err
			}
		}
	} else {
		resp, err = i.CreateTLSCAConfig(instance)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (i *Initialize) CreateEnrollmentCAConfig(instance *current.IBPCA) (*initializer.Response, error) {
	log.Info(fmt.Sprintf("Creating Enrollment CA config '%s'", instance.GetName()))
	bytes, err := ioutil.ReadFile(i.Config.CADefaultConfigPath)
	if err != nil {
		return nil, err
	}

	sca, err := i.GetEnrollmentInitCA(instance, bytes)
	if err != nil {
		return nil, err
	}

	var caOverrides *cav1.ServerConfig
	if instance.Spec.ConfigOverride != nil && instance.Spec.ConfigOverride.CA != nil {
		caOverrides = &cav1.ServerConfig{}
		err = json.Unmarshal(instance.Spec.ConfigOverride.CA.Raw, caOverrides)
		if err != nil {
			return nil, err
		}
	}

	resp, err := i.Initializer.Create(instance, caOverrides, sca)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (i *Initialize) UpdateEnrollmentCAConfig(instance *current.IBPCA) (*initializer.Response, error) {
	log.Info(fmt.Sprintf("Updating Enrollment CA config '%s'", instance.GetName()))
	cmname := fmt.Sprintf("%s-ca-config", instance.GetName())
	cm, err := i.ReadConfigMap(instance, cmname)
	if err != nil {
		return nil, err
	}

	sca, err := i.GetEnrollmentInitCA(instance, cm.BinaryData["fabric-ca-server-config.yaml"])
	if err != nil {
		return nil, err
	}

	var caOverrides *cav1.ServerConfig
	if instance.Spec.ConfigOverride != nil && instance.Spec.ConfigOverride.CA != nil {
		caOverrides = &cav1.ServerConfig{}
		err = json.Unmarshal(instance.Spec.ConfigOverride.CA.Raw, caOverrides)
		if err != nil {
			return nil, err
		}
	}

	resp, err := i.Initializer.Update(instance, caOverrides, sca)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (i *Initialize) GetEnrollmentInitCA(instance *current.IBPCA, data []byte) (*initializer.CA, error) {
	serverConfig := &cav1.ServerConfig{}
	err := yaml.Unmarshal(data, serverConfig)
	if err != nil {
		return nil, err
	}

	initCAConfig := &caconfig.Config{
		ServerConfig: serverConfig,
		HomeDir:      filepath.Join(i.Config.SharedPath, instance.GetName(), "ca"),
		MountPath:    "/crypto/ca",
		SqlitePath:   instance.Spec.CustomNames.Sqlite,
	}

	cn := instance.GetName() + "-ca"
	if instance.Spec.ConfigOverride != nil && instance.Spec.ConfigOverride.CA != nil {
		configOverride, err := config.ReadFrom(&instance.Spec.ConfigOverride.CA.Raw)
		if err != nil {
			return nil, err
		}
		if configOverride.ServerConfig.CSR.CN != "" {
			cn = configOverride.ServerConfig.CSR.CN
		}
	}

	sca := initializer.NewCA(initCAConfig, caconfig.EnrollmentCA, i.Config.SharedPath, instance.UsingHSMProxy(), cn)

	return sca, nil
}

func (i *Initialize) CreateTLSCAConfig(instance *current.IBPCA) (*initializer.Response, error) {
	log.Info(fmt.Sprintf("Creating TLS CA config '%s'", instance.GetName()))
	bytes, err := ioutil.ReadFile(i.Config.TLSCADefaultConfigPath)
	if err != nil {
		return nil, err
	}

	sca, err := i.GetTLSInitCA(instance, bytes)
	if err != nil {
		return nil, err
	}

	var tlscaOverrides *cav1.ServerConfig
	if instance.Spec.ConfigOverride != nil && instance.Spec.ConfigOverride.TLSCA != nil {
		tlscaOverrides = &cav1.ServerConfig{}
		err = json.Unmarshal(instance.Spec.ConfigOverride.TLSCA.Raw, tlscaOverrides)
		if err != nil {
			return nil, err
		}
	}

	resp, err := i.Initializer.Create(instance, tlscaOverrides, sca)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (i *Initialize) UpdateTLSCAConfig(instance *current.IBPCA) (*initializer.Response, error) {
	log.Info(fmt.Sprintf("Updating TLSCA config '%s'", instance.GetName()))
	cmname := fmt.Sprintf("%s-tlsca-config", instance.GetName())
	cm, err := i.ReadConfigMap(instance, cmname)
	if err != nil {
		return nil, err
	}

	tca, err := i.GetTLSInitCA(instance, cm.BinaryData["fabric-ca-server-config.yaml"])
	if err != nil {
		return nil, err
	}

	var tlscaOverrides *cav1.ServerConfig
	if instance.Spec.ConfigOverride != nil && instance.Spec.ConfigOverride.TLSCA != nil {
		tlscaOverrides = &cav1.ServerConfig{}
		err = json.Unmarshal(instance.Spec.ConfigOverride.TLSCA.Raw, tlscaOverrides)
		if err != nil {
			return nil, err
		}
	}

	resp, err := i.Initializer.Update(instance, tlscaOverrides, tca)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (i *Initialize) GetTLSInitCA(instance *current.IBPCA, data []byte) (*initializer.CA, error) {
	serverConfig := &cav1.ServerConfig{}
	err := yaml.Unmarshal(data, serverConfig)
	if err != nil {
		return nil, err
	}

	initCAConfig := &caconfig.Config{
		ServerConfig: serverConfig,
		HomeDir:      filepath.Join(i.Config.SharedPath, instance.GetName(), "tlsca"),
		MountPath:    "/crypto/tlsca",
		SqlitePath:   instance.Spec.CustomNames.Sqlite,
	}

	cn := instance.GetName() + "-tlsca"
	if instance.Spec.ConfigOverride != nil && instance.Spec.ConfigOverride.TLSCA != nil {
		configOverride, err := config.ReadFrom(&instance.Spec.ConfigOverride.TLSCA.Raw)
		if err != nil {
			return nil, err
		}
		if configOverride.ServerConfig.CSR.CN != "" {
			cn = configOverride.ServerConfig.CSR.CN
		}
	}

	tca := initializer.NewCA(initCAConfig, caconfig.TLSCA, i.Config.SharedPath, instance.UsingHSMProxy(), cn)

	return tca, nil
}

func (i *Initialize) HandleConfigResources(name string, instance *current.IBPCA, resp *initializer.Response, update Update) error {
	var err error

	if update.CAOverridesUpdated() || update.TLSCAOverridesUpdated() {
		log.Info(fmt.Sprintf("Updating config resources for '%s'", name))
		err = i.UpdateConfigResources(name, instance, resp)
		if err != nil {
			return err
		}
	} else {
		log.Info(fmt.Sprintf("Creating config resources for '%s'", name))
		err = i.CreateConfigResources(name, instance, resp)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Initialize) UpdateConfigResources(name string, instance *current.IBPCA, resp *initializer.Response) error {
	var err error

	secretName := fmt.Sprintf("%s-crypto", name)
	secret, err := i.GetCryptoSecret(instance, secretName)
	if err != nil {
		return err
	}

	mergedCrypto := i.MergeCryptoMaterial(secret.Data, resp.CryptoMap)

	mergedResp := &initializer.Response{
		CryptoMap: mergedCrypto,
		Config:    resp.Config,
	}

	err = i.CreateConfigResources(name, instance, mergedResp)
	if err != nil {
		return err
	}

	return nil
}

func (i *Initialize) CreateConfigResources(name string, instance *current.IBPCA, resp *initializer.Response) error {
	var err error

	if len(resp.CryptoMap) > 0 {
		secretName := fmt.Sprintf("%s-crypto", name)
		err = i.CreateOrUpdateCryptoSecret(instance, resp.CryptoMap, secretName)
		if err != nil {
			return err
		}
	}

	if resp.Config != nil {
		bytes, err := ConfigToBytes(resp.Config)
		if err != nil {
			return err
		}

		data := map[string][]byte{
			"fabric-ca-server-config.yaml": bytes,
		}
		cmName := fmt.Sprintf("%s-config", name)
		err = i.CreateOrUpdateConfigMap(instance, data, cmName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Initialize) ReadConfigMap(instance *current.IBPCA, name string) (*corev1.ConfigMap, error) {
	n := types.NamespacedName{
		Name:      name,
		Namespace: instance.GetNamespace(),
	}

	cm := &corev1.ConfigMap{}
	err := i.Client.Get(context.TODO(), n, cm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config map")
	}

	return cm, nil
}

func (i *Initialize) CreateOrUpdateConfigMap(instance *current.IBPCA, data map[string][]byte, name string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: instance.GetNamespace(),
			Labels:    i.Labels(instance),
		},
		BinaryData: data,
	}

	err := i.Client.CreateOrUpdate(context.TODO(), cm, k8sclient.CreateOrUpdateOption{
		Owner:  instance,
		Scheme: i.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create/update config map")
	}

	return nil
}

func (i *Initialize) CreateOrUpdateCryptoSecret(instance *current.IBPCA, caCrypto map[string][]byte, name string) error {
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: instance.Namespace,
			Labels:    i.Labels(instance),
		},
		Data: caCrypto,
		Type: corev1.SecretTypeOpaque,
	}

	err := i.Client.CreateOrUpdate(context.TODO(), secret, k8sclient.CreateOrUpdateOption{
		Owner:  instance,
		Scheme: i.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create/update secret")
	}

	return nil
}

func (i *Initialize) GetCryptoSecret(instance *current.IBPCA, name string) (*corev1.Secret, error) {
	log.Info(fmt.Sprintf("Getting secret '%s'", name))

	nn := types.NamespacedName{
		Name:      name,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := i.Client.Get(context.TODO(), nn, secret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create/update secret")
	}

	return secret, nil
}

func (i *Initialize) SyncDBConfig(orig *current.IBPCA) (*current.IBPCA, error) {
	instance := orig.DeepCopy()
	if instance.Spec.ConfigOverride != nil {
		if instance.Spec.ConfigOverride.CA != nil {
			eca := &cav1.ServerConfig{}
			err := json.Unmarshal(instance.Spec.ConfigOverride.CA.Raw, eca)
			if err != nil {
				return nil, err
			}

			if instance.Spec.ConfigOverride.TLSCA == nil {
				tca := &cav1.ServerConfig{}
				tca.CAConfig.DB = eca.CAConfig.DB

				tbytes, err := json.Marshal(tca)
				if err != nil {
					return nil, err
				}

				instance.Spec.ConfigOverride.TLSCA = &runtime.RawExtension{Raw: tbytes}
			} else {
				tca := &cav1.ServerConfig{}
				err := json.Unmarshal(instance.Spec.ConfigOverride.TLSCA.Raw, tca)
				if err != nil {
					return nil, err
				}

				tca.CAConfig.DB = eca.CAConfig.DB
				tbytes, err := json.Marshal(tca)
				if err != nil {
					return nil, err
				}

				instance.Spec.ConfigOverride.TLSCA = &runtime.RawExtension{Raw: tbytes}
			}
		}
	}
	return instance, nil
}

func (i *Initialize) MergeCryptoMaterial(current map[string][]byte, updated map[string][]byte) map[string][]byte {
	for ukey, umaterial := range updated {
		if len(umaterial) != 0 {
			current[ukey] = umaterial
		}
	}

	return current
}

func (i *Initialize) SecretExists(instance *current.IBPCA, name string) bool {
	n := types.NamespacedName{
		Name:      name,
		Namespace: instance.GetNamespace(),
	}

	s := &corev1.Secret{}
	err := i.Client.Get(context.TODO(), n, s)

	return err == nil
}

func ConfigToBytes(c *cav1.ServerConfig) ([]byte, error) {
	bytes, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
