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
	"os"
	"path/filepath"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/secretmanager"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer"
	ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v1"
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDirOrderer = "orderer-init-test"
)

var _ = Describe("Orderer init", func() {
	var (
		err         error
		ordererInit *initializer.Initializer
		instance    *current.IBPOrderer
		orderer     *baseorderer.Node
	)

	BeforeEach(func() {
		ordererInit = &initializer.Initializer{
			Config: &initializer.Config{
				OUFile: filepath.Join(defaultConfigs, "orderer/ouconfig.yaml"),
			},
			Client: client,
			Scheme: scheme,
		}
		ordererInit.SecretManager = secretmanager.New(client, scheme, ordererInit.GetLabels)

		orderer = &baseorderer.Node{
			Client:            client,
			Initializer:       ordererInit,
			DeploymentManager: &mocks.DeploymentManager{},
		}

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

			instance = &current.IBPOrderer{
				Spec: current.IBPOrdererSpec{
					Secret: &current.SecretSpec{
						MSP: &current.MSPSpec{
							Component: msp,
							TLS:       msp,
						},
					},
					DisableNodeOU: &current.BoolTrue,
				},
			}
			instance.Namespace = namespace
			instance.Name = "testorderer2node0"

			err := client.Create(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("parses orderer msp", func() {
			BeforeEach(func() {
				ordererinit, err := ordererInit.GetInitOrderer(instance, "")
				Expect(err).NotTo(HaveOccurred())

				oconfig, err := ordererconfig.ReadOrdererFile("../../defaultconfig/orderer/orderer.yaml")
				Expect(err).NotTo(HaveOccurred())

				ordererinit.Config = oconfig

				err = orderer.InitializeCreate(instance, ordererinit)
				Expect(err).NotTo(HaveOccurred())
			})

			It("gets ecert crypto", func() {
				By("creating a secret containing admin certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testorderer2node0-admincerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["admincert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing ca root certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testorderer2node0-cacerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cacert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing ca intermediate certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testorderer2node0-intercerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["intercert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing signed cert", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testorderer2node0-signcert", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cert.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing private key", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testorderer2node0-keystore", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					keyBytes := secret.Data["key.pem"]
					VerifyKeyData(keyBytes)
				})

				By("creating a secret containing TLS ca root certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testorderer2node0-cacerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cacert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS ca intermediate certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testorderer2node0-intercerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["intercert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS signed cert", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testorderer2node0-signcert", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cert.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS private key", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testorderer2node0-keystore", metav1.GetOptions{})
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

			instance = &current.IBPOrderer{
				Spec: current.IBPOrdererSpec{
					Secret: &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{
							Component: enrollment,
							TLS:       enrollment,
						},
					},
					DisableNodeOU: &current.BoolTrue,
				},
			}
			instance.Namespace = namespace
			instance.Name = "testorderer1node0"

			err := client.Create(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = os.RemoveAll(testDirOrderer)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("enrolls orderer with fabric ca server", func() {
			BeforeEach(func() {
				ordererinit, err := ordererInit.GetInitOrderer(instance, testDirOrderer)
				Expect(err).NotTo(HaveOccurred())

				oconfig, err := ordererconfig.ReadOrdererFile("../../defaultconfig/orderer/orderer.yaml")
				Expect(err).NotTo(HaveOccurred())

				ordererinit.Config = oconfig

				err = orderer.InitializeCreate(instance, ordererinit)
				Expect(err).NotTo(HaveOccurred())
			})

			It("gets enrollment crypto", func() {
				By("creating a secret containing ca root certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testorderer1node0-cacerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cacert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing ca intermediate certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testorderer1node0-intercerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["intercert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing signed cert", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testorderer1node0-signcert", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cert.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing private key", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-testorderer1node0-keystore", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					keyBytes := secret.Data["key.pem"]
					VerifyKeyData(keyBytes)
				})

				By("creating a secret containing TLS ca root certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testorderer1node0-cacerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cacert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS ca intermediate certs", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testorderer1node0-intercerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["intercert-0.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS signed cert", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testorderer1node0-signcert", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					certBytes := secret.Data["cert.pem"]
					VerifyCertData(certBytes)
				})

				By("creating a secret containing TLS private key", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-testorderer1node0-keystore", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(secret.Data)).To(Equal(1))
					keyBytes := secret.Data["key.pem"]
					VerifyKeyData(keyBytes)
				})
			})
		})
	})
})
