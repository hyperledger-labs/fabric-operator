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

package console_test

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
)

var (
	console  *Console
	console2 *Console // DISABLED
	console3 *Console
)

var (
	defaultRequestsConsole = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("30m"),
		corev1.ResourceMemory:           resource.MustParse("60M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
	}

	defaultLimitsConsole = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("300m"),
		corev1.ResourceMemory:           resource.MustParse("600M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
	}

	defaultRequestsConfigtxlator = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("25m"),
		corev1.ResourceMemory:           resource.MustParse("50M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
	}

	defaultLimitsConfigtxlator = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("25m"),
		corev1.ResourceMemory:           resource.MustParse("50M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
	}

	defaultRequestsCouchdb = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("30m"),
		corev1.ResourceMemory:           resource.MustParse("60M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
	}

	defaultLimitsCouchdb = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("300m"),
		corev1.ResourceMemory:           resource.MustParse("600M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
	}

	defaultRequestsDeployer = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("10m"),
		corev1.ResourceMemory:           resource.MustParse("20M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
	}

	defaultLimitsDeployer = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("100m"),
		corev1.ResourceMemory:           resource.MustParse("200M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
	}

	useTagsFlag = true
)

var _ = Describe("Interaction between IBP-Operator and Kubernetes cluster", func() {
	SetDefaultEventuallyTimeout(240 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("IBPConsole controller", func() {
		Context("applying incorrectly configured third instance of IBPConsole CR", func() {
			It("should set the CR status to error", func() {
				Eventually(console3.pollForCRStatus).Should((Equal(current.Error)))

				crStatus := &current.IBPConsole{}
				result := ibpCRClient.Get().Namespace(namespace).Resource("ibpconsoles").Name(console3.Name).Do(context.TODO())
				result.Into(crStatus)

				Expect(crStatus.Status.Message).To(ContainSubstring("Service account name not provided"))
			})

			It("should delete the third instance of IBPConsole CR", func() {
				result := ibpCRClient.Delete().Namespace(namespace).Resource("ibpconsoles").Name(console3.Name).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())
			})
		})

		// This test is disabled as it doesn't test anything interesting AND it consumes
		// too many resources on the GHA pipeline, causing the primary test flow to starve
		// and eventually time out.
		PContext("applying the second instance of IBPConsole CR", func() {
			var (
				err error
				dep *appsv1.Deployment
			)

			It("creates a second IBPConsole custom resource", func() {
				By("starting a pod", func() {
					Eventually(console2.PodIsRunning).Should((Equal(true)))
				})
			})

			PIt("should find zone and region", func() {
				// Wait for new deployment before querying deployment for updates
				err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console2.Name, metav1.GetOptions{})
					if dep != nil {
						if dep.Status.UpdatedReplicas == 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
							return true, nil
						}
					}
					return false, nil
				})
				Expect(err).NotTo(HaveOccurred())
				dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console2.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("checking zone", func() {
					Expect(console2.TestAffinityZone(dep)).To((Equal(true)))
				})

				By("checking region", func() {
					Expect(console2.TestAffinityRegion(dep)).To((Equal(true)))
				})
			})

			It("should delete the second instance of IBPConsole CR", func() {
				result := ibpCRClient.Delete().Namespace(namespace).Resource("ibpconsoles").Name(console2.Name).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())
			})
		})

		Context("applying the first instance of IBPConsole CR", func() {
			var (
				err error
				dep *appsv1.Deployment
			)

			It("creates a IBPConsole custom resource", func() {
				By("setting the CR status to deploying", func() {
					Eventually(console.pollForCRStatus).Should(Equal(current.Deploying))
				})

				By("creating a service", func() {
					Eventually(console.ServiceExists).Should((Equal(true)))
				})

				By("creating a pvc", func() {
					Eventually(console.PVCExists).Should((Equal(true)))
				})

				By("creating a configmap", func() {
					Eventually(console.ConfigMapExists).Should((Equal(true)))
				})

				By("starting a ingress", func() {
					Eventually(console.IngressExists).Should((Equal(true)))
				})

				By("creating a deployment", func() {
					Eventually(console.DeploymentExists).Should((Equal(true)))
				})

				By("starting a pod", func() {
					Eventually(console.PodIsRunning).Should((Equal(true)))
				})

				By("setting the CR status to deployed when pod is running", func() {
					Eventually(console.pollForCRStatus).Should((Equal(current.Deployed)))
				})
			})

			It("should not find zone and region", func() {
				// Wait for new deployment before querying deployment for updates
				err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console.Name, metav1.GetOptions{})
					if dep != nil {
						if dep.Status.UpdatedReplicas == 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
							return true, nil
						}
					}
					return false, nil
				})
				Expect(err).NotTo(HaveOccurred())
				dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("checking zone", func() {
					Expect(console.TestAffinityZone(dep)).Should((Equal(false)))
				})

				By("checking region", func() {
					Expect(console.TestAffinityRegion(dep)).Should((Equal(false)))
				})
			})

			When("the custom resource is updated", func() {
				var (
					err                              error
					dep                              *appsv1.Deployment
					newResourceRequestsConsole       corev1.ResourceList
					newResourceLimitsConsole         corev1.ResourceList
					newResourceRequestsConfigtxlator corev1.ResourceList
					newResourceLimitsConfigtxlator   corev1.ResourceList
					newResourceRequestsCouchdb       corev1.ResourceList
					newResourceLimitsCouchdb         corev1.ResourceList
					newResourceRequestsDeployer      corev1.ResourceList
					newResourceLimitsDeployer        corev1.ResourceList
				)

				BeforeEach(func() {
					newResourceRequestsConsole = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("25m"),
						corev1.ResourceMemory:           resource.MustParse("50M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					}
					newResourceLimitsConsole = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("250m"),
						corev1.ResourceMemory:           resource.MustParse("500M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					}

					newResourceRequestsConfigtxlator = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("30m"),
						corev1.ResourceMemory:           resource.MustParse("60M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					}
					newResourceLimitsConfigtxlator = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("30m"),
						corev1.ResourceMemory:           resource.MustParse("60M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					}

					newResourceRequestsCouchdb = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("35m"),
						corev1.ResourceMemory:           resource.MustParse("70M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					}
					newResourceLimitsCouchdb = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("350m"),
						corev1.ResourceMemory:           resource.MustParse("700M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					}

					newResourceRequestsDeployer = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("9m"),
						corev1.ResourceMemory:           resource.MustParse("18M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					}
					newResourceLimitsDeployer = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("90m"),
						corev1.ResourceMemory:           resource.MustParse("180M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					}

					Eventually(console.DeploymentExists).Should((Equal(true)))
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
				})

				It("updates the instance of IBPConsole if resources are updated in CR", func() {
					consoleResources := dep.Spec.Template.Spec.Containers[0].Resources
					Expect(consoleResources.Requests).To(Equal(defaultRequestsConsole))
					Expect(consoleResources.Limits).To(Equal(defaultLimitsConsole))

					deployerResources := dep.Spec.Template.Spec.Containers[1].Resources
					Expect(deployerResources.Requests).To(Equal(defaultRequestsDeployer))
					Expect(deployerResources.Limits).To(Equal(defaultLimitsDeployer))

					configtxResources := dep.Spec.Template.Spec.Containers[2].Resources
					Expect(configtxResources.Requests).To(Equal(defaultRequestsConfigtxlator))
					Expect(configtxResources.Limits).To(Equal(defaultLimitsConfigtxlator))

					couchdbResources := dep.Spec.Template.Spec.Containers[3].Resources
					Expect(couchdbResources.Requests).To(Equal(defaultRequestsCouchdb))
					Expect(couchdbResources.Limits).To(Equal(defaultLimitsCouchdb))

					console.CR.Spec.Resources = &current.ConsoleResources{
						Console: &corev1.ResourceRequirements{
							Requests: newResourceRequestsConsole,
							Limits:   newResourceLimitsConsole,
						},
						Configtxlator: &corev1.ResourceRequirements{
							Requests: newResourceRequestsConfigtxlator,
							Limits:   newResourceLimitsConfigtxlator,
						},
						CouchDB: &corev1.ResourceRequirements{
							Requests: newResourceRequestsCouchdb,
							Limits:   newResourceLimitsCouchdb,
						},
						Deployer: &corev1.ResourceRequirements{
							Requests: newResourceRequestsDeployer,
							Limits:   newResourceLimitsDeployer,
						},
					}
					console.CR.Spec.Password = ""
					bytes, err := json.Marshal(console.CR)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibpconsoles").Name(console.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					// Wait for new deployment before querying deployment for updates
					err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console.Name, metav1.GetOptions{})
						if dep != nil {
							if dep.Status.UpdatedReplicas == 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
								if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() == newResourceRequestsConsole.Cpu().MilliValue() {
									return true, nil
								}
							}
						}
						return false, nil
					})
					Expect(err).NotTo(HaveOccurred())

					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					updatedConsoleResources := dep.Spec.Template.Spec.Containers[0].Resources
					Expect(updatedConsoleResources.Requests).To(Equal(newResourceRequestsConsole))
					Expect(updatedConsoleResources.Limits).To(Equal(newResourceLimitsConsole))

					updatedDeployerResources := dep.Spec.Template.Spec.Containers[1].Resources
					Expect(updatedDeployerResources.Requests).To(Equal(newResourceRequestsDeployer))
					Expect(updatedDeployerResources.Limits).To(Equal(newResourceLimitsDeployer))

					updatedConfigtxResources := dep.Spec.Template.Spec.Containers[2].Resources
					Expect(updatedConfigtxResources.Requests).To(Equal(newResourceRequestsConfigtxlator))
					Expect(updatedConfigtxResources.Limits).To(Equal(newResourceLimitsConfigtxlator))

					updatedCouchDBResources := dep.Spec.Template.Spec.Containers[3].Resources
					Expect(updatedCouchDBResources.Requests).To(Equal(newResourceRequestsCouchdb))
					Expect(updatedCouchDBResources.Limits).To(Equal(newResourceLimitsCouchdb))
				})
			})

			When("a deployment managed by operator is manually edited", func() {
				var (
					err error
					dep *appsv1.Deployment
				)

				BeforeEach(func() {
					Eventually(console.DeploymentExists).Should((Equal(true)))
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
				})

				It("restores states", func() {
					origRequests := dep.Spec.Template.Spec.Containers[0].Resources.Requests
					dep.Spec.Template.Spec.Containers[0].Resources.Requests = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("107m"),
						corev1.ResourceMemory: resource.MustParse("107M"),
					}

					depBytes, err := json.Marshal(dep)
					Expect(err).NotTo(HaveOccurred())

					_, err = kclient.AppsV1().Deployments(namespace).Patch(context.TODO(), console.Name, types.MergePatchType, depBytes, metav1.PatchOptions{})
					Expect(err).NotTo(HaveOccurred())

					// Wait for new deployment before querying deployment for updates
					err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console.Name, metav1.GetOptions{})
						if dep != nil {
							if dep.Status.UpdatedReplicas == 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
								if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() == origRequests.Cpu().MilliValue() {
									return true, nil
								}
							}
						}
						return false, nil
					})
					Expect(err).NotTo(HaveOccurred())
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), console.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					Expect(dep.Spec.Template.Spec.Containers[0].Resources.Requests).To(Equal(origRequests))
				})
			})

			It("should delete the first instance of IBPConsole CR", func() {
				result := ibpCRClient.Delete().Namespace(namespace).Resource("ibpconsoles").Name(console.Name).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())
			})
		})
	})
})

