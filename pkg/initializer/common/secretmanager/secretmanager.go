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

package secretmanager

import (
	"context"
	"fmt"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("secret_manager")

type SecretManager struct {
	Client    k8sclient.Client
	Scheme    *runtime.Scheme
	GetLabels func(instance v1.Object) map[string]string
}

func New(client k8sclient.Client, scheme *runtime.Scheme, labels func(instance v1.Object) map[string]string) *SecretManager {
	return &SecretManager{
		Client:    client,
		Scheme:    scheme,
		GetLabels: labels,
	}
}

func (s *SecretManager) GenerateSecrets(prefix common.SecretType, instance v1.Object, crypto *config.Response) error {
	if crypto == nil {
		return nil
	}

	if prefix != common.TLS {
		err := s.CreateAdminSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "admincerts"), instance, crypto.AdminCerts)
		if err != nil {
			return errors.Wrap(err, "failed to create admin certs secret")
		}
	}

	err := s.CreateCACertsSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "cacerts"), instance, crypto.CACerts)
	if err != nil {
		return errors.Wrap(err, "failed to create ca certs secret")
	}

	err = s.CreateIntermediateCertsSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "intercerts"), instance, crypto.IntermediateCerts)
	if err != nil {
		return errors.Wrap(err, "failed to create intermediate ca certs secret")
	}

	err = s.CreateSignCert(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "signcert"), instance, crypto.SignCert)
	if err != nil {
		return errors.Wrap(err, "failed to create signing cert secret")
	}

	err = s.CreateKey(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "keystore"), instance, crypto.Keystore)
	if err != nil {
		return errors.Wrap(err, "failed to create key secret")
	}

	return nil
}

func (s *SecretManager) UpdateSecrets(prefix common.SecretType, instance v1.Object, crypto *config.Response) error {
	// AdminCert updates are checked in base Initialize() code

	err := s.CreateCACertsSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "cacerts"), instance, crypto.CACerts)
	if err != nil {
		return errors.Wrap(err, "failed to create ca certs secret")
	}

	err = s.CreateIntermediateCertsSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "intercerts"), instance, crypto.IntermediateCerts)
	if err != nil {
		return errors.Wrap(err, "failed to create intermediate ca certs secret")
	}

	err = s.CreateSignCert(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "signcert"), instance, crypto.SignCert)
	if err != nil {
		return errors.Wrap(err, "failed to create signing cert secret")
	}

	err = s.CreateKey(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "keystore"), instance, crypto.Keystore)
	if err != nil {
		return errors.Wrap(err, "failed to create key secret")
	}

	return nil
}

func (s *SecretManager) CreateAdminSecret(name string, instance v1.Object, adminCerts [][]byte) error {
	if len(adminCerts) == 0 || string(adminCerts[0]) == "" {
		return nil
	}

	data := s.GetCertsData("admincert", adminCerts)
	err := s.CreateOrUpdateSecret(instance, name, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *SecretManager) CreateCACertsSecret(name string, instance v1.Object, caCerts [][]byte) error {
	if len(caCerts) == 0 {
		return nil
	}

	data := s.GetCertsData("cacert", caCerts)
	err := s.CreateOrUpdateSecret(instance, name, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *SecretManager) CreateIntermediateCertsSecret(name string, instance v1.Object, interCerts [][]byte) error {
	if len(interCerts) == 0 {
		return nil
	}

	data := s.GetCertsData("intercert", interCerts)
	err := s.CreateOrUpdateSecret(instance, name, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *SecretManager) CreateSignCert(name string, instance v1.Object, cert []byte) error {
	if len(cert) == 0 {
		return nil
	}

	data := map[string][]byte{
		"cert.pem": cert,
	}
	err := s.CreateOrUpdateSecret(instance, name, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *SecretManager) CreateKey(name string, instance v1.Object, key []byte) error {
	if key == nil {
		return nil
	}

	data := map[string][]byte{
		"key.pem": key,
	}
	err := s.CreateOrUpdateSecret(instance, name, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *SecretManager) CreateOrUpdateSecret(instance v1.Object, name string, data map[string][]byte) error {
	log.Info(fmt.Sprintf("Create/Update secret '%s'", name))

	secret := s.BuildSecret(instance, name, data, s.GetLabels(instance))
	err := s.Client.CreateOrUpdate(context.TODO(), secret, k8sclient.CreateOrUpdateOption{
		Owner:  instance,
		Scheme: s.Scheme,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *SecretManager) UpdateAdminCertSecret(instance v1.Object, secretSpec *current.SecretSpec) error {
	name := fmt.Sprintf("ecert-%s-admincerts", instance.GetName())

	adminCerts := common.GetAdminCertsFromSpec(secretSpec)

	if len(adminCerts) == 0 || string(adminCerts[0]) == "" {
		return nil
	}

	adminCertsBytes, err := common.ConvertCertsToBytes(adminCerts)
	if err != nil {
		return err
	}

	data := s.GetCertsData("admincert", adminCertsBytes)
	err = s.CreateOrUpdateSecret(instance, name, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *SecretManager) BuildSecret(instance v1.Object, name string, data map[string][]byte, labels map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: instance.GetNamespace(),
			Labels:    labels,
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
}

func (s *SecretManager) GetSecret(name string, instance v1.Object) (*corev1.Secret, error) {
	n := types.NamespacedName{
		Name:      name,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := s.Client.Get(context.TODO(), n, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return secret, nil
}

func (s *SecretManager) GetCertsData(certType string, certs [][]byte) map[string][]byte {
	data := map[string][]byte{}
	for i, cert := range certs {
		if string(cert) == "" {
			continue
		}
		data[fmt.Sprintf("%s-%d.pem", certType, i)] = cert
	}

	return data
}

func (s *SecretManager) DeleteSecrets(prefix string, instance v1.Object, name string) error {
	secret := &corev1.Secret{}
	secret.Namespace = instance.GetNamespace()

	secret.Name = fmt.Sprintf("%s-%s-%s", prefix, name, "admincerts")
	err := s.Client.Delete(context.TODO(), secret)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete secret '%s'", secret.Name)
		}
	}

	secret.Name = fmt.Sprintf("%s-%s-%s", prefix, name, "cacerts")
	err = s.Client.Delete(context.TODO(), secret)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete secret '%s'", secret.Name)
		}
	}

	secret.Name = fmt.Sprintf("%s-%s-%s", prefix, name, "intercerts")
	err = s.Client.Delete(context.TODO(), secret)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete secret '%s'", secret.Name)
		}
	}

	secret.Name = fmt.Sprintf("%s-%s-%s", prefix, name, "signcert")
	err = s.Client.Delete(context.TODO(), secret)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete secret '%s'", secret.Name)
		}
	}

	secret.Name = fmt.Sprintf("%s-%s-%s", prefix, name, "keystore")
	err = s.Client.Delete(context.TODO(), secret)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete secret '%s'", secret.Name)
		}
	}

	return nil
}

