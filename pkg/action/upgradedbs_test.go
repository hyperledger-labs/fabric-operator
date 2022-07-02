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

package action_test

import (
	"context"
	"errors"
	"strings"

	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/action"
	"github.com/IBM-Blockchain/fabric-operator/pkg/action/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
)

var _ = Describe("actions", func() {
	var (
		depMgr   *mocks.DeploymentReset
		client   *controllermocks.Client
		instance *current.IBPPeer
	)

	BeforeEach(func() {
		depMgr = &mocks.DeploymentReset{}
		instance = &current.IBPPeer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "peer",
			},
			Spec: current.IBPPeerSpec{
				Images: &current.PeerImages{
					PeerImage: "peerimage",
					PeerTag:   "peertag",
				},
			},
		}

		replicas := int32(1)
		dep := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							v1.Container{
								Name: "peer",
							},
						},
					},
				},
			},
		}
		depMgr.GetReturnsOnCall(0, dep, nil)
		depMgr.GetReturnsOnCall(1, &appsv1.Deployment{}, nil)
		depMgr.GetReturnsOnCall(2, &appsv1.Deployment{}, nil)
		depMgr.GetSchemeReturns(&runtime.Scheme{})

		status := appsv1.DeploymentStatus{
			Replicas: int32(0),
		}
		depMgr.DeploymentStatusReturnsOnCall(0, status, nil)

		status.Replicas = 1
		depMgr.DeploymentStatusReturnsOnCall(1, status, nil)

		client = &controllermocks.Client{
			GetStub: func(ctx context.Context, types types.NamespacedName, obj k8sclient.Object) error {
				switch obj.(type) {
				case *batchv1.Job:
					job := obj.(*batchv1.Job)
					job.Status.Active = int32(1)
				}
				return nil
			},
			ListStub: func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
				switch obj.(type) {
				case *corev1.PodList:
					pods := obj.(*corev1.PodList)
					if strings.Contains(opts[0].(*k8sclient.ListOptions).LabelSelector.String(), "job-name") {
						pods.Items = []corev1.Pod{
							{
								Status: corev1.PodStatus{
									ContainerStatuses: []corev1.ContainerStatus{
										{
											State: corev1.ContainerState{
												Terminated: &corev1.ContainerStateTerminated{},
												// Running: &corev1.ContainerStateRunning{},
											},
										},
									},
								},
							},
						}
					}
				}
				return nil
			},
		}
	})

	Context("peer upgrade dbs", func() {
		It("returns error if failed to set replica to zero", func() {
			client.PatchReturnsOnCall(0, errors.New("update error"))
			err := action.UpgradeDBs(depMgr, client, instance, config.DBMigrationTimeouts{})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("update error")))
		})

		It("returns error if failed to set replica to original value", func() {
			client.PatchReturnsOnCall(1, errors.New("update error"))
			err := action.UpgradeDBs(depMgr, client, instance, config.DBMigrationTimeouts{
				JobStart:      common.MustParseDuration("1s"),
				JobCompletion: common.MustParseDuration("1s"),
			})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("update error")))
		})

		It("returns error if failed start job", func() {
			client.CreateReturns(errors.New("job create error"))
			err := action.UpgradeDBs(depMgr, client, instance, config.DBMigrationTimeouts{})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("job create error")))
		})

		It("upgrade dbs", func() {
			err := action.UpgradeDBs(depMgr, client, instance, config.DBMigrationTimeouts{})
			Expect(err).NotTo(HaveOccurred())

			By("starting job", func() {
				Expect(client.CreateCallCount()).To(Equal(1))
			})

			By("updating deployments to update replicas", func() {
				_, dep, _, _ := client.PatchArgsForCall(0)
				Expect(*dep.(*appsv1.Deployment).Spec.Replicas).To(Equal(int32(0)))

				_, dep, _, _ = client.PatchArgsForCall(1)
				Expect(*dep.(*appsv1.Deployment).Spec.Replicas).To(Equal(int32(1)))

				Expect(client.PatchCallCount()).To(Equal(2))
			})
		})
	})
})
