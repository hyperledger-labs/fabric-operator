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
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GetTLSSignCertEncoded(client k8sclient.Client, instance v1.Object) (string, error) {
	return getSignCertEncoded("tls", client, instance)
}

func GetTLSKeystoreEncoded(client k8sclient.Client, instance v1.Object) (string, error) {
	return getKeystoreEncoded("tls", client, instance)
}

func GetTLSCACertEncoded(client k8sclient.Client, instance v1.Object) ([]string, error) {
	return getCACertEncoded("tls", client, instance)
}

func GetEcertSignCertEncoded(client k8sclient.Client, instance v1.Object) (string, error) {
	return getSignCertEncoded("ecert", client, instance)
}

func GetEcertKeystoreEncoded(client k8sclient.Client, instance v1.Object) (string, error) {
	return getKeystoreEncoded("ecert", client, instance)
}

func GetEcertCACertEncoded(client k8sclient.Client, instance v1.Object) ([]string, error) {
	return getCACertEncoded("ecert", client, instance)
}

func GetEcertAdmincertEncoded(client k8sclient.Client, instance v1.Object) ([]string, error) {
	return getAdmincertEncoded("ecert", client, instance)
}

func GetEcertIntercertEncoded(client k8sclient.Client, instance v1.Object) ([]string, error) {
	return getIntermediateCertEncoded("ecert", client, instance)
}

func GetTLSIntercertEncoded(client k8sclient.Client, instance v1.Object) ([]string, error) {
	return getIntermediateCertEncoded("tls", client, instance)
}

func getSignCertBytes(prefix common.SecretType, client k8sclient.Client, instance v1.Object) ([]byte, error) {
	secretName := fmt.Sprintf("%s-%s-signcert", prefix, instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      secretName,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data) == 0 {
		return nil, fmt.Errorf("%s signcert secret is blank", prefix)
	}

	if secret.Data["cert.pem"] != nil {
		return secret.Data["cert.pem"], nil
	}

	return nil, fmt.Errorf("cannot get %s signcert", prefix)
}

func getKeystoreBytes(prefix common.SecretType, client k8sclient.Client, instance v1.Object) ([]byte, error) {
	secretName := fmt.Sprintf("%s-%s-keystore", prefix, instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      secretName,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data) == 0 {
		return nil, fmt.Errorf("%s keystore secret is blank", prefix)
	}

	if secret.Data["key.pem"] != nil {
		return secret.Data["key.pem"], nil
	}

	return nil, fmt.Errorf("cannot get %s keystore", prefix)
}

func getCACertBytes(prefix common.SecretType, client k8sclient.Client, instance v1.Object) ([][]byte, error) {
	secretName := fmt.Sprintf("%s-%s-cacerts", prefix, instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      secretName,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data) == 0 {
		return nil, fmt.Errorf("%s cacert secret is blank", prefix)
	}

	var certs [][]byte
	for _, cert := range secret.Data {
		if cert != nil {
			certs = append(certs, cert)
		}
	}

	return certs, nil
}

