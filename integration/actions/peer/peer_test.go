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

package peer_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

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

var _ = Describe("trigger peer actions", func() {
	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	var (
		podName string
		ibppeer *current.IBPPeer
	)

	BeforeEach(func() {
		Eventually(func() int { return len(org1peer.GetRunningPods()) }).Should(Equal(1))
		podName = org1peer.GetRunningPods()[0].Name

		integration.ClearOperatorConfig(kclient, namespace)
	})

	When("spec has restart flag set to true", func() {
		It("performs restart action", func() {
			patch := func(o client.Object) {
				ibppeer = o.(*current.IBPPeer)
				ibppeer.Spec.Action.Restart = true
			}

			err := integration.ResilientPatch(ibpCRClient, org1peer.Name, namespace, IBPPEERS, 3, &current.IBPPeer{}, patch)
			Expect(err).NotTo(HaveOccurred())

			Eventually(org1peer.PodIsRunning).Should((Equal(true)))

			By("restarting peer pods", func() {
				Eventually(func() bool {
					pods := org1peer.GetRunningPods()
					if len(pods) == 0 {
						return false
					}

					newPodName := pods[0].Name
					if newPodName != podName {
						return true
					}

					return false
				}).Should(Equal(true))
			})

			By("setting restart flag back to false after restart", func() {
				Eventually(func() bool {
					result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
					ibppeer := &current.IBPPeer{}
					result.Into(ibppeer)

					return ibppeer.Spec.Action.Restart
				}).Should(Equal(false))
			})
		})
	})

	When("spec has ecert reenroll flag set to true", func() {
		var (
			ecert, ekey []byte

			commonAssertions = func() {
				By("restarting peer pods", func() {
					Eventually(func() bool {
						pods := org1peer.GetRunningPods()
						if len(pods) == 0 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName != podName {
							return true
						}

						return false
					}).Should(Equal(true))
				})

				By("backing up old signcert", func() {
					backup := GetBackup("ecert", org1peer.Name)
					Expect(len(backup.List)).NotTo(Equal(0))
					Expect(backup.List[len(backup.List)-1].SignCerts).To(Equal(base64.StdEncoding.EncodeToString(ecert)))
				})

				By("updating ecert signcert secret", func() {
					updatedEcertSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					Expect(bytes.Equal(ecert, updatedEcertSecret.Data["cert.pem"])).To(Equal(false))
				})
			}
		)

		BeforeEach(func() {
			ecertSecret, err := kclient.CoreV1().Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			ecert = ecertSecret.Data["cert.pem"]

			ecertSecret, err = kclient.CoreV1().Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			ekey = ecertSecret.Data["key.pem"]
		})

		It("gets a new certificate and key", func() {
			patch := func(o client.Object) {
				ibppeer = o.(*current.IBPPeer)
				ibppeer.Spec.Action.Reenroll.EcertNewKey = true
			}

			err := integration.ResilientPatch(
				ibpCRClient,
				org1peer.Name,
				namespace,
				IBPPEERS,
				3,
				&current.IBPPeer{},
				patch)
			Expect(err).NotTo(HaveOccurred())

			commonAssertions()

			By("generating a new key", func() {
				updatedEcertKey, err := kclient.CoreV1().Secrets(namespace).
					Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", org1peer.Name), metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(bytes.Equal(ekey, updatedEcertKey.Data["key.pem"])).To(Equal(false))
			})

			By("setting reenroll flag back to false after restart", func() {
				ibppeer := &current.IBPPeer{}
				Eventually(func() bool {
					result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
					result.Into(ibppeer)

					return ibppeer.Spec.Action.Reenroll.EcertNewKey
				}).Should(Equal(false))
			})
		})

		It("gets a new certificate", func() {
			patch := func(o client.Object) {
				ibppeer = o.(*current.IBPPeer)
				ibppeer.Spec.Action.Reenroll.Ecert = true
			}

			err := integration.ResilientPatch(
				ibpCRClient,
				org1peer.Name,
				namespace,
				IBPPEERS,
				3,
				&current.IBPPeer{},
				patch)
			Expect(err).NotTo(HaveOccurred())

			commonAssertions()

			By("not generating a new key", func() {
				updatedEcertKey, err := kclient.CoreV1().Secrets(namespace).
					Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", org1peer.Name), metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(bytes.Equal(ekey, updatedEcertKey.Data["key.pem"])).To(Equal(true))
			})

			By("setting reenroll flag back to false after restart", func() {
				ibppeer := &current.IBPPeer{}
				Eventually(func() bool {
					result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
					result.Into(ibppeer)

					return ibppeer.Spec.Action.Reenroll.Ecert
				}).Should(Equal(false))
			})
		})
	})

	When("spec has TLS reenroll flag set to true", func() {
		var (
			cert, key []byte

			commonAssertions = func() {
				By("restarting peer pods", func() {
					Eventually(func() bool {
						pods := org1peer.GetRunningPods()
						if len(pods) == 0 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName != podName {
							return true
						}

						return false
					}).Should(Equal(true))
				})

				By("backing up old signcert", func() {
					backup := GetBackup("tls", org1peer.Name)
					Expect(len(backup.List)).NotTo(Equal(0))
					Expect(backup.List[len(backup.List)-1].SignCerts).To(Equal(base64.StdEncoding.EncodeToString(cert)))
				})

				By("updating tls signcert secret", func() {
					updatedTLSSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					Expect(bytes.Equal(cert, updatedTLSSecret.Data["cert.pem"])).To(Equal(false))
				})
			}
		)

		BeforeEach(func() {
			tlsSecret, err := kclient.CoreV1().Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			cert = tlsSecret.Data["cert.pem"]

			tlsSecret, err = kclient.CoreV1().Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("tls-%s-keystore", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			key = tlsSecret.Data["key.pem"]
		})

		When("requesting a new key", func() {
			It("gets a new key and certificate", func() {
				patch := func(o client.Object) {
					ibppeer = o.(*current.IBPPeer)
					ibppeer.Spec.Action.Reenroll.TLSCertNewKey = true
				}

				err := integration.ResilientPatch(
					ibpCRClient,
					org1peer.Name,
					namespace,
					IBPPEERS,
					3,
					&current.IBPPeer{},
					patch)
				Expect(err).NotTo(HaveOccurred())

				commonAssertions()

				By("generating a new key", func() {
					updatedKey, err := kclient.CoreV1().Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("tls-%s-keystore", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					Expect(bytes.Equal(key, updatedKey.Data["key.pem"])).To(Equal(false))
				})

				By("setting reenroll flag back to false after restart", func() {
					Eventually(func() bool {
						result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
						ibppeer := &current.IBPPeer{}
						result.Into(ibppeer)

						return ibppeer.Spec.Action.Reenroll.TLSCertNewKey
					}).Should(Equal(false))
				})
			})
		})

		When("reusing existing key", func() {
			It("gets a new certificate", func() {
				patch := func(o client.Object) {
					ibppeer = o.(*current.IBPPeer)
					ibppeer.Spec.Action.Reenroll.TLSCert = true
				}

				err := integration.ResilientPatch(
					ibpCRClient,
					org1peer.Name,
					namespace,
					IBPPEERS,
					3,
					&current.IBPPeer{},
					patch)
				Expect(err).NotTo(HaveOccurred())

				commonAssertions()

				By("not generating a new key", func() {
					updatedKey, err := kclient.CoreV1().Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("tls-%s-keystore", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(bytes.Equal(key, updatedKey.Data["key.pem"])).To(Equal(true))
				})

				By("setting reenroll flag back to false after restart", func() {
					Eventually(func() bool {
						result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
						ibppeer := &current.IBPPeer{}
						result.Into(ibppeer)

						return ibppeer.Spec.Action.Reenroll.TLSCert
					}).Should(Equal(false))
				})
			})
		})
	})

	When("spec has ecert enroll flag set to true", func() {
		var (
			ecert    []byte
			ecertKey []byte
		)

		BeforeEach(func() {
			ecertSecret, err := kclient.CoreV1().
				Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			ecertKeySecret, err := kclient.CoreV1().
				Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			ecert = ecertSecret.Data["cert.pem"]
			ecertKey = ecertKeySecret.Data["key.pem"]
		})

		It("generates new crypto", func() {
			patch := func(o client.Object) {
				ibppeer = o.(*current.IBPPeer)
				ibppeer.Spec.Action.Enroll.Ecert = true
			}

			err := integration.ResilientPatch(ibpCRClient, org1peer.Name, namespace, IBPPEERS, 3, &current.IBPPeer{}, patch)
			Expect(err).NotTo(HaveOccurred())

			By("backing up old crypto", func() {
				Eventually(func() bool {
					backup := GetBackup("ecert", org1peer.Name)
					if len(backup.List) == 0 {
						return false
					}

					return backup.List[len(backup.List)-1].SignCerts == base64.StdEncoding.EncodeToString(ecert) &&
						backup.List[len(backup.List)-1].KeyStore == base64.StdEncoding.EncodeToString(ecertKey)
				}).Should(Equal(true))
			})

			By("updating ecert signcert secret", func() {
				Eventually(func() bool {
					updatedEcertSecret, err := kclient.CoreV1().
						Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return bytes.Equal(ecert, updatedEcertSecret.Data["cert.pem"])
				}).Should(Equal(false))
			})

			By("updating ecert key secret", func() {
				Eventually(func() bool {
					updatedEcertSecret, err := kclient.CoreV1().
						Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return bytes.Equal(ecertKey, updatedEcertSecret.Data["key.pem"])
				}).Should(Equal(false))
			})

			By("setting ecert action flag back to false in spec after completion", func() {
				Eventually(func() bool {
					result := ibpCRClient.Get().Namespace(namespace).
						Resource(IBPPEERS).
						Name(org1peer.Name).
						Do(context.TODO())
					ibppeer := &current.IBPPeer{}
					result.Into(ibppeer)

					return ibppeer.Spec.Action.Enroll.Ecert
				}).Should(Equal(false))
			})
		})
	})

	When("spec has tls enroll flag set to true", func() {
		var (
			tlscert []byte
			tlskey  []byte
		)

		BeforeEach(func() {
			tlscertSecret, err := kclient.CoreV1().
				Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			tlskeySecret, err := kclient.CoreV1().
				Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("tls-%s-keystore", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			tlscert = tlscertSecret.Data["cert.pem"]
			tlskey = tlskeySecret.Data["key.pem"]
		})

		It("generates new crypto", func() {
			patch := func(o client.Object) {
				ibppeer = o.(*current.IBPPeer)
				ibppeer.Spec.Action.Enroll.TLSCert = true
			}

			err := integration.ResilientPatch(ibpCRClient, org1peer.Name, namespace, IBPPEERS, 3, &current.IBPPeer{}, patch)
			Expect(err).NotTo(HaveOccurred())

			By("backing up old crypto", func() {
				Eventually(func() bool {
					backup := GetBackup("tls", org1peer.Name)
					Expect(len(backup.List)).NotTo(Equal(0))
					return backup.List[len(backup.List)-1].SignCerts == base64.StdEncoding.EncodeToString(tlscert) &&
						backup.List[len(backup.List)-1].KeyStore == base64.StdEncoding.EncodeToString(tlskey)
				}).Should(Equal(true))
			})

			By("updating ecert signcert secret", func() {
				Eventually(func() bool {
					updatedTlscertSecret, err := kclient.CoreV1().
						Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return bytes.Equal(tlscert, updatedTlscertSecret.Data["cert.pem"])
				}).Should(Equal(false))
			})

			By("updating ecert key secret", func() {
				Eventually(func() bool {
					updatedTlskeySecret, err := kclient.CoreV1().
						Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("tls-%s-keystore", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					return bytes.Equal(tlskey, updatedTlskeySecret.Data["key.pem"])
				}).Should(Equal(false))
			})

			By("setting TLS action flag back to false in spec after completion", func() {
				Eventually(func() bool {
					result := ibpCRClient.Get().Namespace(namespace).
						Resource(IBPPEERS).
						Name(org1peer.Name).
						Do(context.TODO())
					ibppeer := &current.IBPPeer{}
					result.Into(ibppeer)

					return ibppeer.Spec.Action.Enroll.TLSCert
				}).Should(Equal(false))
			})
		})
	})

	Context("upgrade dbs", func() {
		var (
			migrationJobName string
			err              error
		)

		It("performs db reset job", func() {
			patch := func(o client.Object) {
				ibppeer = o.(*current.IBPPeer)
				ibppeer.Spec.Action.UpgradeDBs = true
			}

			err = integration.ResilientPatch(ibpCRClient, org1peer.Name, namespace, IBPPEERS, 3, &current.IBPPeer{}, patch)
			Expect(err).NotTo(HaveOccurred())

			By("starting migration job", func() {
				Eventually(func() bool {
					migrationJobName, err = helper.GetJobID(kclient, namespace, fmt.Sprintf("%s-dbmigration", ibppeer.Name))
					if err != nil {
						return false
					}

					_, err = kclient.BatchV1().Jobs(namespace).
						Get(context.TODO(), migrationJobName, metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("clearing out reset value after completion", func() {
				Eventually(func() bool {
					result := ibpCRClient.Get().Namespace(namespace).
						Resource(IBPPEERS).
						Name(org1peer.Name).
						Do(context.TODO())

					Expect(result.Error()).NotTo(HaveOccurred())

					ibppeer = &current.IBPPeer{}
					result.Into(ibppeer)

					return ibppeer.Spec.Action.UpgradeDBs
				}).Should(Equal(false))
			})

			By("removing migration job", func() {
				Eventually(func() bool {
					_, err := kclient.BatchV1().Jobs(namespace).
						Get(context.TODO(), migrationJobName, metav1.GetOptions{})
					if err != nil {
						return true
					}
					return false
				}).Should(Equal(true))
			})

			By("removing migration pod", func() {
				Eventually(func() bool {
					podList, err := kclient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
						LabelSelector: fmt.Sprintf("job-name=%s-dbmigration", ibppeer.Name),
					})
					if err != nil {
						return true
					}

					if len(podList.Items) == 0 {
						return true
					}

					return false
				}).Should(Equal(true))
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
