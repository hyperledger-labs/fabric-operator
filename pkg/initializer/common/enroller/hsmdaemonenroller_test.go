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

package enroller_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	ccmocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller/mocks"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("HSM Daemon sidecar enroller", func() {
	var (
		e           *enroller.HSMDaemonEnroller
		ccClient    *ccmocks.Client
		hsmcaClient *mocks.HSMCAClient
		instance    *mocks.Instance
	)

	BeforeEach(func() {
		instance = &mocks.Instance{}
		instance.GetNameReturns("test")
		instance.PVCNameReturns("test-pvc")

		ccClient = &ccmocks.Client{
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
					p := obj.(*corev1.PodList)
					p.Items = []corev1.Pod{{
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Name: enroller.CertGen,
									State: corev1.ContainerState{
										Terminated: &corev1.ContainerStateTerminated{
											ExitCode: int32(0),
										},
									},
								},
							},
							Phase: corev1.PodSucceeded,
						},
					}}
				}
				return nil
			},
		}

		hsmcaClient = &mocks.HSMCAClient{}
		hsmcaClient.GetEnrollmentRequestReturns(&current.Enrollment{
			CATLS: &current.CATLS{
				CACert: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNGakNDQWIyZ0F3SUJBZ0lVZi84bk94M2NqM1htVzNDSUo1L0Q1ejRRcUVvd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRBek1ERTNNamd3TUZvWERUTTBNVEF5TmpFM01qZ3dNRm93YURFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1Sa3dGd1lEVlFRREV4Qm1ZV0p5YVdNdFkyRXRjMlZ5CmRtVnlNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUVSbzNmbUc2UHkyUHd6cUMwNnFWZDlFOFgKZ044eldqZzFMb3lnMmsxdkQ4MXY1dENRRytCTVozSUJGQnI2VTRhc0tZTUREakd6TElERmdUUTRjVDd1VktORgpNRU13RGdZRFZSMFBBUUgvQkFRREFnRUdNQklHQTFVZEV3RUIvd1FJTUFZQkFmOENBUUV3SFFZRFZSME9CQllFCkZFa0RtUHhjbTdGcXZSMXllN0tNNGdLLy9KZ1JNQW9HQ0NxR1NNNDlCQU1DQTBjQU1FUUNJRC92QVFVSEh2SWwKQWZZLzM5UWdEU2ltTWpMZnhPTG44NllyR1EvWHpkQVpBaUFpUmlyZmlMdzVGbXBpRDhtYmlmRjV4bzdFUzdqNApaUWQyT0FUNCt5OWE0Zz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K",
			},
		})

		hsmConfig := &config.HSMConfig{
			Type:    "hsm",
			Version: "v1",
			Library: config.Library{
				FilePath: "/usr/lib/libCryptoki2_64.so",
				Image:    "ghcr.io/ibm-blockchain/ibp-pkcs11-proxy/gemalto-client:skarim-amd64",
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
			Daemon: &config.Daemon{
				Image: "ghcr.io/ibm-blockchain/ibp-pkcs11-proxy/hsmdaemon:skarim-amd64",
				Auth: &config.Auth{
					ImagePullSecret: "hsmpullsecret",
				},
				Envs: []corev1.EnvVar{
					{
						Name:  "DAEMON_ENV_NAME",
						Value: "DAEMON_ENV_VALUE",
					},
				},
			},
			MountPaths: []config.MountPath{
				{
					MountPath: "/pvc/mount/path",
					UsePVC:    true,
				},
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

		e = &enroller.HSMDaemonEnroller{
			Config:   hsmConfig,
			Client:   ccClient,
			Instance: instance,
			CAClient: hsmcaClient,
			Timeouts: enroller.HSMEnrollJobTimeouts{
				JobStart:      common.MustParseDuration("1s"),
				JobCompletion: common.MustParseDuration("1s"),
			},
		}
	})

	Context("enroll", func() {
		It("returns error if creating ca crypto secret fails", func() {
			ccClient.CreateReturnsOnCall(0, errors.New("failed to create root TLS secret"))
			_, err := e.Enroll()
			Expect(err).To(MatchError(ContainSubstring("failed to create root TLS secret")))
		})

		It("returns error if creating ca config map fails", func() {
			ccClient.CreateReturnsOnCall(1, errors.New("failed to create ca config map"))
			_, err := e.Enroll()
			Expect(err).To(MatchError(ContainSubstring("failed to create ca config map")))
		})

		It("returns error if creating job fails", func() {
			ccClient.CreateReturnsOnCall(2, errors.New("failed to create job"))
			_, err := e.Enroll()
			Expect(err).To(MatchError(ContainSubstring("failed to create job")))
		})

		Context("job start timeout", func() {
			BeforeEach(func() {
				ccClient.GetStub = func(ctx context.Context, nn types.NamespacedName, obj k8sclient.Object) error {
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
				_, err := e.Enroll()
				Expect(err).To(MatchError(ContainSubstring("job failed to start")))
			})
		})

		Context("job fails", func() {
			When("job timesout", func() {
				BeforeEach(func() {
					ccClient.ListStub = func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						switch obj.(type) {
						case *corev1.PodList:
							p := obj.(*corev1.PodList)
							p.Items = []corev1.Pod{
								{
									Status: corev1.PodStatus{
										ContainerStatuses: []corev1.ContainerStatus{
											{
												Name:  enroller.CertGen,
												State: corev1.ContainerState{},
											},
										},
									},
								},
							}
						}
						return nil
					}
				})

				It("returns error", func() {
					_, err := e.Enroll()
					Expect(err).To(MatchError(ContainSubstring("failed to finish")))
				})
			})

			When("pod enters failed state", func() {
				BeforeEach(func() {
					ccClient.ListStub = func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						switch obj.(type) {
						case *corev1.PodList:
							p := obj.(*corev1.PodList)
							p.Items = []corev1.Pod{
								{
									Status: corev1.PodStatus{
										ContainerStatuses: []corev1.ContainerStatus{
											{
												Name: enroller.CertGen,
												State: corev1.ContainerState{
													Terminated: &corev1.ContainerStateTerminated{
														ExitCode: int32(1),
													},
												},
											},
										},
									},
								},
							}
						}
						return nil
					}
				})

				It("returns error", func() {
					_, err := e.Enroll()
					Expect(err).To(MatchError(ContainSubstring("finished unsuccessfully, not cleaning up pods to allow for error")))
				})
			})
		})

		It("returns no error on successfull enroll", func() {
			resp, err := e.Enroll()
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())

			By("creating a job resource", func() {
				_, obj, _ := ccClient.CreateArgsForCall(2)
				Expect(obj).NotTo(BeNil())

				job := obj.(*batchv1.Job)
				Expect(len(job.Spec.Template.Spec.Containers)).To(Equal(2))

				Expect(job.Spec.Template.Spec.Containers[0].Env).To(Equal([]corev1.EnvVar{
					{
						Name:  "DUMMY_ENV_NAME",
						Value: "DUMMY_ENV_VALUE",
					},
				}))

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
					{
						Name:      fmt.Sprintf("%s-pvc-volume", instance.GetName()),
						MountPath: "/pvc/mount/path",
					},
				}))

				Expect(job.Spec.Template.Spec.Containers[1].Env).To(Equal([]corev1.EnvVar{
					{
						Name:  "DAEMON_ENV_NAME",
						Value: "DAEMON_ENV_VALUE",
					},
				}))

				Expect(job.Spec.Template.Spec.Containers[1].VolumeMounts).To(ContainElements([]corev1.VolumeMount{
					{
						Name:      "shared",
						MountPath: "/shared",
					},
					{
						Name:      "hsmcrypto",
						MountPath: "/hsm",
					},
					{
						Name:      "hsmconfig",
						MountPath: "/etc/Chrystoki.conf",
						SubPath:   "Chrystoki.conf",
					},
					{
						Name:      fmt.Sprintf("%s-pvc-volume", instance.GetName()),
						MountPath: "/pvc/mount/path",
					},
				}))

				Expect(job.Spec.Template.Spec.Volumes).To(ContainElements([]corev1.Volume{
					{
						Name: fmt.Sprintf("%s-pvc-volume", instance.GetName()),
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "test-pvc",
							},
						},
					},
					{
						Name: "shared",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{
								Medium: corev1.StorageMediumMemory,
							},
						},
					},
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
				// One delete to clean up ca config map before starting job
				// Second delete to delete job
				// Third delete to delete associated pod
				// Fourth delete to delete root tls secret
				// Fifth delete to delete ca config map
				Expect(ccClient.DeleteCallCount()).To(Equal(5))
			})

			By("setting controller reference on resources created by enroll job", func() {
				Expect(ccClient.UpdateCallCount()).To(Equal(4))
			})
		})
	})

	// TODO: Add more tests for error path testing
})
