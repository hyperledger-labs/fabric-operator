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

package ingress_test

import (
	"context"

	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	ingress "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/ingress"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	networkingv1 "k8s.io/api/networking/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Ingress manager", func() {
	var (
		mockKubeClient *mocks.Client
		manager        *ingress.Manager
		instance       metav1.Object
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{}

		instance = &metav1.ObjectMeta{}

		manager = &ingress.Manager{
			IngressFile: "../../../../definitions/ca/ingress.yaml",
			Client:      mockKubeClient,
			OverrideFunc: func(object v1.Object, ingress *networkingv1.Ingress, action resources.Action) error {
				return nil
			},
			LabelsFunc: func(v1.Object) map[string]string {
				return map[string]string{}
			},
		}

		instance = &metav1.ObjectMeta{}
	})

	Context("reconciles the ingress instance", func() {
		It("does not try to create ingress if the get request returns an error other than 'not found'", func() {
			errMsg := "connection refused"
			mockKubeClient.GetReturns(errors.New(errMsg))
			err := manager.Reconcile(instance, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errMsg))
		})

		When("ingress does not exist", func() {
			BeforeEach(func() {
				notFoundErr := &k8serror.StatusError{
					ErrStatus: metav1.Status{
						Reason: metav1.StatusReasonNotFound,
					},
				}
				mockKubeClient.GetReturns(notFoundErr)
			})

			It("returns an error if fails to load default config", func() {
				manager.IngressFile = "bad.yaml"
				err := manager.Reconcile(instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no such file or directory"))
			})

			It("returns an error if override ingress value fails", func() {
				manager.OverrideFunc = func(v1.Object, *networkingv1.Ingress, resources.Action) error {
					return errors.New("creation override failed")
				}
				err := manager.Reconcile(instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("creation override failed"))
			})

			It("returns an error if the creation of the Ingress fails", func() {
				errMsg := "unable to create ingress"
				mockKubeClient.CreateReturns(errors.New(errMsg))
				err := manager.Reconcile(instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(errMsg))
			})

			It("does not return an error on a successfull ingress creation", func() {
				ing := networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      instance.GetName(),
						Namespace: instance.GetNamespace(),
						Annotations: map[string]string{
							"test": "test value",
						},
					},
				}

				count := 0
				mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {

					switch obj.(type) {
					case *networkingv1.Ingress:
						if count == 0 {
							// Send not found the first time to go to creation path
							notFoundErr := &k8serror.StatusError{
								ErrStatus: metav1.Status{
									Reason: metav1.StatusReasonNotFound,
								},
							}
							count++
							return notFoundErr
						}

						i := obj.(*networkingv1.Ingress)
						i.ObjectMeta = ing.ObjectMeta
					}

					return nil
				}

				err := manager.Reconcile(instance, false)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
