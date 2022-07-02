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

package cclauncher_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/yaml"
)

var _ = Describe("chaincode launcher", func() {
	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("V2 Peer", func() {
		It("creates peer resources", func() {
			By("creating deployment that contains four containers", func() {
				dep, err := kclient.AppsV1().Deployments(namespace).Get(context.TODO(), org1peer.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				Expect(dep.Spec.Template.Spec.Containers).To(HaveLen(4))
			})

			By("creating config map with external builders", func() {
				cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), fmt.Sprintf("%s-config", org1peer.Name), metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				v2Core := &v2.Core{}
				coreBytes := cm.BinaryData["core.yaml"]
				err = yaml.Unmarshal(coreBytes, v2Core)
				Expect(err).NotTo(HaveOccurred())

				extBuilder := v2.ExternalBuilder{
					Path: "/usr/local",
					Name: "ibp-builder",
					EnvironmentWhiteList: []string{
						"IBP_BUILDER_ENDPOINT",
						"IBP_BUILDER_SHARED_DIR",
					},
					PropogateEnvironment: []string{
						"IBP_BUILDER_ENDPOINT",
						"IBP_BUILDER_SHARED_DIR",
						"PEER_NAME",
					},
				}
				Expect(v2Core.Chaincode.ExternalBuilders).To(ContainElement(extBuilder))
			})

			By("setting builders environment variables", func() {
				dep, err := kclient.AppsV1().Deployments(namespace).Get(context.TODO(), org1peer.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				peerContainer := dep.Spec.Template.Spec.Containers[0]

				dirEnvVar := corev1.EnvVar{
					Name:  "IBP_BUILDER_SHARED_DIR",
					Value: "/cclauncher",
				}
				Expect(peerContainer.Env).To(ContainElement(dirEnvVar))

				endpointEnvVar := corev1.EnvVar{
					Name:  "IBP_BUILDER_ENDPOINT",
					Value: "127.0.0.1:11111",
				}
				Expect(peerContainer.Env).To(ContainElement(endpointEnvVar))
			})
		})
	})
})
