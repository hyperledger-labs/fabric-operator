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

package orderer_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("trigger orderer actions", func() {
	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	var (
		node1 helper.Orderer
		node2 helper.Orderer
		node3 helper.Orderer

		podNameNode1 string
		podNameNode2 string
		podNameNode3 string

		ibpordererNode1 *current.IBPOrderer
	)

	BeforeEach(func() {
		node1 = orderer.Nodes[0]
		node2 = orderer.Nodes[1]
		node3 = orderer.Nodes[2]

		Eventually(node1.PodIsRunning, time.Second*60, time.Second*2).Should((Equal(true)))

		// NOTE: Need to keep same operator config for duration of test to ensure that the correct
		// reason string is passed into operator-restart-config CM.
		// integration.ClearOperatorConfig(kclient, namespace)

		Eventually(func() int { return len(node1.GetPods()) }).Should(Equal(1))
		Eventually(func() int { return len(node2.GetPods()) }).Should(Equal(1))
		Eventually(func() int { return len(node3.GetPods()) }).Should(Equal(1))

		podNameNode1 = node1.GetPods()[0].Name
		podNameNode2 = node2.GetPods()[0].Name
		podNameNode3 = node3.GetPods()[0].Name

		result := ibpCRClient.Get().Namespace(namespace).
			Resource(IBPORDERERS).
			Name(node1.Name).
			Do(context.TODO())
		Expect(result.Error()).NotTo(HaveOccurred())

		ibpordererNode1 = &current.IBPOrderer{}
		result.Into(ibpordererNode1)
	})

	Context("spec has restart flag set to true", func() {
		It("performs restart action", func() {
			patch := func(o client.Object) {
				ibporderer := o.(*current.IBPOrderer)
				ibporderer.Spec.Action.Restart = true
			}

			err := integration.ResilientPatch(ibpCRClient, node1.Name, namespace, IBPORDERERS, 3, &current.IBPOrderer{}, patch)
			Expect(err).NotTo(HaveOccurred())

			Eventually(node1.PodIsRunning).Should((Equal(true)))

			By("restarting orderer pods", func() {
				Eventually(func() bool {
					pods := node1.GetRunningPods()
					if len(pods) == 0 {
						return false
					}

					newPodName := pods[0].Name
					if newPodName != podNameNode1 {
						return true
					}

					return false
				}).Should(Equal(true))
			})

			By("setting restart flag back to false after restart", func() {
				Eventually(func() bool {
					result := ibpCRClient.Get().Namespace(namespace).Resource(IBPORDERERS).Name(node1.Name).Do(context.TODO())
					ibporderer := &current.IBPOrderer{}
					result.Into(ibporderer)

					return ibporderer.Spec.Action.Restart
				}).Should(Equal(false))
			})
		})
	})

	Context("spec has ecert reenroll flag set to true", func() {
		var (
			ecert, ekey []byte

			commonAssertions = func() {
				By("restarting orderer pods", func() {
					Eventually(func() bool {
						pods := node1.GetRunningPods()
						if len(pods) != 1 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName != podNameNode1 {
							return true
						}

						return false
					}).Should(Equal(true))
				})

				By("backing up old signcert", func() {
					backup := GetBackup("ecert", node1.Name)
					Expect(len(backup.List)).NotTo(Equal(0))
					Expect(backup.List[len(backup.List)-1].SignCerts).To(Equal(base64.StdEncoding.EncodeToString(ecert)))
				})

				By("updating ecert signcert secret", func() {
					updatedEcertSecret, err := kclient.CoreV1().Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", node1.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(bytes.Equal(ecert, updatedEcertSecret.Data["cert.pem"])).To(Equal(false))
				})
			}
		)

		BeforeEach(func() {
			ecertSecret, err := kclient.CoreV1().Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", node1.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			ecert = ecertSecret.Data["cert.pem"]

			ecertSecret, err = kclient.CoreV1().Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", node1.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			ekey = ecertSecret.Data["key.pem"]
		})

		When("requesting a new key", func() {
			It("gets a new key and certificate", func() {
				patch := func(o client.Object) {
					ibporderer := o.(*current.IBPOrderer)
					ibporderer.Spec.Action.Reenroll.EcertNewKey = true
				}

				err := integration.ResilientPatch(ibpCRClient,
					node1.Name,
					namespace,
					IBPORDERERS,
					3,
					&current.IBPOrderer{},
					patch)
				Expect(err).NotTo(HaveOccurred())

				commonAssertions()

				By("generating a new key", func() {
					updatedEcertKey, err := kclient.CoreV1().Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", node1.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(bytes.Equal(ekey, updatedEcertKey.Data["key.pem"])).To(Equal(false))
				})

				By("setting reenroll flag back to false after restart", func() {
					ibporderer := &current.IBPOrderer{}
					Eventually(func() bool {
						result := ibpCRClient.Get().Namespace(namespace).Resource(IBPORDERERS).
							Name(node1.Name).Do(context.TODO())
						result.Into(ibporderer)

						return ibporderer.Spec.Action.Reenroll.EcertNewKey
					}).Should(Equal(false))
				})
			})
		})

		When("reusing existing key", func() {
			It("gets a new certificate", func() {
				patch := func(o client.Object) {
					ibporderer := o.(*current.IBPOrderer)
					ibporderer.Spec.Action.Reenroll.Ecert = true
				}

				err := integration.ResilientPatch(ibpCRClient,
					node1.Name,
					namespace,
					IBPORDERERS,
					3,
					&current.IBPOrderer{},
					patch)
				Expect(err).NotTo(HaveOccurred())

				commonAssertions()

				By("not generating a new key", func() {
					updatedEcertKey, err := kclient.CoreV1().Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", node1.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(bytes.Equal(ekey, updatedEcertKey.Data["key.pem"])).To(Equal(true))
				})

				By("setting reenroll flag back to false after restart", func() {
					ibporderer := &current.IBPOrderer{}
					Eventually(func() bool {
						result := ibpCRClient.Get().Namespace(namespace).Resource(IBPORDERERS).
							Name(node1.Name).Do(context.TODO())
						result.Into(ibporderer)

						return ibporderer.Spec.Action.Reenroll.Ecert
					}).Should(Equal(false))
				})
			})
		})
	})

	Context("spec has ecert enroll flag set to true", func() {
		var (
			ecert    []byte
			ecertKey []byte
		)

		BeforeEach(func() {
			ecertSecret, err := kclient.CoreV1().
				Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", node1.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			ecertKeySecret, err := kclient.CoreV1().
				Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", node1.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			ecert = ecertSecret.Data["cert.pem"]
			ecertKey = ecertKeySecret.Data["key.pem"]
		})

		It("generates new crypto", func() {
			patch := func(o client.Object) {
				ibporderer := o.(*current.IBPOrderer)
				ibporderer.Spec.Action.Enroll.Ecert = true
			}

			err := integration.ResilientPatch(ibpCRClient, node1.Name, namespace, IBPORDERERS, 3, &current.IBPOrderer{}, patch)
			Expect(err).NotTo(HaveOccurred())

			By("backing up old crypto", func() {
				Eventually(func() bool {
					backup := GetBackup("ecert", node1.Name)
					Expect(len(backup.List)).NotTo(Equal(0))
					return backup.List[len(backup.List)-1].SignCerts == base64.StdEncoding.EncodeToString(ecert) &&
						backup.List[len(backup.List)-1].KeyStore == base64.StdEncoding.EncodeToString(ecertKey)
				}).Should(Equal(true))
			})

			By("updating ecert signcert secret", func() {
				Eventually(func() bool {
					updatedEcertSecret, err := kclient.CoreV1().
						Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", node1.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return bytes.Equal(ecert, updatedEcertSecret.Data["cert.pem"])
				}).Should(Equal(false))
			})

			By("updating ecert key secret", func() {
				Eventually(func() bool {
					updatedEcertSecret, err := kclient.CoreV1().
						Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", node1.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return bytes.Equal(ecertKey, updatedEcertSecret.Data["key.pem"])
				}).Should(Equal(false))
			})

			By("setting enroll flag back to false after restart", func() {
				Eventually(func() bool {
					result := ibpCRClient.Get().Namespace(namespace).Resource(IBPORDERERS).Name(node1.Name).Do(context.TODO())
					ibporderer := &current.IBPOrderer{}
					result.Into(ibporderer)

					return ibporderer.Spec.Action.Enroll.Ecert
				}).Should(Equal(false))
			})
		})
	})

	Context("spec has tlscert reenroll flag set to true", func() {
		var (
			tlsCert, tlsKey []byte

			commonAssertions = func() {
				By("restarting orderer pods", func() {
					Eventually(func() bool {
						pods := node2.GetRunningPods()
						if len(pods) != 1 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName != podNameNode2 {
							return true
						}

						return false
					}).Should(Equal(true))
				})

				By("updating tls signcert secret", func() {
					updatedTLSSecret, err := kclient.CoreV1().Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", node2.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(bytes.Equal(tlsCert, updatedTLSSecret.Data["cert.pem"])).To(Equal(false))
				})
			}
		)

		BeforeEach(func() {
			tlsSecret, err := kclient.CoreV1().Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", node2.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			tlsCert = tlsSecret.Data["cert.pem"]

			tlsSecret, err = kclient.CoreV1().Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("tls-%s-keystore", node2.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			tlsKey = tlsSecret.Data["key.pem"]
		})

		When("requesting a new key", func() {
			It("gets a new key and certificate", func() {
				patch := func(o client.Object) {
					ibporderer := o.(*current.IBPOrderer)
					ibporderer.Spec.Action.Reenroll.TLSCertNewKey = true
				}

				err := integration.ResilientPatch(
					ibpCRClient,
					node2.Name,
					namespace,
					IBPORDERERS,
					3,
					&current.IBPOrderer{},
					patch)
				Expect(err).NotTo(HaveOccurred())

				commonAssertions()

				By("generating a new key", func() {
					updatedEcertKey, err := kclient.CoreV1().Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("tls-%s-keystore", node2.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(bytes.Equal(tlsKey, updatedEcertKey.Data["key.pem"])).To(Equal(false))
				})

				By("setting reenroll flag back to false after restart", func() {
					ibporderer := &current.IBPOrderer{}
					Eventually(func() bool {
						result := ibpCRClient.Get().Namespace(namespace).Resource(IBPORDERERS).
							Name(node2.Name).Do(context.TODO())
						result.Into(ibporderer)

						return ibporderer.Spec.Action.Reenroll.TLSCertNewKey
					}).Should(Equal(false))
				})
			})
		})

		When("reusing existing key", func() {
			It("gets a new certificate", func() {
				patch := func(o client.Object) {
					ibporderer := o.(*current.IBPOrderer)
					ibporderer.Spec.Action.Reenroll.TLSCert = true
				}

				err := integration.ResilientPatch(
					ibpCRClient,
					node2.Name,
					namespace,
					IBPORDERERS,
					3,
					&current.IBPOrderer{},
					patch)
				Expect(err).NotTo(HaveOccurred())

				commonAssertions()

				By("not generating a new key", func() {
					updatedEcertKey, err := kclient.CoreV1().Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("tls-%s-keystore", node2.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(bytes.Equal(tlsKey, updatedEcertKey.Data["key.pem"])).To(Equal(true))
				})

				By("setting reenroll flag back to false after restart", func() {
					ibporderer := &current.IBPOrderer{}
					Eventually(func() bool {
						result := ibpCRClient.Get().Namespace(namespace).Resource(IBPORDERERS).
							Name(node2.Name).Do(context.TODO())
						result.Into(ibporderer)

						return ibporderer.Spec.Action.Reenroll.TLSCert
					}).Should(Equal(false))
				})
			})
		})
	})

	Context("spec has tlscert enroll flag set to true", func() {
		var (
			tls []byte
		)

		BeforeEach(func() {
			tlsSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", node3.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			tls = tlsSecret.Data["cert.pem"]
		})

		It("gets a new certificate", func() {
			patch := func(o client.Object) {
				ibporderer := o.(*current.IBPOrderer)
				ibporderer.Spec.Action.Enroll.TLSCert = true
			}

			err := integration.ResilientPatch(ibpCRClient, node3.Name, namespace, IBPORDERERS, 3, &current.IBPOrderer{}, patch)
			Expect(err).NotTo(HaveOccurred())

			By("restarting orderer pods", func() {
				Eventually(func() bool {
					pods := node3.GetPods()
					if len(pods) != 1 {
						return false
					}

					newPodName := pods[0].Name
					if newPodName != podNameNode3 {
						return true
					}

					return false
				}).Should(Equal(true))
			})

			By("backing up old signcert", func() {
				backup := GetBackup("tls", node3.Name)
				Expect(len(backup.List)).NotTo(Equal(0))
				Expect(backup.List[len(backup.List)-1].SignCerts).To(Equal(base64.StdEncoding.EncodeToString(tls)))
			})

			By("updating tls signcert secret", func() {
				updatedTLSSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", node3.Name), metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				Expect(bytes.Equal(tls, updatedTLSSecret.Data["cert.pem"])).To(Equal(false))
			})

			By("setting reenroll flag back to false after restart", func() {
				Eventually(func() bool {
					result := ibpCRClient.Get().Namespace(namespace).Resource(IBPORDERERS).Name(node3.Name).Do(context.TODO())
					ibporderer := &current.IBPOrderer{}
					result.Into(ibporderer)

					return ibporderer.Spec.Action.Enroll.TLSCert
				}).Should(Equal(false))
			})
		})
	})

})

func GetBackup(certType, name string) *common.Backup {
	backupSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("%s-crypto-backup", name), metav1.GetOptions{})
	if err != nil {
		Expect(k8serrors.IsNotFound(err)).To(Equal(true))
		return &common.Backup{}
	}

	backup := &common.Backup{}
	key := fmt.Sprintf("%s-backup.json", certType)
	err = json.Unmarshal(backupSecret.Data[key], backup)
	Expect(err).NotTo(HaveOccurred())

	return backup
}
