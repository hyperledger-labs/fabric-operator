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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/orderer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("K8s Orderer Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPOrderer
	)

	BeforeEach(func() {
		overrider = &override.Override{}
	})

	Context("Ingress", func() {
		var (
			ingress *networkingv1.Ingress
		)

		BeforeEach(func() {
			var err error

			ingress, err = util.GetIngressFromFile("../../../../../definitions/orderer/ingress.yaml")
			Expect(err).NotTo(HaveOccurred())

			instance = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress1",
					Namespace: "namespace1",
				},
				Spec: current.IBPOrdererSpec{
					Domain: "domain1",
				},
			}
		})

		When("creating ingress", func() {
			It("sets appropriate values", func() {
				err := overrider.Ingress(instance, ingress, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				VerifyIngressCommonOverrides(instance, ingress)
			})
		})

		When("creating ingress with custom class", func() {
			It("sets appropriate values", func() {
				instance.Spec.Ingress = current.Ingress{
					Class: "custom",
				}
				err := overrider.Ingress(instance, ingress, resources.Create)
				Expect(err).NotTo(HaveOccurred())
				VerifyIngressCommonOverrides(instance, ingress)
			})
		})

		When("updating ingress", func() {
			It("sets appropriate values", func() {
				err := overrider.Ingress(instance, ingress, resources.Update)
				Expect(err).NotTo(HaveOccurred())
				VerifyIngressCommonOverrides(instance, ingress)
			})
		})

		When("updating ingress with custom class", func() {
			It("sets appropriate values", func() {
				instance.Spec.Ingress = current.Ingress{
					Class: "custom",
				}
				err := overrider.Ingress(instance, ingress, resources.Update)
				Expect(err).NotTo(HaveOccurred())
				VerifyIngressCommonOverrides(instance, ingress)
			})
		})
	})
})

func VerifyIngressCommonOverrides(instance *current.IBPOrderer, ingress *networkingv1.Ingress) {
	By("setting annotation for custom ingress class", func() {
		if instance.Spec.Ingress.Class != "" {
			Expect(ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"]).To(Equal(instance.Spec.Ingress.Class))
		} else {
			Expect(ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"]).To(Equal("nginx"))
		}
	})

	By("setting api host in rules host", func() {
		Expect(ingress.Spec.Rules[0].Host).To(Equal(instance.Namespace + "-" + instance.Name + "-orderer" + "." + instance.Spec.Domain))
	})

	By("setting api tls host", func() {
		Expect(ingress.Spec.TLS[0].Hosts).To(Equal([]string{instance.Namespace + "-" + instance.Name + "-orderer" + "." + instance.Spec.Domain}))
	})

	By("setting backend service name", func() {
		Expect(ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name).To(Equal(instance.Name))
	})

	By("setting backend service port", func() {
		Expect(ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Name).To(Equal("orderer-grpc"))
	})

	By("setting operations host in rules host", func() {
		Expect(ingress.Spec.Rules[1].Host).To(Equal(instance.Namespace + "-" + instance.Name + "-operations" + "." + instance.Spec.Domain))
	})

	By("setting operations tls host", func() {
		Expect(ingress.Spec.TLS[1].Hosts).To(Equal([]string{instance.Namespace + "-" + instance.Name + "-operations" + "." + instance.Spec.Domain}))
	})

	By("setting backend service name", func() {
		Expect(ingress.Spec.Rules[1].HTTP.Paths[0].Backend.Service.Name).To(Equal(instance.Name))
	})

	By("setting backend service port", func() {
		Expect(ingress.Spec.Rules[1].HTTP.Paths[0].Backend.Service.Port.Name).To(Equal("operations"))
	})

	By("setting operations host in rules host", func() {
		Expect(ingress.Spec.Rules[2].Host).To(Equal(instance.Namespace + "-" + instance.Name + "-grpcweb" + "." + instance.Spec.Domain))
	})

	By("setting operations tls host", func() {
		Expect(ingress.Spec.TLS[2].Hosts).To(Equal([]string{instance.Namespace + "-" + instance.Name + "-grpcweb" + "." + instance.Spec.Domain}))
	})

	By("setting backend service name", func() {
		Expect(ingress.Spec.Rules[2].HTTP.Paths[0].Backend.Service.Name).To(Equal(instance.Name))
	})

	By("setting backend service port", func() {
		Expect(ingress.Spec.Rules[2].HTTP.Paths[0].Backend.Service.Port.Name).To(Equal("grpcweb"))
	})
}
