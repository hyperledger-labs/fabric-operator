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

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("Base Orderer Service Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPOrderer
		service   *corev1.Service
	)

	BeforeEach(func() {
		var err error

		service, err = util.GetServiceFromFile("../../../../../definitions/orderer/service.yaml")
		Expect(err).NotTo(HaveOccurred())

		overrider = &override.Override{}
		instance = &current.IBPOrderer{
			Spec: current.IBPOrdererSpec{
				Service: &current.Service{
					Type: corev1.ServiceTypeNodePort,
				},
			},
		}
	})

	Context("create", func() {
		It("overrides values based on spec", func() {
			err := overrider.Service(instance, service, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			By("setting service type", func() {
				Expect(service.Spec.Type).To(Equal(instance.Spec.Service.Type))
			})
		})
	})
})
