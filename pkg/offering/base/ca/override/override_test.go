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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/ca/override"
)

var _ = Describe("Base CA Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPCA
	)

	BeforeEach(func() {
		overrider = &override.Override{}
	})

	Context("Affnity", func() {
		BeforeEach(func() {
			instance = &current.IBPCA{
				Spec: current.IBPCASpec{
					Arch:   []string{"test-arch"},
					Zone:   "dal",
					Region: "us-south",
				},
			}
			instance.Name = "ca1"
		})

		It("returns an proper affinity when arch is passed", func() {
			instance.Spec.Arch = []string{"test-arch"}
			a := overrider.GetAffinity(instance)

			By("setting node affinity", func() {
				Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(Equal([]string{"test-arch"}))
				Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1].Values).To(Equal([]string{"dal"}))
				Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[2].Values).To(Equal([]string{"us-south"}))
			})

			By("setting pod anti affinity", func() {
				Expect(a.PodAntiAffinity).NotTo(BeNil())
				Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Values).To(Equal([]string{"ca1"}))
			})
		})

		It("returns a proper affinity when no arch is passed", func() {
			instance.Spec.Arch = []string{}
			a := overrider.GetAffinity(instance)
			Expect(a.NodeAffinity).NotTo(BeNil())

			By("setting node affinity", func() {
				Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(Equal([]string{"dal"}))
				Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1].Values).To(Equal([]string{"us-south"}))
			})

			By("setting pod anti affinity", func() {
				Expect(a.PodAntiAffinity).NotTo(BeNil())
				Expect(len(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(2))
				Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Values).To(Equal([]string{"ca1"}))
			})
		})

		It("returns a proper affinity for postgres CA", func() {
			caOverrides := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					DB: &v1.CAConfigDB{
						Type: "postgres",
					},
				},
			}
			bytes, err := json.Marshal(caOverrides)
			Expect(err).NotTo(HaveOccurred())
			rawMessage := json.RawMessage(bytes)
			instance.Spec.ConfigOverride = &current.ConfigOverride{
				CA: &runtime.RawExtension{Raw: rawMessage},
			}

			a := overrider.GetAffinity(instance)

			By("not setting zone or region in node affinity", func() {
				Expect(a.NodeAffinity).NotTo(BeNil())
				Expect(len(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions)).To(Equal(1))
				Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(Equal([]string{"test-arch"}))
			})

			By("setting pod anti affinity with hostname topology key", func() {
				Expect(a.PodAntiAffinity).NotTo(BeNil())
				Expect(len(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(3))
				Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Values).To(Equal([]string{"ca1"}))
				Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[2].PodAffinityTerm.TopologyKey).To(Equal("kubernetes.io/hostname"))
			})
		})
	})
})
