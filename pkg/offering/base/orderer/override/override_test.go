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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/override"
)

var _ = Describe("K8S Orderer Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPOrderer
	)

	BeforeEach(func() {
		overrider = &override.Override{
			Client: &mocks.Client{},
		}
		instance = &current.IBPOrderer{}
	})

	Context("Affnity", func() {
		BeforeEach(func() {
			instance = &current.IBPOrderer{
				Spec: current.IBPOrdererSpec{
					OrgName: "orderermsp",
					Arch:    []string{"test-arch"},
					Zone:    "dal",
					Region:  "us-south",
				},
			}
		})

		It("returns an proper affinity when arch is passed", func() {
			a := overrider.GetAffinity(instance)
			Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(Equal([]string{"test-arch"}))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Key).To(Equal("orgname"))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Values).To(Equal([]string{"orderermsp"}))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight).To(Equal(int32(100)))
		})

		It("returns an proper affinity when no arch is passed", func() {
			instance.Spec.Arch = []string{}
			a := overrider.GetAffinity(instance)
			Expect(a.NodeAffinity).NotTo(BeNil())
			Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(Equal([]string{"dal"}))
			Expect(a.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1].Values).To(Equal([]string{"us-south"}))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Key).To(Equal("orgname"))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchExpressions[0].Values).To(Equal([]string{"orderermsp"}))
			Expect(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight).To(Equal(int32(100)))
		})
	})

	Context("Deployment", func() {
		var (
			orderernode *current.IBPOrderer
		)

		nodenum := 2

		BeforeEach(func() {
			instance = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "os1",
				},
				Spec: current.IBPOrdererSpec{
					License: current.License{
						Accept: true,
					},
					OrgName:           "orderermsp",
					MSPID:             "orderermsp",
					OrdererType:       "solo",
					ExternalAddress:   "0.0.0.0",
					GenesisProfile:    "Initial",
					Storage:           &current.OrdererStorages{},
					Service:           &current.Service{},
					Images:            &current.OrdererImages{},
					Resources:         &current.OrdererResources{},
					SystemChannelName: "testchainid",
					Arch:              []string{"test-arch"},
					Zone:              "dal",
					Region:            "us-south",
					ClusterSize:       2,
					NodeNumber:        &nodenum,
					ClusterLocation: []current.IBPOrdererClusterLocation{
						current.IBPOrdererClusterLocation{
							Zone:   "dal1",
							Region: "us-south1",
						},
						current.IBPOrdererClusterLocation{
							Zone:   "dal2",
							Region: "us-south2",
						},
					},
				},
			}

			orderernode = &current.IBPOrderer{
				Spec: current.IBPOrdererSpec{
					License: current.License{
						Accept: true,
					},
					OrgName:           "orderermsp",
					MSPID:             "orderermsp",
					OrdererType:       "solo",
					ExternalAddress:   "0.0.0.0",
					GenesisProfile:    "Initial",
					Storage:           &current.OrdererStorages{},
					Service:           &current.Service{},
					Images:            &current.OrdererImages{},
					Resources:         &current.OrdererResources{},
					SystemChannelName: "testchainid",
					Arch:              []string{"test-arch"},
					Zone:              "dal",
					Region:            "us-south",
				},
			}
		})

		Context("Create overrides", func() {
			It("overides things correctly", func() {
				err := overrider.OrdererNode(instance, orderernode, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				Expect(orderernode.Spec.Zone).To(Equal(instance.Spec.ClusterLocation[1].Zone))
				Expect(orderernode.Spec.Region).To(Equal(instance.Spec.ClusterLocation[1].Region))
				Expect(orderernode.GetName()).To(Equal(instance.GetName() + "node2"))
				Expect(orderernode.Labels["parent"]).To(Equal(instance.Name))
			})
		})
	})
})
