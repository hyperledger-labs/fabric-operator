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
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v2"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	v2peerconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	IBPPEERS = "ibppeers"
)

var _ = Describe("peer", func() {
	BeforeEach(func() {
		Eventually(org1peer.PodIsRunning).Should((Equal(true)))

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
			bytes []byte
		)

		BeforeEach(func() {
			// Make sure the config is in expected state
			cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), org1peer.Name+"-config", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			coreBytes := cm.BinaryData["core.yaml"]
			peerConfig, err := v2peerconfig.ReadCoreFromBytes(coreBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Peer.ID).To(Equal("testPeerID"))

			// Update the config overrides
			result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			peer := &current.IBPPeer{}
			result.Into(peer)

			configOverride := &v2peerconfig.Core{
				Core: v2.Core{
					Peer: v2.Peer{
						Keepalive: v2.KeepAlive{
							MinInterval: common.MustParseDuration("20h"),
						},
					},
				},
			}

			configBytes, err := json.Marshal(configOverride)
			Expect(err).NotTo(HaveOccurred())
			peer.Spec.ConfigOverride = &runtime.RawExtension{Raw: configBytes}

			bytes, err = json.Marshal(peer)
			Expect(err).NotTo(HaveOccurred())
		})

		It("updates config based on overrides", func() {
			result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Body(bytes).Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			var peerConfig *v2peerconfig.Core
			Eventually(func() bool {
				cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), org1peer.Name+"-config", metav1.GetOptions{})
				if err != nil {
					return false
				}

				coreBytes := cm.BinaryData["core.yaml"]
				peerConfig, err = v2peerconfig.ReadCoreFromBytes(coreBytes)
				if err != nil {
					return false
				}

				if peerConfig.Peer.Keepalive.MinInterval.Duration == common.MustParseDuration("20h").Duration {
					return true
				}

				return false
			}).Should(Equal(true))

			Expect(peerConfig.Peer.ID).To(Equal("testPeerID"))
			Expect(peerConfig.Peer.Keepalive.MinInterval.Duration).To(Equal(common.MustParseDuration("20h").Duration))
		})
	})

	Context("node ou updated", func() {
		var (
			podName string
			bytes   []byte
		)

		BeforeEach(func() {
			// Pods seem to run slower and restart slower when running test in Travis.
			SetDefaultEventuallyTimeout(540 * time.Second)

			Eventually(func() int { return len(org1peer.GetRunningPods()) }).Should(Equal(1))
			podName = org1peer.GetRunningPods()[0].Name

			// Make sure config is in expected state
			cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), org1peer.Name+"-config", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			configBytes := cm.BinaryData["config.yaml"]
			cfg, err := config.NodeOUConfigFromBytes(configBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.NodeOUs.Enable).To(Equal(true))

			// Update the config overrides
			result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			peer := &current.IBPPeer{}
			result.Into(peer)

			// Disable node ou
			peer.Spec.DisableNodeOU = &current.BoolTrue
			bytes, err = json.Marshal(peer)
			Expect(err).NotTo(HaveOccurred())
		})

		It("disables nodeOU", func() {
			result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Body(bytes).Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			peer := &current.IBPPeer{}
			result.Into(peer)
			Expect(peer.Spec.NodeOUDisabled()).To(Equal(true))

			By("updating config map", func() {
				Eventually(func() bool {
					cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), org1peer.Name+"-config", metav1.GetOptions{})
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
		})
	})
})
