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
	"k8s.io/apimachinery/pkg/api/resource"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("Base Peer PVC Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPPeer
		pvc       *corev1.PersistentVolumeClaim
	)

	BeforeEach(func() {
		var err error

		pvc, err = util.GetPVCFromFile("../../../../../definitions/peer/pvc.yaml")
		Expect(err).NotTo(HaveOccurred())

		overrider = &override.Override{}
		instance = &current.IBPPeer{
			Spec: current.IBPPeerSpec{
				Zone:   "zone1",
				Region: "region1",
				Storage: &current.PeerStorages{
					Peer: &current.StorageSpec{
						Size:  "100m",
						Class: "manual",
					},
				},
			},
		}
	})

	Context("create", func() {
		It("overrides values based on spec", func() {
			err := overrider.PVC(instance, pvc, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			By("setting storage class", func() {
				Expect(pvc.Spec.StorageClassName).To(Equal(&instance.Spec.Storage.Peer.Class))
			})

			By("setting requested storage size", func() {
				expectedRequests, err := resource.ParseQuantity(instance.Spec.Storage.Peer.Size)
				Expect(err).NotTo(HaveOccurred())
				Expect(pvc.Spec.Resources.Requests).To(Equal(corev1.ResourceList{corev1.ResourceStorage: expectedRequests}))
			})

			By("setting zone labels", func() {
				Expect(pvc.ObjectMeta.Labels["zone"]).To(Equal(instance.Spec.Zone))
			})

			By("setting region labels", func() {
				Expect(pvc.ObjectMeta.Labels["region"]).To(Equal(instance.Spec.Region))
			})
		})

		It("sets class to manual if spec used local", func() {
			instance.Spec.Storage.Peer.Class = "manual"
			err := overrider.PVC(instance, pvc, resources.Create)
			Expect(err).NotTo(HaveOccurred())
			Expect(*pvc.Spec.StorageClassName).To(Equal("manual"))
		})

		It("returns an error if invalid value for size is used", func() {
			instance.Spec.Storage.Peer.Size = "10x"
			err := overrider.PVC(instance, pvc, resources.Create)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("quantities must match the regular expression"))
		})
	})
})
