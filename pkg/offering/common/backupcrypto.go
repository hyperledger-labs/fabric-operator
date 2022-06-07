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
	"encoding/json"
	"fmt"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("backup_crypto")

// Number of iterations we are storing
const ITERATIONS = 10

type Backup struct {
	List      []*current.MSP `json:"list"`
	Timestamp string         `json:"timestamp"`
}

type Crypto struct {
	TLS        *current.MSP
	Ecert      *current.MSP
	Operations *current.MSP
	CA         *current.MSP
}

func BackupCrypto(client k8sclient.Client, scheme *runtime.Scheme, instance v1.Object, labels map[string]string) error {
	tlsCrypto := GetCrypto("tls", client, instance)
	ecertCrypto := GetCrypto("ecert", client, instance)

	if tlsCrypto == nil && ecertCrypto == nil {
		// No backup required if crypto doesn't exist/no found
		log.Info(fmt.Sprintf("No TLS or ecert crypto found for %s, not performing backup", instance.GetName()))
		return nil
	}

	crypto := &Crypto{
		TLS:   tlsCrypto,
		Ecert: ecertCrypto,
	}

	return backupCrypto(client, scheme, instance, labels, crypto)
}

func BackupCACrypto(client k8sclient.Client, scheme *runtime.Scheme, instance v1.Object, labels map[string]string) error {
	caCrypto, operationsCrypto, tlsCrypto := GetCACrypto(client, instance)

	if caCrypto == nil && operationsCrypto == nil && tlsCrypto == nil {
		log.Info(fmt.Sprintf("No crypto found for %s, not performing backup", instance.GetName()))
		return nil
	}

	crypto := &Crypto{
		CA:         caCrypto,
		Operations: operationsCrypto,
		TLS:        tlsCrypto,
	}
	return backupCrypto(client, scheme, instance, labels, crypto)
}

func backupCrypto(client k8sclient.Client, scheme *runtime.Scheme, instance v1.Object, labels map[string]string, crypto *Crypto) error {
	backupSecret, err := GetBackupSecret(client, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create secret
			data, err := CreateBackupSecretData(crypto)
			if err != nil {
				return errors.Wrap(err, "failed to create backup secret data")
			}

			newSecret := &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      fmt.Sprintf("%s-crypto-backup", instance.GetName()),
					Namespace: instance.GetNamespace(),
					Labels:    labels,
				},
				Data: data,
				Type: corev1.SecretTypeOpaque,
			}

			err = CreateBackupSecret(client, scheme, instance, newSecret)
			if err != nil {
				return errors.Wrap(err, "failed to create backup secret")
			}
			return nil
		}
		return errors.Wrap(err, "failed to get backup secret")
	}

	// Update secret
	data, err := UpdateBackupSecretData(backupSecret.Data, crypto)
	if err != nil {
		return errors.Wrap(err, "failed to update backup secret data")
	}
	backupSecret.Data = data

	err = UpdateBackupSecret(client, scheme, instance, backupSecret)
	if err != nil {
		return errors.Wrap(err, "failed to update backup secret")
	}

	return nil
}

func CreateBackupSecretData(crypto *Crypto) (map[string][]byte, error) {
	data := map[string][]byte{}

	if crypto.TLS != nil {
		tlsBackup := &Backup{
			List:      []*current.MSP{crypto.TLS},
			Timestamp: time.Now().String(),
		}
		tlsBytes, err := json.Marshal(tlsBackup)
		if err != nil {
			return nil, err
		}
		data["tls-backup.json"] = tlsBytes
	}

	if crypto.Ecert != nil {
		ecertBackup := &Backup{
			List:      []*current.MSP{crypto.Ecert},
			Timestamp: time.Now().String(),
		}
		ecertBytes, err := json.Marshal(ecertBackup)
		if err != nil {
			return nil, err
		}
		data["ecert-backup.json"] = ecertBytes
	}

	if crypto.Operations != nil {
		opBackup := &Backup{
			List:      []*current.MSP{crypto.Operations},
			Timestamp: time.Now().String(),
		}
		opBytes, err := json.Marshal(opBackup)
		if err != nil {
			return nil, err
		}
		data["operations-backup.json"] = opBytes
	}

	if crypto.CA != nil {
		caBackup := &Backup{
			List:      []*current.MSP{crypto.CA},
			Timestamp: time.Now().String(),
		}
		caBytes, err := json.Marshal(caBackup)
		if err != nil {
			return nil, err
		}
		data["ca-backup.json"] = caBytes
	}

	return data, nil
}