func getAdmincertBytes(prefix common.SecretType, client k8sclient.Client, instance v1.Object) ([][]byte, error) {
	secretName := fmt.Sprintf("%s-%s-admincerts", prefix, instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      secretName,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		// if admincert secret is not found, admincerts dont exist
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if secret.Data == nil || len(secret.Data) == 0 {
		// do not throw error
		return nil, nil // errors.New("Ecert admincert secret is blank")
	}

	var certs [][]byte
	for _, cert := range secret.Data {
		if cert != nil {
			certs = append(certs, cert)
		}
	}

	return certs, nil
}

func getIntermediateCertBytes(prefix common.SecretType, client k8sclient.Client, instance v1.Object) ([][]byte, error) {
	secretName := fmt.Sprintf("%s-%s-intercerts", prefix, instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      secretName,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		// if intercert secret is not found, intercerts dont exist
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if secret.Data == nil || len(secret.Data) == 0 {
		// do not throw error
		return nil, nil
	}

	var certs [][]byte
	for _, cert := range secret.Data {
		if cert != nil {
			certs = append(certs, cert)
		}
	}

	return certs, nil
}

func getSignCertEncoded(prefix common.SecretType, client k8sclient.Client, instance v1.Object) (string, error) {
	certBytes, err := getSignCertBytes(prefix, client, instance)
	if err != nil {
		return "", err
	}

	cert := base64.StdEncoding.EncodeToString(certBytes)
	return cert, nil
}

func getKeystoreEncoded(prefix common.SecretType, client k8sclient.Client, instance v1.Object) (string, error) {
	keyBytes, err := getKeystoreBytes(prefix, client, instance)
	if err != nil {
		return "", err
	}

	cert := base64.StdEncoding.EncodeToString(keyBytes)
	return cert, nil
}

func getCACertEncoded(prefix common.SecretType, client k8sclient.Client, instance v1.Object) ([]string, error) {
	certBytes, err := getCACertBytes(prefix, client, instance)
	if err != nil {
		return nil, err
	}

	var certs []string
	for _, certByte := range certBytes {
		cert := base64.StdEncoding.EncodeToString(certByte)
		certs = append(certs, cert)
	}
	return certs, nil
}

func getAdmincertEncoded(prefix common.SecretType, client k8sclient.Client, instance v1.Object) ([]string, error) {
	certBytes, err := getAdmincertBytes(prefix, client, instance)
	if err != nil {
		return nil, err
	}

	var certs []string
	for _, certByte := range certBytes {
		cert := base64.StdEncoding.EncodeToString(certByte)
		certs = append(certs, cert)
	}
	return certs, nil
}

func getIntermediateCertEncoded(prefix common.SecretType, client k8sclient.Client, instance v1.Object) ([]string, error) {
	certBytes, err := getIntermediateCertBytes(prefix, client, instance)
	if err != nil {
		return nil, err
	}

	var certs []string
	for _, certByte := range certBytes {
		cert := base64.StdEncoding.EncodeToString(certByte)
		certs = append(certs, cert)
	}
	return certs, nil
}

type CACryptoBytes struct {
	Cert           []byte
	Key            []byte
	OperationsCert []byte
	OperationsKey  []byte
	TLSCert        []byte
	TLSKey         []byte
}

func GetCACryptoBytes(client k8sclient.Client, instance v1.Object) (*CACryptoBytes, error) {
	secretName := fmt.Sprintf("%s-ca-crypto", instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      secretName,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data) == 0 {
		return nil, errors.New("CA crypto secret is blank")
	}

	if secret.Data["tls-cert.pem"] == nil {
		return nil, errors.New("cannot get tlscert")
	}

	return &CACryptoBytes{
		TLSCert:        secret.Data["tls-cert.pem"],
		TLSKey:         secret.Data["tls-key.pem"],
		Cert:           secret.Data["cert.pem"],
		Key:            secret.Data["key.pem"],
		OperationsCert: secret.Data["operations-cert.pem"],
		OperationsKey:  secret.Data["operations-key.pem"],
	}, nil
}

func GetTLSCACryptoBytes(client k8sclient.Client, instance v1.Object) (*CACryptoBytes, error) {
	secretName := fmt.Sprintf("%s-tlsca-crypto", instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      secretName,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data) == 0 {
		return nil, errors.New("TLSCA crypto secret is blank")
	}
	if secret.Data["cert.pem"] == nil {
		return nil, errors.New("cannot get root TLSCA cert")
	}
	return &CACryptoBytes{
		Cert: secret.Data["cert.pem"],
		Key:  secret.Data["key.pem"],
	}, nil
}

type CACryptoEncoded struct {
	Cert           string
	Key            string
	OperationsCert string
	OperationsKey  string
	TLSCert        string
	TLSKey         string
}

func GetCACryptoEncoded(client k8sclient.Client, instance v1.Object) (*CACryptoEncoded, error) {
	bytes, err := GetCACryptoBytes(client, instance)
	if err != nil {
		return nil, err
	}

	encoded := &CACryptoEncoded{}
	encoded.Cert = base64.StdEncoding.EncodeToString(bytes.Cert)
	encoded.Key = base64.StdEncoding.EncodeToString(bytes.Key)
	encoded.OperationsCert = base64.StdEncoding.EncodeToString(bytes.OperationsCert)
	encoded.OperationsKey = base64.StdEncoding.EncodeToString(bytes.OperationsKey)
	encoded.TLSCert = base64.StdEncoding.EncodeToString(bytes.TLSCert)
	encoded.TLSKey = base64.StdEncoding.EncodeToString(bytes.TLSKey)

	return encoded, err
}

func GetTLSCACryptoEncoded(client k8sclient.Client, instance v1.Object) (*CACryptoEncoded, error) {
	bytes, err := GetTLSCACryptoBytes(client, instance)
	if err != nil {
		return nil, err
	}

	encoded := &CACryptoEncoded{}
	encoded.Cert = base64.StdEncoding.EncodeToString(bytes.Cert)
	encoded.Key = base64.StdEncoding.EncodeToString(bytes.Key)

	return encoded, err
}
