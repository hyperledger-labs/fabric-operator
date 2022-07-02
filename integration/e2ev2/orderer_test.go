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

package e2ev2_test

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
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v1"
	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v2"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	v2config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v2"
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	IBPORDERERS = "ibporderers"

	signCert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNUekNDQWZXZ0F3SUJBZ0lVQWNnakVkOHBkOE43Vjg0YmFleG4yQzU0dWtzd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRFeE1qRTRNell3TUZvWERUSTBNVEV4TURFNE5ERXdNRm93WHpFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdZMnhwWlc1ME1SQXdEZ1lEVlFRREV3ZHZjbVJsY21WeU1Ga3dFd1lICktvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUU2NFJwK1pvVnYyaTg0cE5KUUFNUHJpenJmZVlNT2Y0UnZ1eHkKNHZOUU1Pd3JEemlIZkFLTnZmdUJlbDhpQ2dndHRXM2paZTVkSEFZaFVIS2Ryb3FodmFPQmhUQ0JnakFPQmdOVgpIUThCQWY4RUJBTUNCNEF3REFZRFZSMFRBUUgvQkFJd0FEQWRCZ05WSFE0RUZnUVVWakl3Y1YwYXRNZmZWV1E5CnhtenpXVG9uYmlJd0h3WURWUjBqQkJnd0ZvQVVTUU9ZL0Z5YnNXcTlIWEo3c296aUFyLzhtQkV3SWdZRFZSMFIKQkJzd0dZSVhVMkZoWkhNdFRXRmpRbTl2YXkxUWNtOHViRzlqWVd3d0NnWUlLb1pJemowRUF3SURTQUF3UlFJaApBUGE4Y3VjL3QvOW45ZDZlSHZoUWdialNBK1k2dytERW1ka2RpdnJHaGE5RUFpQXdTZStlVGdsQWJYQVNoTnhwCkJpR0Rjc2IwZ1pmRmhQd1pIN1VnQW1IQjN3PT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
	certKey  = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ1p2VWRsUVZ6QlVSc3I2STMKZEVvd0ZlVGkvVkNLZVZqMmFwN2x3QWNYSzJLaFJBTkNBQVRyaEduNW1oVy9hTHppazBsQUF3K3VMT3Q5NWd3NQovaEcrN0hMaTgxQXc3Q3NQT0lkOEFvMjkrNEY2WHlJS0NDMjFiZU5sN2wwY0JpRlFjcDJ1aXFHOQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg=="
	caCert   = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNGakNDQWIyZ0F3SUJBZ0lVZi84bk94M2NqM1htVzNDSUo1L0Q1ejRRcUVvd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRBek1ERTNNamd3TUZvWERUTTBNVEF5TmpFM01qZ3dNRm93YURFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1Sa3dGd1lEVlFRREV4Qm1ZV0p5YVdNdFkyRXRjMlZ5CmRtVnlNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUVSbzNmbUc2UHkyUHd6cUMwNnFWZDlFOFgKZ044eldqZzFMb3lnMmsxdkQ4MXY1dENRRytCTVozSUJGQnI2VTRhc0tZTUREakd6TElERmdUUTRjVDd1VktORgpNRU13RGdZRFZSMFBBUUgvQkFRREFnRUdNQklHQTFVZEV3RUIvd1FJTUFZQkFmOENBUUV3SFFZRFZSME9CQllFCkZFa0RtUHhjbTdGcXZSMXllN0tNNGdLLy9KZ1JNQW9HQ0NxR1NNNDlCQU1DQTBjQU1FUUNJRC92QVFVSEh2SWwKQWZZLzM5UWdEU2ltTWpMZnhPTG44NllyR1EvWHpkQVpBaUFpUmlyZmlMdzVGbXBpRDhtYmlmRjV4bzdFUzdqNApaUWQyT0FUNCt5OWE0Zz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
)

