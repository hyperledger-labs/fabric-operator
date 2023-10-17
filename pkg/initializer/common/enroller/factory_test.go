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

package enroller_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller/mocks"

	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Enroller factory", func() {
	var instance *mocks.CryptoInstance

	BeforeEach(func() {
		instance = &mocks.CryptoInstance{}
	})

	Context("software enroller", func() {
		It("returns software type enroller", func() {
			e, err := enroller.Factory(&current.Enrollment{}, &mocks.Client{}, instance, "/tmp", &runtime.Scheme{}, []byte("cert"), enroller.HSMEnrollJobTimeouts{})
			Expect(err).NotTo(HaveOccurred())

			_, sw := e.Enroller.(*enroller.SWEnroller)
			Expect(sw).To(Equal(true))
		})
	})

	Context("HSM", func() {
		BeforeEach(func() {
			instance.IsHSMEnabledReturns(true)
		})

		Context("sidecar enroller", func() {
			It("returns sidecar type enroller", func() {
				e, err := enroller.Factory(&current.Enrollment{}, &mocks.Client{}, instance, "/tmp", &runtime.Scheme{}, []byte("cert"), enroller.HSMEnrollJobTimeouts{})
				Expect(err).NotTo(HaveOccurred())

				_, hsm := e.Enroller.(*enroller.HSMEnroller)
				Expect(hsm).To(Equal(true))
			})
		})

		Context("proxy enroller", func() {
			BeforeEach(func() {
				instance.UsingHSMProxyReturns(true)
			})

			It("returns sidecar type enroller", func() {
				e, err := enroller.Factory(&current.Enrollment{}, &mocks.Client{}, instance, "/tmp", &runtime.Scheme{}, []byte("cert"), enroller.HSMEnrollJobTimeouts{})
				Expect(err).NotTo(HaveOccurred())

				_, hsm := e.Enroller.(*enroller.HSMProxyEnroller)
				Expect(hsm).To(Equal(true))
			})
		})
	})
})
