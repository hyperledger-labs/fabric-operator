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

package v25_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	v2peer "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v2"
	v25config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v25"
	v25 "github.com/IBM-Blockchain/fabric-operator/pkg/migrator/peer/fabric/v25"
	"github.com/IBM-Blockchain/fabric-operator/pkg/migrator/peer/fabric/v25/mocks"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var _ = Describe("V2 peer migrator", func() {
	var (
		deploymentManager *mocks.DeploymentManager
		configMapManager  *mocks.ConfigMapManager
		client            *controllermocks.Client
		migrator          *v25.Migrate
		instance          *current.IBPPeer
	)
	const FABRIC_V2 = "2.2.5-1"
	BeforeEach(func() {
		deploymentManager = &mocks.DeploymentManager{}
		configMapManager = &mocks.ConfigMapManager{}
		client = &controllermocks.Client{}

		instance = &current.IBPPeer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ibppeer",
			},
			Spec: current.IBPPeerSpec{
				Images: &current.PeerImages{
					PeerImage: "peerimage",
					PeerTag:   "peertag",
				},
				Resources: &current.PeerResources{},
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
							v1.Container{
								Name: "dind",
							},
						},
					},
				},
			},
		}
		deploymentManager.GetReturns(dep, nil)
		deploymentManager.DeploymentStatusReturns(appsv1.DeploymentStatus{}, nil)
		deploymentManager.GetSchemeReturns(&runtime.Scheme{})

		client.GetStub = func(ctx context.Context, types types.NamespacedName, obj k8sclient.Object) error {
			switch obj.(type) {
			case *batchv1.Job:
				job := obj.(*batchv1.Job)
				job.Status.Active = int32(1)
			}
			return nil
		}

		configMapManager.GetCoreConfigReturns(&corev1.ConfigMap{
			BinaryData: map[string][]byte{
				"core.yaml": []byte{},
			},
		}, nil)

		migrator = &v25.Migrate{
			DeploymentManager: deploymentManager,
			ConfigMapManager:  configMapManager,
			Client:            client,
		}
	})

	Context("migration needed", func() {
		It("returns false if deployment not found", func() {
			deploymentManager.GetReturns(nil, errors.New("not found"))
			needed := migrator.MigrationNeeded(instance)
			Expect(needed).To(Equal(false))
		})

		It("returns true if config map not updated", func() {
			dep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
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
			deploymentManager.GetReturns(dep, nil)

			needed := migrator.MigrationNeeded(instance)
			Expect(needed).To(Equal(true))
		})

		It("returns true if deployment has dind container", func() {
			needed := migrator.MigrationNeeded(instance)
			Expect(needed).To(Equal(true))
		})
	})

	Context("upgrade dbs peer", func() {
		BeforeEach(func() {
			client.ListStub = func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
				if strings.Contains(opts[0].(*k8sclient.ListOptions).LabelSelector.String(), "app") {
					pods := obj.(*corev1.PodList)
					pods.Items = []corev1.Pod{}
				}
				if strings.Contains(opts[0].(*k8sclient.ListOptions).LabelSelector.String(), "job-name") {
					pods := obj.(*corev1.PodList)
					pods.Items = []corev1.Pod{
						corev1.Pod{
							Status: corev1.PodStatus{
								ContainerStatuses: []corev1.ContainerStatus{
									corev1.ContainerStatus{
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
			}
		})

		It("returns an error if unable to reset peer", func() {
			deploymentManager.GetReturns(nil, errors.New("restore failed"))
			err := migrator.UpgradeDBs(instance, config.DBMigrationTimeouts{
				JobStart:      common.MustParseDuration("1s"),
				JobCompletion: common.MustParseDuration("1s"),
			})
			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(ContainSubstring("restore failed")))
		})

		It("upgrade dbs", func() {
			status := appsv1.DeploymentStatus{
				Replicas: int32(0),
			}
			deploymentManager.DeploymentStatusReturnsOnCall(0, status, nil)

			status.Replicas = 1
			deploymentManager.DeploymentStatusReturnsOnCall(1, status, nil)

			err := migrator.UpgradeDBs(instance, config.DBMigrationTimeouts{
				JobStart:      common.MustParseDuration("1s"),
				JobCompletion: common.MustParseDuration("1s"),
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("update config", func() {
		It("returns an error if unable to get config map", func() {
			configMapManager.GetCoreConfigReturns(nil, errors.New("get config map failed"))
			err := migrator.UpdateConfig(instance, FABRIC_V2)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(ContainSubstring("get config map failed")))
		})

		It("returns an error if unable to update config map", func() {
			configMapManager.CreateOrUpdateReturns(errors.New("update config map failed"))
			err := migrator.UpdateConfig(instance, FABRIC_V2)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(ContainSubstring("update config map failed")))
		})

		It("sets relevant v25.x fields in config", func() {
			err := migrator.UpdateConfig(instance, FABRIC_V2)
			Expect(err).NotTo(HaveOccurred())

			_, config := configMapManager.CreateOrUpdateArgsForCall(0)
			core := config.(*v25config.Core)

			By("setting external builder", func() {
				Expect(core.Chaincode.ExternalBuilders).To(ContainElement(
					v2peer.ExternalBuilder{
						Name: "ibp-builder",
						Path: "/usr/local",
						EnvironmentWhiteList: []string{
							"IBP_BUILDER_ENDPOINT",
							"IBP_BUILDER_SHARED_DIR",
						},
						PropogateEnvironment: []string{
							"IBP_BUILDER_ENDPOINT",
							"IBP_BUILDER_SHARED_DIR",
							"PEER_NAME",
						},
					},
				))
			})

			By("setting install timeout", func() {
				Expect(core.Chaincode.InstallTimeout).To(Equal(common.MustParseDuration("300s")))
			})

			By("setting lifecycle chaincode", func() {
				Expect(core.Chaincode.System["_lifecycle"]).To(Equal("enable"))
			})

			By("setting limits", func() {
				Expect(core.Peer.Limits).To(Equal(v2peer.Limits{
					Concurrency: v2peer.Concurrency{
						DeliverService:  2500,
						EndorserService: 2500,
					},
				}))
			})

			By("setting implicit collection dissemination policy", func() {
				Expect(core.Peer.Gossip.PvtData.ImplicitCollectionDisseminationPolicy).To(Equal(v2peer.ImplicitCollectionDisseminationPolicy{
					RequiredPeerCount: 0,
					MaxPeerCount:      1,
				}))
			})

		})

		It("updates config map", func() {
			err := migrator.UpdateConfig(instance, FABRIC_V2)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("set chaincode launcher resource on CR", func() {
		BeforeEach(func() {
			client.GetStub = func(ctx context.Context, nn types.NamespacedName, obj k8sclient.Object) error {
				switch obj.(type) {
				case *corev1.ConfigMap:
					dep := &deployer.Config{
						Defaults: &deployer.Defaults{
							Resources: &deployer.Resources{
								Peer: &current.PeerResources{
									CCLauncher: &corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("2"),
											corev1.ResourceMemory: resource.MustParse("200Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("3"),
											corev1.ResourceMemory: resource.MustParse("3Gi"),
										},
									},
								},
							},
						},
					}

					bytes, err := yaml.Marshal(dep)
					Expect(err).NotTo(HaveOccurred())

					cm := obj.(*corev1.ConfigMap)
					cm.Data = map[string]string{
						"settings.yaml": string(bytes),
					}
				}

				return nil
			}

			client.ListStub = func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
				switch obj.(type) {
				case *current.IBPConsoleList:
					list := obj.(*current.IBPConsoleList)
					list.Items = []current.IBPConsole{current.IBPConsole{}}
				}

				return nil
			}
		})

		It("sets resources based on deployer config map", func() {
			err := migrator.SetChaincodeLauncherResourceOnCR(instance)
			Expect(err).NotTo(HaveOccurred())

			_, cr, _ := client.UpdateArgsForCall(0)
			Expect(cr).NotTo(BeNil())
			Expect(*cr.(*current.IBPPeer).Spec.Resources.CCLauncher).To(Equal(corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("3"),
					corev1.ResourceMemory: resource.MustParse("3Gi"),
				}},
			))
		})

		It("sets resources default config map", func() {
			client.GetStub = nil

			err := migrator.SetChaincodeLauncherResourceOnCR(instance)
			Expect(err).NotTo(HaveOccurred())

			_, cr, _ := client.UpdateArgsForCall(0)
			Expect(cr).NotTo(BeNil())
			Expect(*cr.(*current.IBPPeer).Spec.Resources.CCLauncher).To(Equal(corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0.1"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				}},
			))
		})
	})
})