func (s *SecretManager) GetCryptoFromSecrets(prefix common.SecretType, instance v1.Object) (*config.Response, error) {
	resp := &config.Response{}

	admincerts, err := s.GetSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "admincerts"), instance)
	if err != nil {
		return nil, err
	}
	if admincerts != nil {
		resp.AdminCerts = s.GetCertBytesFromData(admincerts.Data)
	}

	cacerts, err := s.GetSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "cacerts"), instance)
	if err != nil {
		return nil, err
	}
	if cacerts != nil {
		resp.CACerts = s.GetCertBytesFromData(cacerts.Data)
	}

	intercerts, err := s.GetSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "intercerts"), instance)
	if err != nil {
		return nil, err
	}
	if intercerts != nil {
		resp.IntermediateCerts = s.GetCertBytesFromData(intercerts.Data)
	}

	signcert, err := s.GetSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "signcert"), instance)
	if err != nil {
		return nil, err
	}
	if signcert != nil {
		resp.SignCert = signcert.Data["cert.pem"]
	}

	keystore, err := s.GetSecret(fmt.Sprintf("%s-%s-%s", prefix, instance.GetName(), "keystore"), instance)
	if err != nil {
		return nil, err
	}
	if keystore != nil {
		resp.Keystore = keystore.Data["key.pem"]
	}

	return resp, nil
}

func (s *SecretManager) GetCertBytesFromData(data map[string][]byte) [][]byte {
	bytes := [][]byte{}
	for _, cert := range data {
		bytes = append(bytes, cert)
	}
	return bytes
}

func (s *SecretManager) GenerateSecretsFromResponse(instance v1.Object, cryptoResponse *config.CryptoResponse) error {
	if cryptoResponse != nil {
		err := s.GenerateSecrets("ecert", instance, cryptoResponse.Enrollment)
		if err != nil {
			return errors.Wrap(err, "failed to generate ecert secrets")
		}

		err = s.GenerateSecrets("tls", instance, cryptoResponse.TLS)
		if err != nil {
			return errors.Wrap(err, "failed to generate tls secrets")
		}

		err = s.GenerateSecrets("clientauth", instance, cryptoResponse.ClientAuth)
		if err != nil {
			return errors.Wrap(err, "failed to generate client auth secrets")
		}
	}
	return nil
}

func (s *SecretManager) UpdateSecretsFromResponse(instance v1.Object, cryptoResponse *config.CryptoResponse) error {
	if cryptoResponse != nil {
		err := s.UpdateSecrets("ecert", instance, cryptoResponse.Enrollment)
		if err != nil {
			return errors.Wrap(err, "failed to update ecert secrets")
		}

		err = s.UpdateSecrets("tls", instance, cryptoResponse.TLS)
		if err != nil {
			return errors.Wrap(err, "failed to update tls secrets")
		}

		err = s.UpdateSecrets("clientauth", instance, cryptoResponse.ClientAuth)
		if err != nil {
			return errors.Wrap(err, "failed to update client auth secrets")
		}
	}
	return nil
}

func (s *SecretManager) GetCryptoResponseFromSecrets(instance v1.Object) (*config.CryptoResponse, error) {
	var err error
	cryptoResponse := &config.CryptoResponse{}

	cryptoResponse.Enrollment, err = s.GetCryptoFromSecrets("ecert", instance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ecert crypto")
	}
	cryptoResponse.TLS, err = s.GetCryptoFromSecrets("tls", instance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tls crypto")
	}
	cryptoResponse.ClientAuth, err = s.GetCryptoFromSecrets("clientauth", instance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client auth crypto")
	}

	return cryptoResponse, nil
}
