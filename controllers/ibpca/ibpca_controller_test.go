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

package ibpca

import (
	"context"
	"errors"
	"fmt"
	"sync"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	camocks "github.com/IBM-Blockchain/fabric-operator/controllers/ibpca/mocks"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
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

var _ = Describe("ReconcileIBPCA", func() {
	var (
		reconciler      *ReconcileIBPCA
		request         reconcile.Request
		mockKubeClient  *mocks.Client
		mockCAReconcile *camocks.CAReconcile
		instance        *current.IBPCA
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{}
		mockCAReconcile = &camocks.CAReconcile{}
		instance = &current.IBPCA{
			Spec: current.IBPCASpec{},
		}
		instance.Name = "test-ca"

		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *current.IBPCA:
				o := obj.(*current.IBPCA)
				o.Kind = "IBPCA"
				o.Name = instance.Name

				instance.Status = o.Status
			}
			return nil
		}

		mockKubeClient.UpdateStatusStub = func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
			switch obj.(type) {
			case *current.IBPCA:
				o := obj.(*current.IBPCA)
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
			case *current.IBPCAList:
				caList := obj.(*current.IBPCAList)
				ca1 := current.IBPCA{}
				ca1.Name = "test-ca1"
				ca2 := current.IBPCA{}
				ca2.Name = "test-ca2"
				ca3 := current.IBPCA{}
				ca3.Name = "test-ca2"
				caList.Items = []current.IBPCA{ca1, ca2, ca3}
			case *current.IBPPeerList:
				caList := obj.(*current.IBPPeerList)
				p1 := current.IBPPeer{}
				p1.Name = "test-peer"
				caList.Items = []current.IBPPeer{p1}
			}
			return nil
		}

		reconciler = &ReconcileIBPCA{
			Offering: mockCAReconcile,
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
			mockCAReconcile.ReconcileReturns(common.Result{}, errors.New(errMsg))
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("CA instance '%s' encountered error: %s", instance.Name, errMsg)))
		})

		It("does not return an error if it encountered a breaking error", func() {
			mockCAReconcile.ReconcileReturns(common.Result{}, operatorerrors.New(operatorerrors.InvalidDeploymentCreateRequest, "failed to reconcile deployment encountered breaking error"))
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("update reconcile", func() {
		var (
			oldCA *current.IBPCA
			newCA *current.IBPCA
			e     event.UpdateEvent
		)

		BeforeEach(func() {
			caConfig := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					CA: v1.CAInfo{
						Name: "old-ca-name",
					},
				},
			}
			caJson, err := util.ConvertToJsonMessage(caConfig)
			Expect(err).NotTo(HaveOccurred())

			oldCA = &current.IBPCA{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPCASpec{
					ConfigOverride: &current.ConfigOverride{
						CA: &runtime.RawExtension{Raw: *caJson},
					},
				},
			}

			newcaConfig := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					CA: v1.CAInfo{
						Name: "new-ca-name",
					},
				},
			}
			newcaJson, err := util.ConvertToJsonMessage(newcaConfig)
			Expect(err).NotTo(HaveOccurred())

			newCA = &current.IBPCA{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPCASpec{
					ConfigOverride: &current.ConfigOverride{
						CA: &runtime.RawExtension{Raw: *newcaJson},
					},
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldCA,
				ObjectNew: newCA,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))

			oldCA = &current.IBPCA{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPCASpec{
					ImagePullSecrets: []string{"old-secret"},
				},
			}

			newCA = &current.IBPCA{
				ObjectMeta: metav1.ObjectMeta{
					Name: instance.Name,
				},
				Spec: current.IBPCASpec{
					ImagePullSecrets: []string{"new-secret"},
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldCA,
				ObjectNew: newCA,
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
		})

		It("properly pops update flags from stack", func() {
			Expect(reconciler.GetUpdateStatus(instance).CAOverridesUpdated()).To(Equal(true))

			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).NotTo(HaveOccurred())

			Expect(reconciler.GetUpdateStatus(instance).CAOverridesUpdated()).To(Equal(false))
			Expect(reconciler.GetUpdateStatus(instance).SpecUpdated()).To(Equal(true))

			_, err = reconciler.Reconcile(context.TODO(), request)
			Expect(err).NotTo(HaveOccurred())

			Expect(reconciler.GetUpdateStatus(instance).CAOverridesUpdated()).To(Equal(false))
			Expect(reconciler.GetUpdateStatus(instance).TLSCAOverridesUpdated()).To(Equal(false))
			Expect(reconciler.GetUpdateStatus(instance).SpecUpdated()).To(Equal(false))
		})
	})

	Context("set status", func() {
		It("sets the status to error if error occured during IBPCA reconciliation", func() {
			reconciler.SetStatus(instance, nil, errors.New("ibpca error"))
			Expect(instance.Status.Type).To(Equal(current.Error))
			Expect(instance.Status.Message).To(Equal("ibpca error"))
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

	Context("add owner reference to secret", func() {
		var (
			secret *corev1.Secret
		)

		BeforeEach(func() {
			secret = &corev1.Secret{}
			secret.Name = "test-ca1-ca-crypto"
		})

		It("returns error if fails to get list of CAs", func() {
			mockKubeClient.ListReturns(errors.New("list error"))
			_, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("list error"))
		})

		It("returns false if secret doesn't belong to any CAs in list", func() {
			secret.Name = "invalidca-ca-crypto"
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(false))
		})

		It("returns true if owner references added to ca crypto secret", func() {
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(true))
		})

		It("returns true if owner references added to tlsca crypto secret", func() {
			secret.Name = "test-ca2-tlsca-crypto"
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(true))
		})

		It("returns true if owner references added to ca secret", func() {
			secret.Name = "test-ca2-ca"
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(true))
		})

		It("returns true if owner references added to tlsca secret", func() {
			secret.Name = "test-ca2-tlsca"
			added, err := reconciler.AddOwnerReferenceToSecret(secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(Equal(true))
		})
	})
})
