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

package validator

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"strings"

	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type Validator struct {
	Client     k8sclient.Client
	HSMEnabled bool
}

func (v *Validator) CheckAdminCerts(instance v1.Object, prefix string) error {
	//No-op
	return nil
}

func (v *Validator) CheckEcertCrypto(instance v1.Object, name string) error {
	prefix := "ecert-" + name

	// CA certs verification
	err := v.CheckCACerts(instance, prefix)
	if err != nil {
		return err
	}

	if v.HSMEnabled {
		err = v.CheckCert(instance, prefix)
		if err != nil {
			return err
		}
	} else {
		err = v.CheckCertAndKey(instance, prefix)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *Validator) CheckTLSCrypto(instance v1.Object, name string) error {
	prefix := "tls-" + name

	// CA certs verification
	err := v.CheckCACerts(instance, prefix)
	if err != nil {
		return err
	}

	err = v.CheckCertAndKey(instance, prefix)
	if err != nil {
		return err
	}

	return nil
}

func (v *Validator) CheckClientAuthCrypto(instance v1.Object, name string) error {
	prefix := "clientauth-" + name

	// CA cert verification
	err := v.CheckCACerts(instance, prefix)
	if err != nil {
		return err
	}

	err = v.CheckCertAndKey(instance, prefix)
	if err != nil {
		return err
	}

	return nil
}

func (v *Validator) CheckCACerts(instance v1.Object, prefix string) error {
	namespacedName := types.NamespacedName{
		Name:      prefix + "-cacerts",
		Namespace: instance.GetNamespace(),
	}

	caCerts := &corev1.Secret{}
	err := v.Client.Get(context.TODO(), namespacedName, caCerts)
	if err != nil {
		return err
	}

	if caCerts.Data == nil || len(caCerts.Data) == 0 {
		return errors.New("no ca certificates found in cacerts secret")
	}

	err = ValidateCerts(caCerts.Data)
	if err != nil {
		return errors.Wrap(err, "not a proper ca cert")
	}

	return nil
}

func (v *Validator) CheckCertAndKey(instance v1.Object, prefix string) error {
	var err error

	// Sign cert verification
	err = v.CheckCert(instance, prefix)
	if err != nil {
		return err
	}

	// Key verification
	err = v.CheckKey(instance, prefix)
	if err != nil {
		return err
	}

	return nil
}

func (v *Validator) CheckCert(instance v1.Object, prefix string) error {
	namespacedName := types.NamespacedName{
		Namespace: instance.GetNamespace(),
	}

	signCert := &corev1.Secret{}
	namespacedName.Name = prefix + "-signcert"
	err := v.Client.Get(context.TODO(), namespacedName, signCert)
	if err != nil {
		return err
	}

	err = ValidateCert(signCert.Data["cert.pem"])
	if err != nil {
		return errors.Wrap(err, "not a proper sign cert")
	}

	return nil
}

func (v *Validator) CheckKey(instance v1.Object, prefix string) error {
	namespacedName := types.NamespacedName{
		Namespace: instance.GetNamespace(),
	}

	key := &corev1.Secret{}
	namespacedName.Name = prefix + "-keystore"
	err := v.Client.Get(context.TODO(), namespacedName, key)
	if err != nil {
		return err
	}

	err = ValidateKey(key.Data["key.pem"])
	if err != nil {
		return errors.Wrap(err, "not a proper key")
	}

	return nil
}

func (v *Validator) SetHSMEnabled(enabled bool) {
	v.HSMEnabled = enabled
}

func CheckError(err error) bool {
	if err == nil {
		return false
	}

	if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not valid") {
		return true
	}

	return false
}

func ValidateCerts(certs map[string][]byte) error {
	for _, cert := range certs {
		err := ValidateCert(cert)
		if err != nil {
			return err
		}
	}

	return nil
}

func ValidateCert(cert []byte) error {
	block, _ := pem.Decode(cert)
	if block == nil {
		return errors.New("failed to get certificate block")
	}

	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return errors.Wrap(err, "not valid")
	}

	return nil
}

func ValidateKey(key []byte) error {
	block, _ := pem.Decode(key)
	if block == nil {
		return errors.New("failed to get key block")
	}

	_, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return errors.Wrap(err, "not valid")
	}

	return nil
}
