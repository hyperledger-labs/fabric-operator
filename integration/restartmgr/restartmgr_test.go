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

package restartmgr_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/staggerrestarts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("restart manager", func() {
	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("peer", func() {
		Context("admin certs", func() {
			var (
				podName     string
				peer        *current.IBPPeer
				tlsbackup   *common.Backup
				ecertbackup *common.Backup
			)

			BeforeEach(func() {
				Eventually(func() int { return len(org1peer.GetRunningPods()) }).Should(Equal(1))

				podName = org1peer.GetRunningPods()[0].Name

				// Get peer's custom resource (CR)
				result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				peer = &current.IBPPeer{}
				result.Into(peer)

				tlsbackup = GetBackup("tls", org1peer.Name)
				ecertbackup = GetBackup("ecert", org1peer.Name)
			})

			It("restarts the peer after admin cert update", func() {
				// Update the admin cert in the peer's CR spec
				adminCertBytes, err := ioutil.ReadFile(filepath.Join(wd, "org1peer", peerAdminUsername+"2", "msp", "signcerts", "cert.pem"))
				Expect(err).NotTo(HaveOccurred())
				adminCertB64 := base64.StdEncoding.EncodeToString(adminCertBytes)
				peer.Spec.Secret.Enrollment.Component.AdminCerts = []string{peer.Spec.Secret.Enrollment.Component.AdminCerts[0], adminCertB64}

				bytes, err := json.Marshal(peer)
				Expect(err).NotTo(HaveOccurred())

				// Update the peer's CR spec
				result := ibpCRClient.Put().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Body(bytes).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				Eventually(org1peer.PodIsRunning).Should((Equal(true)))

				By("restarting peer pods", func() {
					Eventually(func() bool {
						pods := org1peer.GetRunningPods()
						if len(pods) != 1 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName == podName {
							return false
						}

						return true
					}).Should(Equal(true))
				})

				By("not performing backup of crypto beforehand", func() {
					newTLSBackup := GetBackup("tls", org1peer.Name)
					newEcertBackup := GetBackup("ecert", org1peer.Name)
					Expect(newTLSBackup).To(Equal(tlsbackup))
					Expect(newEcertBackup).To(Equal(ecertbackup))
				})

				By("removing instance from restart queue", func() {
					Eventually(func() bool {
						restartConfig := GetRestartConfigFor("peer")
						if len(restartConfig.Queues[org1peer.CR.GetMSPID()]) != 0 {
							return false
						}
						if restartConfig.Log[org1peer.Name] == nil {
							return false
						}
						if len(restartConfig.Log[org1peer.Name]) != 1 {
							return false
						}
						if restartConfig.Log[org1peer.Name][0].CRName != org1peer.Name {
							return false
						}

						return true
					}).Should(Equal(true))
				})
			})

			It("does not restart the peer if spec is updated with empty list of admin certs", func() {
				// Update the admin cert in the peer's CR spec to be empty
				peer.Spec.Secret.Enrollment.Component.AdminCerts = []string{}
				bytes, err := json.Marshal(peer)
				Expect(err).NotTo(HaveOccurred())

				// Update the peer's CR spec
				result := ibpCRClient.Put().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Body(bytes).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				Eventually(org1peer.PodIsRunning).Should((Equal(true)))

				Eventually(func() bool {
					pods := org1peer.GetRunningPods()
					if len(pods) != 1 {
						return false
					}

					newPodName := pods[0].Name
					if newPodName == podName {
						return true
					}

					return false
				}).Should(Equal(true))

			})
		})

		Context("request deployment restart", func() {
			var (
				podName     string
				peer        *current.IBPPeer
				restartTime string
			)

			BeforeEach(func() {
				Eventually(func() int { return len(org1peer.GetPods()) }).Should(Equal(1))

				podName = org1peer.GetRunningPods()[0].Name

				// Get peer's custom resource (CR)
				result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				peer = &current.IBPPeer{}
				result.Into(peer)

			})

			When("peer was restarted less than 10 min ago for admin cert updates", func() {
				BeforeEach(func() {
					// Create operator-config map to indicate that peer was restarted recently for admin cert update
					restartTime = time.Now().UTC().Format(time.RFC3339)
					CreateOrUpdateOperatorConfig(peer.Name, restart.ADMINCERT, restartTime)

					Eventually(func() bool {
						_, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "operator-config", metav1.GetOptions{})
						if err != nil {
							return false
						}
						return true
					}).Should(Equal(true))
				})

				It("does not restart the peer when admin certs are updated", func() {
					By("updating peer's admin certs", func() {
						adminCertBytes, err := ioutil.ReadFile(filepath.Join(wd, "org1peer", peerAdminUsername+"2", "msp", "signcerts", "cert.pem"))
						Expect(err).NotTo(HaveOccurred())
						adminCertB64 := base64.StdEncoding.EncodeToString(adminCertBytes)
						peer.Spec.Secret.Enrollment.Component.AdminCerts = []string{adminCertB64}

						bytes, err := json.Marshal(peer)
						Expect(err).NotTo(HaveOccurred())

						// Update the peer's CR spec
						result := ibpCRClient.Put().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Body(bytes).Do(context.TODO())
						Expect(result.Error()).NotTo(HaveOccurred())

						Eventually(org1peer.PodIsRunning).Should((Equal(true)))
					})

					By("not restarting peer pods again", func() {
						Consistently(func() bool {
							pods := org1peer.GetRunningPods()
							if len(pods) != 1 {
								return false
							}

							newPodName := pods[0].Name
							if newPodName == podName {
								return true
							}

							return false
						}, 5*time.Second).Should(Equal(true))
					})

					// TODO: This test is failing, there seems to be a couple seconds difference between actual and expected time values. Needs investigation.
					By("adding a pending restart request to config map", func() {
						Skip("Skipping test, needs revision as it currently fails")
						cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "operator-config", metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						cfg := &restart.Config{}
						err = json.Unmarshal(cm.BinaryData["restart-config.yaml"], cfg)
						Expect(err).NotTo(HaveOccurred())

						Expect(cfg.Instances[peer.Name].Requests[restart.ADMINCERT].LastActionTimestamp).To(Equal(restartTime))
						Expect(cfg.Instances[peer.Name].Requests[restart.ADMINCERT].Status).To(Equal(restart.Pending))
					})
				})
			})
		})
	})

	Context("orderer - request deployment restart", func() {
		var (
			node1 helper.Orderer

			podName     string
			ibporderer  *current.IBPOrderer
			restartTime string
		)

		BeforeEach(func() {
			ClearOperatorConfig()

			node1 = orderer.Nodes[0]
			Eventually(node1.PodIsRunning, time.Second*60, time.Second*2).Should((Equal(true)))
			Eventually(func() int { return len(node1.GetPods()) }).Should(Equal(1))

			podName = node1.GetPods()[0].Name
			result := ibpCRClient.Get().Namespace(namespace).
				Resource(IBPORDERERS).
				Name(node1.Name).
				Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			ibporderer = &current.IBPOrderer{}
			result.Into(ibporderer)
		})

		When("reenroll is triggered", func() {
			It("restarts", func() {
				ibporderer.Spec.Action.Reenroll.Ecert = true
				ordererbytes, err := json.Marshal(ibporderer)
				Expect(err).NotTo(HaveOccurred())

				result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource(IBPORDERERS).Name(node1.Name).Body(ordererbytes).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				Eventually(func() bool {
					restartConfig := GetRestartConfigFor("orderer")
					if restartConfig == nil {
						return false
					}
					if len(restartConfig.Queues["orderermsp"]) != 0 {
						return false
					}
					if restartConfig.Log["ibporderer1node1"] == nil || len(restartConfig.Log["ibporderer1node1"]) != 1 {
						return false
					}
					return true
				}).Should(Equal(true))

			})
		})

		When("orderer was restarted less than 10 min ago for ecert reenroll", func() {
			BeforeEach(func() {
				// Create operator-config map to indicate that peer was restarted recently for ecert reenroll
				restartTime = time.Now().UTC().Format(time.RFC3339)
				CreateOrUpdateOperatorConfig(ibporderer.Name, restart.ECERTUPDATE, restartTime)

				Eventually(func() bool {
					_, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "operator-config", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			It("does not restart orderer when ecerts reenroll occurs", func() {
				By("triggering ecert reenroll", func() {
					ibporderer.Spec.Action.Reenroll.Ecert = true
					ordererbytes, err := json.Marshal(ibporderer)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Put().Namespace(namespace).Resource(IBPORDERERS).Name(node1.Name).Body(ordererbytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					Eventually(node1.PodIsRunning).Should(Equal(true))
				})

				By("not restarting orderer pods again", func() {
					Eventually(func() bool {
						pods := node1.GetRunningPods()
						if len(pods) != 1 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName == podName {
							return true
						}

						return false
					}).Should(Equal(true))
				})

				By("adding a pending restart request to config map", func() {
					Skip("Skipping test, needs revision as it currently fails")

					Eventually(func() bool {
						cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "operator-config", metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						cfg := &restart.Config{}
						err = json.Unmarshal(cm.BinaryData["restart-config.yaml"], cfg)
						Expect(err).NotTo(HaveOccurred())

						status := cfg.Instances[ibporderer.Name].Requests[restart.ECERTUPDATE].Status
						lastTimestamp := cfg.Instances[ibporderer.Name].Requests[restart.ECERTUPDATE].LastActionTimestamp
						if status == restart.Pending && lastTimestamp == restartTime {
							return true
						}

						return false
					}).Should(Equal(true))
				})
			})
		})
	})

	Context("CA - request deployment restart", func() {
		var (
			podName     string
			ca          *current.IBPCA
			restartTime string
		)

		BeforeEach(func() {
			Eventually(func() int {
				return len(org1ca.GetPods())
			}).Should(Equal(1))

			podName = org1ca.GetPods()[0].Name

			result := ibpCRClient.Get().Namespace(namespace).Resource(IBPCAS).Name(org1ca.Name).Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			ca = &current.IBPCA{}
			result.Into(ca)
		})

		Context("staggering ca restarts", func() {
			var (
				bytes []byte
				err   error
			)

			BeforeEach(func() {
				ca.Spec.Action.Renew.TLSCert = true

				bytes, err = json.Marshal(ca)
				Expect(err).NotTo(HaveOccurred())
			})

			It("restarts nodes one at a time in same org", func() {
				result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource(IBPCAS).Name(org1ca.Name).Body(bytes).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				Eventually(func() bool {
					restartConfig := GetRestartConfigFor("ca")
					if restartConfig == nil {
						return false
					}
					if len(restartConfig.Queues[""]) != 0 {
						return false
					}
					if restartConfig.Log["org1ca"] == nil || len(restartConfig.Log["org1ca"]) != 1 {
						return false
					}

					return true
				}).Should(Equal(true))
			})
		})

		When("ca was restarted less than 10 min ago for config override", func() {
			BeforeEach(func() {
				// Create operator-config map to indicate that peer was restarted recently for ecert reenroll
				restartTime = time.Now().UTC().Format(time.RFC3339)
				CreateOrUpdateOperatorConfig(ca.Name, restart.CONFIGOVERRIDE, restartTime)

				Eventually(func() bool {
					_, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "operator-config", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			It("does not restart ca when config override occurs", func() {
				override := &v1.ServerConfig{
					CAConfig: v1.CAConfig{
						Version: "1.4.8",
					},
				}
				overrideBytes, err := json.Marshal(override)
				Expect(err).NotTo(HaveOccurred())
				ca.Spec.ConfigOverride = &current.ConfigOverride{
					CA: &runtime.RawExtension{Raw: overrideBytes},
				}

				bytes, err := json.Marshal(ca)
				Expect(err).NotTo(HaveOccurred())

				result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource(IBPCAS).Name(org1ca.Name).Body(bytes).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				Eventually(org1ca.PodIsRunning).Should((Equal(true)))

				By("not restarting ca pod", func() {
					Eventually(func() bool {
						pods := org1ca.GetPods()
						if len(pods) != 1 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName == podName {
							return true
						}

						return false
					}).Should(Equal(true))
				})

				By("adding a pending restart request to config map", func() {
					Eventually(func() bool {
						cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "operator-config", metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						cfg := &restart.Config{}
						err = json.Unmarshal(cm.BinaryData["restart-config.yaml"], cfg)
						Expect(err).NotTo(HaveOccurred())

						status := cfg.Instances[ca.Name].Requests[restart.CONFIGOVERRIDE].Status
						lastTimestamp := cfg.Instances[ca.Name].Requests[restart.CONFIGOVERRIDE].LastActionTimestamp
						if status == restart.Pending && lastTimestamp == restartTime {
							return true
						}

						return false
					}).Should(Equal(true))
				})
			})
		})
	})
})

func CreateOrUpdateOperatorConfig(instance string, reason restart.Reason, lastRestart string) {
	oldCM := GetOperatorConfigMap(instance, reason, lastRestart)

	cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "operator-config", metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		_, err = kclient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), oldCM, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	} else {

		cm.BinaryData = oldCM.BinaryData
		_, err = kclient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())
	}
}

func GetOperatorConfigMap(instance string, reason restart.Reason, lastRestart string) *corev1.ConfigMap {
	cfg := &restart.Config{
		Instances: map[string]*restart.Restart{
			instance: {
				Requests: map[restart.Reason]*restart.Request{
					reason: {
						LastActionTimestamp: lastRestart,
					},
				},
			},
		},
	}
	bytes, err := json.Marshal(cfg)
	Expect(err).NotTo(HaveOccurred())

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "operator-config",
			Namespace: namespace,
		},
		BinaryData: map[string][]byte{
			"restart-config.yaml": bytes,
		},
	}
}

func ClearOperatorConfig() {
	err := kclient.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), "operator-config", *metav1.NewDeleteOptions(0))
	if !k8serrors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

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

func GetRestartConfigFor(componentType string) *staggerrestarts.RestartConfig {
	cmName := componentType + "-restart-config"
	cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), cmName, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	restartConfig := &staggerrestarts.RestartConfig{}
	err = json.Unmarshal(cm.BinaryData["restart-config.yaml"], restartConfig)
	Expect(err).NotTo(HaveOccurred())

	return restartConfig
}
