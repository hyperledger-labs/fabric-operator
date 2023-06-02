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

package init

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	peerinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/validator"
	basepeer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	testDir        = "peer-init-test"
	testcert       = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNiekNDQWhhZ0F3SUJBZ0lVUE1MTUZ3cmMwZUV2ZlhWV3FEN0pCVnNrdVQ4d0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEl3TVRBd09ERTNNelF3TUZvWERUSTFNVEF3TnpFM016UXdNRm93YnpFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1TQXdIZ1lEVlFRREV4ZFRZV0ZrY3kxTllXTkNiMjlyCkxWQnlieTVzYjJOaGJEQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJLcHdXMTNsY2hBbXBuVlUKbWZXUi9TYXR5b3hSYkpZL1ZtZDQ3RlZtVFRRelA2b3phczlrdzdZZFU4cHV1U0JSWlV5c2paS29nNlpJaFAxaQpwcmt0VmlHamdaWXdnWk13RGdZRFZSMFBBUUgvQkFRREFnT29NQjBHQTFVZEpRUVdNQlFHQ0NzR0FRVUZCd01CCkJnZ3JCZ0VGQlFjREFqQU1CZ05WSFJNQkFmOEVBakFBTUIwR0ExVWREZ1FXQkJRQVJWTlVRU0dCVEJvbmhTa3gKSDNVK3VtYlg5akFmQmdOVkhTTUVHREFXZ0JSWkdVRktPNk9qL2NXY29vUFVxM1p1blBUeWpqQVVCZ05WSFJFRQpEVEFMZ2dsc2IyTmhiR2h2YzNRd0NnWUlLb1pJemowRUF3SURSd0F3UkFJZ2ExZk9Od3VicWFlVWlPNGdhVjZICld1QW9TQ1haU2NTNWNkWEo1WUJER2djQ0lGNUNPQVNzekZJbEJBSTJ1VnltaHVhWnlyVFJIVEZHUzJ5OHBPMWcKSG5VNgotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	defaultConfigs = "../../defaultconfig"
)

