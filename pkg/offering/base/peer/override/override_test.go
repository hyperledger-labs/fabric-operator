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

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/override"
)

var _ = Describe("Base Peer Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPPeer
	)

	BeforeEach(func() {
		overrider = &override.Override{
			Client: &mocks.Client{},
		}
		instance = &current.IBPPeer{}
	})

	Context("Affnity", func() {
		BeforeEach(func() {
			instance = &current.IBPPeer{
				Spec: current.IBPPeerSpec{
					MSPID:  "peer-msp-id",
					Arch:   []string{"test-arch"},
					Zone:   "dal",
					Region: "us-south",
				},
			}
		})

		It("returns an proper affinity when arch is passed", func() {
			a := overrider.GetAffinity(instance)
			Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(Equal([]string{"test-arch"}))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Key).To(Equal("orgname"))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Values).To(Equal([]string{"peer-msp-id"}))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight).To(Equal(int32(100)))
		})

		It("returns an proper affinity when no arch is passed", func() {
			instance.Spec.Arch = []string{}
			a := overrider.GetAffinity(instance)
			Expect(a.NodeAffinity).NotTo(BeNil())
			Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(Equal([]string{"dal"}))
			Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1].Values).To(Equal([]string{"us-south"}))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Key).To(Equal("orgname"))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Values).To(Equal([]string{"peer-msp-id"}))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight).To(Equal(int32(100)))
		})
	})
})
