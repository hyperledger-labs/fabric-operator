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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This test is designed to stress-test reenroll functionality
// NOTE: need to set Restart.WaitTime = 0 in operator config
var _ = PDescribe("reenroll action", func() {
	BeforeEach(func() {
		Eventually(org1peer.PodIsRunning).Should((Equal(true)))
	})

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("reenroll peer", func() {
		const (
			// Modify to stress-test reenroll functionality
			numReenrolls = 1
		)

		When("spec has ecert &tlscert reenroll flag set to true", func() {
			var (
				ecert []byte
				tcert []byte
			)

			It("reenrolls ecert & tlscert for numReenrolls amount of times", func() {
				count := 1
				for count <= numReenrolls {
					fmt.Printf("REENROLL COUNT: %d\n", count)

					ecertSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					ecert = ecertSecret.Data["cert.pem"]

					tlsSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", org1peer.Name), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					tcert = tlsSecret.Data["cert.pem"]

					patch := func(o client.Object) {
						ibppeer := o.(*current.IBPPeer)
						ibppeer.Spec.Action.Reenroll.Ecert = true
						ibppeer.Spec.Action.Reenroll.TLSCert = true
					}

					err = integration.ResilientPatch(ibpCRClient, org1peer.Name, namespace, IBPPEERS, 3, &current.IBPPeer{}, patch)
					Expect(err).NotTo(HaveOccurred())

					fmt.Printf("APPLIED PATCH NUMBER: %d\n", count)

					By("updating ecert signcert secret", func() {
						Eventually(func() bool {
							updatedEcertSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", org1peer.Name), metav1.GetOptions{})
							Expect(err).NotTo(HaveOccurred())

							return bytes.Equal(ecert, updatedEcertSecret.Data["cert.pem"])
						}).Should(Equal(false))
					})

					By("updating tls signcert secret", func() {
						Eventually(func() bool {
							updatedTLSSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", org1peer.Name), metav1.GetOptions{})
							Expect(err).NotTo(HaveOccurred())

							return bytes.Equal(tcert, updatedTLSSecret.Data["cert.pem"])
						}).Should(Equal(false))
					})

					time.Sleep(10 * time.Second)

					By("setting reenroll flag back to false after restart", func() {
						Eventually(func() bool {
							result := ibpCRClient.Get().Namespace(namespace).Resource(IBPPEERS).Name(org1peer.Name).Do(context.TODO())
							ibppeer := &current.IBPPeer{}
							result.Into(ibppeer)

							return ibppeer.Spec.Action.Reenroll.Ecert &&
								ibppeer.Spec.Action.Reenroll.TLSCert
						}).Should(Equal(false))
					})

					count++
				}

			})
		})
	})
})
