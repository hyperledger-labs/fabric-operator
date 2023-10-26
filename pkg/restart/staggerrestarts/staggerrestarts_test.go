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

package staggerrestarts_test

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/staggerrestarts"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Staggerrestarts", func() {

	var (
		mockClient *controllermocks.Client
		service    *staggerrestarts.StaggerRestartsService
		instance   *current.IBPPeer
	)

	BeforeEach(func() {
		mockClient = &controllermocks.Client{}
		service = staggerrestarts.New(mockClient, 5*time.Minute)

		instance = &current.IBPPeer{}
		instance.Name = "org1peer1"
		instance.Namespace = "namespace"
		instance.Spec.MSPID = "org1"
	})

	Context("add to queue", func() {
		It("returns error if failed to get restart config", func() {
			mockClient.GetReturns(errors.New("get error"))
			err := service.AddToQueue(instance, "reason")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to add org1peer1 to queue"))
		})

		It("returns error if failed to update restart config", func() {
			mockClient.CreateOrUpdateReturns(errors.New("update error"))
			err := service.AddToQueue(instance, "reason")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to add org1peer1 to queue"))
		})

		It("adds restart request to queue in restart config", func() {
			err := service.AddToQueue(instance, "reason")
			Expect(err).NotTo(HaveOccurred())

			_, cm, _ := mockClient.CreateOrUpdateArgsForCall(0)
			cfg := getRestartConfig(cm.(*corev1.ConfigMap))

			Expect(len(cfg.Queues["org1"])).To(Equal(1))
			comp := cfg.Queues["org1"][0]
			Expect(comp.CRName).To(Equal("org1peer1"))
			Expect(comp.Reason).To(Equal("reason"))
			Expect(comp.Status).To(Equal(staggerrestarts.Pending))

		})
	})

	Context("reconcile", func() {
		var (
			restartConfig *staggerrestarts.RestartConfig
			component1    *staggerrestarts.Component
			component2    *staggerrestarts.Component
			component3    *staggerrestarts.Component

			pod *corev1.Pod
			dep *appsv1.Deployment
		)

		BeforeEach(func() {
			component1 = &staggerrestarts.Component{
				CRName: "org1peer1",
				Reason: "migration",
				Status: staggerrestarts.Pending,
			}
			component2 = &staggerrestarts.Component{
				CRName: "org1peer2",
				Reason: "migration",
				Status: staggerrestarts.Pending,
			}
			component3 = &staggerrestarts.Component{
				CRName: "org2peer1",
				Reason: "migration",
				Status: staggerrestarts.Pending,
			}

			restartConfig = &staggerrestarts.RestartConfig{
				Queues: map[string][]*staggerrestarts.Component{
					"org1": {component1, component2},
					"org2": {component3},
				},
			}

			pod = &corev1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "pod1",
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Ready: true,
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						},
					},
					Phase: corev1.PodRunning,
				},
			}
			replicas := int32(1)
			dep = &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: &replicas,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								corev1.Container{
									Name: "org1peer1",
								},
							},
						},
					},
				},
			}
			bytes, err := json.Marshal(restartConfig)
			Expect(err).NotTo(HaveOccurred())

			mockClient.GetStub = func(ctx context.Context, ns types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *corev1.ConfigMap:
					o := obj.(*corev1.ConfigMap)
					o.Name = ns.Name
					o.Namespace = instance.Namespace
					o.BinaryData = map[string][]byte{
						"restart-config.yaml": bytes,
					}
				case *appsv1.Deployment:
					o := obj.(*appsv1.Deployment)
					o.Name = ns.Name
					o.Namespace = instance.Namespace
				}

				return nil
			}

			mockClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...k8sclient.ListOption) error {
				switch obj.(type) {
				case *corev1.PodList:
					pods := obj.(*corev1.PodList)
					pods.Items = []corev1.Pod{*pod}
				case *appsv1.DeploymentList:
					deployments := obj.(*appsv1.DeploymentList)
					deployments.Items = []appsv1.Deployment{*dep}
				}
				return nil
			}
		})

		Context("pending", func() {
			It("returns empty pod list if failed to get running pods", func() {
				mockClient.ListReturns(errors.New("list error"))
				mockClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...k8sclient.ListOption) error {
					switch obj.(type) {
					case *appsv1.DeploymentList:
						deployments := obj.(*appsv1.DeploymentList)
						deployments.Items = []appsv1.Deployment{*dep}
					}
					return nil
				}
				requeue, err := service.Reconcile("peer", "namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(requeue).To(Equal(false))

				_, cm, _ := mockClient.CreateOrUpdateArgsForCall(0)
				cfg := getRestartConfig(cm.(*corev1.ConfigMap))

				By("restarting first component in queue but not setting Pod Name", func() {
					Expect(cfg.Queues["org1"][0].CRName).To(Equal("org1peer1"))
					Expect(cfg.Queues["org1"][0].Status).To(Equal(staggerrestarts.Waiting))
					Expect(cfg.Queues["org1"][0].PodName).To(Equal(""))
				})
			})

			It("check deleted status when pods/deployments list is empty", func() {
				mockClient.ListReturns(errors.New("list error"))
				mockClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...k8sclient.ListOption) error {
					switch obj.(type) {
					case *appsv1.DeploymentList:
						deployments := obj.(*appsv1.DeploymentList)
						deployments.Items = []appsv1.Deployment{}
					}
					return nil
				}
				requeue, err := service.Reconcile("peer", "namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(requeue).To(Equal(false))

				_, cm, _ := mockClient.CreateOrUpdateArgsForCall(0)
				cfg := getRestartConfig(cm.(*corev1.ConfigMap))
				By("deleting first component from queue, immediate second component will be in pending state", func() {
					Expect(cfg.Queues["org1"][0].CRName).To(Equal("org1peer2"))
					Expect(cfg.Queues["org1"][0].Status).To(Equal(staggerrestarts.Pending))
					Expect(cfg.Queues["org1"][0].PodName).To(Equal(""))
				})

				By("moving the component to the log and setting status to deleted", func() {
					Expect(len(cfg.Log)).To(Equal(2)) // since org1peer1 and org2peer1 has been deleted

					for _, components := range cfg.Log {
						Expect(components[0].CRName).To(ContainSubstring("peer1")) // org1peer1 and org2peer1
						Expect(components[0].Status).To(Equal(staggerrestarts.Deleted))
					}
				})
			})

			It("returns error if fails to restart deployment", func() {
				mockClient.PatchReturns(errors.New("patch error"))
				requeue, err := service.Reconcile("peer", "namespace")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to restart deployment"))
				Expect(requeue).To(Equal(false))
			})

			It("restarts deployment for pending component", func() {
				requeue, err := service.Reconcile("peer", "namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(requeue).To(Equal(false))

				_, cm, _ := mockClient.CreateOrUpdateArgsForCall(0)
				cfg := getRestartConfig(cm.(*corev1.ConfigMap))

				By("restarting first component in org1 queue", func() {
					Expect(cfg.Queues["org1"][0].CRName).To(Equal("org1peer1"))
					Expect(cfg.Queues["org1"][0].Status).To(Equal(staggerrestarts.Waiting))
					Expect(cfg.Queues["org1"][0].PodName).To(Equal("pod1"))
					Expect(cfg.Queues["org1"][0].LastCheckedTimestamp).NotTo(Equal(""))
					Expect(cfg.Queues["org1"][0].CheckUntilTimestamp).NotTo(Equal(""))
				})

				By("restarting first component in org2 queue", func() {
					Expect(cfg.Queues["org2"][0].CRName).To(Equal("org2peer1"))
					Expect(cfg.Queues["org2"][0].Status).To(Equal(staggerrestarts.Waiting))
					Expect(cfg.Queues["org2"][0].PodName).To(Equal("pod1"))
					Expect(cfg.Queues["org2"][0].LastCheckedTimestamp).NotTo(Equal(""))
					Expect(cfg.Queues["org2"][0].CheckUntilTimestamp).NotTo(Equal(""))
				})

			})
		})

		Context("waiting", func() {
			var (
				originalLastChecked string
			)
			BeforeEach(func() {
				originalLastChecked = time.Now().Add(-35 * time.Second).UTC().String()
				checkUntil := time.Now().Add(5 * time.Minute).UTC().String()

				component1.Status = staggerrestarts.Waiting
				component1.LastCheckedTimestamp = originalLastChecked
				component1.CheckUntilTimestamp = checkUntil
				component1.PodName = "pod1"

				component3.Status = staggerrestarts.Waiting
				component3.LastCheckedTimestamp = originalLastChecked
				component3.CheckUntilTimestamp = checkUntil
				component3.PodName = "pod1"

				// Make sure returned restartConfig contains updated components
				bytes, err := json.Marshal(restartConfig)
				Expect(err).NotTo(HaveOccurred())

				mockClient.GetStub = func(ctx context.Context, ns types.NamespacedName, obj client.Object) error {
					o := obj.(*corev1.ConfigMap)
					o.Name = ns.Name
					o.Namespace = instance.Namespace
					o.BinaryData = map[string][]byte{
						"restart-config.yaml": bytes,
					}

					return nil
				}
			})

			It("keeps components in Waiting status if unable to get list of pods", func() {
				mockClient.ListReturns(errors.New("list error"))
				requeue, err := service.Reconcile("peer", "namespace")
				Expect(err).NotTo(HaveOccurred())

				By("returning false to requeue the restart reconcile request if LastCheckedTimestamp was last updated more than 10-30 seconds ago", func() {
					Expect(requeue).To(Equal(false))
				})

				_, cm, _ := mockClient.CreateOrUpdateArgsForCall(0)
				cfg := getRestartConfig(cm.(*corev1.ConfigMap))

				By("keeping first component of each queue in Waiting status and updating LastCheckedTime", func() {
					for _, q := range cfg.Queues {
						comp := q[0]
						Expect(comp.Status).To(Equal(staggerrestarts.Waiting))
						Expect(comp.PodName).To(Equal("pod1"))
						Expect(comp.LastCheckedTimestamp).NotTo(Equal(originalLastChecked))
					}
				})
			})

			It("keeps components in Waiting status if there is more than one running pod for the instance", func() {
				pod2 := pod.DeepCopy()
				pod2.Name = "pod2"
				mockClient.ListStub = func(ctx context.Context, obj client.ObjectList, opts ...k8sclient.ListOption) error {
					switch obj.(type) {
					case *corev1.PodList:
						pods := obj.(*corev1.PodList)
						pods.Items = []corev1.Pod{*pod, *pod2}
					case *appsv1.DeploymentList:
						deployments := obj.(*appsv1.DeploymentList)
						deployments.Items = []appsv1.Deployment{*dep}
					}
					return nil
				}

				requeue, err := service.Reconcile("peer", "namespace")
				Expect(err).NotTo(HaveOccurred())

				By("returning false to requeue the restart reconcile request if LastCheckedTimestamp was last updated more than 10-30 seconds ago", func() {
					Expect(requeue).To(Equal(false))
				})

				_, cm, _ := mockClient.CreateOrUpdateArgsForCall(0)
				cfg := getRestartConfig(cm.(*corev1.ConfigMap))

				By("keeping first component of each queue in Waiting status and updating LastCheckedTime", func() {
					for _, q := range cfg.Queues {
						comp := q[0]
						Expect(comp.Status).To(Equal(staggerrestarts.Waiting))
						Expect(comp.PodName).To(Equal("pod1"))
						Expect(comp.LastCheckedTimestamp).NotTo(Equal(originalLastChecked))
					}
				})
			})

			It("sets component to Completed and moves it to the log if pod has restarted", func() {
				pod.Name = "newpod"

				requeue, err := service.Reconcile("peer", "namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(requeue).To(Equal(false))

				_, cm, _ := mockClient.CreateOrUpdateArgsForCall(0)
				cfg := getRestartConfig(cm.(*corev1.ConfigMap))

				By("removing the component from its queue", func() {
					Expect(len(cfg.Queues["org1"])).To(Equal(1)) // now contains only org1peer2
					Expect(len(cfg.Queues["org2"])).To(Equal(0)) // now contains no peers
				})

				By("moving the component to the log and setting status to Completed", func() {
					Expect(len(cfg.Log)).To(Equal(2)) // since both org1peer1 and org2peer1 restarted
					Expect(len(cfg.Log["org1peer1"])).To(Equal(1))
					Expect(len(cfg.Log["org2peer1"])).To(Equal(1))

					for _, components := range cfg.Log {
						Expect(components[0].CRName).To(ContainSubstring("peer1"))
						Expect(components[0].Status).To(Equal(staggerrestarts.Completed))
					}
				})
			})

			It("sets component to Expired and moves it to the log if pod has not restarted within timeout window", func() {
				component1.CheckUntilTimestamp = time.Now().Add(-5 * time.Second).UTC().String()
				bytes, err := json.Marshal(restartConfig)
				Expect(err).NotTo(HaveOccurred())

				mockClient.GetStub = func(ctx context.Context, ns types.NamespacedName, obj client.Object) error {
					o := obj.(*corev1.ConfigMap)
					o.Name = ns.Name
					o.Namespace = instance.Namespace
					o.BinaryData = map[string][]byte{
						"restart-config.yaml": bytes,
					}

					return nil
				}

				requeue, err := service.Reconcile("peer", "namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(requeue).To(Equal(false))

				_, cm, _ := mockClient.CreateOrUpdateArgsForCall(0)
				cfg := getRestartConfig(cm.(*corev1.ConfigMap))

				By("removing org1peer1 from its queue", func() {
					Expect(len(cfg.Queues["org1"])).To(Equal(1)) // now contains only org1peer2
				})

				By("keeping org2peer1 in its queue as it's timeout window has not expired yet", func() {
					Expect(len(cfg.Queues["org2"])).To(Equal(1))
				})

				By("moving org1peer1 to the log and setting Status to Expired", func() {
					Expect(len(cfg.Log["org1peer1"])).To(Equal(1))
					comp1 := cfg.Log["org1peer1"][0]
					Expect(comp1.CRName).To(Equal("org1peer1"))
					Expect(comp1.Status).To(Equal(staggerrestarts.Expired))
				})
			})
		})
	})
})

func getRestartConfig(cm *corev1.ConfigMap) *staggerrestarts.RestartConfig {
	cfgBytes := cm.BinaryData["restart-config.yaml"]
	cfg := &staggerrestarts.RestartConfig{}
	err := json.Unmarshal(cfgBytes, cfg)
	Expect(err).NotTo(HaveOccurred())

	return cfg
}