func shuf(min, max int) int32 {
	rand.Seed(time.Now().UnixNano())
	return int32(rand.Intn(max-min+1) + min)
}

func GetConsole() *Console {
	consolePort := shuf(30000, 32768)
	proxyPort := shuf(30000, 32768)

	name := "ibpconsole1"
	cr := &current.IBPConsole{
		Spec: current.IBPConsoleSpec{
			License: current.License{
				Accept: true,
			},
			ConnectionString:   "http://localhost:5984",
			ServiceAccountName: "ibpconsole1",
			NetworkInfo: &current.NetworkInfo{
				Domain:      integration.TestAutomation1IngressDomain,
				ConsolePort: consolePort,
				ProxyPort:   proxyPort,
			},
			Email:    "admin@ibm.com",
			Password: "cGFzc3dvcmQ=",
			Resources: &current.ConsoleResources{
				Console: &corev1.ResourceRequirements{
					Requests: defaultRequestsConsole,
					Limits:   defaultLimitsConsole,
				},
				Configtxlator: &corev1.ResourceRequirements{
					Requests: defaultRequestsConfigtxlator,
					Limits:   defaultLimitsConfigtxlator,
				},
				CouchDB: &corev1.ResourceRequirements{
					Requests: defaultRequestsCouchdb,
					Limits:   defaultLimitsCouchdb,
				},
				Deployer: &corev1.ResourceRequirements{
					Requests: defaultRequestsDeployer,
					Limits:   defaultLimitsDeployer,
				},
			},
			ImagePullSecrets: []string{"regcred"},
			Images: &current.ConsoleImages{
				ConfigtxlatorImage: integration.ConfigtxlatorImage,
				ConfigtxlatorTag:   integration.ConfigtxlatorTag,
				ConsoleImage:       integration.ConsoleImage,
				ConsoleTag:         integration.ConsoleTag,
				ConsoleInitImage:   integration.InitImage,
				ConsoleInitTag:     integration.InitTag,
				CouchDBImage:       integration.CouchdbImage,
				CouchDBTag:         integration.CouchdbTag,
				DeployerImage:      integration.DeployerImage,
				DeployerTag:        integration.DeployerTag,
			},
			Versions: &current.Versions{
				CA: map[string]current.VersionCA{
					integration.FabricCAVersion: current.VersionCA{
						Default: true,
						Version: integration.FabricCAVersion,
						Image: current.CAImages{
							CAInitImage: integration.InitImage,
							CAInitTag:   integration.InitTag,
							CAImage:     integration.CaImage,
							CATag:       integration.CaTag,
						},
					},
				},
				Peer: map[string]current.VersionPeer{
					integration.FabricVersion: current.VersionPeer{
						Default: true,
						Version: integration.FabricVersion,
						Image: current.PeerImages{
							PeerInitImage: integration.InitImage,
							PeerInitTag:   integration.InitTag,
							PeerImage:     integration.PeerImage,
							PeerTag:       integration.PeerTag,
							GRPCWebImage:  integration.GrpcwebImage,
							GRPCWebTag:    integration.GrpcwebTag,
							CouchDBImage:  integration.CouchdbImage,
							CouchDBTag:    integration.CouchdbTag,
						},
					},
				},
				Orderer: map[string]current.VersionOrderer{
					integration.FabricVersion: current.VersionOrderer{
						Default: true,
						Version: integration.FabricVersion,
						Image: current.OrdererImages{
							OrdererInitImage: integration.InitImage,
							OrdererInitTag:   integration.InitTag,
							OrdererImage:     integration.OrdererImage,
							OrdererTag:       integration.OrdererTag,
							GRPCWebImage:     integration.GrpcwebImage,
							GRPCWebTag:       integration.GrpcwebTag,
						},
					},
				},
			},
			UseTags: &useTagsFlag,
		},
	}
	cr.Name = name

	return &Console{
		Name: name,
		CR:   cr,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}

// DISABLED
func GetConsole2() *Console {
	consolePort := shuf(30000, 32768)
	proxyPort := shuf(30000, 32768)

	name := "ibpconsole2"
	cr := &current.IBPConsole{
		Spec: current.IBPConsoleSpec{
			License: current.License{
				Accept: true,
			},
			ConnectionString:   "http://localhost:5984",
			ServiceAccountName: "ibpconsole1",
			NetworkInfo: &current.NetworkInfo{
				Domain:      integration.TestAutomation1IngressDomain,
				ConsolePort: consolePort,
				ProxyPort:   proxyPort,
			},
			Email:            "admin@ibm.com",
			Password:         "cGFzc3dvcmQ=",
			Zone:             "select",
			Region:           "select",
			ImagePullSecrets: []string{"regcred"},
			Images: &current.ConsoleImages{
				ConfigtxlatorImage: integration.ConfigtxlatorImage,
				ConfigtxlatorTag:   integration.ConfigtxlatorTag,
				ConsoleImage:       integration.ConsoleImage,
				ConsoleTag:         integration.ConsoleTag,
				ConsoleInitImage:   integration.InitImage,
				ConsoleInitTag:     integration.InitTag,
				CouchDBImage:       integration.CouchdbImage,
				CouchDBTag:         integration.CouchdbTag,
				DeployerImage:      integration.DeployerImage,
				DeployerTag:        integration.DeployerTag,
			},
			Versions: &current.Versions{
				CA: map[string]current.VersionCA{
					integration.FabricCAVersion: current.VersionCA{
						Default: true,
						Version: integration.FabricCAVersion,
						Image: current.CAImages{
							CAInitImage: integration.InitImage,
							CAInitTag:   integration.InitTag,
							CAImage:     integration.CaImage,
							CATag:       integration.CaTag,
						},
					},
				},
				Peer: map[string]current.VersionPeer{
					integration.FabricVersion: current.VersionPeer{
						Default: true,
						Version: integration.FabricVersion,
						Image: current.PeerImages{
							PeerInitImage: integration.InitImage,
							PeerInitTag:   integration.InitTag,
							PeerImage:     integration.PeerImage,
							PeerTag:       integration.PeerTag,
							GRPCWebImage:  integration.GrpcwebImage,
							GRPCWebTag:    integration.GrpcwebTag,
							CouchDBImage:  integration.CouchdbImage,
							CouchDBTag:    integration.CouchdbTag,
						},
					},
				},
				Orderer: map[string]current.VersionOrderer{
					integration.FabricVersion: current.VersionOrderer{
						Default: true,
						Version: integration.FabricVersion,
						Image: current.OrdererImages{
							OrdererInitImage: integration.InitImage,
							OrdererInitTag:   integration.InitTag,
							OrdererImage:     integration.OrdererImage,
							OrdererTag:       integration.OrdererTag,
							GRPCWebImage:     integration.GrpcwebImage,
							GRPCWebTag:       integration.GrpcwebTag,
						},
					},
				},
			},
			UseTags: &useTagsFlag,
		},
	}
	cr.Name = name

	return &Console{
		Name: name,
		CR:   cr,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}

func GetConsole3() *Console {
	consolePort := shuf(30000, 32768)
	proxyPort := shuf(30000, 32768)

	name := "ibpconsole3"
	cr := &current.IBPConsole{
		Spec: current.IBPConsoleSpec{
			License: current.License{
				Accept: true,
			},
			ServiceAccountName: "", // Will cause error
			NetworkInfo: &current.NetworkInfo{
				Domain:      integration.TestAutomation1IngressDomain,
				ConsolePort: consolePort,
				ProxyPort:   proxyPort,
			},
			Images: &current.ConsoleImages{
				CouchDBImage: integration.CouchdbImage,
				CouchDBTag:   integration.CouchdbTag,
			},
			UseTags: &useTagsFlag,
		},
	}
	cr.Name = name

	return &Console{
		Name: name,
		CR:   cr,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}

type Console struct {
	Name string
	CR   *current.IBPConsole
	integration.NativeResourcePoller
}

func (console *Console) pollForCRStatus() current.IBPCRStatusType {
	crStatus := &current.IBPConsole{}

	result := ibpCRClient.Get().Namespace(namespace).Resource("ibpconsoles").Name(console.Name).Do(context.TODO())
	result.Into(crStatus)

	return crStatus.Status.Type
}
