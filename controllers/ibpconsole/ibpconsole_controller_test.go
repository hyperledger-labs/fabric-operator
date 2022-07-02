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

package ibpconsole

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	consolemocks "github.com/IBM-Blockchain/fabric-operator/controllers/ibpconsole/mocks"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
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

var _ = Describe("ReconcileIBPConsole", func() {
	const (
		testRoleBindingFile    = "../../../definitions/console/rolebinding.yaml"
		testServiceAccountFile = "../../../definitions/console/serviceaccount.yaml"
	)

	var (
		reconciler           *ReconcileIBPConsole
		request              reconcile.Request
		mockKubeClient       *mocks.Client
		mockConsoleReconcile *consolemocks.ConsoleReconcile
		instance             *current.IBPConsole
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{}
		mockConsoleReconcile = &consolemocks.ConsoleReconcile{}
		instance = &current.IBPConsole{
			Spec: current.IBPConsoleSpec{},
		}
		instance.Name = "test-console"
		instance.Namespace = "test-namespace"

		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *current.IBPConsole:
				o := obj.(*current.IBPConsole)
				o.Kind = "IBPConsole"
				o.Spec = instance.Spec
				o.Name = instance.Name

				instance.Status = o.Status
			}
			return nil
		}

		mockKubeClient.UpdateStatusStub = func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
			switch obj.(type) {
			case *current.IBPConsole:
				o := obj.(*current.IBPConsole)
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
			case *current.IBPConsoleList:
				list := obj.(*current.IBPConsoleList)
				console1 := current.IBPConsole{}
				console1.Name = "test-console1"
				console2 := current.IBPConsole{}
				console2.Name = "test-console1"
				list.Items = []current.IBPConsole{console1, console2}
			case *current.IBPPeerList:
				caList := obj.(*current.IBPPeerList)
				p1 := current.IBPPeer{}
				p1.Name = "test-peer"
				caList.Items = []current.IBPPeer{p1}
			}
			return nil
		}

		reconciler = &ReconcileIBPConsole{
			Config:   &config.Config{},
			Offering: mockConsoleReconcile,
			client:   mockKubeClient,
			scheme:   &runtime.Scheme{},
		}
		request = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "test-namespace",
				Name:      "test",
			},
		}
	})

	Context("Reconciles", func() {
		It("does not return an error if the custom resource is 'not fonund'", func() {
			notFoundErr := &k8serror.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonNotFound,
				},
			}
			mockKubeClient.GetReturns(notFoundErr)
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error if the request to get custom resource return any other errors besides 'not found'", func() {
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
			mockConsoleReconcile.ReconcileReturns(common.Result{}, errors.New(errMsg))
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Console instance '%s' encountered error: %s", instance.Name, errMsg)))
		})

		It("does not return an error if it encountered a breaking error", func() {
			mockConsoleReconcile.ReconcileReturns(common.Result{}, operatorerrors.New(operatorerrors.InvalidDeploymentCreateRequest, "failed to reconcile deployment encountered breaking error"))
			_, err := reconciler.Reconcile(context.TODO(), request)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("set status", func() {
		It("returns an error if the custom resource is not found", func() {
			notFoundErr := &k8serror.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonNotFound,
				},
			}
			mockKubeClient.GetReturns(notFoundErr)
			err := reconciler.SetStatus(instance, notFoundErr)
			Expect(err).To(HaveOccurred())
		})

		It("sets the status to error if error occured during IBPConsole reconciliation", func() {
			reconciler.SetStatus(instance, errors.New("ibpconsole error"))
			Expect(instance.Status.Type).To(Equal(current.Error))
			Expect(instance.Status.Message).To(Equal("ibpconsole error"))
		})

		It("sets the status to deploying if pod is not yet running", func() {
			mockKubeClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
				podList := obj.(*corev1.PodList)
				pod := corev1.Pod{}
				podList.Items = append(podList.Items, pod)
				return nil
			}
			reconciler.SetStatus(instance, nil)
			Expect(instance.Status.Type).To(Equal(current.Deploying))
		})

		It("sets the status to deployed if pod is running", func() {
			reconciler.SetStatus(instance, nil)
			Expect(instance.Status.Type).To(Equal(current.Deployed))
		})
	})

	Context("create func predicate", func() {
		var (
			e event.CreateEvent
		)

		BeforeEach(func() {
			e = event.CreateEvent{
				Object: instance,
			}
		})

		It("returns false if new console's name already exists for another IBPConsole", func() {
			instance.Name = "test-console1"
			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(false))
			Expect(instance.Status.Type).To(Equal(current.Error))
		})

		It("returns false if new console's name already exists for another custom resource", func() {
			instance.Name = "test-peer"
			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(false))
			Expect(instance.Status.Type).To(Equal(current.Error))
		})

		It("returns true if new console with valid name created", func() {
			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(true))
		})
	})
})
