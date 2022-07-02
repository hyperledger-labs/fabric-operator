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

package ibppeer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	peermocks "github.com/IBM-Blockchain/fabric-operator/controllers/ibppeer/mocks"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	yaml "sigs.k8s.io/yaml"
)

var _ = Describe("ReconcileIBPPeer", func() {
	var (
		reconciler        *ReconcileIBPPeer
		request           reconcile.Request
		mockKubeClient    *mocks.Client
		mockPeerReconcile *peermocks.PeerReconcile
		instance          *current.IBPPeer
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{}
		mockPeerReconcile = &peermocks.PeerReconcile{}
		instance = &current.IBPPeer{
			Spec: current.IBPPeerSpec{
				Images: &current.PeerImages{
					PeerTag: "1.4.9-2511004",
				},
			},
		}
		instance.Name = "test-peer"

		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *current.IBPPeer:
				o := obj.(*current.IBPPeer)
				o.Kind = "IBPPeer"
				o.Name = instance.Name

				instance = o
			case *corev1.Service:
				o := obj.(*corev1.Service)
				o.Spec.Type = corev1.ServiceTypeNodePort
				o.Spec.Ports = append(o.Spec.Ports, corev1.ServicePort{
					Name: "peer-api",
					TargetPort: intstr.IntOrString{
						IntVal: 7051,
					},
					NodePort: int32(7051),
				})
			}
			return nil
		}

		mockKubeClient.UpdateStatusStub = func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
			switch obj.(type) {
			case *current.IBPPeer:
				o := obj.(*current.IBPPeer)
				instance = o
			}
			return nil
		}

		mockKubeClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
			switch obj.(type) {
			case *corev1.NodeList:
				nodeList := obj.(*corev1.NodeList)
				node := corev1.Node{}
				node.Labels = map[string]string{}
				node.Labels["topology.kubernetes.io/zone"] = "dal"
				node.Labels["topology.kubernetes.io/region"] = "us-south"
				nodeList.Items = append(nodeList.Items, node)
			case *current.IBPPeerList:
				peerList := obj.(*current.IBPPeerList)
				p1 := current.IBPPeer{}
				p1.Name = "test-peer1"
				p2 := current.IBPPeer{}
				p2.Name = "test-peer2"
				p3 := current.IBPPeer{}
				p3.Name = "test-peer2"
				peerList.Items = []current.IBPPeer{p1, p2, p3}
			}
			return nil
		}

		reconciler = &ReconcileIBPPeer{
			Offering: mockPeerReconcile,
			client:   mockKubeClient,
			scheme:   &runtime.Scheme{},
			update:   map[string][]Update{},
			mutex:    &sync.Mutex{},
		}
		request = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "test-namespace",
				Name:      "test",
			},
		}
	})

	Context("Reconciles", func() {
		It("does not return an error if the custom resource is 'not found'", func() {
			notFoundErr := &k8serror.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonNotFound,
				},
			}
			mockKubeClient.GetReturns(notFoundErr)
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error if the request to get custom resource return any other error besides 'not found'", func() {
			alreadyExistsErr := &k8serror.StatusError{
				ErrStatus: metav1.Status{
					Message: "already exists",
					Reason:  metav1.StatusReasonAlreadyExists,
				},
			}
			mockKubeClient.GetReturns(alreadyExistsErr)
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("already exists"))
		})

		It("returns an error if it encountered a non-breaking error", func() {
			errMsg := "failed to reconcile deployment encountered breaking error"
			mockPeerReconcile.ReconcileReturns(common.Result{}, errors.New(errMsg))
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Peer instance '%s' encountered error: %s", instance.Name, errMsg)))
		})

		It("does not return an error if it encountered a breaking error", func() {
			mockPeerReconcile.ReconcileReturns(common.Result{}, operatorerrors.New(operatorerrors.InvalidDeploymentCreateRequest, "failed to reconcile deployment encountered breaking error"))
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("update reconcile", func() {
		var (
			oldPeer   *current.IBPPeer
			newPeer   *current.IBPPeer
			oldSecret *corev1.Secret
			newSecret *corev1.Secret
			e         event.UpdateEvent
		)

		BeforeEach(func() {

			configoverride := []byte(`{"peer": {"id": "peer1"} }`)

			oldPeer = &current.IBPPeer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPPeerSpec{
					Images: &current.PeerImages{
						PeerTag: "1.4.6-20200101",
					},
					ConfigOverride: &runtime.RawExtension{Raw: configoverride},
				},
			}

			configoverride2 := []byte(`{"peer": {"id": "peer2"} }`)

			newPeer = &current.IBPPeer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPPeerSpec{
					Images: &current.PeerImages{
						PeerTag: "1.4.9-2511004",
					},
					ConfigOverride: &runtime.RawExtension{Raw: configoverride2},
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldPeer,
				ObjectNew: newPeer,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))

			oldPeer = &current.IBPPeer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPPeerSpec{
					Images: &current.PeerImages{
						PeerTag: "1.4.6-20200101",
					},
					MSPID: "old-mspid",
				},
			}

			newPeer = &current.IBPPeer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPPeerSpec{
					Images: &current.PeerImages{
						PeerTag: "1.4.9-2511004",
					},
					MSPID: "new-mspid",
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldPeer,
				ObjectNew: newPeer,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))

			oldSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("tls-%s-signcert", instance.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: instance.Name,
							Kind: "IBPPeer",
						},
					},
				},
			}

			newSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("tls-%s-signcert", instance.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: instance.Name,
							Kind: "IBPPeer",
						},
					},
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldSecret,
				ObjectNew: newSecret,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(false))

			oldSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("ecert-%s-signcert", instance.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: instance.Name,
							Kind: "IBPPeer",
						},
					},
				},
				Data: map[string][]byte{
					"test": []byte("data"),
				},
			}

			newSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("ecert-%s-signcert", instance.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: instance.Name,
							Kind: "IBPPeer",
						},
					},
				},
				Data: map[string][]byte{
					"test": []byte("newdata"),
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldSecret,
				ObjectNew: newSecret,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))

			oldSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("tls-%s-admincerts", instance.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: instance.Name,
							Kind: "IBPPeer",
						},
					},
				},
			}

			newSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("tls-%s-admincerts", instance.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: instance.Name,
							Kind: "IBPPeer",
						},
					},
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldSecret,
				ObjectNew: newSecret,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(false))

			oldPeer = &current.IBPPeer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPPeerSpec{
					Secret: &current.SecretSpec{
						MSP: &current.MSPSpec{
							Component: &current.MSP{
								SignCerts: "testcert",
							},
						},
					},
				},
			}

			newPeer = &current.IBPPeer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPPeerSpec{
					Secret: &current.SecretSpec{
						MSP: &current.MSPSpec{
							TLS: &current.MSP{
								SignCerts: "testcert",
							},
						},
					},
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldPeer,
				ObjectNew: newPeer,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
		})

		It("properly pops update flags from stack", func() {
			By("popping first update - config overrides", func() {
				Expect(reconciler.GetUpdateStatus(instance).overridesUpdated).To(Equal(true))
				Expect(reconciler.GetUpdateStatus(instance).peerTagUpdated).To(Equal(true))

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).NotTo(HaveOccurred())

			})

			By("popping second update - spec updated", func() {
				Expect(reconciler.GetUpdateStatus(instance).overridesUpdated).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(instance).specUpdated).To(Equal(true))

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).NotTo(HaveOccurred())
			})

			By("popping third update - ecert updated", func() {
				Expect(reconciler.GetUpdateStatus(instance).tlsCertUpdated).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(instance).ecertUpdated).To(Equal(true))

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).NotTo(HaveOccurred())
			})

			By("popping fourth update - msp updated", func() {
				Expect(reconciler.GetUpdateStatus(instance).tlsCertUpdated).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(instance).ecertUpdated).To(Equal(false))

				Expect(reconciler.GetUpdateStatus(instance).mspUpdated).To(Equal(true))

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).NotTo(HaveOccurred())

				Expect(reconciler.GetUpdateStatus(instance).mspUpdated).To(Equal(false))
			})

		})

		Context("num seconds warning period updated", func() {
			BeforeEach(func() {
				oldPeer = &current.IBPPeer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
					Spec: current.IBPPeerSpec{
						NumSecondsWarningPeriod: 10,
					},
				}

				newPeer = &current.IBPPeer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
					Spec: current.IBPPeerSpec{
						NumSecondsWarningPeriod: 20,
					},
				}

				e = event.UpdateEvent{
					ObjectOld: oldPeer,
					ObjectNew: newPeer,
				}

				Expect(reconciler.UpdateFunc(e)).To(Equal(true))
			})

			It("returns true if numSecondsWarningPeriod changed", func() {
				Expect(reconciler.GetUpdateStatusAtElement(instance, 4).TLSCertUpdated()).To(Equal(true))
				Expect(reconciler.GetUpdateStatusAtElement(instance, 4).EcertUpdated()).To(Equal(true))

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("enrollment information changes detection", func() {
			BeforeEach(func() {
				oldPeer = &current.IBPPeer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}

				newPeer = &current.IBPPeer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}

				e = event.UpdateEvent{
					ObjectOld: oldPeer,
					ObjectNew: newPeer,
				}
			})

			Context("ecert", func() {
				It("returns false if new secret is nil", func() {
					Expect(reconciler.UpdateFunc(e)).To(Equal(false))
					Expect(reconciler.GetUpdateStatus(instance).EcertEnroll()).To(Equal(false))
				})

				It("returns false if new secret has ecert msp set along with enrollment inforamtion", func() {
					oldPeer.Spec.Secret = &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{
							Component: &current.Enrollment{
								EnrollID: "id1",
							},
						},
					}
					newPeer.Spec.Secret = &current.SecretSpec{
						MSP: &current.MSPSpec{
							Component: &current.MSP{},
						},
						Enrollment: &current.EnrollmentSpec{
							Component: &current.Enrollment{
								EnrollID: "id2",
							},
						},
					}

					newPeer.Spec.Action = current.PeerAction{
						Restart: true,
					}

					reconciler.UpdateFunc(e)
					Expect(reconciler.GetUpdateStatusAtElement(instance, 4).EcertEnroll()).To(Equal(false))
				})
			})

			Context("TLS", func() {
				It("returns false if new secret is nil", func() {
					Expect(reconciler.UpdateFunc(e)).To(Equal(false))
					Expect(reconciler.GetUpdateStatus(instance).EcertEnroll()).To(Equal(false))
				})

				It("returns false if new secret has TLS msp set along with enrollment inforamtion", func() {
					oldPeer.Spec.Secret = &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{
							Component: &current.Enrollment{
								EnrollID: "id1",
							},
						},
					}
					newPeer.Spec.Secret = &current.SecretSpec{
						MSP: &current.MSPSpec{
							Component: &current.MSP{},
						},
						Enrollment: &current.EnrollmentSpec{
							Component: &current.Enrollment{
								EnrollID: "id2",
							},
						},
					}

					newPeer.Spec.Action = current.PeerAction{
						Restart: true,
					}

					reconciler.UpdateFunc(e)
					Expect(reconciler.GetUpdateStatusAtElement(instance, 4).EcertEnroll()).To(Equal(false))
				})
			})
		})

		Context("detect MSP updates", func() {
			BeforeEach(func() {
				oldPeer = &current.IBPPeer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}

				newPeer = &current.IBPPeer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}

				e = event.UpdateEvent{
					ObjectOld: oldPeer,
					ObjectNew: newPeer,
				}
			})

			It("returns false if only admin certs updated in new msp", func() {
				oldPeer.Spec.Secret = &current.SecretSpec{
					MSP: &current.MSPSpec{
						Component: &current.MSP{
							AdminCerts: []string{"oldcert"},
						},
					},
				}
				newPeer.Spec.Secret = &current.SecretSpec{
					MSP: &current.MSPSpec{
						Component: &current.MSP{
							AdminCerts: []string{"newcert"},
						},
					},
				}
				reconciler.UpdateFunc(e)
				Expect(reconciler.GetUpdateStatusAtElement(instance, 4).MSPUpdated()).To(Equal(false))
			})
		})

		Context("update node OU", func() {
			BeforeEach(func() {
				oldPeer = &current.IBPPeer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}

				newPeer = &current.IBPPeer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}
				newPeer.Spec.DisableNodeOU = &current.BoolTrue

				e = event.UpdateEvent{
					ObjectOld: oldPeer,
					ObjectNew: newPeer,
				}
			})

			It("returns true if node ou updated in spec", func() {
				reconciler.UpdateFunc(e)
				Expect(reconciler.GetUpdateStatusAtElement(instance, 4).NodeOUUpdated()).To(Equal(true))
			})
		})
	})

	Context("set status", func() {
		It("sets the status to error if error occured during IPBPPeer reconciliation", func() {
			reconciler.SetStatus(instance, nil, errors.New("ibppeer error"))
			Expect(instance.Status.Type).To(Equal(current.Error))
			Expect(instance.Status.Message).To(Equal("ibppeer error"))
		})

		It("sets the status to deploying if pod is not yet running", func() {
			mockKubeClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
				podList := obj.(*corev1.PodList)
				pod := corev1.Pod{}
				podList.Items = append(podList.Items, pod)
				return nil
			}
			reconciler.SetStatus(instance, nil, nil)
			Expect(instance.Status.Type).To(Equal(current.Deploying))
		})

		It("sets the status to deployed if pod is running", func() {
			mockKubeClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
				podList := obj.(*corev1.PodList)
				pod := corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				}
				podList.Items = append(podList.Items, pod)
				return nil
			}

			reconciler.SetStatus(instance, nil, nil)
			Expect(instance.Status.Type).To(Equal(current.Deployed))
		})
	})

	Context("create func predicate", func() {
		Context("case: peer", func() {
			var (
				peer *current.IBPPeer
				e    event.CreateEvent
			)

			BeforeEach(func() {
				peer = &current.IBPPeer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.GetName(),
					},
					Status: current.IBPPeerStatus{
						CRStatus: current.CRStatus{
							Type: current.Deployed,
						},
					},
				}
				e = event.CreateEvent{
					Object: peer,
				}
			})

			It("sets update flags to false if instance has status type and a create event is detected but no spec changes are detected", func() {
				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(true))

				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{
					specUpdated:      false,
					overridesUpdated: false,
					dindArgsUpdated:  false,
				}))
			})

			It("sets update flags to true if instance has status type and a create event is detected and spec changes detected", func() {
				override := []byte("{}")

				spec := current.IBPPeerSpec{
					ImagePullSecrets: []string{"pullsecret1"},
					ConfigOverride:   &runtime.RawExtension{Raw: override},
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
				peer.Status.Type = ""

				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(true))

				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{}))
			})

			It("returns true but doesn't trigger update if new instance's name is unique to one IBPPeer in the list of IBPPeers", func() {
				peer.Status.Type = ""
				peer.Name = "test-peer1"

				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(true))
				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{}))

			})

			It("returns false if new instance's name already exists for another IBPPeer custom resource", func() {
				peer.Status.Type = ""
				peer.Name = "test-peer2"

				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(false))
				Expect(peer.Status.Type).To(Equal(current.Error))
			})
		})

		Context("case: secret", func() {
			var (
				cert *corev1.Secret
				e    event.CreateEvent
			)

			BeforeEach(func() {
				cert = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{Name: instance.Name,
								Kind: "IBPPeer"},
						},
					},
				}
				e = event.CreateEvent{}
			})

			It("sets create flags to true if create event is detected for secret and secret is a TLS signcert", func() {
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

			It("does not set update flags and doesn't trigger create event if create event is detected for non-peer secret", func() {
				cert.Name = "tls-orderer1-signcert"
				cert.OwnerReferences = nil
				e.Object = cert
				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(false))

				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{}))
			})

			It("does not set update flags if create event is detected for secret with non-peer owner", func() {
				cert.Name = "tls-orderer1-signcert"
				cert.OwnerReferences[0].Kind = "IBPOrderer"
				e.Object = cert
				create := reconciler.CreateFunc(e)
				Expect(create).To(Equal(true))

				Expect(reconciler.GetUpdateStatus(instance)).To(Equal(&Update{
					tlsCertCreated: false,
				}))
			})
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

	Context("append update if missing", func() {
		It("appends update", func() {
			updates := []Update{{tlsCertUpdated: true}}
			updates = reconciler.AppendUpdateIfMissing(updates, Update{ecertUpdated: true})
			Expect(len(updates)).To(Equal(2))
		})

		It("doesn't append update that is already in stack", func() {
			updates := []Update{{tlsCertUpdated: true}}
			updates = reconciler.AppendUpdateIfMissing(updates, Update{tlsCertUpdated: true})
			Expect(len(updates)).To(Equal(1))
		})
	})

	Context("push update", func() {
		It("pushes update only if missing from stack of updates", func() {
			reconciler.PushUpdate(instance.Name, Update{specUpdated: true})
			Expect(len(reconciler.update[instance.Name])).To(Equal(1))
			reconciler.PushUpdate(instance.Name, Update{tlsCertUpdated: true})
			Expect(len(reconciler.update[instance.Name])).To(Equal(2))
			reconciler.PushUpdate(instance.Name, Update{ecertUpdated: true})
			Expect(len(reconciler.update[instance.Name])).To(Equal(3))
			reconciler.PushUpdate(instance.Name, Update{tlsCertUpdated: true})
			Expect(len(reconciler.update[instance.Name])).To(Equal(3))
			reconciler.PushUpdate(instance.Name, Update{tlsCertUpdated: true, specUpdated: true})
			Expect(len(reconciler.update[instance.Name])).To(Equal(4))
		})
	})

	Context("add owner reference to secret", func() {
		var (
			secret *corev1.Secret
		)

		BeforeEach(func() {
			secret = &corev1.Secret{}
			secret.Name = "ecert-test-peer1-signcert"
		})

		It("returns error if fails to get list of peers", func() {
			mockKubeClient.ListReturns(errors.New("list error"))
			_, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("list error"))
		})

		It("returns false if secret doesn't belong to any peers in list", func() {
			secret.Name = "tls-orderer1-signcert"
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(false))
		})

		It("returns true if owner references added to secret", func() {
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(true))
		})

		It("returns true if owner references added to init-rootcert secret", func() {
			secret.Name = "test-peer1-init-rootcert"
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(true))
		})
	})
})
