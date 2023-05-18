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
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (o *Override) Ingressv1beta1(object v1.Object, ingress *networkingv1beta1.Ingress, action resources.Action) error {
	instance := object.(*current.IBPConsole)

	switch action {
	case resources.Create:
		return o.CreateIngressv1beta1(instance, ingress)
	case resources.Update:
		return o.UpdateIngressv1beta1(instance, ingress)
	}

	return nil
}

func (o *Override) CreateIngressv1beta1(instance *current.IBPConsole, ingress *networkingv1beta1.Ingress) error {
	return o.CommonIngressv1beta1(instance, ingress)
}

func (o *Override) UpdateIngressv1beta1(instance *current.IBPConsole, ingress *networkingv1beta1.Ingress) error {
	return o.CommonIngressv1beta1(instance, ingress)
}

func (o *Override) CommonIngressv1beta1(instance *current.IBPConsole, ingress *networkingv1beta1.Ingress) error {
	ingressClass := "nginx"
	if instance.Spec.Ingress.Class != "" {
		ingressClass = instance.Spec.Ingress.Class
	}
	ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"] = ingressClass

	consolehost := instance.Namespace + "-" + instance.Name + "-console" + "." + instance.Spec.NetworkInfo.Domain

	ingress.Spec = networkingv1beta1.IngressSpec{
		Rules: []networkingv1beta1.IngressRule{
			{
				Host: consolehost,
				IngressRuleValue: networkingv1beta1.IngressRuleValue{
					HTTP: &networkingv1beta1.HTTPIngressRuleValue{
						Paths: []networkingv1beta1.HTTPIngressPath{
							{
								Backend: networkingv1beta1.IngressBackend{
									ServiceName: instance.GetName(),
									ServicePort: intstr.FromString("optools"),
								},
								Path: "/",
							},
						},
					},
				},
			},
		},
		TLS: []networkingv1beta1.IngressTLS{
			{
				Hosts: []string{consolehost},
			},
		},
	}

	return nil
}
