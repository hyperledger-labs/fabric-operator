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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
)

const (
	IBPCAS = "ibpcas"
)

var (
	org1ca *helper.CA
)

var _ = Describe("ca", func() {
	BeforeEach(func() {
		Eventually(org1ca.PodIsRunning).Should((Equal(true)))

		ClearOperatorConfig()
	})

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	// Marked as Pending because slow clusters makes this test flaky as the CR
	// doesn't get created, so there is a small window of time to catch
	// its error status before it disappears.
	PContext("validate CR name when created", func() {
		BeforeEach(func() {
			Eventually(org1peer.PodIsRunning).Should((Equal(true)))
		})

		When("creating a CA with a pre-existing CR name", func() {
			It("puts CA in error phase", func() {
				org1ca2 := Org1CA2()
				helper.CreateCA(ibpCRClient, org1ca2.CR)

				Eventually(org1ca2.PollForCRStatus).Should((Equal(current.Error)))
			})
		})
	})
})

func Org1CA2() *helper.CA {
	cr := helper.Org1CACR(namespace, domain)
	// Set CR name to existing cr name for test
	cr.Name = "org1peer"

	return &helper.CA{
		Domain:     domain,
		Name:       cr.Name,
		Namespace:  namespace,
		WorkingDir: wd,
		CR:         cr,
		CRClient:   ibpCRClient,
		KClient:    kclient,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      cr.Name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}
