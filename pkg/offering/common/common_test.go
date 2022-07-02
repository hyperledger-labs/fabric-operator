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

package common_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
)

var _ = Describe("Common", func() {

	const testcert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo"

	var (
		mockKubeClient *mocks.Client
		instance       *current.IBPPeer

		crypto1 *current.MSP
		crypto2 *current.MSP
		crypto3 *current.MSP

		encodedtestcert string

		backupData map[string][]byte
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{}

		instance = &current.IBPPeer{}
		instance.Name = "peer1"

		crypto1 = &current.MSP{SignCerts: "signcert1"}
		crypto2 = &current.MSP{SignCerts: "signcert2"}
		crypto3 = &current.MSP{SignCerts: "signcert3"}

		backup := &common.Backup{
			List:      []*current.MSP{crypto1},
			Timestamp: time.Now().String(),
		}
		backupBytes, err := json.Marshal(backup)
		Expect(err).NotTo(HaveOccurred())

		backupData = map[string][]byte{
			"tls-backup.json":   backupBytes,
			"ecert-backup.json": backupBytes,
		}

		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			o := obj.(*corev1.Secret)
			switch types.Name {
			case "tls-" + instance.Name + "-signcert":
				o.Name = "tls-" + instance.Name + "-signcert"
				o.Namespace = instance.Namespace
				o.Data = map[string][]byte{"cert.pem": []byte(testcert)}
			case "tls-" + instance.Name + "-keystore":
				o.Name = "tls-" + instance.Name + "-keystore"
				o.Namespace = instance.Namespace
				o.Data = map[string][]byte{"key.pem": []byte(testcert)}
			case "tls-" + instance.Name + "-cacerts":
				o.Name = "tls-" + instance.Name + "-cacerts"
				o.Namespace = instance.Namespace
				o.Data = map[string][]byte{"key.pem": []byte(testcert)}
			case "ecert-" + instance.Name + "-signcert":
				o.Name = "ecert-" + instance.Name + "-signcert"
				o.Namespace = instance.Namespace
				o.Data = map[string][]byte{"cert.pem": []byte(testcert)}
			case "ecert-" + instance.Name + "-cacerts":
				o.Name = "ecert-" + instance.Name + "-cacerts"
				o.Namespace = instance.Namespace
				o.Data = map[string][]byte{"cacert-0.pem": []byte(testcert)}
			case "peer1-crypto-backup":
				o.Name = instance.Name + "-crypto-backup"
				o.Namespace = instance.Namespace
				o.Data = backupData
			case "ca1-ca-crypto":
				o.Name = instance.Name + "-ca-crypto"
				o.Namespace = instance.Namespace
				o.Data = map[string][]byte{
					"tls-cert.pem":        []byte(testcert),
					"tls-key.pem":         []byte(testcert),
					"cert.pem":            []byte(testcert),
					"key.pem":             []byte(testcert),
					"operations-cert.pem": []byte(testcert),
					"operations-key.pem":  []byte(testcert),
				}
			case "ca1-crypto-backup":
				o.Name = "ca1-crypto-backup"
				o.Namespace = instance.Namespace
				o.Data = backupData
			}
			return nil
		}

		encodedtestcert = base64.StdEncoding.EncodeToString([]byte(testcert))
	})

	Context("backup crypto", func() {

		Context("get crypto", func() {
			It("returns nil if fails to get secret", func() {
				mockKubeClient.GetReturns(errors.New("get error"))
				crypto := common.GetCrypto("tls", mockKubeClient, instance)
				Expect(crypto).To(BeNil())
			})

			It("returns nil if no secrets are found", func() {
				mockKubeClient.GetReturns(k8serrors.NewNotFound(schema.GroupResource{}, "not found"))
				crypto := common.GetCrypto("tls", mockKubeClient, instance)
				Expect(crypto).To(BeNil())
			})

			It("returns tls crypto", func() {
				crypto := common.GetCrypto("tls", mockKubeClient, instance)
				Expect(crypto).NotTo(BeNil())
				Expect(crypto).To(Equal(&current.MSP{
					SignCerts: encodedtestcert,
					KeyStore:  encodedtestcert,
					CACerts:   []string{encodedtestcert},
				}))
			})

			It("returns ecert crypto", func() {
				crypto := common.GetCrypto("ecert", mockKubeClient, instance)
				Expect(crypto).NotTo(BeNil())
				Expect(crypto).To(Equal(&current.MSP{
					SignCerts: encodedtestcert,
					CACerts:   []string{encodedtestcert},
				}))
			})
		})

		Context("udpate secret data", func() {
			var (
				data map[string][]byte
			)

			BeforeEach(func() {

				backup := &common.Backup{
					List:      []*current.MSP{crypto1},
					Timestamp: time.Now().String(),
				}
				backupBytes, err := json.Marshal(backup)
				Expect(err).NotTo(HaveOccurred())

				data = map[string][]byte{
					"tls-backup.json":   backupBytes,
					"ecert-backup.json": backupBytes,
				}
			})

			It("adds crypto to backup list", func() {
				crypto := &common.Crypto{
					TLS:   crypto2,
					Ecert: crypto2,
				}
				updatedData, err := common.UpdateBackupSecretData(data, crypto)
				Expect(err).NotTo(HaveOccurred())

				By("updating tls backup", func() {
					tlsbackup := &common.Backup{}
					err = json.Unmarshal(updatedData["tls-backup.json"], tlsbackup)
					Expect(err).NotTo(HaveOccurred())
					Expect(tlsbackup.List).To(Equal([]*current.MSP{crypto1, crypto2}))
					Expect(tlsbackup.Timestamp).NotTo(Equal(""))
				})

				By("updating ecert backup", func() {
					ecertbackup := &common.Backup{}
					err = json.Unmarshal(updatedData["ecert-backup.json"], ecertbackup)
					Expect(err).NotTo(HaveOccurred())
					Expect(ecertbackup.List).To(Equal([]*current.MSP{crypto1, crypto2}))
					Expect(ecertbackup.Timestamp).NotTo(Equal(""))
				})
			})

			It("removes oldest crypto from queue and inserts new crypto if list is longer than 10", func() {
				backup := &common.Backup{
					List:      []*current.MSP{crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto2},
					Timestamp: time.Now().String(),
				}
				backupBytes, err := json.Marshal(backup)
				Expect(err).NotTo(HaveOccurred())

				data = map[string][]byte{
					"tls-backup.json":   backupBytes,
					"ecert-backup.json": backupBytes,
				}

				crypto := &common.Crypto{
					TLS:   crypto3,
					Ecert: crypto3,
				}
				updatedData, err := common.UpdateBackupSecretData(data, crypto)
				Expect(err).NotTo(HaveOccurred())

				By("updating tls backup to contain 10 most recent backups", func() {
					tlsbackup := &common.Backup{}
					err = json.Unmarshal(updatedData["tls-backup.json"], tlsbackup)
					Expect(err).NotTo(HaveOccurred())
					Expect(tlsbackup.List).To(Equal([]*current.MSP{crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto2, crypto3}))
					Expect(tlsbackup.Timestamp).NotTo(Equal(""))
				})

				By("updating ecert backup to contain 10 most recent backups", func() {
					ecertbackup := &common.Backup{}
					err = json.Unmarshal(updatedData["ecert-backup.json"], ecertbackup)
					Expect(err).NotTo(HaveOccurred())
					Expect(ecertbackup.List).To(Equal([]*current.MSP{crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto1, crypto2, crypto3}))
					Expect(ecertbackup.Timestamp).NotTo(Equal(""))
				})
			})
		})

		Context("backup crypto", func() {
			It("returns nil if neither TLS nor ecert crypto is found", func() {
				mockKubeClient.GetReturns(errors.New("get error"))
				err := common.BackupCrypto(mockKubeClient, &runtime.Scheme{}, instance, map[string]string{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns error if fails to update backup secret", func() {
				mockKubeClient.UpdateReturns(errors.New("create or update error"))
				err := common.BackupCrypto(mockKubeClient, &runtime.Scheme{}, instance, map[string]string{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to update backup secret: create or update error"))
			})

			It("updates backup secret if one exists for instance", func() {
				err := common.BackupCrypto(mockKubeClient, &runtime.Scheme{}, instance, map[string]string{})
				Expect(err).NotTo(HaveOccurred())
				Expect(mockKubeClient.UpdateCallCount()).To(Equal(1))

				newCrypto := &current.MSP{
					SignCerts: encodedtestcert,
					KeyStore:  encodedtestcert,
					CACerts:   []string{encodedtestcert},
				}

				By("updating tls backup", func() {
					tlsbackup := &common.Backup{}
					err = json.Unmarshal(backupData["tls-backup.json"], tlsbackup)
					Expect(err).NotTo(HaveOccurred())
					Expect(tlsbackup.List).To(Equal([]*current.MSP{crypto1, newCrypto}))
					Expect(tlsbackup.Timestamp).NotTo(Equal(""))
				})

				By("updating ecert backup", func() {
					newCrypto.KeyStore = ""
					ecertbackup := &common.Backup{}
					err = json.Unmarshal(backupData["ecert-backup.json"], ecertbackup)
					Expect(err).NotTo(HaveOccurred())
					Expect(ecertbackup.List).To(Equal([]*current.MSP{crypto1, newCrypto}))
					Expect(ecertbackup.Timestamp).NotTo(Equal(""))
				})
			})
		})

		Context("backup CA crypto", func() {
			var (
				instance *current.IBPCA
			)

			BeforeEach(func() {
				instance = &current.IBPCA{}
				instance.Name = "ca1"
			})

			It("returns nil if CA TLS crypto is not found", func() {
				mockKubeClient.GetReturns(errors.New("get error"))
				err := common.BackupCACrypto(mockKubeClient, &runtime.Scheme{}, instance, map[string]string{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("updates backup secret if one exists for instance", func() {
				err := common.BackupCACrypto(mockKubeClient, &runtime.Scheme{}, instance, map[string]string{})
				Expect(err).NotTo(HaveOccurred())
				Expect(mockKubeClient.UpdateCallCount()).To(Equal(1))

				newCrypto := &current.MSP{
					SignCerts: encodedtestcert,
					KeyStore:  encodedtestcert,
				}

				By("updating tls backup", func() {
					tlsbackup := &common.Backup{}
					err = json.Unmarshal(backupData["tls-backup.json"], tlsbackup)
					Expect(err).NotTo(HaveOccurred())
					Expect(tlsbackup.List).To(Equal([]*current.MSP{crypto1, newCrypto}))
					Expect(tlsbackup.Timestamp).NotTo(Equal(""))
				})

				By("creating operations backup", func() {
					opbackup := &common.Backup{}
					err = json.Unmarshal(backupData["operations-backup.json"], opbackup)
					Expect(err).NotTo(HaveOccurred())
					Expect(opbackup.List).To(Equal([]*current.MSP{newCrypto}))
					Expect(opbackup.Timestamp).NotTo(Equal(""))
				})

				By("creating ca backup", func() {
					cabackup := &common.Backup{}
					err = json.Unmarshal(backupData["ca-backup.json"], cabackup)
					Expect(err).NotTo(HaveOccurred())
					Expect(cabackup.List).To(Equal([]*current.MSP{newCrypto}))
					Expect(cabackup.Timestamp).NotTo(Equal(""))
				})
			})
		})

	})

})
