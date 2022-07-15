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

package action_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"

	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/action"
	"github.com/IBM-Blockchain/fabric-operator/pkg/action/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
)

var _ = Describe("actions", func() {
	var (
		client *controllermocks.Client
	)

	BeforeEach(func() {
		client = &controllermocks.Client{}
	})

	Context("restart", func() {

		It("returns an error if failed to get deployment", func() {
			client.GetReturns(errors.New("get error"))
			err := action.Restart(client, "name", "namespace")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get error"))
		})

		It("returns error if fails to patch deployment", func() {
			client.PatchReturns(errors.New("patch error"))
			err := action.Restart(client, "name", "namespace")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("patch error"))
		})

		It("restarts deployment by updating annotations", func() {
			err := action.Restart(client, "name", "namespace")
			Expect(err).NotTo(HaveOccurred())
			_, dep, _, _ := client.PatchArgsForCall(0)
			deployment := dep.(*appsv1.Deployment)
			annotation := deployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"]
			Expect(annotation).NotTo(Equal(""))

		})
	})

	Context("reenroll", func() {
		var (
			instance   *mocks.ReenrollInstance
			reenroller *mocks.Reenroller
		)

		BeforeEach(func() {
			reenroller = &mocks.Reenroller{}
			instance = &mocks.ReenrollInstance{}
		})

		It("returns an error if pod deletion fails", func() {
			reenroller.RenewCertReturns(errors.New("renew failed"))
			err := action.Reenroll(reenroller, client, common.ECERT, instance, true)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(ContainSubstring("renew failed")))
		})

		It("reenrolls ecert successfully", func() {
			err := action.Reenroll(reenroller, client, common.ECERT, instance, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(reenroller.RenewCertCallCount()).To(Equal(1))
		})

		It("reenrolls TLS successfully", func() {
			err := action.Reenroll(reenroller, client, common.TLS, instance, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(reenroller.RenewCertCallCount()).To(Equal(1))
		})
	})
})
