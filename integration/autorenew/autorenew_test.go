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

package autorenew_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Autorenew", func() {
	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("orderer", func() {
		var (
			node1 helper.Orderer

			tlscert []byte
			ecert   []byte
		)

		BeforeEach(func() {
			node1 = orderer.Nodes[0]
			Eventually(node1.PodCreated, time.Second*60, time.Second*2).Should((Equal(true)))
		})

		AfterEach(func() {
			// Set flag if a test falls
			if CurrentGinkgoTestDescription().Failed {
				testFailed = true
			}
		})

		BeforeEach(func() {
			ecertSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", node1.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			ecert = ecertSecret.Data["cert.pem"]

			tlsSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", node1.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			tlscert = tlsSecret.Data["cert.pem"]
		})

		When("signcert certificate is up for renewal and enrollment spec exists", func() {
			It("only renews the ecert when timer goes off", func() {
				// signcert certificates expire in 1 year (31536000s) from creation;
				// NumSecondsWarningPeriod has been set to 1 year - 60s to make
				// renewal occur when test runs

				By("setting status to warning", func() {
					Eventually(orderer.PollForParentCRStatus).Should(Equal(current.Warning))
				})

				By("backing up old ecert signcert", func() {
					Eventually(func() bool {
						backup := GetBackup("ecert", node1.Name)
						if len(backup.List) > 0 {
							return backup.List[len(backup.List)-1].SignCerts == base64.StdEncoding.EncodeToString(ecert)
						}

						return false
					}).Should(Equal(true))

				})

				By("reenrolling identity and updating ecert certificate secret", func() {
					Eventually(func() bool {
						updatedEcertSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", node1.Name), metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						return bytes.Equal(ecert, updatedEcertSecret.Data["cert.pem"])
					}).Should(Equal(false))
				})

				By("not updating tls signcert secret", func() {
					Eventually(func() bool {
						updatedTLSSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", node1.Name), metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						return bytes.Equal(tlscert, updatedTLSSecret.Data["cert.pem"])
					}).Should(Equal(true))
				})

				By("returning to Deployed status as tls cert won't expire for 10 years", func() {
					Eventually(orderer.PollForParentCRStatus).Should(Equal(current.Deployed))
				})

			})
		})
	})

	Context("peer", func() {
		var (
			tlscert []byte
			ecert   []byte
		)

		BeforeEach(func() {
			Eventually(org1peer.PodCreated, time.Second*60, time.Second*2).Should((Equal(true)))
		})

		AfterEach(func() {
			// Set flag if a test falls
			if CurrentGinkgoTestDescription().Failed {
				testFailed = true
			}
		})

		BeforeEach(func() {
			ecertSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			ecert = ecertSecret.Data["cert.pem"]

			tlsSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", org1peer.Name), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			tlscert = tlsSecret.Data["cert.pem"]
		})

		When("signcert certificate is up for renewal and enrollment spec exists", func() {
			It("only renews the ecert when timer goes off", func() {
				// signcert certificates expire in 1 year (31536000s) from creation;
				// NumSecondsWarningPeriod has been set to 1 year - 60s to make
				// renewal occur when test runs

				By("setting status to warning", func() {
					Eventually(org1peer.PollForCRStatus).Should(Equal(current.Warning))
				})

				By("backing up old ecert signcert", func() {
					Eventually(func() bool {
						backup := GetBackup("ecert", org1peer.Name)
						if len(backup.List) > 0 {
							return backup.List[len(backup.List)-1].SignCerts == base64.StdEncoding.EncodeToString(ecert)
						}

						return false
					}).Should(Equal(true))
				})

				By("reenrolling identity and updating ecert certificate secret", func() {
					Eventually(func() bool {
						updatedEcertSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", org1peer.Name), metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						return bytes.Equal(ecert, updatedEcertSecret.Data["cert.pem"])
					}).Should(Equal(false))
				})

				By("not updating tls signcert secret", func() {
					Eventually(func() bool {
						updatedTLSSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", org1peer.Name), metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						return bytes.Equal(tlscert, updatedTLSSecret.Data["cert.pem"])
					}).Should(Equal(true))
				})

				By("returning to Deployed status as tls cert won't expire for 10 years", func() {
					Eventually(org1peer.PollForCRStatus).Should(Equal(current.Deployed))
				})

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
