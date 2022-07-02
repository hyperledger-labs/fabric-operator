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

package override_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("Base Orderer Service Account Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPOrderer
		sa        *corev1.ServiceAccount
	)

	BeforeEach(func() {
		var err error

		sa, err = util.GetServiceAccountFromFile("../../../../../definitions/orderer/serviceaccount.yaml")
		Expect(err).NotTo(HaveOccurred())

		overrider = &override.Override{}
		instance = &current.IBPOrderer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "override1",
				Namespace: "namespace1",
			},
			Spec: current.IBPOrdererSpec{
				ImagePullSecrets: []string{"pullsecret1"},
			},
		}
	})

	Context("create", func() {
		It("overrides values in service account, based on Orderer's instance spec", func() {
			err := overrider.ServiceAccount(instance, sa, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			By("setting the image pull secret", func() {
				Expect(sa.ImagePullSecrets[1].Name).To(Equal(instance.Spec.ImagePullSecrets[0]))
			})
		})

		Context("update", func() {
			It("overrides values in service account, based on Orderer's instance spec", func() {
				err := overrider.ServiceAccount(instance, sa, resources.Update)
				Expect(err).NotTo(HaveOccurred())

				By("setting the image pull secret", func() {
					Expect(sa.ImagePullSecrets[1].Name).To(Equal(instance.Spec.ImagePullSecrets[0]))
				})
			})
		})
	})
})
