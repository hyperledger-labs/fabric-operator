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

package override

import (
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *Override) Ingress(object v1.Object, ingress *networkingv1.Ingress, action resources.Action) error {
	instance := object.(*current.IBPPeer)

	switch action {
	case resources.Create:
		return o.CreateIngress(instance, ingress)
	case resources.Update:
		return o.UpdateIngress(instance, ingress)
	}

	return nil
}

func (o *Override) CreateIngress(instance *current.IBPPeer, ingress *networkingv1.Ingress) error {
	return o.CommonIngress(instance, ingress)
}

func (o *Override) UpdateIngress(instance *current.IBPPeer, ingress *networkingv1.Ingress) error {
	return o.CommonIngress(instance, ingress)
}

func (o *Override) CommonIngress(instance *current.IBPPeer, ingress *networkingv1.Ingress) error {
	ingressClass := "nginx"
	if instance.Spec.Ingress.Class != "" {
		ingressClass = instance.Spec.Ingress.Class
	}
	ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"] = ingressClass

	apihost := instance.Namespace + "-" + instance.Name + "-peer" + "." + instance.Spec.Domain
	operationshost := instance.Namespace + "-" + instance.Name + "-operations" + "." + instance.Spec.Domain
	grpcwebhost := instance.Namespace + "-" + instance.Name + "-grpcweb" + "." + instance.Spec.Domain

	pathType := networkingv1.PathTypeImplementationSpecific
	ingress.Spec = networkingv1.IngressSpec{
		Rules: []networkingv1.IngressRule{
			{
				Host: apihost,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
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
			},
			{
				Host: operationshost,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
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
			},
			{
				Host: grpcwebhost,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
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
			},
		},
		TLS: []networkingv1.IngressTLS{
			{
				Hosts: []string{apihost},
			},
			{
				Hosts: []string{operationshost},
			},
			{
				Hosts: []string{grpcwebhost},
			},
		},
	}

	return nil
}
