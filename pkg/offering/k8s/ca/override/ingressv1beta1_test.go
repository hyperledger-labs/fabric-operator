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
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/ca/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("K8s CA Ingress Overrides", func() {
	var (
		err       error
		overrider *override.Override
		instance  *current.IBPCA
		ingress   *networkingv1beta1.Ingress
		cahost    string
		operhost  string
	)

	BeforeEach(func() {
		overrider = &override.Override{}
		instance = &current.IBPCA{
			Spec: current.IBPCASpec{
				Domain: "test.domain",
			},
		}
		ingress, err = util.GetIngressv1beta1FromFile("../../../../../definitions/ca/ingressv1beta1.yaml")
		Expect(err).NotTo(HaveOccurred())

		cahost = instance.Namespace + "-" + instance.Name + "-ca" + "." + instance.Spec.Domain
		operhost = instance.Namespace + "-" + instance.Name + "-operations" + "." + instance.Spec.Domain
	})

	Context("create", func() {
		It("appropriately overrides the respective values for ingress", func() {
			err := overrider.Ingressv1beta1(instance, ingress, resources.Create)
			Expect(err).NotTo(HaveOccurred())

			By("setting rule", func() {
				Expect(ingress.Spec.Rules).To(HaveLen(2))
				Expect(ingress.Spec.Rules[0]).To(Equal(networkingv1beta1.IngressRule{
					Host: cahost,
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								networkingv1beta1.HTTPIngressPath{
									Backend: networkingv1beta1.IngressBackend{
										ServiceName: instance.GetName(),
										ServicePort: intstr.FromString("http"),
									},
									Path: "/",
								},
							},
						},
					},
				}))
				Expect(ingress.Spec.Rules[1]).To(Equal(networkingv1beta1.IngressRule{
					Host: operhost,
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								networkingv1beta1.HTTPIngressPath{
									Backend: networkingv1beta1.IngressBackend{
										ServiceName: instance.GetName(),
										ServicePort: intstr.FromString("operations"),
									},
									Path: "/",
								},
							},
						},
					},
				}))
			})

			By("setting TLS hosts", func() {
				Expect(ingress.Spec.TLS).To(HaveLen(2))
				Expect(ingress.Spec.TLS[0].Hosts).To(Equal([]string{cahost}))
				Expect(ingress.Spec.TLS[1].Hosts).To(Equal([]string{operhost}))
			})
		})
	})
})
