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
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/ca/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("PVC Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPCA
		pvc       *corev1.PersistentVolumeClaim
	)

	BeforeEach(func() {
		var err error

		overrider = &override.Override{}
		pvc, err = util.GetPVCFromFile("../../../../../definitions/ca/pvc.yaml")
		Expect(err).NotTo(HaveOccurred())

		instance = &current.IBPCA{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "override1",
				Namespace: "namespace1",
			},
			Spec: current.IBPCASpec{
				Storage: &current.CAStorages{
					CA: &current.StorageSpec{
						Size:  "200M",
						Class: "not-manual",
					},
				},
				Region: "fakeregion",
				Zone:   "fakezone",
			},
		}
	})

	Context("creating a new pvc", func() {
		It("returns an error if improperly formatted value for size is used", func() {
			instance.Spec.Storage.CA.Size = "123b"
			err := overrider.PVC(instance, pvc, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("quantities must match the regular expression"))
		})

		It("overrides values in pvc, based on CA's instance spec", func() {
			Expect(pvc.Spec.StorageClassName).To(BeNil())
			err := overrider.PVC(instance, pvc, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			By("setting the labels for zone and region", func() {
				Expect(pvc.ObjectMeta.Labels["region"]).To(Equal("fakeregion"))
				Expect(pvc.ObjectMeta.Labels["zone"]).To(Equal("fakezone"))
			})

			By("setting the storage class name and size", func() {
				Expect(*pvc.Spec.StorageClassName).To(Equal("not-manual"))
				q := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
				Expect(q.String()).To(Equal("200M"))
			})

		})
	})
})
