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

package baseorderer_test

import (
	"context"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	cmocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	ordererinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer"
	managermocks "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/mocks"
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/mocks"
	orderermocks "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Base Orderer", func() {
	var (
		orderer        *baseorderer.Orderer
		instance       *current.IBPOrderer
		mockKubeClient *cmocks.Client
		nodeManager    *orderermocks.NodeManager

		ordererNodeMgr *managermocks.ResourceManager
		update         *mocks.Update
	)

	BeforeEach(func() {
		mockKubeClient = &cmocks.Client{}
		update = &mocks.Update{}
		instance = &current.IBPOrderer{
			Spec: current.IBPOrdererSpec{
				ClusterSize: 1,
				License: current.License{
					Accept: true,
				},
				OrdererType:       "etcdraft",
				SystemChannelName: "testchainid",
				OrgName:           "orderermsp",
				MSPID:             "orderermsp",
				ExternalAddress:   "ibporderer:7050",
				ImagePullSecrets:  []string{"regcred"},
			},
		}
		instance.Kind = "IBPOrderer"

		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *current.IBPOrderer:
				o := obj.(*current.IBPOrderer)
				o.Kind = "IBPOrderer"
				instance = o
			case *corev1.Service:
				o := obj.(*corev1.Service)
				o.Spec.Type = corev1.ServiceTypeNodePort
				o.Spec.Ports = append(o.Spec.Ports, corev1.ServicePort{
					Name: "orderer-api",
					TargetPort: intstr.IntOrString{
						IntVal: 7051,
					},
					NodePort: int32(7051),
				})
			}
			return nil
		}

		ordererNodeMgr = &managermocks.ResourceManager{}

		nodeManager = &orderermocks.NodeManager{}
		orderer = &baseorderer.Orderer{
			Client: mockKubeClient,
			Scheme: &runtime.Scheme{},
			Config: &config.Config{
				OrdererInitConfig: &ordererinit.Config{
					ConfigTxFile: "../../../../defaultconfig/orderer/configtx.yaml",
					OUFile:       "../../../../defaultconfig/orderer/ouconfig.yaml",
				},
			},

			NodeManager:        nodeManager,
			OrdererNodeManager: ordererNodeMgr,
		}
	})

	Context("Reconciles", func() {
		PIt("reconciles IBPOrderer", func() {
			_, err := orderer.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("check csr hosts", func() {
		It("adds csr hosts if not present", func() {
			instance = &current.IBPOrderer{
				Spec: current.IBPOrdererSpec{
					Secret: &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{},
					},
				},
			}
			hosts := []string{"test.com", "127.0.0.1"}
			orderer.CheckCSRHosts(instance, hosts)
			Expect(instance.Spec.Secret.Enrollment.TLS).NotTo(BeNil())
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR).NotTo(BeNil())
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts).To(Equal(hosts))
		})

		It("appends csr hosts if passed", func() {
			hostsCustom := []string{"custom.domain.com"}
			hosts := []string{"test.com", "127.0.0.1"}
			instance = &current.IBPOrderer{
				Spec: current.IBPOrdererSpec{
					Secret: &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{
							TLS: &current.Enrollment{
								CSR: &current.CSR{
									Hosts: hostsCustom,
								},
							},
						},
					},
				},
			}
			orderer.CheckCSRHosts(instance, hosts)
			Expect(instance.Spec.Secret.Enrollment.TLS).NotTo(BeNil())
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR).NotTo(BeNil())
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts).To(ContainElement(hostsCustom[0]))
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts).To(ContainElement(hosts[0]))
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts).To(ContainElement(hosts[1]))
		})
	})

	Context("images overrides", func() {
		var images *current.OrdererImages

		Context("using registry url", func() {
			BeforeEach(func() {
				images = &current.OrdererImages{
					OrdererInitImage: "ordererinitimage",
					OrdererInitTag:   "2.0.0",
					OrdererImage:     "ordererimage",
					OrdererTag:       "2.0.0",
					GRPCWebImage:     "grpcimage",
					GRPCWebTag:       "2.0.0",
				}
			})

			It("overrides images based with registry url and does not append more value on each call", func() {
				images.Override(images, "ghcr.io/ibm-blockchain/", "amd64")
				Expect(images.OrdererInitImage).To(Equal("ghcr.io/ibm-blockchain/ordererinitimage"))
				Expect(images.OrdererInitTag).To(Equal("2.0.0"))
				Expect(images.OrdererImage).To(Equal("ghcr.io/ibm-blockchain/ordererimage"))
				Expect(images.OrdererTag).To(Equal("2.0.0"))
				Expect(images.GRPCWebImage).To(Equal("ghcr.io/ibm-blockchain/grpcimage"))
				Expect(images.GRPCWebTag).To(Equal("2.0.0"))
			})

			It("overrides images based with registry url and does not append more value on each call", func() {
				images.Override(images, "ghcr.io/ibm-blockchain/images/", "s390")
				Expect(images.OrdererInitImage).To(Equal("ghcr.io/ibm-blockchain/images/ordererinitimage"))
				Expect(images.OrdererInitTag).To(Equal("2.0.0"))
				Expect(images.OrdererImage).To(Equal("ghcr.io/ibm-blockchain/images/ordererimage"))
				Expect(images.OrdererTag).To(Equal("2.0.0"))
				Expect(images.GRPCWebImage).To(Equal("ghcr.io/ibm-blockchain/images/grpcimage"))
				Expect(images.GRPCWebTag).To(Equal("2.0.0"))
			})
		})

		Context("using fully qualified path", func() {
			BeforeEach(func() {
				images = &current.OrdererImages{
					OrdererInitImage: "ghcr.io/ibm-blockchain/ordererinitimage",
					OrdererInitTag:   "2.0.0",
					OrdererImage:     "ghcr.io/ibm-blockchain/ordererimage",
					OrdererTag:       "2.0.0",
					GRPCWebImage:     "ghcr.io/ibm-blockchain/grpcimage",
					GRPCWebTag:       "2.0.0",
				}
			})

			It("keeps images and adds arch to tag", func() {
				images.Override(images, "", "amd64")
				Expect(images.OrdererInitImage).To(Equal("ghcr.io/ibm-blockchain/ordererinitimage"))
				Expect(images.OrdererInitTag).To(Equal("2.0.0"))
				Expect(images.OrdererImage).To(Equal("ghcr.io/ibm-blockchain/ordererimage"))
				Expect(images.OrdererTag).To(Equal("2.0.0"))
				Expect(images.GRPCWebImage).To(Equal("ghcr.io/ibm-blockchain/grpcimage"))
				Expect(images.GRPCWebTag).To(Equal("2.0.0"))
			})
		})
	})
})