var _ = Describe("Peer init", func() {
	var (
		err      error
		peerInit *initializer.Initializer
		instance *current.IBPPeer
		peer     *basepeer.Peer
	)

	BeforeEach(func() {
		peer = &basepeer.Peer{
			Client:            client,
			Initializer:       peerInit,
			DeploymentManager: &mocks.DeploymentManager{},
			Config: &operatorconfig.Config{
				PeerInitConfig: &peerinit.Config{},
			},
		}

		config := &initializer.Config{
			OUFile:          filepath.Join(defaultConfigs, "peer/ouconfig.yaml"),
			CorePeerFile:    filepath.Join(defaultConfigs, "peer/core.yaml"),
			CorePeerV2File:  filepath.Join(defaultConfigs, "peer/v2/core.yaml"),
			CorePeerV25File: filepath.Join(defaultConfigs, "peer/v25/core.yaml"),
		}
		validator := &validator.Validator{
			Client: client,
		}

		peerInit = initializer.New(config, scheme, client, peer.GetLabels, validator, enroller.HSMEnrollJobTimeouts{})
		peer.Initializer = peerInit
	})

	Context("msp spec", func() {
		var (
			msp *current.MSP
		)

		BeforeEach(func() {
			msp = &current.MSP{
				KeyStore:          "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ3hRUXdSVFFpVUcwREo1UHoKQTJSclhIUEtCelkxMkxRa0MvbVlveWo1bEhDaFJBTkNBQVN5bE1YLzFqdDlmUGt1RTZ0anpvSTlQbGt4LzZuVQpCMHIvMU56TTdrYnBjUk8zQ3RIeXQ2TXlQR21FOUZUN29pYXphU3J1TW9JTDM0VGdBdUpIOU9ZWQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==",
				SignCerts:         testcert,
				AdminCerts:        []string{testcert},
				CACerts:           []string{testcert},
				IntermediateCerts: []string{testcert},
			}

			instance = &current.IBPPeer{
				Spec: current.IBPPeerSpec{
					Secret: &current.SecretSpec{
						MSP: &current.MSPSpec{
							Component: msp,
							TLS:       msp,
						},
					},
					DisableNodeOU: pointer.Bool(true),
				},
			}
			instance.Namespace = namespace
			instance.Name = "testpeer2"

			err := client.Create(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())

		})

		Context("parses peer msp", func() {
			BeforeEach(func() {
				peerinit, err := peerInit.GetInitPeer(instance, "")
				Expect(err).NotTo(HaveOccurred())
				peerinit.Config = &config.Core{}

				err = peer.InitializeCreate(instance, peerinit)
				Expect(err).NotTo(HaveOccurred())
			})

			It("gets ecert crypto", func() {
				By("creating a secret containing admin certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer2-admincerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["admincert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing ca root certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer2-cacerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cacert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing ca intermediate certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer2-intercerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["intercert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing signed cert", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer2-signcert", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cert.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing private key", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer2-keystore", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					keyBytes := secret.Data["key.pem"]
					VerifyKeyData(keyBytes)
				})

				By("creating a secret containing TLS ca root certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testpeer2-cacerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cacert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS ca intermediate certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testpeer2-intercerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["intercert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS signed cert", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testpeer2-signcert", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cert.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS private key", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testpeer2-keystore", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					keyBytes := secret.Data["key.pem"]
					VerifyKeyData(keyBytes)
				})
			})
		})
	})

	Context("enrollment spec", func() {
		var (
			enrollment *current.Enrollment
		)

		BeforeEach(func() {
			enrollment = &current.Enrollment{
				CAHost:       "localhost",
				CAPort:       "7055",
				EnrollID:     "admin",
				EnrollSecret: "adminpw",
				AdminCerts:   []string{testcert},
				CATLS: &current.CATLS{
					CACert: testcert,
				},
			}

			instance = &current.IBPPeer{
				Spec: current.IBPPeerSpec{
					Secret: &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{
							Component: enrollment,
							TLS:       enrollment,
						},
					},
					DisableNodeOU: pointer.Bool(true),
				},
			}
			instance.Namespace = namespace
			instance.Name = "testpeer1"

			err := client.Create(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = os.RemoveAll(testDir)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("enrolls peer with fabric ca server", func() {
			BeforeEach(func() {
				peerinit, err := peerInit.GetInitPeer(instance, testDir)
				Expect(err).NotTo(HaveOccurred())
				peerinit.Config = &config.Core{}

				err = peer.InitializeCreate(instance, peerinit)
				Expect(err).NotTo(HaveOccurred())
			})

			It("gets enrollment crypto", func() {
				By("creating a secret containing admin certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer1-admincerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["admincert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing ca root certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer1-cacerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cacert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing ca intermediate certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer1-intercerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["intercert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing signed cert", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer1-signcert", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cert.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing private key", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testpeer1-keystore", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					keyBytes := secret.Data["key.pem"]
					VerifyKeyData(keyBytes)
				})

				By("creating a secret containing TLS ca root certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testpeer1-cacerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cacert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS ca intermediate certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testpeer1-intercerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["intercert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS signed cert", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testpeer1-signcert", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cert.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS private key", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testpeer1-keystore", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					keyBytes := secret.Data["key.pem"]
					VerifyKeyData(keyBytes)
				})
			})
		})
	})
})

func VerifyKeyData(data []byte) {
	block, _ := pem.Decode(data)
	Expect(block).NotTo(BeNil())
	_, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	Expect(err).NotTo(HaveOccurred())
}

func VerifyCertData(data []byte) {
	block, _ := pem.Decode(data)
	Expect(block).NotTo(BeNil())
	_, err := x509.ParseCertificate(block.Bytes)
	Expect(err).NotTo(HaveOccurred())
}
