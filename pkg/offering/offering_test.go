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

package offering_test

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Peer configuration", func() {
	Context("get type", func() {

		It("returns type OPENSHIFT", func() {
			t, err := offering.GetType("OPENSHIFT")
			Expect(err).To(BeNil())
			Expect(t).To(Equal(offering.OPENSHIFT))

			t, err = offering.GetType("openshift")
			Expect(err).To(BeNil())
			Expect(t).To(Equal(offering.OPENSHIFT))
		})

		It("returns an error for unrecongized input", func() {
			_, err := offering.GetType("foo")
			Expect(err).NotTo(BeNil())
		})

		It("returns type K8S for input of k8s", func() {
			t, err := offering.GetType("K8S")
			Expect(err).To(BeNil())
			Expect(t).To(Equal(offering.K8S))

			t, err = offering.GetType("k8s")
			Expect(err).To(BeNil())
			Expect(t).To(Equal(offering.K8S))
		})
	})
})
