//go:build !pkcs11
// +build !pkcs11

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

package ca_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Interaction between IBP-Operator and Kubernetes cluster", func() {
	var (
		err error
		ca  *CA
		ca2 *CA
		ca3 *CA
	)

	BeforeEach(func() {
		ca = GetCA1()
		err = helper.CreateCA(ibpCRClient, ca.CR)
		Expect(err).NotTo(HaveOccurred())

		ca2 = GetCA2()
		err = helper.CreateCA(ibpCRClient, ca2.CR)
		Expect(err).NotTo(HaveOccurred())

		ca3 = GetCA3()
		err = helper.CreateCA(ibpCRClient, ca3.CR)
		Expect(err).NotTo(HaveOccurred())

		integration.ClearOperatorConfig(kclient, namespace)
	})

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("IBPCA controller", func() {
		Context("applying the first instance of IBPCA CR", func() {
			var (
				err error
				dep *appsv1.Deployment
			)

			It("creates a IBPCA custom resource", func() {
				By("setting the CR status to deploying", func() {
					Eventually(ca.PollForCRStatus).Should((Equal(current.Deploying)))
				})

				By("creating a service", func() {
					Eventually(ca.ServiceExists).Should((Equal(true)))
				})

				By("creating a configmap", func() {
					Eventually(ca.ConfigMapExists).Should((Equal(true)))
				})

				By("starting a ingress", func() {
					Eventually(ca.IngressExists).Should((Equal(true)))
				})

				By("creating a deployment", func() {
					Eventually(ca.DeploymentExists).Should((Equal(true)))
				})

				By("starting a pod", func() {
					Eventually(ca.PodIsRunning).Should((Equal(true)))
				})

				By("creating config map that contains spec", func() {
					Eventually(func() bool {
						_, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), ca.Name+"-spec", metav1.GetOptions{})
						if err != nil {
							return false
						}
						return true
					}).Should(Equal(true))
				})

				By("creating secret with crypto for CA", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), ca.Name+"-ca-crypto", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(secret).NotTo(BeNil())
					Expect(len(secret.Data)).To(Equal(6))
				})

				By("creating secret with crypto for TLS CA", func() {
					secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), ca.Name+"-tlsca-crypto", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(secret).NotTo(BeNil())
					Expect(len(secret.Data)).To(Equal(2))
				})

				By("setting the CR status to deployed when pod is running", func() {
					Eventually(ca.PollForCRStatus).Should((Equal(current.Deployed)))
				})
			})

			It("should not find zone and region", func() {
				// Wait for new deployment before querying deployment for updates
				err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
					if dep != nil {
						if dep.Status.UpdatedReplicas == 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
							return true, nil
						}
					}
					return false, nil
				})
				Expect(err).NotTo(HaveOccurred())
				dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				// TODO :: only run these when using MZ clusters
				// By("checking zone", func() {
				// 	Expect(ca.TestAffinityZone(dep)).To((Equal(false)))
				// })

				// By("checking region", func() {
				// 	Expect(ca.TestAffinityRegion(dep)).To((Equal(false)))
				// })
			})

			When("the custom resource is updated", func() {
				var (
					err                 error
					dep                 *appsv1.Deployment
					newResourceRequests corev1.ResourceList
					newResourceLimits   corev1.ResourceList
				)

				BeforeEach(func() {
					newResourceRequests = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("55m"),
						corev1.ResourceMemory: resource.MustParse("110M"),
					}
					newResourceLimits = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:    resource.MustParse("55m"),
						corev1.ResourceMemory: resource.MustParse("110M"),
					}
					ca.expectedRequests = newResourceRequests
					ca.expectedLimits = newResourceLimits

					Eventually(ca.DeploymentExists).Should((Equal(true)))
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

				})

				It("updates the instance of IBPCA if resources are updated in CR", func() {
					currentResources := dep.Spec.Template.Spec.Containers[0].Resources
					Expect(currentResources.Requests).To(Equal(defaultRequests))
					Expect(currentResources.Limits).To(Equal(defaultLimits))

					ca.CR.Spec.Resources = &current.CAResources{
						CA: &corev1.ResourceRequirements{
							Requests: newResourceRequests,
							Limits:   newResourceLimits,
						},
					}

					caOverrides := &v1.ServerConfig{}
					err := json.Unmarshal(ca.CR.Spec.ConfigOverride.CA.Raw, caOverrides)
					Expect(err).NotTo(HaveOccurred())
					caOverrides.CAConfig.CA = v1.CAInfo{
						Name: "new-ca",
					}

					caJson, err := util.ConvertToJsonMessage(caOverrides)
					Expect(err).NotTo(HaveOccurred())
					ca.CR.Spec.ConfigOverride.CA = &runtime.RawExtension{Raw: *caJson}

					tlscaOverrides := &v1.ServerConfig{}
					err = json.Unmarshal(ca.CR.Spec.ConfigOverride.TLSCA.Raw, tlscaOverrides)
					Expect(err).NotTo(HaveOccurred())
					tlscaOverrides.CAConfig.CA = v1.CAInfo{
						Name: "new-tlsca",
					}

					tlscaJson, err := util.ConvertToJsonMessage(tlscaOverrides)
					Expect(err).NotTo(HaveOccurred())
					ca.CR.Spec.ConfigOverride.TLSCA = &runtime.RawExtension{Raw: *tlscaJson}

					bytes, err := json.Marshal(ca.CR)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibpcas").Name(ca.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					// Wait for new deployment before querying deployment for updates
					err = wait.Poll(500*time.Millisecond, 120*time.Second, func() (bool, error) {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
						if dep != nil {
							if dep.Status.UpdatedReplicas == 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
								if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() == newResourceRequests.Cpu().MilliValue() {
									return true, nil
								}
							}
						}
						return false, nil
					})
					Expect(err).NotTo(HaveOccurred())

					Eventually(ca.resourcesRequestsUpdated).Should(Equal(true))
					Eventually(ca.resourcesLimitsUpdated).Should(Equal(true))

					By("updating the config map with new values from override for ecert and tls ca", func() {
						cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), fmt.Sprintf("%s-ca-config", ca.Name), metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						serverconfig := &v1.ServerConfig{}
						err = yaml.Unmarshal(cm.BinaryData["fabric-ca-server-config.yaml"], serverconfig)
						Expect(err).NotTo(HaveOccurred())

						Expect(serverconfig.CAConfig.CA.Name).To(Equal("new-ca"))

						cm, err = kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), fmt.Sprintf("%s-tlsca-config", ca.Name), metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						serverconfig = &v1.ServerConfig{}
						err = yaml.Unmarshal(cm.BinaryData["fabric-ca-server-config.yaml"], serverconfig)
						Expect(err).NotTo(HaveOccurred())

						Expect(serverconfig.CAConfig.CA.Name).To(Equal("new-tlsca"))

						By("restarting deployment for ecert ca", func() {
							// Pod should first go away, and deployment is restarted
							// Eventually(ca.PodIsRunning).Should((Equal(false))) // FLAKY TEST
							// Pod should eventually then go into running state
							Eventually(ca.PodIsRunning).Should((Equal(true)))
						})

					})

				})
			})

			When("a deployment managed by operator is manually edited", func() {
				var (
					err error
					dep *appsv1.Deployment
				)

				BeforeEach(func() {
					Eventually(ca.DeploymentExists).Should((Equal(true)))
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
				})

				It("restores states", func() {

					// Reduce the deployment resource requests
					origRequests := dep.Spec.Template.Spec.Containers[0].Resources.Requests
					newResourceRequests := corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("20m"),
						corev1.ResourceMemory: resource.MustParse("50M"),
					}
					Expect(newResourceRequests).ToNot(Equal(origRequests))

					dep.Spec.Template.Spec.Containers[0].Resources.Requests = newResourceRequests
					depBytes, err := json.Marshal(dep)
					Expect(err).NotTo(HaveOccurred())

					// After patching, the resource limits should have been reduced to the lower values
					dep, err = kclient.AppsV1().Deployments(namespace).Patch(context.TODO(), ca.Name, types.MergePatchType, depBytes, metav1.PatchOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(dep.Spec.Template.Spec.Containers[0].Resources.Requests).To(Equal(newResourceRequests))

					// And with get resource, not just the deployment returned by patch
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(dep.Spec.Template.Spec.Containers[0].Resources.Requests).To(Equal(newResourceRequests))

					// But the operator prevails:  resource limits will be reset to the original amount specified in the CRD
					err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
						if dep != nil {
							if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() == origRequests.Cpu().MilliValue() {
								return true, nil
							}
						}
						return false, nil
					})
					Expect(err).NotTo(HaveOccurred())

					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(dep.Spec.Template.Spec.Containers[0].Resources.Requests).To(Equal(origRequests))
				})
			})
		})

		Context("applying the second instance of IBPCA CR", func() {
			var (
				err error
				dep *appsv1.Deployment
			)

			BeforeEach(func() {
				Eventually(ca2.PodIsRunning).Should((Equal(true)))
			})

			It("should find zone and region", func() {
				// Wait for new deployment before querying deployment for updates
				err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca2.Name, metav1.GetOptions{})
					if dep != nil {
						if dep.Status.UpdatedReplicas == 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
							return true, nil
						}
					}
					return false, nil
				})
				Expect(err).NotTo(HaveOccurred())
				dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca2.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				// TODO :: only run these when using MZ clusters
				// By("checking zone", func() {
				// 	Expect(ca2.TestAffinityZone(dep)).To((Equal(true)))
				// })

				// By("checking region", func() {
				// 	Expect(ca2.TestAffinityRegion(dep)).To((Equal(true)))
				// })
			})

			When("fabric version is updated", func() {
				BeforeEach(func() {
					ibpca := &current.IBPCA{}
					result := ibpCRClient.Get().Namespace(namespace).Resource("ibpcas").Name(ca2.Name).Do(context.TODO())
					result.Into(ibpca)

					ibpca.Spec.FabricVersion = integration.FabricCAVersion + "-1"
					bytes, err := json.Marshal(ibpca)
					Expect(err).NotTo(HaveOccurred())

					result = ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibpcas").Name(ca2.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})

				It("sets images mapped to version", func() {
					Eventually(func() current.CAImages {
						ibpca := &current.IBPCA{}
						result := ibpCRClient.Get().Namespace(namespace).Resource("ibpcas").Name(ca2.Name).Do(context.TODO())
						result.Into(ibpca)
						fmt.Println("ca images ")
						fmt.Printf("%+v", *ibpca.Spec.Images)
						return *ibpca.Spec.Images
					}).Should(Equal(current.CAImages{
						CAInitImage: integration.InitImage,
						CAInitTag:   integration.InitTag,
						CAImage:     integration.CaImage,
						CATag:       integration.CaTag,
					}))
				})
			})
		})

		Context("applying incorrectly configured third instance of IBPCA CR", func() {
			It("should set the CR status to error", func() {
				Eventually(ca3.PollForCRStatus).Should((Equal(current.Error)))

				crStatus := &current.IBPCA{}
				result := ibpCRClient.Get().Namespace(namespace).Resource("ibpcas").Name(ca3.Name).Do(context.TODO())
				result.Into(crStatus)

				Expect(crStatus.Status.Message).To(ContainSubstring("Failed to provide database configuration for TLSCA to support greater than 1 replicas"))
			})
		})

		Context("pod restart", func() {
			var (
				oldPodName string
			)
			Context("should not trigger deployment restart if config overrides not updated", func() {
				BeforeEach(func() {
					Eventually(ca.PodIsRunning).Should((Equal(true)))

					Eventually(func() int {
						return len(ca.GetPods())
					}).Should(Equal(1))

					pods := ca.GetPods()
					oldPodName = pods[0].Name
				})

				It("does not restart the ca pod", func() {
					Eventually(ca.PodIsRunning).Should((Equal(true)))

					Eventually(func() bool {
						pods := ca.GetPods()
						if len(pods) != 1 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName == oldPodName {
							return true
						}

						return false
					}).Should(Equal(true))
				})
			})

			Context("should trigger deployment restart if config overrides updated", func() {
				BeforeEach(func() {
					Eventually(ca.PodIsRunning).Should((Equal(true)))

					Eventually(func() int {
						return len(ca.GetPods())
					}).Should(Equal(1))

					pods := ca.GetPods()
					oldPodName = pods[0].Name

					caOverrides := &v1.ServerConfig{}
					err = json.Unmarshal(ca.CR.Spec.ConfigOverride.CA.Raw, caOverrides)
					Expect(err).NotTo(HaveOccurred())
					caOverrides.CAConfig.CA = v1.CAInfo{
						Name: "new-ca",
					}

					caJson, err := util.ConvertToJsonMessage(caOverrides)
					Expect(err).NotTo(HaveOccurred())
					ca.CR.Spec.ConfigOverride.CA = &runtime.RawExtension{Raw: *caJson}

					bytes, err := json.Marshal(ca.CR)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibpcas").Name(ca.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})

				It("restarts the ca pod", func() {

					// FLAKY TEST: Checking for pod not running before pod is running causes test to be flaky
					// due to the rolling restart nature of our component restarts. Sometimes, a new pod
					// comes up quicker than this test can check for a non-running pod, so it will never
					// detect that the pod was being terminated before a new one come up.
					// Eventually(ca.PodIsRunning, 240*time.Second, 500*time.Millisecond).Should((Equal(false)))
					Eventually(ca.PodIsRunning).Should((Equal(true)))

					Eventually(func() bool {
						pods := ca.GetPods()
						if len(pods) != 1 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName == oldPodName {
							return false
						}

						return true
					}).Should(Equal(true))
				})
			})
		})

		//TODO: Disabling the test untill DNS host issues are sorted out with the nginx ingress
		PContext("enroll intermediate ca", func() {
			BeforeEach(func() {
				Eventually(ca.PodIsRunning).Should((Equal(true)))
			})

			It("enrolls with root ca", func() {
				ica := GetIntermediateCA()
				helper.CreateCA(ibpCRClient, ica.CR)

				Eventually(ica.PodIsRunning).Should((Equal(true)))
			})
		})

		Context("delete crs", func() {
			It("should delete IBPCA CR", func() {
				By("deleting the first instance of IBPCA CR", func() {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibpcas").Name(ca.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})

				By("deleting the second instance of IBPCA CR", func() {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibpcas").Name(ca2.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})

				By("deleting the third instance of IBPCA CR", func() {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibpcas").Name(ca3.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})
			})
		})
	})
})

func GetCA1() *CA {
	caOverrides := &v1.ServerConfig{
		Debug: pointer.True(),
		TLS: v1.ServerTLSConfig{
			CertFile: tlsCert,
			KeyFile:  tlsKey,
		},
		CAConfig: v1.CAConfig{
			CA: v1.CAInfo{
				Name:     "ca",
				Certfile: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNVakNDQWZpZ0F3SUJBZ0lSQUtSTFhRQm02WUo5ODlhRGQxVmRxM2d3Q2dZSUtvWkl6ajBFQXdJd2N6RUwKTUFrR0ExVUVCaE1DVlZNeEV6QVJCZ05WQkFnVENrTmhiR2xtYjNKdWFXRXhGakFVQmdOVkJBY1REVk5oYmlCRwpjbUZ1WTJselkyOHhHVEFYQmdOVkJBb1RFRzl5WnpFdVpYaGhiWEJzWlM1amIyMHhIREFhQmdOVkJBTVRFMk5oCkxtOXlaekV1WlhoaGJYQnNaUzVqYjIwd0hoY05NakF3TkRBNU1EQTBOekF3V2hjTk16QXdOREEzTURBME56QXcKV2pCek1Rc3dDUVlEVlFRR0V3SlZVekVUTUJFR0ExVUVDQk1LUTJGc2FXWnZjbTVwWVRFV01CUUdBMVVFQnhNTgpVMkZ1SUVaeVlXNWphWE5qYnpFWk1CY0dBMVVFQ2hNUWIzSm5NUzVsZUdGdGNHeGxMbU52YlRFY01Cb0dBMVVFCkF4TVRZMkV1YjNKbk1TNWxlR0Z0Y0d4bExtTnZiVEJaTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEEwSUEKQkxJMENnNlFMTDZqOWdZQkZsQ3k1RTVWSC8vUHJoSUhwZ0ZNQ3VRUXJ4WUM2Y3dBbGdhS1g3Tmd4QzQrenE2dApUaU54OGtSd3h3NTRrQ2N0ZnZQdU1DMmpiVEJyTUE0R0ExVWREd0VCL3dRRUF3SUJwakFkQmdOVkhTVUVGakFVCkJnZ3JCZ0VGQlFjREFnWUlLd1lCQlFVSEF3RXdEd1lEVlIwVEFRSC9CQVV3QXdFQi96QXBCZ05WSFE0RUlnUWcKRlhXeWVGYlpMaFRHTko5MzVKQm85bFMyM284cm13SjJSQnZXaDlDMldJa3dDZ1lJS29aSXpqMEVBd0lEU0FBdwpSUUloQUxVcUU5a2F2U0NmbEV6U25ERUhIdVh1ZjR4MEhUbnU3eGtNOXArNW5PcnBBaUF1aE5NWXhxbjU5MUpLCjdWRGFPK0k0eVVWZEViNGxiRlFBZUJiR1FTdkxDdz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0==",
				Keyfile:  "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ1FNWnEwdFY4Mjl0UUZQcS8KcSswZnNES0p6MDdnd0dpS0FUNEMwTG9qSnpDaFJBTkNBQVN5TkFvT2tDeStvL1lHQVJaUXN1Uk9WUi8vejY0UwpCNllCVEFya0VLOFdBdW5NQUpZR2lsK3pZTVF1UHM2dXJVNGpjZkpFY01jT2VKQW5MWDd6N2pBdAotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0t",
			},
		},
	}
	caJson, err := util.ConvertToJsonMessage(caOverrides)
	Expect(err).NotTo(HaveOccurred())

	tlscaOverrides := v1.ServerConfig{
		CAConfig: v1.CAConfig{
			CA: v1.CAInfo{
				Name: "tlsca-ca1",
			},
		},
	}
	tlscaJson, err := util.ConvertToJsonMessage(tlscaOverrides)
	Expect(err).NotTo(HaveOccurred())

	name := "ibpca1"
	cr := &current.IBPCA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPCASpec{
			License: current.License{
				Accept: true,
			},
			ImagePullSecrets: []string{"regcred"},
			// TODO:OSS
			Domain: domain,
			Images: &current.CAImages{
				CAImage:     integration.CaImage,
				CATag:       integration.CaTag,
				CAInitImage: integration.InitImage,
				CAInitTag:   integration.InitTag,
			},
			RegistryURL: "no-registry-url",
			Resources: &current.CAResources{
				CA: &corev1.ResourceRequirements{
					Requests: defaultRequests,
					Limits:   defaultLimits,
				},
			},
			ConfigOverride: &current.ConfigOverride{
				CA:    &runtime.RawExtension{Raw: *caJson},
				TLSCA: &runtime.RawExtension{Raw: *tlscaJson},
			},
			FabricVersion: integration.FabricCAVersion,
		},
	}

	return &CA{
		CA: helper.CA{
			Name:      name,
			Namespace: namespace,
			CR:        cr,
			CRClient:  ibpCRClient,
			KClient:   kclient,
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name,
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}
}

func GetCA2() *CA {
	caOverrides := &v1.ServerConfig{
		Debug: pointer.True(),
		TLS: v1.ServerTLSConfig{
			CertFile: tlsCert,
			KeyFile:  tlsKey,
		},
	}
	caJson, err := util.ConvertToJsonMessage(caOverrides)
	Expect(err).NotTo(HaveOccurred())

	name := "ibpca2"
	cr := &current.IBPCA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPCASpec{
			License: current.License{
				Accept: true,
			},
			ImagePullSecrets: []string{"regcred"},
			Images: &current.CAImages{
				CAImage:     integration.CaImage,
				CATag:       integration.CaTag,
				CAInitImage: integration.InitImage,
				CAInitTag:   integration.InitTag,
			},
			RegistryURL:   "no-registry-url",
			FabricVersion: integration.FabricCAVersion,
			Resources: &current.CAResources{
				CA: &corev1.ResourceRequirements{
					Requests: defaultRequests,
					Limits:   defaultLimits,
				},
			},
			ConfigOverride: &current.ConfigOverride{
				CA: &runtime.RawExtension{Raw: *caJson},
			},
			Zone:   "select",
			Region: "select",
			Domain: domain,
			CustomNames: current.CACustomNames{
				Sqlite: "/data/fabric-ca-server.db",
			},
		},
	}
	cr.Name = name

	return &CA{
		CA: helper.CA{
			Name:      name,
			Namespace: namespace,
			CR:        cr,
			CRClient:  ibpCRClient,
			KClient:   kclient,
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name,
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}
}

func GetCA3() *CA {
	caOverrides := &v1.ServerConfig{
		Debug: pointer.True(),
		TLS: v1.ServerTLSConfig{
			CertFile: tlsCert,
			KeyFile:  tlsKey,
		},
	}

	caJson, err := util.ConvertToJsonMessage(caOverrides)
	Expect(err).NotTo(HaveOccurred())
	var replicas int32
	replicas = 3
	name := "ibpca3"
	cr := &current.IBPCA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPCASpec{
			Domain: domain,
			ConfigOverride: &current.ConfigOverride{
				CA: &runtime.RawExtension{Raw: *caJson},
			},
			FabricVersion: integration.FabricCAVersion,
			License: current.License{
				Accept: true,
			},
			Replicas: &replicas,
		},
	}

	return &CA{
		CA: helper.CA{
			Name:      name,
			Namespace: namespace,
			CR:        cr,
			CRClient:  ibpCRClient,
			KClient:   kclient,
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name,
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}
}

func GetIntermediateCA() *CA {
	caOverrides := &v1.ServerConfig{
		Debug: pointer.True(),
		TLS: v1.ServerTLSConfig{
			CertFile: tlsCert,
			KeyFile:  tlsKey,
		},
		CAConfig: v1.CAConfig{
			Intermediate: v1.IntermediateCA{
				ParentServer: v1.ParentServer{
					URL: fmt.Sprintf("https://admin:adminpw@%s-ibpca1-ca.%s", namespace, domain),
				},
				TLS: v1.ClientTLSConfig{
					Enabled:   pointer.True(),
					CertFiles: []string{trustedRootTLSCert},
				},
			},
		},
	}

	caJson, err := util.ConvertToJsonMessage(caOverrides)
	Expect(err).NotTo(HaveOccurred())

	name := "interca"
	cr := &current.IBPCA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPCASpec{
			License: current.License{
				Accept: true,
			},
			ImagePullSecrets: []string{"regcred"},
			Domain:           domain,
			Images: &current.CAImages{
				CAImage:     integration.CaImage,
				CATag:       integration.CaTag,
				CAInitImage: integration.InitImage,
				CAInitTag:   integration.InitTag,
			},
			ConfigOverride: &current.ConfigOverride{
				CA: &runtime.RawExtension{Raw: *caJson},
			},
			FabricVersion: integration.FabricCAVersion,
		},
	}

	return &CA{
		CA: helper.CA{
			Name:      name,
			Namespace: namespace,
			CR:        cr,
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name,
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}
}
