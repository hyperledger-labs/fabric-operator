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

package operatorrestart_test

import (
	"fmt"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("operator restart", func() {
	Context("CA", func() {
		var (
			originalPodName string
		)

		BeforeEach(func() {
			Eventually(func() int {
				return len(org1ca.GetRunningPods())
			}).Should(Equal(1))
			originalPodName = org1ca.GetRunningPods()[0].Name
		})

		It("does not restart ca on operator restart", func() {
			RestartOperator()

			Consistently(func() string {
				fmt.Fprintf(GinkgoWriter, "Making sure '%s' does not restart, original pod name '%s'\n", org1ca.Name, originalPodName)

				if len(org1ca.GetRunningPods()) != 1 {
					return "incorrect number of running pods"
				}

				return org1ca.GetRunningPods()[0].Name
			}, 5*time.Second, time.Second).Should(Equal(originalPodName))
		})
	})

	Context("Peer", func() {
		var (
			originalPodName string
		)

		BeforeEach(func() {
			Eventually(func() int {
				return len(org1peer.GetRunningPods())
			}).Should(Equal(1))
			originalPodName = org1peer.GetRunningPods()[0].Name
		})

		It("does not restart peer on operator restart", func() {
			RestartOperator()

			Consistently(func() string {
				fmt.Fprintf(GinkgoWriter, "Making sure '%s' does not restart, original pod name '%s'\n", org1peer.Name, originalPodName)

				if len(org1peer.GetRunningPods()) != 1 {
					return "incorrect number of running pods"
				}

				return org1peer.GetRunningPods()[0].Name
			}, 5*time.Second, time.Second).Should(Equal(originalPodName))
		})
	})

	Context("Orderer Node", func() {
		var (
			node            helper.Orderer
			originalPodName string
		)

		BeforeEach(func() {
			node = orderer.Nodes[0]

			Eventually(func() int {
				return len(node.GetRunningPods())
			}).Should(Equal(1))
			originalPodName = node.GetRunningPods()[0].Name
		})

		It("does not restart orderer node on operator restart", func() {
			RestartOperator()

			Consistently(func() string {
				fmt.Fprintf(GinkgoWriter, "Making sure '%s' does not restart, original pod name '%s'\n", node.Name, originalPodName)

				if len(org1peer.GetRunningPods()) != 1 {
					return "incorrect number of running pods"
				}

				return node.GetRunningPods()[0].Name
			}, 5*time.Second, time.Second).Should(Equal(originalPodName))
		})
	})
})
