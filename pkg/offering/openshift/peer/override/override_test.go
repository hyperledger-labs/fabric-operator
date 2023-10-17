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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/peer/override"
	routev1 "github.com/openshift/api/route/v1"
)

var _ = Describe("Openshift Peer Overrides", func() {
	var (
		route     *routev1.Route
		overrider *override.Override
		instance  *current.IBPPeer
	)

	BeforeEach(func() {
		route = &routev1.Route{}
		overrider = &override.Override{}

		instance = &current.IBPPeer{
			Spec: current.IBPPeerSpec{
				Domain: "test-domain",
			},
		}
		instance.Name = "route1"
		instance.Namespace = "testNS"
	})

	Context("Peer Route", func() {
		When("creating a new Peer Route", func() {
			It("appropriately overrides the respective values", func() {
				err := overrider.PeerRoute(instance, route, resources.Create)
				Expect(err).NotTo(HaveOccurred())

				Expect(route.Name).To(Equal(fmt.Sprintf("%s-peer", instance.Name)))
				Expect(route.Spec.Host).To(Equal("testNS-route1-peer.test-domain"))
				Expect(route.Spec.To.Kind).To(Equal("Service"))
				Expect(route.Spec.To.Name).To(Equal(instance.Name))
				Expect(*route.Spec.To.Weight).To(Equal(int32(100)))
				Expect(route.Spec.Port.TargetPort).To(Equal(intstr.FromString("peer-api")))
				Expect(route.Spec.TLS.Termination).To(Equal(routev1.TLSTerminationPassthrough))
			})
		})
	})

	Context("Operation Route", func() {
		When("creating a new Operation Route", func() {
			It("appropriately overrides the respective values", func() {
				err := overrider.OperationsRoute(instance, route, resources.Create)
				Expect(err).NotTo(HaveOccurred())

				Expect(route.Name).To(Equal(fmt.Sprintf("%s-operations", instance.Name)))
				Expect(route.Spec.Host).To(Equal("testNS-route1-operations.test-domain"))
				Expect(route.Spec.To.Kind).To(Equal("Service"))
				Expect(route.Spec.To.Name).To(Equal(instance.Name))
				Expect(*route.Spec.To.Weight).To(Equal(int32(100)))
				Expect(route.Spec.Port.TargetPort).To(Equal(intstr.FromString("operations")))
				Expect(route.Spec.TLS.Termination).To(Equal(routev1.TLSTerminationPassthrough))
			})
		})
	})

	Context("GPRC Route", func() {
		When("creating a new GRPC Route", func() {
			It("appropriately overrides the respective values", func() {
				err := overrider.PeerGRPCRoute(instance, route, resources.Create)
				Expect(err).NotTo(HaveOccurred())

				Expect(route.Name).To(Equal(fmt.Sprintf("%s-grpcweb", instance.Name)))
				Expect(route.Spec.Host).To(Equal("testNS-route1-grpcweb.test-domain"))
				Expect(route.Spec.To.Kind).To(Equal("Service"))
				Expect(route.Spec.To.Name).To(Equal(instance.Name))
				Expect(*route.Spec.To.Weight).To(Equal(int32(100)))
				Expect(route.Spec.Port.TargetPort).To(Equal(intstr.FromString("grpcweb")))
				Expect(route.Spec.TLS.Termination).To(Equal(routev1.TLSTerminationPassthrough))
			})
		})
	})
})