var _ = Describe("orderer", func() {
	var (
		node1 helper.Orderer
	)

	BeforeEach(func() {
		node1 = orderer.Nodes[0]
		Eventually(node1.PodIsRunning, time.Second*60, time.Second*2).Should((Equal(true)))

		ClearOperatorConfig()
	})

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("config overrides", func() {
		var (
			podName string
			bytes   []byte
		)

		BeforeEach(func() {
			cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), orderer.Name+"node1-config", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			ordererBytes := cm.BinaryData["orderer.yaml"]
			ordererConfig, err := v2config.ReadOrdererFromBytes(ordererBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(ordererConfig.General.Keepalive.ServerMinInterval.Duration).To(Equal(common.MustParseDuration("30h").Duration))

			configOverride := &v2config.Orderer{
				Orderer: v2.Orderer{
					General: v2.General{
						Keepalive: v1.Keepalive{
							ServerInterval: common.MustParseDuration("20h"),
						},
					},
				},
			}
			configBytes, err := json.Marshal(configOverride)
			Expect(err).NotTo(HaveOccurred())
			orderer.CR.Spec.ConfigOverride = &runtime.RawExtension{Raw: configBytes}

			orderer.CR.Name = orderer.CR.Name + "node1"

			bytes, err = json.Marshal(orderer.CR)
			Expect(err).NotTo(HaveOccurred())

			podName = node1.GetRunningPods()[0].Name
		})

		It("updates config based on overrides", func() {
			result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource(IBPORDERERS).Name(node1.Name).Body(bytes).Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			By("updating config in config map", func() {
				var ordererConfig *v2config.Orderer
				Eventually(func() bool {
					cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), orderer.Name+"node1-config", metav1.GetOptions{})
					if err != nil {
						return false
					}

					ordererBytes := cm.BinaryData["orderer.yaml"]
					ordererConfig, err = v2config.ReadOrdererFromBytes(ordererBytes)
					if err != nil {
						return false
					}

					if ordererConfig.General.Keepalive.ServerInterval.Duration == common.MustParseDuration("20h").Duration {
						return true
					}

					return false
				}).Should(Equal(true))

				Expect(ordererConfig.General.Keepalive.ServerMinInterval.Duration).To(Equal(common.MustParseDuration("30h").Duration))
				Expect(ordererConfig.General.Keepalive.ServerInterval.Duration).To(Equal(common.MustParseDuration("20h").Duration))
			})

			By("restarting orderer pods", func() {
				Eventually(func() bool {
					pods := node1.GetRunningPods()
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
		})
	})

	Context("msp certs", func() {
		var (
			podName     string
			oldsigncert []byte
			oldkeystore []byte
			oldcacert   []byte
		)

		BeforeEach(func() {
			Eventually(func() int { return len(node1.GetRunningPods()) }).Should(Equal(1))

			pods := node1.GetPods()
			podName = pods[0].Name

			// Store original certs
			oldsigncert = EcertSignCert(node1.Name)
			oldkeystore = EcertKeystore(node1.Name)
			oldcacert = EcertCACert(node1.Name)
		})

		It("updates secrets for new certs passed through MSP spec", func() {

			patch := func(i client.Object) {
				testOrderer := i.(*current.IBPOrderer)
				testOrderer.Spec.Secret = &current.SecretSpec{
					MSP: &current.MSPSpec{
						Component: &current.MSP{
							SignCerts: signCert,
							KeyStore:  certKey,
							CACerts:   []string{caCert},
						},
					},
				}
			}

			err := integration.ResilientPatch(ibpCRClient, node1.Name, namespace, "ibporderers", 3, &current.IBPOrderer{}, patch)
			Expect(err).NotTo(HaveOccurred())

			By("restarting node", func() {
				Eventually(func() bool {
					pods := node1.GetPods()
					if len(pods) != 1 {
						return false
					}

					newPodName := pods[0].Name
					if newPodName == podName {
						return false
					}

					return true
				}).Should(Equal(true))

				Eventually(node1.PodIsRunning).Should((Equal(true)))
			})

			By("backing up old signcert", func() {
				backup := GetBackup("ecert", node1.Name)
				Expect(len(backup.List)).NotTo(Equal(0))
				Expect(backup.List[len(backup.List)-1].SignCerts).To(Equal(base64.StdEncoding.EncodeToString(oldsigncert)))
				Expect(backup.List[len(backup.List)-1].KeyStore).To(Equal(base64.StdEncoding.EncodeToString(oldkeystore)))
				Expect(backup.List[len(backup.List)-1].CACerts).To(Equal([]string{base64.StdEncoding.EncodeToString(oldcacert)}))
			})

			By("updating signcert secret", func() {
				Expect(bytes.Equal(oldsigncert, EcertSignCert(node1.Name))).To(Equal(false))
			})

			By("updating keystore secret", func() {
				Expect(bytes.Equal(oldkeystore, EcertKeystore(node1.Name))).To(Equal(false))
			})

			By("updating cacert secret", func() {
				Expect(bytes.Equal(oldcacert, EcertCACert(node1.Name))).To(Equal(false))
			})
		})
	})

	Context("node ou updated", func() {
		var (
			podName    string
			bytes      []byte
			ibporderer *current.IBPOrderer
			secret     *corev1.Secret
		)

		BeforeEach(func() {
			// Pods seem to run slower and restart slower when running test in Travis.
			SetDefaultEventuallyTimeout(540 * time.Second)

			Eventually(func() int { return len(node1.GetRunningPods()) }).Should(Equal(1))
			podName = node1.GetRunningPods()[0].Name

			// Make sure config is in expected state
			cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), node1.Name+"-config", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			configBytes := cm.BinaryData["config.yaml"]
			cfg, err := config.NodeOUConfigFromBytes(configBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.NodeOUs.Enable).To(Equal(true))

			secret, err = kclient.CoreV1().
				Secrets(namespace).
				Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", node1.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			result := ibpCRClient.Get().Namespace(namespace).Resource(IBPORDERERS).Name(node1.Name).Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			ibporderer = &current.IBPOrderer{}
			result.Into(ibporderer)
		})

		It("disables nodeOU", func() {
			By("providing admin certs", func() {
				var err error
				adminCert := base64.StdEncoding.EncodeToString(secret.Data["cert.pem"])

				ibporderer.Spec.Secret.Enrollment.Component.AdminCerts = []string{adminCert}
				ibporderer.Spec.Secret.MSP = nil
				bytes, err = json.Marshal(ibporderer)
				Expect(err).NotTo(HaveOccurred())

				result := ibpCRClient.Put().Namespace(namespace).Resource(IBPORDERERS).Name(node1.Name).Body(bytes).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				Eventually(func() bool {
					_, err := kclient.CoreV1().
						Secrets(namespace).
						Get(context.TODO(), fmt.Sprintf("ecert-%s-admincerts", node1.Name), metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("disabling nodeOU", func() {
				result := ibpCRClient.Get().Namespace(namespace).Resource(IBPORDERERS).Name(node1.Name).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				ibporderer = &current.IBPOrderer{}
				result.Into(ibporderer)

				// Disable node ou
				ibporderer.Spec.DisableNodeOU = &current.BoolTrue
				bytes, err := json.Marshal(ibporderer)
				Expect(err).NotTo(HaveOccurred())

				result = ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource(IBPORDERERS).Name(node1.Name).Body(bytes).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())
			})

			By("updating config map", func() {
				Eventually(func() bool {
					cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), node1.Name+"-config", metav1.GetOptions{})
					if err != nil {
						return false
					}

					configBytes := cm.BinaryData["config.yaml"]
					nodeOUConfig, err := config.NodeOUConfigFromBytes(configBytes)
					if err != nil {
						return false
					}

					return nodeOUConfig.NodeOUs.Enable
				}).Should(Equal(false))
			})

			By("restarting orderer node pods", func() {
				Eventually(func() bool {
					pods := node1.GetRunningPods()
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
		})
	})
})

func GetOrderer(tlsCert, caHost string) *helper.Orderer {
	cr, err := helper.OrdererCR(namespace, domain, ordererUsername, tlsCert, caHost)
	Expect(err).NotTo(HaveOccurred())

	nodes := []helper.Orderer{
		helper.Orderer{
			Name:      cr.Name + "node1",
			Namespace: namespace,
			CR:        cr.DeepCopy(),
			NodeName:  fmt.Sprintf("%s%s%d", cr.Name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      cr.Name + "node1",
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}

	nodes[0].CR.ObjectMeta.Name = cr.Name + "node1"

	return &helper.Orderer{
		Name:      cr.Name,
		Namespace: namespace,
		CR:        cr,
		NodeName:  fmt.Sprintf("%s-%s%d", cr.Name, baseorderer.NODE, 1),
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      cr.Name,
			Namespace: namespace,
			Client:    kclient,
		},
		Nodes: nodes,
	}
}
