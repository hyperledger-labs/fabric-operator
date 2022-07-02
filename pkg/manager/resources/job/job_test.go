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

package job_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/job"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/job/mocks"

	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Job", func() {
	var (
		k8sJob  *v1.Job
		testJob *job.Job
	)

	BeforeEach(func() {
		k8sJob = &v1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "k8sJob",
				Namespace: "default",
			},
		}
		testJob = &job.Job{
			Job: k8sJob,
		}
	})

	It("creates job with defaults", func() {
		testJob = job.NewWithDefaults(k8sJob)
		Expect(testJob.Timeouts).To(Equal(&job.Timeouts{
			WaitUntilActive:   60 * time.Second,
			WaitUntilFinished: 60 * time.Second,
		}))

		By("adding unique id to job name", func() {
			Expect(testJob.Name).To(ContainSubstring("k8sJob-"))
		})
	})

	It("adds container", func() {
		cont := container.Container{
			Container: &corev1.Container{
				Name: "test-cont",
			},
		}

		testJob.AddContainer(cont)
		Expect(len(testJob.Spec.Template.Spec.Containers)).To(Equal(1))
		Expect(testJob.Spec.Template.Spec.Containers[0]).To(Equal(*cont.Container))
	})

	Context("volumes", func() {
		BeforeEach(func() {
			testJob.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "test-volume",
				},
			}
		})

		It("appends volume if missing", func() {
			testJob.AppendVolumeIfMissing(corev1.Volume{Name: "test-volume"})
			testJob.AppendVolumeIfMissing(corev1.Volume{Name: "test-volume2"})

			Expect(len(testJob.Spec.Template.Spec.Volumes)).To(Equal(2))
			Expect(testJob.Spec.Template.Spec.Volumes[1]).To(Equal(corev1.Volume{Name: "test-volume2"}))
		})
	})

	Context("image pull secrets", func() {
		BeforeEach(func() {
			testJob.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: "pullsecret",
				},
			}
		})

		It("appends volume if missing", func() {
			testJob.AppendPullSecret(corev1.LocalObjectReference{Name: "pullsecret"})
			testJob.AppendPullSecret(corev1.LocalObjectReference{Name: "pullsecret2"})

			Expect(len(testJob.Spec.Template.Spec.ImagePullSecrets)).To(Equal(2))
			Expect(testJob.Spec.Template.Spec.ImagePullSecrets[1]).To(
				Equal(corev1.LocalObjectReference{Name: "pullsecret2"}),
			)
		})
	})

	Context("events", func() {
		var (
			client *mocks.Client
		)

		BeforeEach(func() {
			client = &mocks.Client{}

		})

		Context("status", func() {
			Context("failures", func() {
				Context("job", func() {
					When("getting job from API server fails", func() {
						BeforeEach(func() {
							client.GetStub = func(ctx context.Context, nn types.NamespacedName, obj k8sclient.Object) error {
								return errors.New("failed to get job")
							}
						})

						It("returns error and UNKNOWN status", func() {
							status, err := testJob.Status(client)
							Expect(err).To(HaveOccurred())
							Expect(status).To(Equal(job.UNKNOWN))
						})
					})

					When("job has failed", func() {
						BeforeEach(func() {
							client.GetStub = func(ctx context.Context, nn types.NamespacedName, obj k8sclient.Object) error {
								j := obj.(*v1.Job)
								j.Status = v1.JobStatus{
									Failed: int32(1),
								}
								return nil
							}
						})

						It("returns FAILED status", func() {
							status, err := testJob.Status(client)
							Expect(err).NotTo(HaveOccurred())
							Expect(status).To(Equal(job.FAILED))
						})
					})
				})

				Context("pods", func() {
					When("getting pods from API server fails", func() {
						BeforeEach(func() {
							client.ListStub = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
								return errors.New("failed to list pods")
							}
						})

						It("returns error and UNKNOWN status", func() {
							status, err := testJob.Status(client)
							Expect(err).To(HaveOccurred())
							Expect(status).To(Equal(job.UNKNOWN))
						})
					})

					When("job has failed", func() {
						BeforeEach(func() {
							client.ListStub = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
								pods := list.(*corev1.PodList)
								pods.Items = []corev1.Pod{
									{
										Status: corev1.PodStatus{
											Phase: corev1.PodFailed,
										},
									},
								}
								return nil
							}
						})

						It("returns FAILED status", func() {
							status, err := testJob.Status(client)
							Expect(err).NotTo(HaveOccurred())
							Expect(status).To(Equal(job.FAILED))
						})
					})
				})
			})

			It("returns COMPLETED state", func() {
				status, err := testJob.Status(client)
				Expect(err).NotTo(HaveOccurred())
				Expect(status).To(Equal(job.COMPLETED))
			})
		})

		Context("wait until active", func() {
			BeforeEach(func() {
				testJob.Timeouts = &job.Timeouts{
					WaitUntilActive: time.Second,
				}

				client.GetStub = func(ctx context.Context, nn types.NamespacedName, obj k8sclient.Object) error {
					j := obj.(*v1.Job)
					j.Status = v1.JobStatus{
						Active: int32(1),
					}
					return nil
				}
			})

			It("returns before timeout with no errors", func() {
				err := testJob.WaitUntilActive(client)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("wait until finished", func() {
			BeforeEach(func() {
				testJob.Timeouts = &job.Timeouts{
					WaitUntilFinished: time.Second,
				}

				client.ListStub = func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
					pods := list.(*corev1.PodList)
					pods.Items = []corev1.Pod{
						{
							Status: corev1.PodStatus{
								ContainerStatuses: []corev1.ContainerStatus{
									{
										State: corev1.ContainerState{
											Terminated: &corev1.ContainerStateTerminated{},
										},
									},
								},
							},
						},
					}
					return nil
				}
			})

			It("returns before timeout with no errors", func() {
				err := testJob.WaitUntilFinished(client)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
