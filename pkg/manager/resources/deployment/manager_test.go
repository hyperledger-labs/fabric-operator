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

package deployment_test

import (
	"context"

	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Deployment manager", func() {
	var (
		mockKubeClient *mocks.Client
		manager        *deployment.Manager
		instance       metav1.Object
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{}
		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *appsv1.Deployment:
				o := obj.(*appsv1.Deployment)
				manager.BasedOnCR(instance, o)
				o.Status.Replicas = 1
				o.Status.UpdatedReplicas = 1
			}
			return nil
		}

		manager = &deployment.Manager{
			DeploymentFile: "../../../../definitions/ca/deployment.yaml",
			Client:         mockKubeClient,
			OverrideFunc: func(object v1.Object, d *appsv1.Deployment, action resources.Action) error {
				d.Spec = appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{corev1.Container{
								Name: "container",
							}},
						},
					},
				}
				return nil
			},
			LabelsFunc: func(v1.Object) map[string]string {
				return map[string]string{}
			},
		}

		instance = &metav1.ObjectMeta{}
	})

	Context("reconciles the deployment instance", func() {
		It("does not try to create deployment if the get request returns an error other than 'not found'", func() {
			errMsg := "connection refused"
			mockKubeClient.GetReturns(errors.New(errMsg))
			err := manager.Reconcile(instance, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errMsg))
		})

		When("deployment does not exist", func() {
			BeforeEach(func() {
				notFoundErr := &k8serror.StatusError{
					ErrStatus: metav1.Status{
						Reason: metav1.StatusReasonNotFound,
					},
				}
				mockKubeClient.GetReturns(notFoundErr)
			})

			It("returns an error if fails to load default config", func() {
				manager.DeploymentFile = "bad.yaml"
				err := manager.Reconcile(instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no such file or directory"))
			})

			It("returns an error if override deployment value fails", func() {
				manager.OverrideFunc = func(v1.Object, *appsv1.Deployment, resources.Action) error {
					return errors.New("creation override failed")
				}
				err := manager.Reconcile(instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("creation override failed"))
			})

			It("returns an error if the creation of the Deployment fails", func() {
				errMsg := "unable to create service"
				mockKubeClient.CreateReturns(errors.New(errMsg))
				err := manager.Reconcile(instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(errMsg))
			})

			It("does not return an error on a successfull Deployment creation", func() {
				err := manager.Reconcile(instance, false)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("deployment already exists", func() {
			It("returns an error if override deployment value fails", func() {
				manager.OverrideFunc = func(v1.Object, *appsv1.Deployment, resources.Action) error {
					return errors.New("update override failed")
				}
				err := manager.Reconcile(instance, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("update override failed"))
			})

			It("returns an error if the updating of Deployment fails", func() {
				errMsg := "unable to update deployment"
				mockKubeClient.PatchReturns(errors.New(errMsg))
				err := manager.Reconcile(instance, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(errMsg))
			})

			It("does not return an error on a successfull Deployment update", func() {
				err := manager.Reconcile(instance, true)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("check deployment state", func() {
		It("returns an error if an unexpected change in deployment is detected", func() {
			dep := &appsv1.Deployment{}

			mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *appsv1.Deployment:
					dep = obj.(*appsv1.Deployment)
					dep.Spec = appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": ""},
						},
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{corev1.Container{
									Name: "test-container",
								}},
							},
						},
					}
				}
				return nil
			}

			err := manager.CheckState(dep)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("deployment () has been edited manually, and does not match what is expected based on the CR: unexpected mismatch: Template.Spec.Containers.slice[0].Name: test-container != container"))
		})

		It("returns no error if no changes detected for deployment", func() {
			err := manager.CheckState(&appsv1.Deployment{})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("restore deployment state", func() {
		It("returns an error if the restoring deployment state fails", func() {
			errMsg := "unable to restore deployment"
			mockKubeClient.PatchReturns(errors.New(errMsg))
			err := manager.RestoreState(&appsv1.Deployment{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errMsg))
		})

		It("returns no error if able to restore deployment state", func() {
			err := manager.RestoreState(&appsv1.Deployment{})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
