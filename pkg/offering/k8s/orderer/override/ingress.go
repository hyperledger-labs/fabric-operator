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
	"github.com/IBM-Blockchain/fabric-operator/version"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *Override) Ingress(object v1.Object, ingress *networkingv1.Ingress, action resources.Action) error {
	instance := object.(*current.IBPOrderer)

	switch action {
	case resources.Create:
		return o.CreateIngress(instance, ingress)
	case resources.Update:
		return o.UpdateIngress(instance, ingress)
	}

	return nil
}

func (o *Override) CreateIngress(instance *current.IBPOrderer, ingress *networkingv1.Ingress) error {
	return o.CommonIngress(instance, ingress)
}

func (o *Override) UpdateIngress(instance *current.IBPOrderer, ingress *networkingv1.Ingress) error {
	return o.CommonIngress(instance, ingress)
}

func (o *Override) CommonIngress(instance *current.IBPOrderer, ingress *networkingv1.Ingress) error {

	ingressClass := "nginx"
	if instance.Spec.Ingress.Class != "" {
		ingressClass = instance.Spec.Ingress.Class
	}
	ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"] = ingressClass

	apihost := instance.Namespace + "-" + instance.Name + "-orderer" + "." + instance.Spec.Domain
	operationshost := instance.Namespace + "-" + instance.Name + "-operations" + "." + instance.Spec.Domain
	grpcwebhost := instance.Namespace + "-" + instance.Name + "-grpcweb" + "." + instance.Spec.Domain

	pathType := networkingv1.PathTypeImplementationSpecific
	ingress.Spec = networkingv1.IngressSpec{
		Rules: []networkingv1.IngressRule{
			networkingv1.IngressRule{
				Host: apihost,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							networkingv1.HTTPIngressPath{
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: instance.GetName(),
										Port: networkingv1.ServiceBackendPort{
											Name: "orderer-grpc",
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
			networkingv1.IngressRule{
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
			},
			networkingv1.IngressRule{
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
			},
		},
		TLS: []networkingv1.IngressTLS{
			networkingv1.IngressTLS{
				Hosts: []string{apihost},
			},
			networkingv1.IngressTLS{
				Hosts: []string{operationshost},
			},
			networkingv1.IngressTLS{
				Hosts: []string{grpcwebhost},
			},
		},
	}
	currentVer := version.String(instance.Spec.FabricVersion)
	if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_4_1) {
		adminhost := instance.Namespace + "-" + instance.Name + "-admin" + "." + instance.Spec.Domain
		adminIngressRule := []networkingv1.IngressRule{
			networkingv1.IngressRule{
				Host: adminhost,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							networkingv1.HTTPIngressPath{
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: instance.GetName(),
										Port: networkingv1.ServiceBackendPort{
											Name: "orderer-admin",
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
		}

		admintls := []networkingv1.IngressTLS{
			networkingv1.IngressTLS{
				Hosts: []string{adminhost},
			},
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, adminIngressRule...)
		ingress.Spec.TLS = append(ingress.Spec.TLS, admintls...)
	}
	return nil
}
