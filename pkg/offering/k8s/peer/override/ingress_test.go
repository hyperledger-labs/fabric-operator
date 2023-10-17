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
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/peer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("K8s Peer Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPPeer
	)

	BeforeEach(func() {
		overrider = &override.Override{}
	})

	Context("Ingress", func() {
		var (
			err            error
			ingress        *networkingv1.Ingress
			apihost        string
			operationshost string
			grpcwebhost    string
		)

		BeforeEach(func() {
			ingress, err = util.GetIngressFromFile("../../../../../definitions/peer/ingress.yaml")
			Expect(err).NotTo(HaveOccurred())

			instance = &current.IBPPeer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress1",
					Namespace: "namespace1",
				},
				Spec: current.IBPPeerSpec{
					Domain: "domain1",
				},
			}

			apihost = instance.Namespace + "-" + instance.Name + "-peer" + "." + instance.Spec.Domain
			operationshost = instance.Namespace + "-" + instance.Name + "-operations" + "." + instance.Spec.Domain
			grpcwebhost = instance.Namespace + "-" + instance.Name + "-grpcweb" + "." + instance.Spec.Domain
		})

		When("creating ingress", func() {
			It("sets appropriate values", func() {
				err := overrider.Ingress(instance, ingress, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				VerifyIngressCommonOverrides(instance, ingress, apihost, operationshost, grpcwebhost)
			})
		})

		When("creating ingress with custom class", func() {
			It("sets appropriate values", func() {
				instance.Spec.Ingress = current.Ingress{
					Class: "custom",
				}
				err := overrider.Ingress(instance, ingress, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				VerifyIngressCommonOverrides(instance, ingress, apihost, operationshost, grpcwebhost)
			})
		})

		When("updating ingress", func() {
			It("sets appropriate values", func() {
				err := overrider.Ingress(instance, ingress, resources.Update)
				Expect(err).NotTo(HaveOccurred())
				VerifyIngressCommonOverrides(instance, ingress, apihost, operationshost, grpcwebhost)
			})
		})

		When("updating ingress with custom class", func() {
			It("sets appropriate values", func() {
				instance.Spec.Ingress = current.Ingress{
					Class: "custom",
				}
				err := overrider.Ingress(instance, ingress, resources.Update)
				Expect(err).NotTo(HaveOccurred())
				VerifyIngressCommonOverrides(instance, ingress, apihost, operationshost, grpcwebhost)
			})
		})
	})
})

func VerifyIngressCommonOverrides(instance *current.IBPPeer, ingress *networkingv1.Ingress, apihost, operationshost, grpcwebhost string) {
	By("setting annotation for custom ingress class", func() {
		if instance.Spec.Ingress.Class != "" {
			Expect(ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"]).To(Equal(instance.Spec.Ingress.Class))
		} else {
			Expect(ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"]).To(Equal("nginx"))
		}
	})
	By("setting rules", func() {
		pathType := networkingv1.PathTypeImplementationSpecific
		Expect(ingress.Spec.Rules).To(HaveLen(3))
		Expect(ingress.Spec.Rules[0]).To(Equal(networkingv1.IngressRule{
			Host: apihost,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						networkingv1.HTTPIngressPath{
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: instance.GetName(),
									Port: networkingv1.ServiceBackendPort{
										Name: "peer-api",
									},
								},
							},
							Path:     "/",
							PathType: &pathType,
						},
					},
				},
			},
		}))
		Expect(ingress.Spec.Rules[1]).To(Equal(networkingv1.IngressRule{
			Host: operationshost,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						networkingv1.HTTPIngressPath{
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: instance.GetName(),
									Port: networkingv1.ServiceBackendPort{
										Name: "operations",
									},
								},
							},
							Path:     "/",
							PathType: &pathType,
						},
					},
				},
			},
		}))
		Expect(ingress.Spec.Rules[2]).To(Equal(networkingv1.IngressRule{
			Host: grpcwebhost,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						networkingv1.HTTPIngressPath{
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: instance.GetName(),
									Port: networkingv1.ServiceBackendPort{
										Name: "grpcweb",
									},
								},
							},
							Path:     "/",
							PathType: &pathType,
						},
					},
				},
			},
		}))
	})

	By("setting TLS hosts", func() {
		Expect(ingress.Spec.TLS).To(HaveLen(3))
		Expect(ingress.Spec.TLS[0].Hosts).To(Equal([]string{apihost}))
		Expect(ingress.Spec.TLS[1].Hosts).To(Equal([]string{operationshost}))
		Expect(ingress.Spec.TLS[2].Hosts).To(Equal([]string{grpcwebhost}))
	})
}
