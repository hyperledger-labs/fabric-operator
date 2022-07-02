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
	"errors"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	orderermocks "github.com/IBM-Blockchain/fabric-operator/controllers/ibporderer/mocks"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v1"
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("ReconcileIBPOrderer", func() {
	const (
		testRoleBindingFile    = "../../../definitions/orderer/rolebinding.yaml"
		testServiceAccountFile = "../../../definitions/orderer/serviceaccount.yaml"
	)

	var (
		reconciler           *ReconcileIBPOrderer
		request              reconcile.Request
		mockKubeClient       *mocks.Client
		mockOrdererReconcile *orderermocks.OrdererReconcile
		instance             *current.IBPOrderer
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{}
		mockOrdererReconcile = &orderermocks.OrdererReconcile{}
		nodeNumber := 1
		instance = &current.IBPOrderer{
			Spec: current.IBPOrdererSpec{
				ClusterSize: 3,
				NodeNumber:  &nodeNumber,
			},
		}
		instance.Name = "test-orderer"

		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *current.IBPOrderer:
				o := obj.(*current.IBPOrderer)
				o.Kind = "IBPOrderer"
				o.Name = instance.Name

				instance.Status = o.Status
			}
			return nil
		}

		mockKubeClient.UpdateStatusStub = func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
			switch obj.(type) {
			case *current.IBPOrderer:
				o := obj.(*current.IBPOrderer)
				instance.Status = o.Status
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
		}

		reconciler = &ReconcileIBPOrderer{
			Offering: mockOrdererReconcile,
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
			mockOrdererReconcile.ReconcileReturns(common.Result{}, errors.New(errMsg))
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Orderer instance '%s' encountered error: %s", instance.Name, errMsg)))
		})

		It("does not return an error if it encountered a breaking error", func() {
			mockOrdererReconcile.ReconcileReturns(common.Result{}, operatorerrors.New(operatorerrors.InvalidDeploymentCreateRequest, "failed to reconcile deployment encountered breaking error"))
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("set status", func() {
			It("sets the status to error if error occured during IBPOrderer reconciliation", func() {
				reconciler.SetStatus(instance, nil, errors.New("ibporderer error"))
				Expect(instance.Status.Type).To(Equal(current.Error))
				Expect(instance.Status.Message).To(Equal("ibporderer error"))
			})

			It("sets the status to deploying if pod is not yet running", func() {
				mockKubeClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
					switch obj.(type) {
					case *corev1.PodList:
						podList := obj.(*corev1.PodList)
						pod := corev1.Pod{}
						podList.Items = append(podList.Items, pod)
						return nil
					case *current.IBPOrdererList:
						ordererList := obj.(*current.IBPOrdererList)
						orderer := current.IBPOrderer{}
						orderer.Status = current.IBPOrdererStatus{
							CRStatus: current.CRStatus{
								Type: current.Deploying,
							},
						}
						ordererList.Items = append(ordererList.Items, orderer)
						return nil
					}
					return nil
				}

				reconciler.SetStatus(instance, nil, nil)
				Expect(instance.Status.Type).To(Equal(current.Deploying))
			})

			It("sets the status to deployed if pod is running", func() {
				mockKubeClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
					switch obj.(type) {
					case *corev1.PodList:
						podList := obj.(*corev1.PodList)
						pod := corev1.Pod{
							Status: corev1.PodStatus{
								Phase: corev1.PodRunning,
							},
						}
						podList.Items = append(podList.Items, pod)

						return nil
					case *current.IBPOrdererList:
						ordererList := obj.(*current.IBPOrdererList)
						orderer := current.IBPOrderer{}
						orderer.Status = current.IBPOrdererStatus{
							CRStatus: current.CRStatus{
								Type: current.Deployed,
							},
						}
						ordererList.Items = append(ordererList.Items, orderer)
						return nil
					}
					return nil
				}

				instance.Spec.ClusterSize = 1
				reconciler.SetStatus(instance, nil, nil)
				Expect(instance.Status.Type).To(Equal(current.Deployed))
			})

			It("sets the status to warning if the reconcile loop returns a warning status", func() {
				mockKubeClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
					switch obj.(type) {
					case *corev1.PodList:
						podList := obj.(*corev1.PodList)
						pod := corev1.Pod{
							Status: corev1.PodStatus{
								Phase: corev1.PodRunning,
							},
						}
						podList.Items = append(podList.Items, pod)

						return nil
					case *current.IBPOrdererList:
						ordererList := obj.(*current.IBPOrdererList)
						orderer := current.IBPOrderer{}
						orderer.Status = current.IBPOrdererStatus{
							CRStatus: current.CRStatus{
								Type: current.Deployed,
							},
						}
						ordererList.Items = append(ordererList.Items, orderer)
						return nil
					}
					return nil
				}

				result := &common.Result{
					Status: &current.CRStatus{
						Type: current.Warning,
					},
				}

				instance.Spec.ClusterSize = 1
				reconciler.SetStatus(instance, result, nil)
				Expect(instance.Status.Type).To(Equal(current.Warning))
			})

			It("persists warning status if the instance is already in warning state and reconcile loop returns a warning status", func() {
				mockKubeClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
					switch obj.(type) {
					case *corev1.PodList:
						podList := obj.(*corev1.PodList)
						pod := corev1.Pod{
							Status: corev1.PodStatus{
								Phase: corev1.PodRunning,
							},
						}
						podList.Items = append(podList.Items, pod)

						return nil
					case *current.IBPOrdererList:
						ordererList := obj.(*current.IBPOrdererList)
						orderer := current.IBPOrderer{}
						orderer.Status = current.IBPOrdererStatus{
							CRStatus: current.CRStatus{
								Type: current.Deployed,
							},
						}
						ordererList.Items = append(ordererList.Items, orderer)
						return nil
					}
					return nil
				}

				result := &common.Result{
					Status: &current.CRStatus{
						Type: current.Warning,
					},
				}

				instance.Spec.ClusterSize = 1
				instance.Status.Type = current.Warning
				reconciler.SetStatus(instance, result, nil)
				Expect(instance.Status.Type).To(Equal(current.Warning))
			})
		})
	})

	Context("update reconcile", func() {
		var (
			oldOrderer *current.IBPOrderer
			newOrderer *current.IBPOrderer
			oldSecret  *corev1.Secret
			newSecret  *corev1.Secret
			e          event.UpdateEvent
		)

		BeforeEach(func() {
			configOverride := &config.Orderer{
				Orderer: v1.Orderer{
					General: v1.General{
						LedgerType: "type1",
					},
				},
			}
			configBytes, err := json.Marshal(configOverride)
			Expect(err).NotTo(HaveOccurred())

			oldOrderer = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPOrdererSpec{
					Images: &current.OrdererImages{
						OrdererTag: "1.4.6-20200101",
					},
					ConfigOverride: &runtime.RawExtension{Raw: configBytes},
				},
			}

			configOverride2 := &config.Orderer{
				Orderer: v1.Orderer{
					General: v1.General{
						LedgerType: "type2",
					},
				},
			}
			configBytes2, err := json.Marshal(configOverride2)
			Expect(err).NotTo(HaveOccurred())
			newOrderer = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPOrdererSpec{
					Images: &current.OrdererImages{
						OrdererTag: "1.4.9-2511004",
					},
					ConfigOverride: &runtime.RawExtension{Raw: configBytes2},
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldOrderer,
				ObjectNew: newOrderer,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))

			oldOrderer = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPOrdererSpec{
					Images: &current.OrdererImages{
						OrdererTag: "1.4.9-2511004",
					},
					MSPID: "old-mspid",
				},
			}

			newOrderer = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPOrdererSpec{
					Images: &current.OrdererImages{
						OrdererTag: "1.4.9-2511004",
					},
					MSPID: "new-mspid",
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldOrderer,
				ObjectNew: newOrderer,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))

			oldSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("tls-%s-signcert", instance.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: instance.Name,
							Kind: "IBPOrderer",
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
							Kind: "IBPOrderer",
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
							Kind: "IBPOrderer",
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
						{Name: instance.Name},
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
					Name: fmt.Sprintf("tls-%s-admincert", instance.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: instance.Name,
							Kind: "IBPOrderer",
						},
					},
				},
			}
			newSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("tls-%s-admincert", instance.Name),
					OwnerReferences: []metav1.OwnerReference{
						{
							Name: instance.Name,
							Kind: "IBPOrderer",
						},
					},
				},
			}
			e = event.UpdateEvent{
				ObjectOld: oldSecret,
				ObjectNew: newSecret,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(false))

			oldOrderer = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPOrdererSpec{
					Images: &current.OrdererImages{},
					Secret: &current.SecretSpec{
						MSP: &current.MSPSpec{
							Component: &current.MSP{
								SignCerts: "testcert",
							},
						},
					},
				},
			}

			newOrderer = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPOrdererSpec{
					Images: &current.OrdererImages{},
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
				ObjectOld: oldOrderer,
				ObjectNew: newOrderer,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
		})

		It("properly pops update flags from stack", func() {
			By("popping first update - config overrides", func() {
				Expect(reconciler.GetUpdateStatus(instance).ConfigOverridesUpdated()).To(Equal(true))
				Expect(reconciler.GetUpdateStatus(instance).OrdererTagUpdated()).To(Equal(true))

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).NotTo(HaveOccurred())
			})

			By("popping second update - spec updated", func() {
				Expect(reconciler.GetUpdateStatus(instance).ConfigOverridesUpdated()).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(instance).SpecUpdated()).To(Equal(true))

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).NotTo(HaveOccurred())
			})

			By("popping third update - ecert updated", func() {
				Expect(reconciler.GetUpdateStatus(instance).TLSCertUpdated()).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(instance).EcertUpdated()).To(Equal(true))

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).NotTo(HaveOccurred())
			})

			By("popping fourth update - msp spec updated", func() {
				Expect(reconciler.GetUpdateStatus(instance).TLSCertUpdated()).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(instance).EcertUpdated()).To(Equal(false))

				Expect(reconciler.GetUpdateStatus(instance).MSPUpdated()).To(Equal(true))

				_, err := reconciler.Reconcile(context.TODO(), request)
				Expect(err).NotTo(HaveOccurred())

				Expect(reconciler.GetUpdateStatus(instance).MSPUpdated()).To(Equal(false))
			})

		})

		Context("enrollment information changes detection", func() {
			BeforeEach(func() {
				oldOrderer = &current.IBPOrderer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}

				newOrderer = &current.IBPOrderer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}

				e = event.UpdateEvent{
					ObjectOld: oldOrderer,
					ObjectNew: newOrderer,
				}
			})

			It("returns false if new secret is nil", func() {
				Expect(reconciler.UpdateFunc(e)).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(instance).EcertEnroll()).To(Equal(false))
			})

			It("returns false if new secret has ecert msp set along with enrollment inforamtion", func() {
				oldOrderer.Spec.Secret = &current.SecretSpec{
					Enrollment: &current.EnrollmentSpec{
						Component: &current.Enrollment{
							EnrollID: "id1",
						},
					},
				}
				newOrderer.Spec.Secret = &current.SecretSpec{
					MSP: &current.MSPSpec{
						Component: &current.MSP{},
					},
					Enrollment: &current.EnrollmentSpec{
						Component: &current.Enrollment{
							EnrollID: "id2",
						},
					},
				}

				newOrderer.Spec.Action = current.OrdererAction{
					Restart: true,
				}

				reconciler.UpdateFunc(e)
				Expect(reconciler.GetUpdateStatusAtElement(instance, 4).EcertEnroll()).To(Equal(false))
			})
		})

		Context("update node OU", func() {
			BeforeEach(func() {
				oldOrderer = &current.IBPOrderer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}

				newOrderer = &current.IBPOrderer{
					ObjectMeta: metav1.ObjectMeta{
						Name: instance.Name,
					},
				}
				newOrderer.Spec.DisableNodeOU = &current.BoolTrue

				e = event.UpdateEvent{
					ObjectOld: oldOrderer,
					ObjectNew: newOrderer,
				}
			})

			It("returns true if node ou updated in spec", func() {
				reconciler.UpdateFunc(e)
				Expect(reconciler.GetUpdateStatusAtElement(instance, 4).NodeOUUpdated()).To(Equal(true))
			})
		})
	})

	Context("status updated", func() {
		var (
			oldOrderer *current.IBPOrderer
			newOrderer *current.IBPOrderer
			e          event.UpdateEvent
		)

		BeforeEach(func() {
			oldOrderer = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPOrdererSpec{
					Images: &current.OrdererImages{},
				},
			}
			newOrderer = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPOrdererSpec{
					Images: &current.OrdererImages{},
				},
			}
			e = event.UpdateEvent{
				ObjectOld: oldOrderer,
				ObjectNew: newOrderer,
			}
		})

		It("does not set StatusUpdate flag if only heartbeat has changed", func() {
			oldOrderer.Status.LastHeartbeatTime = time.Now().String()
			newOrderer.Status.LastHeartbeatTime = time.Now().String()

			Expect(reconciler.UpdateFunc(e)).To(Equal(false))
			Expect(reconciler.GetUpdateStatus(instance).StatusUpdated()).To(Equal(false))
		})

		It("sets StatusUpdated flag to true if status type has changed", func() {
			oldOrderer.Status.Type = "old"
			newOrderer.Status.Type = "new"

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(instance).StatusUpdated()).To(Equal(true))
		})

		It("sets StatusUpdated flag to true if status reason has changed", func() {
			oldOrderer.Status.Reason = "oldreason"
			newOrderer.Status.Reason = "newreason"

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(instance).StatusUpdated()).To(Equal(true))
		})

		It("sets StatusUpdated flag to true if status message has changed", func() {
			oldOrderer.Status.Message = "oldmessage"
			newOrderer.Status.Message = "newmessage"

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(instance).StatusUpdated()).To(Equal(true))
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
		It("pushes update only if missing for certificate update", func() {
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
			secret.Name = "ecert-test-orderer1-signcert"
		})

		It("returns error if fails to get list of orderers", func() {
			mockKubeClient.ListReturns(errors.New("list error"))
			_, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("list error"))
		})

		It("returns false if secret doesn't belong to any orderers in list", func() {
			secret.Name = "tls-peer1-signcert"
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(false))
		})

		It("returns false if secret's name doesn't match expected format", func() {
			secret.Name = "orderersecret"
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
			secret.Name = "test-orderer1-init-rootcert"
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(true))
		})
	})
})
