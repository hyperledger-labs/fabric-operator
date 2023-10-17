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

package ibporderer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	orderermocks "github.com/IBM-Blockchain/fabric-operator/controllers/ibporderer/mocks"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v1"
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	yaml "sigs.k8s.io/yaml"
)

var _ = Describe("predicate", func() {
	var (
		reconciler           *ReconcileIBPOrderer
		instance             *current.IBPOrderer
		mockKubeClient       *mocks.Client
		mockOrdererReconcile *orderermocks.OrdererReconcile
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{
			ListStub: func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
				switch obj.(type) {
				case *corev1.NodeList:
					nodeList := obj.(*corev1.NodeList)
					node := corev1.Node{}
					node.Labels = map[string]string{}
					node.Labels["topology.kubernetes.io/zone"] = "dal"
					node.Labels["topology.kubernetes.io/region"] = "us-south"
					nodeList.Items = append(nodeList.Items, node)
				case *current.IBPOrdererList:
					ordererList := obj.(*current.IBPOrdererList)
					o1 := current.IBPOrderer{}
					o1.Name = "test-orderer1"
					o2 := current.IBPOrderer{}
					o2.Name = "test-orderer2"
					o3 := current.IBPOrderer{}
					o3.Name = "test-orderer2"
					ordererList.Items = []current.IBPOrderer{o1, o2, o3}
				}
				return nil
			},
		}

		mockOrdererReconcile = &orderermocks.OrdererReconcile{}
		nodeNumber := 1
		instance = &current.IBPOrderer{
			Spec: current.IBPOrdererSpec{
				ClusterSize: 3,
				NodeNumber:  &nodeNumber,
			},
		}

		reconciler = &ReconcileIBPOrderer{
			Offering: mockOrdererReconcile,
			client:   mockKubeClient,
			scheme:   &runtime.Scheme{},
			update:   map[string][]Update{},
			mutex:    &sync.Mutex{},
		}
	})

	Context("create func predicate", func() {
		var (
			orderer *current.IBPOrderer
			e       event.CreateEvent
		)

		BeforeEach(func() {
			orderer = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.GetName(),
				},
				Status: current.IBPOrdererStatus{
					CRStatus: current.CRStatus{
						Type: current.Deployed,
					},
				},
			}
			e = event.CreateEvent{
				Object: orderer,
			}
		})

		It("sets update flags to false if instance has status type and a create event is detected but no spec changes are detected", func() {
			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(true))

			Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{
				specUpdated:      false,
				overridesUpdated: false,
			}))
		})

		It("sets update flags to true if instance has status type and a create event is detected and spec changes detected", func() {
			configOverride := &config.Orderer{
				Orderer: v1.Orderer{
					General: v1.General{
						LedgerType: "type1",
					},
				},
			}
			configBytes, err := json.Marshal(configOverride)
			Expect(err).NotTo(HaveOccurred())
			spec := current.IBPOrdererSpec{
				ImagePullSecrets: []string{"pullsecret1"},
				ConfigOverride:   &runtime.RawExtension{Raw: configBytes},
			}
			binaryData, err := yaml.Marshal(spec)
			Expect(err).NotTo(HaveOccurred())

			mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *corev1.ConfigMap:
					o := obj.(*corev1.ConfigMap)
					o.BinaryData = map[string][]byte{
						"spec": binaryData,
					}
				}
				return nil
			}
			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(true))

			Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{
				specUpdated:      true,
				overridesUpdated: true,
			}))
		})

		It("does not trigger update if instance does not have status type and a create event is detected", func() {
			orderer.Status.Type = ""

			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(true))

			Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{}))
		})

		It("returns true  but does not trigger update if new instance's name is unique to one IBPOrderer in list of IBPOrderers", func() {
			orderer.Status.Type = ""
			orderer.Name = "test-orderer1"

			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{}))
		})

		It("returns false if new instance's name already exists for another IBPOrderer custom resource", func() {
			orderer.Status.Type = ""
			orderer.Name = "test-orderer2"

			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(false))
			Expect(orderer.Status.Type).To(Equal(current.Error))
		})

		Context("secret created", func() {
			var (
				cert *corev1.Secret
				e    event.CreateEvent
			)

			BeforeEach(func() {
				cert = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{Name: instance.Name,
								Kind: "IBPOrderer"},
						},
					},
				}
				e = event.CreateEvent{}
			})

			It("sets update flags to true if create event is detected for secret and secret is a TLS signcert", func() {
				cert.Name = fmt.Sprintf("tls-%s-signcert", instance.Name)
				e.Object = cert
				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(true))

				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{
					tlsCertCreated: true,
				}))
			})

			It("sets update flags to true if create event is detected for secret and secret is an ecert signcert", func() {
				cert.Name = fmt.Sprintf("ecert-%s-signcert", instance.Name)
				e.Object = cert
				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(true))

				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{
					ecertCreated: true,
				}))
			})

			It("does not set update flags and doesn't trigger create event if create event is detected for secret and secret is not a signcert", func() {
				cert.Name = fmt.Sprintf("tls-%s-admincert", instance.Name)
				e.Object = cert
				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(false))

				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{}))
			})

			It("does not set update flags and doesn't trigger create event if create event is detected for non-orderer secret", func() {
				cert.Name = "tls-peer1-signcert"
				cert.OwnerReferences = nil
				e.Object = cert
				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(false))

				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{}))
			})

			It("does not set update flags if create event is detected for secret with non-orderer owner", func() {
				cert.Name = "tls-peer1-signcert"
				cert.OwnerReferences[0].Kind = "IBPPeer"
				e.Object = cert
				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(true))

				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{
					tlsCertCreated: false,
				}))
			})
		})

		Context("remove element", func() {
			BeforeEach(func() {
				reconciler.PushUpdate(instance.Name, Update{
					overridesUpdated: true,
				})

				reconciler.PushUpdate(instance.Name, Update{
					specUpdated: true,
				})

				Expect(reconciler.GetUpdateStatus(instance).ConfigOverridesUpdated()).To(Equal(true))
				Expect(reconciler.GetUpdateStatusAtElement(instance, 1).SpecUpdated()).To(Equal(true))
			})

			It("removes top element", func() {
				reconciler.PopUpdate(instance.Name)
				Expect(reconciler.GetUpdateStatus(instance).ConfigOverridesUpdated()).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(instance).SpecUpdated()).To(Equal(true))
			})

			It("removing more elements than in slice should not panic", func() {
				reconciler.PopUpdate(instance.Name)
				reconciler.PopUpdate(instance.Name)
				reconciler.PopUpdate(instance.Name)
				Expect(reconciler.GetUpdateStatus(instance).SpecUpdated()).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(instance).ConfigOverridesUpdated()).To(Equal(false))
			})
		})
	})
})
