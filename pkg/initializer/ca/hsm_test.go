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

package initializer_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	caconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("HSM CA initializer", func() {
	var (
		client *mocks.Client
		ca     *mocks.IBPCA
		// defaultConfig *mocks.CAConfig

		hsmca    *initializer.HSM
		instance *current.IBPCA
	)

	BeforeEach(func() {
		client = &mocks.Client{
			GetStub: func(ctx context.Context, nn types.NamespacedName, obj k8sclient.Object) error {
				switch obj.(type) {
				case *batchv1.Job:
					j := obj.(*batchv1.Job)
					j.Status.Active = int32(1)
					j.Name = "test-job"
				}
				return nil
			},
			ListStub: func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
				switch obj.(type) {
				case *corev1.PodList:
					pods := obj.(*corev1.PodList)
					pods.Items = []corev1.Pod{
						{
							Status: corev1.PodStatus{
								Phase: corev1.PodSucceeded,
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
				}
				return nil
			},
		}

		hsmConfig := &config.HSMConfig{
			Type:    "hsm",
			Version: "v1",
			Library: config.Library{
				FilePath: "/usr/lib/libCryptoki2_64.so",
				Image:    "ghcr.io/ibm-blockchain/gemalto-client:skarim-amd64",
				Auth: &config.Auth{
					ImagePullSecret: "hsmpullsecret",
				},
			},
			Envs: []corev1.EnvVar{
				{
					Name:  "DUMMY_ENV_NAME",
					Value: "DUMMY_ENV_VALUE",
				},
			},
			MountPaths: []config.MountPath{
				{
					Name:      "hsmcrypto",
					Secret:    "hsmcrypto",
					MountPath: "/hsm",
					Paths: []config.Path{
						{
							Key:  "cafile.pem",
							Path: "cafile.pem",
						},
					},
				},
				{
					Name:      "hsmconfig",
					Secret:    "hsmcrypto",
					MountPath: "/etc/Chrystoki.conf",
					SubPath:   "Chrystoki.conf",
				},
			},
		}

		ca = &mocks.IBPCA{}
		ca.GetServerConfigReturns(&v1.ServerConfig{
			CAConfig: v1.CAConfig{
				CSP: &v1.BCCSP{
					PKCS11: &v1.PKCS11Opts{},
				},
			},
		})
		ca.GetTypeReturns(caconfig.EnrollmentCA)

		instance = &current.IBPCA{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ibpca",
			},
			Spec: current.IBPCASpec{
				Resources: &current.CAResources{
					CA:   &corev1.ResourceRequirements{},
					Init: &corev1.ResourceRequirements{},
				},
				Images: &current.CAImages{},
			},
		}

		hsmca = &initializer.HSM{
			Config: hsmConfig,
			Timeouts: initializer.HSMInitJobTimeouts{
				JobStart:      common.MustParseDuration("1s"),
				JobCompletion: common.MustParseDuration("1s"),
			},
			Client: client,
		}
	})

	Context("creates", func() {
		It("returns error if overriding server config fails", func() {
			ca.OverrideServerConfigReturns(errors.New("override failed"))
			_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
			Expect(err).To(MatchError("override failed"))
		})

		It("returns error if creating ca crypto secret fails", func() {
			client.CreateReturnsOnCall(0, errors.New("failed to create crypto secret"))
			_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
			Expect(err).To(MatchError(ContainSubstring("failed to create crypto secret")))
		})

		It("returns error if creating ca config map fails", func() {
			client.CreateReturnsOnCall(1, errors.New("failed to create config map"))
			_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
			Expect(err).To(MatchError(ContainSubstring("failed to create config map")))
		})

		It("returns error if creating job fails", func() {
			client.CreateReturnsOnCall(2, errors.New("failed to create job"))
			_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
			Expect(err).To(MatchError(ContainSubstring("failed to create job")))
		})

		Context("job start timeout", func() {
			BeforeEach(func() {
				client.GetStub = func(ctx context.Context, nn types.NamespacedName, obj k8sclient.Object) error {
					switch obj.(type) {
					case *batchv1.Job:
						j := obj.(*batchv1.Job)
						j.Status.Active = int32(0)
						j.Name = "test-job"

					}
					return nil
				}
			})

			It("returns error if job doesn't start before timeout", func() {
				_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
				Expect(err).To(MatchError(ContainSubstring("job failed to start")))
			})
		})

		Context("job fails", func() {
			When("job timesout", func() {
				BeforeEach(func() {
					client.ListStub = func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						switch obj.(type) {
						case *corev1.PodList:
							p := obj.(*corev1.PodList)
							p.Items = []corev1.Pod{}
						}
						return nil
					}
				})

				It("returns error", func() {
					_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
					Expect(err).To(MatchError(ContainSubstring("failed to finish")))
				})
			})

			When("pod enters failed state", func() {
				BeforeEach(func() {
					client.ListStub = func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						switch obj.(type) {
						case *corev1.PodList:
							p := obj.(*corev1.PodList)
							p.Items = []corev1.Pod{{
								Status: corev1.PodStatus{
									Phase: corev1.PodFailed,
								},
							}}
						}
						return nil
					}
				})

				It("returns error", func() {
					_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("check job '%s' pods for errors", instance.GetName()+"-ca-init"))))
				})
			})
		})

		It("returns error if unable to delete job after success", func() {
			client.DeleteReturns(errors.New("failed to delete job"))
			_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
			Expect(err).To(MatchError(ContainSubstring("failed to delete job")))
		})

		It("returns error if unable to update ca config map", func() {
			client.UpdateReturns(errors.New("failed to update ca config map"))
			_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
			Expect(err).To(MatchError(ContainSubstring("failed to update ca config map")))
		})

		It("returns sucessfully with no error and nil response", func() {
			_, err := hsmca.Create(instance, &v1.ServerConfig{}, ca)
			Expect(err).NotTo(HaveOccurred())

			By("creating a job resource", func() {
				_, obj, _ := client.CreateArgsForCall(2)
				Expect(obj).NotTo(BeNil())

				job := obj.(*batchv1.Job)
				Expect(job.Spec.Template.Spec.Containers[0].Env).To(ContainElements(
					corev1.EnvVar{
						Name:  "DUMMY_ENV_NAME",
						Value: "DUMMY_ENV_VALUE",
					},
				))

				Expect(job.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElements([]corev1.VolumeMount{
					{
						Name:      "hsmcrypto",
						MountPath: "/hsm",
					},
					{
						Name:      "hsmconfig",
						MountPath: "/etc/Chrystoki.conf",
						SubPath:   "Chrystoki.conf",
					},
				}))

				Expect(job.Spec.Template.Spec.Volumes).To(ContainElements([]corev1.Volume{
					{
						Name: "hsmconfig",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "hsmcrypto",
							},
						},
					},
					{
						Name: "hsmcrypto",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "hsmcrypto",
								Items: []corev1.KeyToPath{
									{
										Key:  "cafile.pem",
										Path: "cafile.pem",
									},
								},
							},
						},
					},
				}))
			})

			By("deleting completed job", func() {
				// One delete count to delete job and second delete count to delete associated pod
				Expect(client.DeleteCallCount()).To(Equal(2))
			})

			By("updating config map if enrollment CA", func() {
				Expect(client.UpdateCallCount()).To(Equal(1))
			})
		})
	})
})