func UpdateBackupSecretData(data map[string][]byte, crypto *Crypto) (map[string][]byte, error) {
	if crypto.TLS != nil {
		tlsBackup, err := getUpdatedBackup(data["tls-backup.json"], crypto.TLS)
		if err != nil {
			return nil, err
		}
		tlsBytes, err := json.Marshal(tlsBackup)
		if err != nil {
			return nil, err
		}
		data["tls-backup.json"] = tlsBytes
	}

	if crypto.Ecert != nil {
		ecertBackup, err := getUpdatedBackup(data["ecert-backup.json"], crypto.Ecert)
		if err != nil {
			return nil, err
		}
		ecertBytes, err := json.Marshal(ecertBackup)
		if err != nil {
			return nil, err
		}
		data["ecert-backup.json"] = ecertBytes
	}

	if crypto.Operations != nil {
		opBackup, err := getUpdatedBackup(data["operations-backup.json"], crypto.Operations)
		if err != nil {
			return nil, err
		}
		opBytes, err := json.Marshal(opBackup)
		if err != nil {
			return nil, err
		}
		data["operations-backup.json"] = opBytes
	}

	if crypto.CA != nil {
		caBackup, err := getUpdatedBackup(data["ca-backup.json"], crypto.CA)
		if err != nil {
			return nil, err
		}
		caBytes, err := json.Marshal(caBackup)
		if err != nil {
			return nil, err
		}
		data["ca-backup.json"] = caBytes
	}

	return data, nil
}

func getUpdatedBackup(data []byte, crypto *current.MSP) (*Backup, error) {
	backup := &Backup{}
	if data != nil {
		err := json.Unmarshal(data, backup)
		if err != nil {
			return nil, err
		}

		if len(backup.List) < ITERATIONS {
			// Insert to back of queue
			backup.List = append(backup.List, crypto)
		} else {
			// Remove oldest backup and insert new crypto
			backup.List = append(backup.List[1:], crypto)
		}
	} else {
		// Create backup
		backup.List = []*current.MSP{crypto}
	}

	backup.Timestamp = time.Now().String()

	return backup, nil
}

func GetCrypto(prefix common.SecretType, client k8sclient.Client, instance v1.Object) *current.MSP {
	var cryptoExists bool

	// Doesn't return error if can't get secret/secret not found
	signcert, err := getSignCertEncoded(prefix, client, instance)
	if err == nil && signcert != "" {
		cryptoExists = true
	}

	keystore, err := getKeystoreEncoded(prefix, client, instance)
	if err == nil && keystore != "" {
		cryptoExists = true
	}

	cacerts, err := getCACertEncoded(prefix, client, instance)
	if err == nil && cacerts != nil {
		cryptoExists = true
	}

	admincerts, err := getAdmincertEncoded(prefix, client, instance)
	if err == nil && admincerts != nil {
		cryptoExists = true
	}

	intercerts, err := getIntermediateCertEncoded(prefix, client, instance)
	if err == nil && intercerts != nil {
		cryptoExists = true
	}

	if cryptoExists {
		return &current.MSP{
			SignCerts:         signcert,
			KeyStore:          keystore,
			CACerts:           cacerts,
			AdminCerts:        admincerts,
			IntermediateCerts: intercerts,
		}
	}

	return nil
}

func GetCACrypto(client k8sclient.Client, instance v1.Object) (*current.MSP, *current.MSP, *current.MSP) {
	encoded, err := GetCACryptoEncoded(client, instance)
	if err != nil || encoded == nil {
		return nil, nil, nil
	}

	caMSP := &current.MSP{
		SignCerts: encoded.Cert,
		KeyStore:  encoded.Key,
	}

	operationsMSP := &current.MSP{
		SignCerts: encoded.OperationsCert,
		KeyStore:  encoded.OperationsKey,
	}

	tlsMSP := &current.MSP{
		SignCerts: encoded.TLSCert,
		KeyStore:  encoded.TLSKey,
	}

	return caMSP, operationsMSP, tlsMSP
}

func GetBackupSecret(client k8sclient.Client, instance v1.Object) (*corev1.Secret, error) {
	secretName := fmt.Sprintf("%s-crypto-backup", instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      secretName,
		Namespace: instance.GetNamespace(),
	}

	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func CreateBackupSecret(client k8sclient.Client, scheme *runtime.Scheme, instance v1.Object, secret *corev1.Secret) error {
	err := client.Create(context.TODO(), secret, k8sclient.CreateOption{
		Owner:  instance,
		Scheme: scheme,
	})
	if err != nil {
		return err
	}
	return nil
}

func UpdateBackupSecret(client k8sclient.Client, scheme *runtime.Scheme, instance v1.Object, secret *corev1.Secret) error {
	err := client.Update(context.TODO(), secret, k8sclient.UpdateOption{
		Owner:  instance,
		Scheme: scheme,
	})
	if err != nil {
		return err
	}
	return nil
}
