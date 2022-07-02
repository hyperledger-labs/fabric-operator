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

package orderernode_test

import (
	"context"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/orderernode"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Orderernode manager", func() {
	var (
		mockKubeClient *mocks.Client
		manager        *orderernode.Manager
		instance       metav1.Object
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{}
		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *current.IBPOrderer:
				o := obj.(*current.IBPOrderer)
				manager.BasedOnCR(instance, o)
			}
			return nil
		}

		manager = &orderernode.Manager{
			OrdererNodeFile: "../../../../definitions/orderer/orderernode.yaml",
			Client:          mockKubeClient,
			OverrideFunc: func(object v1.Object, d *current.IBPOrderer, action resources.Action) error {
				return nil
			},
			LabelsFunc: func(v1.Object) map[string]string {
				return map[string]string{}
			},
		}

		instance = &metav1.ObjectMeta{}

	})

	Context("reconciles the orderernode instance", func() {
		It("does not try to create orderernode if the get request returns an error other than 'not found'", func() {
			errMsg := "connection refused"
			mockKubeClient.GetReturns(errors.New(errMsg))
			err := manager.Reconcile(instance, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errMsg))
		})

		When("orderernode does not exist", func() {
			BeforeEach(func() {
				notFoundErr := &k8serror.StatusError{
					ErrStatus: metav1.Status{
						Reason: metav1.StatusReasonNotFound,
					},
				}
				mockKubeClient.GetReturns(notFoundErr)
			})

			It("returns an error if the creation of the Orderernode fails", func() {
				errMsg := "unable to create orderernode"
				mockKubeClient.CreateReturns(errors.New(errMsg))
				err := manager.Reconcile(instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(errMsg))
			})

			It("does not return an error on a successfull Orderernode creation", func() {
				err := manager.Reconcile(instance, false)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("orderernode already exists", func() {
			It("returns an error if orderernode is updated", func() {
				errMsg := "Updating orderer node is not allowed programmatically"
				err := manager.Reconcile(instance, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errMsg))
			})
		})
	})

	Context("check orderernode state", func() {
		// TODO fix this test
		// 	It("returns an error if an unexpected change in orderernode is detected", func() {
		// 		num := 1
		// 		dep := &current.IBPOrderer{
		// 			Spec: current.IBPOrdererSpec{
		// 				NodeNumber: &num,
		// 			},
		// 		}
		// 		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj runtime.Object) error {
		// 			switch obj.(type) {
		// 			case *current.IBPOrderer:
		// 				on := obj.(*current.IBPOrderer)
		// 				on.Spec = current.IBPOrdererSpec{
		// 					NodeNumber: &num,
		// 					Arch:       []string{"s390x"},
		// 				}
		// 			}
		// 			return nil
		// 		}

		// 		err := manager.CheckState(dep)
		// 		Expect(err).To(HaveOccurred())
		// 		Expect(err.Error()).To(ContainSubstring("orderernode has been edited manually, and does not match what is expected based on the CR: unexpected mismatch"))
		// 	})

		// 	It("returns no error if no changes detected for orderernode", func() {
		// 		err := manager.CheckState(&appsv1.Deployment{})
		// 		Expect(err).NotTo(HaveOccurred())
		// 	})
	})

	Context("restore orderernode state", func() {
		It("returns an error if the restoring orderernode state fails", func() {
			errMsg := "unable to restore orderernode"
			mockKubeClient.UpdateReturns(errors.New(errMsg))
			num := 1
			err := manager.RestoreState(&current.IBPOrderer{
				Spec: current.IBPOrdererSpec{
					NodeNumber: &num,
				},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errMsg))
		})

		It("returns no error if able to restore orderernode state", func() {
			num := 1
			err := manager.RestoreState(&current.IBPOrderer{
				Spec: current.IBPOrdererSpec{
					NodeNumber: &num,
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
