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

package common

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/mspparser"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/validator"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("initializer")

type SecretType string

var (
	ECERT SecretType = "ecert"
	TLS   SecretType = "tls"
)

type Instance interface {
	metav1.Object
	runtime.Object
	EnrollerImage() string
	GetPullSecrets() []corev1.LocalObjectReference
	IsHSMEnabled() bool
	UsingHSMProxy() bool
	GetConfigOverride() (interface{}, error)
}

// NOTE: Modifies cryptos object passed as param
func GetCommonEnrollers(cryptos *config.Cryptos, enrollmentSpec *current.EnrollmentSpec, storagePath string) error {
	if enrollmentSpec.TLS != nil && cryptos.TLS == nil {
		bytes, err := enrollmentSpec.TLS.GetCATLSBytes()
		if err != nil {
			return err
		}

		caClient := enroller.NewFabCAClient(
			enrollmentSpec.TLS,
			filepath.Join(storagePath, "tls"),
			nil,
			bytes,
		)
		cryptos.TLS = enroller.New(enroller.NewSWEnroller(caClient))
	}

	if enrollmentSpec.ClientAuth != nil && cryptos.ClientAuth == nil {
		bytes, err := enrollmentSpec.ClientAuth.GetCATLSBytes()
		if err != nil {
			return err
		}

		caClient := enroller.NewFabCAClient(
			enrollmentSpec.ClientAuth,
			filepath.Join(storagePath, "clientauth"),
			nil,
			bytes,
		)
		cryptos.ClientAuth = enroller.New(enroller.NewSWEnroller(caClient))
	}

	return nil
}

// NOTE: Modifies cryptos object passed as param
func GetMSPCrypto(cryptos *config.Cryptos, mspSpec *current.MSPSpec) error {
	if mspSpec != nil {
		if mspSpec.Component != nil {
			cryptos.Enrollment = mspparser.New(mspSpec.Component)
		}

		if mspSpec.TLS != nil {
			cryptos.TLS = mspparser.New(mspSpec.TLS)
		}

		if mspSpec.ClientAuth != nil {
			cryptos.ClientAuth = mspparser.New(mspSpec.ClientAuth)
		}
	}

	return nil
}

//go:generate counterfeiter -o mocks/cryptovalidator.go -fake-name CryptoValidator . CryptoValidator
type CryptoValidator interface {
	CheckEcertCrypto(v1.Object, string) error
	CheckTLSCrypto(v1.Object, string) error
	CheckClientAuthCrypto(v1.Object, string) error
	SetHSMEnabled(bool)
}

func CheckCrypto(cryptoValidator CryptoValidator, instance v1.Object, checkClientAuth bool) error {
	name := instance.GetName()

	err := cryptoValidator.CheckEcertCrypto(instance, name)
	if err != nil {
		if validator.CheckError(err) {
			return errors.Wrap(err, "missing ecert crypto")
		}
	}

	err = cryptoValidator.CheckTLSCrypto(instance, name)
	if err != nil {
		if validator.CheckError(err) {
			log.Info(fmt.Sprintf("missing TLS crypto: %s", err.Error()))
			return errors.Wrap(err, "missing TLS crypto")
		}
	}

	if checkClientAuth {
		err := cryptoValidator.CheckClientAuthCrypto(instance, name)
		if validator.CheckError(err) {
			log.Info(fmt.Sprintf("missing Client Auth crypto: %s", err.Error()))
			return errors.Wrap(err, "missing Client Auth crypto")
		}
	}

	return nil
}

func GetAdminCertsFromSecret(client k8sclient.Client, instance v1.Object) map[string][]byte {
	prefix := "ecert-" + instance.GetName()
	namespacedName := types.NamespacedName{
		Name:      prefix + "-admincerts",
		Namespace: instance.GetNamespace(),
	}

	certs := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, certs)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil
		}

		return map[string][]byte{}
	}

	return certs.Data
}

func GetAdminCertsFromSpec(spec *current.SecretSpec) []string {
	adminCerts := []string{}
	if spec != nil {
		if spec.MSP != nil {
			if spec.MSP.Component != nil {
				adminCerts = append(adminCerts, spec.MSP.Component.AdminCerts...)
			}
		} else if spec.Enrollment != nil {
			if spec.Enrollment.Component != nil {
				adminCerts = append(adminCerts, spec.Enrollment.Component.AdminCerts...)
			}
		}
	}

	return adminCerts
}

// Check for equality between two list of certificates. Order of certificates in the lists
// is ignored, if the two lists contain the same exact certificates this returns true
func CheckIfCertsDifferent(current map[string][]byte, updated []string) (bool, error) {
	// Only detect a difference if the list of updated certificates is not empty
	if len(current) != len(updated) && len(updated) > 0 {
		return true, nil
	}

	for _, newCert := range updated {
		certFound := false
		newCertBytes, err := util.Base64ToBytes(newCert)
		if err != nil {
			return false, err
		}

		for _, certBytes := range current {
			if bytes.Equal(certBytes, newCertBytes) {
				certFound = true
				break
			}
		}

		if !certFound {
			return true, nil
		}
	}

	return false, nil
}

func ConvertCertsToBytes(certs []string) ([][]byte, error) {
	certBytes := [][]byte{}
	for _, cert := range certs {
		bytes, err := util.Base64ToBytes(cert)
		if err != nil {
			return nil, err
		}
		certBytes = append(certBytes, bytes)
	}
	return certBytes, nil
}

func GetConfigFromConfigMap(client k8sclient.Client, instance v1.Object) (*corev1.ConfigMap, error) {
	name := fmt.Sprintf("%s-config", instance.GetName())
	log.Info(fmt.Sprintf("Get config map '%s'...", name))

	cm := &corev1.ConfigMap{}
	n := types.NamespacedName{
		Name:      name,
		Namespace: instance.GetNamespace(),
	}

	err := client.Get(context.TODO(), n, cm)
	if err != nil {
		return nil, err
	}

	return cm, nil

}
