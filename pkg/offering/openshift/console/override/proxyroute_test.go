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
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/console/override"
	routev1 "github.com/openshift/api/route/v1"
)

var _ = Describe("Openshift Proxy Route Overrides", func() {
	var (
		route     *routev1.Route
		overrider *override.Override
		instance  *current.IBPConsole
	)

	BeforeEach(func() {
		route = &routev1.Route{}
		overrider = &override.Override{}

		instance = &current.IBPConsole{
			Spec: current.IBPConsoleSpec{
				NetworkInfo: &current.NetworkInfo{
					Domain: "test-domain",
				},
			},
		}
		instance.Name = "route1"
		instance.Namespace = "testNS"
	})

	Context("create", func() {
		It("appropriately overrides the respective values", func() {
			err := overrider.ProxyRoute(instance, route, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			Expect(route.Name).To(Equal(fmt.Sprintf("%s-proxy", instance.Name)))
			Expect(route.Spec.Host).To(Equal("testNS-route1-proxy.test-domain"))
			Expect(route.Spec.To.Kind).To(Equal("Service"))
			Expect(route.Spec.To.Name).To(Equal("route1"))
			Expect(*route.Spec.To.Weight).To(Equal(int32(100)))
			Expect(route.Spec.Port.TargetPort).To(Equal(intstr.FromString("optools")))
			Expect(route.Spec.TLS.Termination).To(Equal(routev1.TLSTerminationPassthrough))
		})
	})
})
