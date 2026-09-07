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

package orderer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v1"
	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v2"
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v2"
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

type OrdererConfig interface {
	ToBytes() ([]byte, error)
}

var (
	orderer       *Orderer
	orderer2      *Orderer
	orderer3      *Orderer
	orderer4      *Orderer
	orderer5      *Orderer
	orderer1nodes []Orderer
	orderer2nodes []Orderer
	orderer3nodes []Orderer
	orderer4nodes []Orderer
	orderer5nodes []Orderer
)

var (
	defaultRequestsOrderer = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("20m"),
		corev1.ResourceMemory:           resource.MustParse("40M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
	}

	defaultLimitsOrderer = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("200m"),
		corev1.ResourceMemory:           resource.MustParse("400M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
	}

	defaultRequestsProxy = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("10m"),
		corev1.ResourceMemory:           resource.MustParse("20M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
	}

	defaultLimitsProxy = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("100m"),
		corev1.ResourceMemory:           resource.MustParse("200M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
	}

	testMSPSpec = &current.MSPSpec{
		Component: &current.MSP{
			KeyStore:   "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ1pwdXhrM3FCSzBpTzM1THIKQ3RXMWhBaHlETmM4Z1JCNkhWU1FkelB1S0d1aFJBTkNBQVFaZzhlUEk3QTIyVW51NGtEemNsR1BSTWFlcmM1Qgo5RGUrTjBxUzd1R05BVTh3cnloTTFzcERPSWhLQXZnOUw2Ny9ybGJSTUZHSUpvdzlaZVZVQzc1WgotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==",
			SignCerts:  "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNOVENDQWR1Z0F3SUJBZ0lVYTNXekt6dldtOXRYWFFhbnRHOUNiSjdHNWVNd0NnWUlLb1pJemowRUF3SXcKYVRFTE1Ba0dBMVVFQmhNQ1ZWTXhFekFSQmdOVkJBZ01Da05oYkdsbWIzSnVhV0V4RmpBVUJnTlZCQWNNRFZOaApiaUJHY21GdVkybHpZMjh4RkRBU0JnTlZCQW9NQzJWNFlXMXdiR1V1WTI5dE1SY3dGUVlEVlFRRERBNWpZUzVsCmVHRnRjR3hsTG1OdmJUQWVGdzB5TmpBNU1EY3hOREEyTWpOYUZ3MHpOakE1TURReE5EQTJNak5hTUdveEN6QUoKQmdOVkJBWVRBbFZUTVJNd0VRWURWUVFJREFwRFlXeHBabTl5Ym1saE1SWXdGQVlEVlFRSERBMVRZVzRnUm5KaApibU5wYzJOdk1SQXdEZ1lEVlFRTERBZHZjbVJsY21WeU1Sd3dHZ1lEVlFRRERCTnZjbVJsY21WeUxtVjRZVzF3CmJHVXVZMjl0TUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFR1lQSGp5T3dOdGxKN3VKQTgzSlIKajBUR25xM09RZlEzdmpkS2t1N2hqUUZQTUs4b1ROYktRemlJU2dMNFBTK3UvNjVXMFRCUmlDYU1QV1hsVkF1KwpXYU5nTUY0d0RnWURWUjBQQVFIL0JBUURBZ2VBTUF3R0ExVWRFd0VCL3dRQ01BQXdId1lEVlIwakJCZ3dGb0FVCno3eVhJZi9nSkVqbzhwS0duSmd5WXVUMWVuQXdIUVlEVlIwT0JCWUVGUFB4OGVqOERYWStFZFJUN1Ftdm5Mb0QKb1ZjSk1Bb0dDQ3FHU000OUJBTUNBMGdBTUVVQ0lHbWZ2VEt6Mi9aL2t1ZGFGT3pDWnJEd0VicjlNTkJkRmdDKwpiejRvcW5VTEFpRUFyT1Y5NmplSWtKb3hpY0pDWHFHNDAxbkh4bmg3VVVQZnB6TmVub3F3bFkwPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
			CACerts:    []string{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNOVENDQWR1Z0F3SUJBZ0lVT0lLaGEzSjRtZSt1TFVFOGFDY3V0TzA5cjJnd0NnWUlLb1pJemowRUF3SXcKYVRFTE1Ba0dBMVVFQmhNQ1ZWTXhFekFSQmdOVkJBZ01Da05oYkdsbWIzSnVhV0V4RmpBVUJnTlZCQWNNRFZOaApiaUJHY21GdVkybHpZMjh4RkRBU0JnTlZCQW9NQzJWNFlXMXdiR1V1WTI5dE1SY3dGUVlEVlFRRERBNWpZUzVsCmVHRnRjR3hsTG1OdmJUQWVGdzB5TmpBNU1EY3hOREEyTWpOYUZ3MHpOakE1TURReE5EQTJNak5hTUdreEN6QUoKQmdOVkJBWVRBbFZUTVJNd0VRWURWUVFJREFwRFlXeHBabTl5Ym1saE1SWXdGQVlEVlFRSERBMVRZVzRnUm5KaApibU5wYzJOdk1SUXdFZ1lEVlFRS0RBdGxlR0Z0Y0d4bExtTnZiVEVYTUJVR0ExVUVBd3dPWTJFdVpYaGhiWEJzClpTNWpiMjB3V1RBVEJnY3Foa2pPUFFJQkJnZ3Foa2pPUFFNQkJ3TkNBQVRJNnFvWWNUYnlIbkJFNGV4ZTZIcEgKdS9qVytGb3oxR3Y2cGhoYlVFVEE5eE81cWYwcEJPR055OXVrNmE0R3RqY1hnVlRxZkxSck1sSE5CL1FvWXhzcgpvMkV3WHpBT0JnTlZIUThCQWY4RUJBTUNBYVl3SFFZRFZSMGxCQll3RkFZSUt3WUJCUVVIQXdJR0NDc0dBUVVGCkJ3TUJNQThHQTFVZEV3RUIvd1FGTUFNQkFmOHdIUVlEVlIwT0JCWUVGTSs4bHlILzRDUkk2UEtTaHB5WU1tTGsKOVhwd01Bb0dDQ3FHU000OUJBTUNBMGdBTUVVQ0lIdi9zSE9FdXNvbXNKeWVDRG9GT01QSlh3TzVZd2N6Zm9ibwo2NjNwWmxZRkFpRUFzRjRCc1NXUWxISmtReTRXNnRpcmhlM0lXeWNuOS94ZmxzbWxoL3RGRkRJPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="},
			AdminCerts: []string{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNNVENDQWRlZ0F3SUJBZ0lVYTNXekt6dldtOXRYWFFhbnRHOUNiSjdHNWVRd0NnWUlLb1pJemowRUF3SXcKYVRFTE1Ba0dBMVVFQmhNQ1ZWTXhFekFSQmdOVkJBZ01Da05oYkdsbWIzSnVhV0V4RmpBVUJnTlZCQWNNRFZOaApiaUJHY21GdVkybHpZMjh4RkRBU0JnTlZCQW9NQzJWNFlXMXdiR1V1WTI5dE1SY3dGUVlEVlFRRERBNWpZUzVsCmVHRnRjR3hsTG1OdmJUQWVGdzB5TmpBNU1EY3hOREEyTWpOYUZ3MHpOakE1TURReE5EQTJNak5hTUdZeEN6QUoKQmdOVkJBWVRBbFZUTVJNd0VRWURWUVFJREFwRFlXeHBabTl5Ym1saE1SWXdGQVlEVlFRSERBMVRZVzRnUm5KaApibU5wYzJOdk1RNHdEQVlEVlFRTERBVmhaRzFwYmpFYU1CZ0dBMVVFQXd3UlFXUnRhVzVBWlhoaGJYQnNaUzVqCmIyMHdXVEFUQmdjcWhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUXN1bURIdE9RVUlXdU5NcnR2OXEzRi9ITFAKVmx6aytkTkVvZUhlczlQcVJQNTNwQTJ1UjgwUUpuNXBrRFlhRHh3VU5LMjN6RG10Y1JRWmNnc29abFBobzJBdwpYakFPQmdOVkhROEJBZjhFQkFNQ0I0QXdEQVlEVlIwVEFRSC9CQUl3QURBZkJnTlZIU01FR0RBV2dCVFB2SmNoCi8rQWtTT2p5a29hY21ESmk1UFY2Y0RBZEJnTlZIUTRFRmdRVVBjb05WOTM4QWlCRlA1dE5VVTljenJqUXV3b3cKQ2dZSUtvWkl6ajBFQXdJRFNBQXdSUUloQU5wSGE1bFJYaVJiRmM3YXBGUW5HZy9yanB1Snp0Qy9veEFjWElndApvWlhxQWlCOFd5VUVzTnBWVDBVTVc4amJjaUUzWXowMHl5TVdwYlFzMmtpcVBQZ0RsUT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"},
		},
		TLS: &current.MSP{
			KeyStore:  "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ1pwdXhrM3FCSzBpTzM1THIKQ3RXMWhBaHlETmM4Z1JCNkhWU1FkelB1S0d1aFJBTkNBQVFaZzhlUEk3QTIyVW51NGtEemNsR1BSTWFlcmM1Qgo5RGUrTjBxUzd1R05BVTh3cnloTTFzcERPSWhLQXZnOUw2Ny9ybGJSTUZHSUpvdzlaZVZVQzc1WgotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==",
			SignCerts: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNOVENDQWR1Z0F3SUJBZ0lVYTNXekt6dldtOXRYWFFhbnRHOUNiSjdHNWVNd0NnWUlLb1pJemowRUF3SXcKYVRFTE1Ba0dBMVVFQmhNQ1ZWTXhFekFSQmdOVkJBZ01Da05oYkdsbWIzSnVhV0V4RmpBVUJnTlZCQWNNRFZOaApiaUJHY21GdVkybHpZMjh4RkRBU0JnTlZCQW9NQzJWNFlXMXdiR1V1WTI5dE1SY3dGUVlEVlFRRERBNWpZUzVsCmVHRnRjR3hsTG1OdmJUQWVGdzB5TmpBNU1EY3hOREEyTWpOYUZ3MHpOakE1TURReE5EQTJNak5hTUdveEN6QUoKQmdOVkJBWVRBbFZUTVJNd0VRWURWUVFJREFwRFlXeHBabTl5Ym1saE1SWXdGQVlEVlFRSERBMVRZVzRnUm5KaApibU5wYzJOdk1SQXdEZ1lEVlFRTERBZHZjbVJsY21WeU1Sd3dHZ1lEVlFRRERCTnZjbVJsY21WeUxtVjRZVzF3CmJHVXVZMjl0TUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFR1lQSGp5T3dOdGxKN3VKQTgzSlIKajBUR25xM09RZlEzdmpkS2t1N2hqUUZQTUs4b1ROYktRemlJU2dMNFBTK3UvNjVXMFRCUmlDYU1QV1hsVkF1KwpXYU5nTUY0d0RnWURWUjBQQVFIL0JBUURBZ2VBTUF3R0ExVWRFd0VCL3dRQ01BQXdId1lEVlIwakJCZ3dGb0FVCno3eVhJZi9nSkVqbzhwS0duSmd5WXVUMWVuQXdIUVlEVlIwT0JCWUVGUFB4OGVqOERYWStFZFJUN1Ftdm5Mb0QKb1ZjSk1Bb0dDQ3FHU000OUJBTUNBMGdBTUVVQ0lHbWZ2VEt6Mi9aL2t1ZGFGT3pDWnJEd0VicjlNTkJkRmdDKwpiejRvcW5VTEFpRUFyT1Y5NmplSWtKb3hpY0pDWHFHNDAxbkh4bmg3VVVQZnB6TmVub3F3bFkwPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
			CACerts:   []string{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNOVENDQWR1Z0F3SUJBZ0lVT0lLaGEzSjRtZSt1TFVFOGFDY3V0TzA5cjJnd0NnWUlLb1pJemowRUF3SXcKYVRFTE1Ba0dBMVVFQmhNQ1ZWTXhFekFSQmdOVkJBZ01Da05oYkdsbWIzSnVhV0V4RmpBVUJnTlZCQWNNRFZOaApiaUJHY21GdVkybHpZMjh4RkRBU0JnTlZCQW9NQzJWNFlXMXdiR1V1WTI5dE1SY3dGUVlEVlFRRERBNWpZUzVsCmVHRnRjR3hsTG1OdmJUQWVGdzB5TmpBNU1EY3hOREEyTWpOYUZ3MHpOakE1TURReE5EQTJNak5hTUdreEN6QUoKQmdOVkJBWVRBbFZUTVJNd0VRWURWUVFJREFwRFlXeHBabTl5Ym1saE1SWXdGQVlEVlFRSERBMVRZVzRnUm5KaApibU5wYzJOdk1SUXdFZ1lEVlFRS0RBdGxlR0Z0Y0d4bExtTnZiVEVYTUJVR0ExVUVBd3dPWTJFdVpYaGhiWEJzClpTNWpiMjB3V1RBVEJnY3Foa2pPUFFJQkJnZ3Foa2pPUFFNQkJ3TkNBQVRJNnFvWWNUYnlIbkJFNGV4ZTZIcEgKdS9qVytGb3oxR3Y2cGhoYlVFVEE5eE81cWYwcEJPR055OXVrNmE0R3RqY1hnVlRxZkxSck1sSE5CL1FvWXhzcgpvMkV3WHpBT0JnTlZIUThCQWY4RUJBTUNBYVl3SFFZRFZSMGxCQll3RkFZSUt3WUJCUVVIQXdJR0NDc0dBUVVGCkJ3TUJNQThHQTFVZEV3RUIvd1FGTUFNQkFmOHdIUVlEVlIwT0JCWUVGTSs4bHlILzRDUkk2UEtTaHB5WU1tTGsKOVhwd01Bb0dDQ3FHU000OUJBTUNBMGdBTUVVQ0lIdi9zSE9FdXNvbXNKeWVDRG9GT01QSlh3TzVZd2N6Zm9ibwo2NjNwWmxZRkFpRUFzRjRCc1NXUWxISmtReTRXNnRpcmhlM0lXeWNuOS94ZmxzbWxoL3RGRkRJPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="},
		},
	}
)

var _ = Describe("Interaction between IBP-Operator and Kubernetes cluster", func() {
	SetDefaultEventuallyTimeout(420 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	BeforeEach(func() {
		orderer, orderer1nodes = GetOrderer()
		err := helper.CreateOrderer(ibpCRClient, orderer.CR)
		Expect(err).NotTo(HaveOccurred())

		integration.ClearOperatorConfig(kclient, namespace)
	})

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentSpecReport().Failed() {
			testFailed = true
		}
	})

	Context("IBPOrderer controller", func() {

		Context("applying first instance of IBPOrderer CR", func() {
			var (
				err error
				dep *appsv1.Deployment
			)

			It("creates a IBPOrderer custom resource", func() {
				By("setting the CR status to precreate", func() {
					// Precreated is a transient state on the way to Deployed, and
					// the operator can pass through it faster than the polling
					// interval, so a poll is not guaranteed to observe it.  Accept
					// either rather than requiring the intermediate state to be
					// sampled; the assertions that follow cover the deployment.
					for _, node := range orderer1nodes {
						Eventually(node.pollForCRStatus).Should(BeElementOf(current.Precreated, current.Deployed))
					}
				})

				By("creating a pvc", func() {
					for _, node := range orderer1nodes {
						Eventually(node.PVCExists).Should((Equal(true)))
					}
				})

				By("creating a service", func() {
					for _, node := range orderer1nodes {
						Eventually(node.ServiceExists).Should((Equal(true)))
					}
				})

				By("creating a configmap", func() {
					for _, node := range orderer1nodes {
						Eventually(node.ConfigMapExists).Should((Equal(true)))
					}
				})

				By("starting a ingress", func() {
					for _, node := range orderer1nodes {
						Eventually(node.IngressExists).Should((Equal(true)))
					}
				})

				By("creating a deployment", func() {
					for _, node := range orderer1nodes {
						Eventually(node.DeploymentExists).Should((Equal(true)))
					}
				})

				By("creating init secrets", func() {
					for _, node := range orderer1nodes {
						Eventually(node.allInitSecretsExist).Should((Equal(true)))
					}
				})

				By("starting a pod", func() {
					for _, node := range orderer1nodes {
						Eventually(node.PodIsRunning).Should((Equal(true)))
					}
				})

				By("creating config map that contains spec", func() {
					for _, node := range orderer1nodes {
						Eventually(func() bool {
							_, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), node.Name+"-spec", metav1.GetOptions{})
							if err != nil {
								return false
							}
							return true
						}).Should(Equal(true))
					}
				})

				By("setting the CR status to deployed when pod is running", func() {
					for _, node := range orderer1nodes {
						Eventually(node.pollForCRStatus).Should((Equal(current.Deployed)))
					}
					Eventually(orderer.pollForCRStatus).Should((Equal(current.Deployed)))
				})

				By("overriding general section in orderer.yaml", func() {
					cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), orderer.Name+"node1-config", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					ordererBytes := cm.BinaryData["orderer.yaml"]
					ordererConfig, err := config.ReadOrdererFromBytes(ordererBytes)
					Expect(err).NotTo(HaveOccurred())
					configOverride, err := orderer.CR.GetConfigOverride()
					Expect(err).NotTo(HaveOccurred())
					bytes, err := configOverride.(OrdererConfig).ToBytes()
					Expect(err).NotTo(HaveOccurred())
					oConfig := &config.Orderer{}
					err = yaml.Unmarshal(bytes, oConfig)
					Expect(err).NotTo(HaveOccurred())
					Expect(ordererConfig.General.ListenPort).To(Equal(oConfig.General.ListenPort))
				})
			})

			It("should not find zone and region", func() {
				// Wait for new deployment before querying deployment for updates
				err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
					ready := true
					for _, node := range orderer1nodes {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
						if dep != nil {
							if dep.Status.UpdatedReplicas != 1 || dep.Status.Conditions[0].Type != appsv1.DeploymentAvailable {
								ready = false
							}
						}
					}

					return ready, nil
				})
				Expect(err).NotTo(HaveOccurred())
				for _, node := range orderer1nodes {
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("checking zone", func() {
						Expect(node.TestAffinityZone(dep)).To((Equal(false)))
					})

					By("checking region", func() {
						Expect(node.TestAffinityRegion(dep)).To((Equal(false)))
					})
				}
			})

			When("the custom resource is updated", func() {
				var (
					dep                        *appsv1.Deployment
					newResourceRequestsOrderer corev1.ResourceList
					newResourceLimitsOrderer   corev1.ResourceList
					newResourceRequestsProxy   corev1.ResourceList
					newResourceLimitsProxy     corev1.ResourceList
				)

				BeforeEach(func() {
					newResourceRequestsOrderer = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("240m"),
						corev1.ResourceMemory:           resource.MustParse("480M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					}
					newResourceLimitsOrderer = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("240m"),
						corev1.ResourceMemory:           resource.MustParse("480M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					}

					newResourceRequestsProxy = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("90m"),
						corev1.ResourceMemory:           resource.MustParse("180M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					}
					newResourceLimitsProxy = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("90m"),
						corev1.ResourceMemory:           resource.MustParse("180M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					}

					for _, node := range orderer1nodes {
						Eventually(node.DeploymentExists).Should((Equal(true)))
					}
				})

				It("updates the instance of IBPOrderer if resources are updated in CR", func() {
					for _, node := range orderer1nodes {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})

						ordererResources := dep.Spec.Template.Spec.Containers[0].Resources
						Expect(ordererResources.Requests).To(Equal(defaultRequestsOrderer))
						Expect(ordererResources.Limits).To(Equal(defaultLimitsOrderer))

						proxyResources := dep.Spec.Template.Spec.Containers[1].Resources
						Expect(proxyResources.Requests).To(Equal(defaultRequestsProxy))
						Expect(proxyResources.Limits).To(Equal(defaultLimitsProxy))

						updatenode := &current.IBPOrderer{}
						result := ibpCRClient.Get().Namespace(namespace).Resource("ibporderers").Name(node.Name).Do(context.TODO())
						result.Into(updatenode)

						updatenode.Spec.Resources = &current.OrdererResources{
							Orderer: &corev1.ResourceRequirements{
								Requests: newResourceRequestsOrderer,
								Limits:   newResourceLimitsOrderer,
							},
							GRPCProxy: &corev1.ResourceRequirements{
								Requests: newResourceRequestsProxy,
								Limits:   newResourceLimitsProxy,
							},
						}
						configOverride := &config.Orderer{
							Orderer: v2.Orderer{
								FileLedger: v1.FileLedger{
									Location: "/temp",
								},
							},
						}
						configBytes, err := json.Marshal(configOverride)
						Expect(err).NotTo(HaveOccurred())
						updatenode.Spec.ConfigOverride = &runtime.RawExtension{Raw: configBytes}

						bytes, err := json.Marshal(updatenode)
						Expect(err).NotTo(HaveOccurred())

						result = ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibporderers").Name(node.Name).Body(bytes).Do(context.TODO())
						Expect(result.Error()).NotTo(HaveOccurred())

						// Wait for new deployment before querying deployment for updates
						Eventually(func() bool {
							dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
							if dep != nil {
								if dep.Status.UpdatedReplicas == 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
									if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() == newResourceRequestsOrderer.Cpu().MilliValue() {
										return true
									}
								}
							}
							return false
						}).Should(Equal(true))

						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						updatedOrdererResources := dep.Spec.Template.Spec.Containers[0].Resources
						Expect(updatedOrdererResources.Requests).To(Equal(newResourceRequestsOrderer))
						Expect(updatedOrdererResources.Limits).To(Equal(newResourceLimitsOrderer))

						updatedProxyResources := dep.Spec.Template.Spec.Containers[1].Resources
						Expect(updatedProxyResources.Requests).To(Equal(newResourceRequestsProxy))
						Expect(updatedProxyResources.Limits).To(Equal(newResourceLimitsProxy))

						By("updating the config map with new values from override", func() {
							Eventually(func() bool {
								cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), orderer.Name+"node1-config", metav1.GetOptions{})
								Expect(err).NotTo(HaveOccurred())

								configBytes := cm.BinaryData["orderer.yaml"]
								ordererConfig, err := config.ReadOrdererFromBytes(configBytes)
								Expect(err).NotTo(HaveOccurred())

								if ordererConfig.FileLedger.Location == "/temp" {
									return true
								}

								return false
							}).Should(Equal(true))
						})
					}
				})
			})

			When("a deployment managed by operator is manually edited", func() {
				var (
					err error
					dep *appsv1.Deployment
				)

				BeforeEach(func() {
					for _, node := range orderer1nodes {
						Eventually(node.DeploymentExists).Should((Equal(true)))
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
					}
				})

				It("restores states", func() {
					for _, node := range orderer1nodes {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						origRequests := dep.Spec.Template.Spec.Containers[0].Resources.Requests
						dep.Spec.Template.Spec.Containers[0].Resources.Requests = map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("200M"),
						}

						depBytes, err := json.Marshal(dep)
						Expect(err).NotTo(HaveOccurred())

						_, err = kclient.AppsV1().Deployments(namespace).Patch(context.TODO(), node.NodeName, types.MergePatchType, depBytes, metav1.PatchOptions{})
						Expect(util.IgnoreOutdatedResourceVersion(err)).NotTo(HaveOccurred())

						// Wait for new deployment before querying deployment for updates
						wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
							dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
							if dep != nil {
								if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() == origRequests.Cpu().MilliValue() {
									return true, nil
								}
							}
							return false, nil
						})

						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						Expect(dep.Spec.Template.Spec.Containers[0].Resources.Requests).To(Equal(origRequests))
					}
				})
			})
		})

		Context("applying last instance of IBPOrderer CR, with channel-less config", func() {

			// NOTE: THIS COUNTER MUST BE EQUAL TO THE NUMBER OF It() ROUTINES IN THIS CONTEXT
			checks_remaining := 2

			// Set up the orderer before the FIRST It() of this context
			BeforeEach(func() {
				if orderer5 == nil {
					orderer5, orderer5nodes = GetOrderer5()
					err := helper.CreateOrderer(ibpCRClient, orderer5.CR)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			// Tear down the orderer after the LAST It() in this context
			AfterEach(func() {
				checks_remaining--
				if checks_remaining == 0 {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibporderers").Name(orderer5.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					orderer5 = nil
					orderer5nodes = nil
				}
			})

			It("creates a IBPOrderer custom resource", func() {
				By("creating a pvc", func() {
					for _, node := range orderer5nodes {
						Eventually(node.PVCExists).Should((Equal(true)))
					}
				})

				By("creating a service", func() {
					for _, node := range orderer5nodes {
						Eventually(node.ServiceExists).Should((Equal(true)))
					}
				})

				By("creating a configmap", func() {
					for _, node := range orderer5nodes {
						Eventually(node.ConfigMapExists).Should((Equal(true)))
					}
				})

				By("starting a ingress", func() {
					for _, node := range orderer5nodes {
						Eventually(node.IngressExists).Should((Equal(true)))
					}
				})

				By("creating a deployment", func() {
					for _, node := range orderer5nodes {
						Eventually(node.DeploymentExists).Should((Equal(true)))
					}
				})

				By("creating init secrets", func() {
					for _, node := range orderer5nodes {
						Eventually(node.allInitSecretsExist).Should((Equal(true)))
					}
				})

				By("starting a pod", func() {
					for _, node := range orderer5nodes {
						Eventually(node.PodIsRunning).Should((Equal(true)))
					}
				})

				By("creating config map that contains spec", func() {
					for _, node := range orderer5nodes {
						Eventually(func() bool {
							_, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), node.Name+"-spec", metav1.GetOptions{})
							if err != nil {
								return false
							}
							return true
						}).Should(Equal(true))
					}
				})

				By("setting the CR status to deployed when pod is running", func() {
					for _, node := range orderer5nodes {
						Eventually(node.pollForCRStatus).Should((Equal(current.Deployed)))
					}
					Eventually(orderer5.pollForCRStatus).Should((Equal(current.Deployed)))
				})
			})

			When("a deployment managed by operator is manually edited", func() {
				var (
					err error
					dep *appsv1.Deployment
				)

				BeforeEach(func() {
					for _, node := range orderer5nodes {
						Eventually(node.DeploymentExists).Should((Equal(true)))
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
					}
				})

				It("restores states", func() {
					for _, node := range orderer5nodes {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						origRequests := dep.Spec.Template.Spec.Containers[0].Resources.Requests
						dep.Spec.Template.Spec.Containers[0].Resources.Requests = map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("200M"),
						}

						depBytes, err := json.Marshal(dep)
						Expect(err).NotTo(HaveOccurred())

						_, err = kclient.AppsV1().Deployments(namespace).Patch(context.TODO(), node.NodeName, types.MergePatchType, depBytes, metav1.PatchOptions{})
						Expect(util.IgnoreOutdatedResourceVersion(err)).NotTo(HaveOccurred())

						// Wait for new deployment before querying deployment for updates
						wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
							dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
							if dep != nil {
								if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() == origRequests.Cpu().MilliValue() {
									return true, nil
								}
							}
							return false, nil
						})

						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						Expect(dep.Spec.Template.Spec.Containers[0].Resources.Requests).To(Equal(origRequests))
					}
				})
			})
		})

		Context("applying the second instance of IBPOrderer CR", func() {
			var (
				err error
				dep *appsv1.Deployment
			)

			// NOTE: THIS COUNTER MUST BE EQUAL TO THE NUMBER OF It() ROUTINES IN THIS CONTEXT
			checks_remaining := 2

			// Set up the orderer before the FIRST It() of this context
			BeforeEach(func() {
				if orderer2 == nil {
					orderer2, orderer2nodes = GetOrderer2()
					err := helper.CreateOrderer(ibpCRClient, orderer2.CR)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			// Tear down the orderer after the LAST It() in this context
			AfterEach(func() {
				checks_remaining--
				if checks_remaining == 0 {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibporderers").Name(orderer2.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					orderer2 = nil
					orderer2nodes = nil
				}
			})

			It("creates a second IBPOrderer custom resource", func() {
				By("starting a pod", func() {
					for _, node := range orderer2nodes {
						Eventually(node.PodIsRunning).Should((Equal(true)))
					}
				})
			})

			PIt("should find zone and region", func() {
				for _, node := range orderer2nodes {
					// Wait for new deployment before querying deployment for updates
					wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
						if dep != nil {
							if dep.Status.UpdatedReplicas >= 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
								return true, nil
							}
						}
						return false, nil
					})

					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), node.NodeName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					By("checking zone", func() {
						Expect(orderer2.TestAffinityZone(dep)).To((Equal(true)))
					})

					By("checking region", func() {
						Expect(orderer2.TestAffinityRegion(dep)).To((Equal(true)))
					})
				}
			})

			It("adjust cluster size should not change number of orderers", func() {
				By("increase number of nodes", func() {
					orderer2.CR.Spec.ClusterSize = 5
					bytes, err := json.Marshal(orderer2.CR)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibporderers").Name(orderer2.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					Eventually(orderer2.NumberOfOrdererNodeDeployments).Should((Equal(3)))
				})

				By("reducing cluster size should not change the number of nodes", func() {
					orderer2.CR.Spec.ClusterSize = 1
					bytes, err := json.Marshal(orderer2.CR)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibporderers").Name(orderer2.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					Eventually(orderer2.NumberOfOrdererNodeDeployments).Should((Equal(3)))

					secretResult := ibpCRClient.Get().Namespace(namespace).Resource("secrets").Name(fmt.Sprintf("ecert-%s%s%d-signcert", orderer2.Name, baseorderer.NODE, 3)).Do(context.TODO())
					Expect(secretResult.Error()).To(HaveOccurred())

					serviceResult := ibpCRClient.Get().Namespace(namespace).Resource("services").Name(fmt.Sprintf("%s%s%dservice", orderer2.Name, baseorderer.NODE, 3)).Do(context.TODO())
					Expect(serviceResult.Error()).To(HaveOccurred())

					cm := ibpCRClient.Get().Namespace(namespace).Resource("configmaps").Name(fmt.Sprintf("%s-%s%d-cm", orderer2.Name, baseorderer.NODE, 3)).Do(context.TODO())
					Expect(cm.Error()).To(HaveOccurred())

					pvc := ibpCRClient.Get().Namespace(namespace).Resource("persistentvolumeclaims").Name(fmt.Sprintf("%s-%s%d-pvc", orderer2.Name, baseorderer.NODE, 3)).Do(context.TODO())
					Expect(pvc.Error()).To(HaveOccurred())
				})
			})
		})

		Context("applying incorrectly configured third instance of IBPOrderer CR", func() {

			// NOTE: THIS COUNTER MUST BE EQUAL TO THE NUMBER OF It() ROUTINES IN THIS CONTEXT
			checks_remaining := 1

			// Set up the orderer before the FIRST It() of this context
			BeforeEach(func() {
				if orderer3 == nil {
					orderer3, orderer3nodes = GetOrderer3()
					err := helper.CreateOrderer(ibpCRClient, orderer3.CR)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			// Tear down the orderer after the LAST It() in this context
			AfterEach(func() {
				checks_remaining--
				if checks_remaining == 0 {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibporderers").Name(orderer3.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					orderer3 = nil
					orderer3nodes = nil
				}
			})

			It("should set the CR status to error", func() {
				Eventually(orderer3.pollForCRStatus).Should((Equal(current.Error)))

				crStatus := &current.IBPOrderer{}
				result := ibpCRClient.Get().Namespace(namespace).Resource("ibporderers").Name(orderer3.Name).Do(context.TODO())
				result.Into(crStatus)

				Expect(crStatus.Status.Message).To(ContainSubstring("Number of Cluster Node Locations does not match cluster size"))
			})
		})

		Context("deleting all child nodes should delete parent of fourth instance of IBPOrderer CR", func() {

			// NOTE: THIS COUNTER MUST BE EQUAL TO THE NUMBER OF It() ROUTINES IN THIS CONTEXT
			checks_remaining := 3

			// Set up the orderer before the FIRST It() of this context
			BeforeEach(func() {
				if orderer4 == nil {
					orderer4, orderer4nodes = GetOrderer4()
					err := helper.CreateOrderer(ibpCRClient, orderer4.CR)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			// Tear down the orderer after the LAST It() in this context
			AfterEach(func() {
				checks_remaining--
				if checks_remaining == 0 {
					// Orderer4 will have been deleted during the test context - expect an error on get()
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibporderers").Name(orderer4.Name).Do(context.TODO())
					Expect(result.Error()).To(HaveOccurred())

					orderer4 = nil
					orderer4nodes = nil
				}
			})

			It("creates a fourth IBPOrderer custom resource", func() {
				By("starting a pod", func() {
					for _, node := range orderer4nodes {
						Eventually(node.PodIsRunning).Should((Equal(true)))
					}
				})
			})

			It("does not delete the parent if few child nodes are deleted", func() {
				node := orderer4nodes[0]
				result := ibpCRClient.Delete().Namespace(namespace).Resource("ibporderers").Name(node.Name).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				node = orderer4nodes[1]
				result = ibpCRClient.Delete().Namespace(namespace).Resource("ibporderers").Name(node.Name).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				// Wait for second node to be deleted
				err := wait.Poll(500*time.Millisecond, 30*time.Second, func() (bool, error) {
					result := ibpCRClient.Get().Namespace(namespace).Resource("ibporderers").Name(node.Name).Do(context.TODO())

					if result.Error() == nil {
						return false, nil
					}
					return true, nil
				})
				Expect(err).NotTo(HaveOccurred())

				parent := &current.IBPOrderer{}
				result = ibpCRClient.Get().Namespace(namespace).Resource("ibporderers").Name(orderer4.CR.GetName()).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())
				err = result.Into(parent)
				Expect(err).NotTo(HaveOccurred())
			})

			It("deletes the parent if all child nodes are deleted", func() {
				node := orderer4nodes[2]
				result := ibpCRClient.Delete().Namespace(namespace).Resource("ibporderers").Name(node.Name).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				err := wait.Poll(500*time.Millisecond, 30*time.Second, func() (bool, error) {
					parent := &current.IBPOrderer{}
					result := ibpCRClient.Get().Namespace(namespace).Resource("ibporderers").Name(orderer4.CR.Name).Do(context.TODO())
					if result.Error() == nil {
						err := result.Into(parent)
						Expect(err).NotTo(HaveOccurred())
						return false, nil
					}
					return true, nil
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("pod restart", func() {
			var (
				orderernode *Orderer
			)

			BeforeEach(func() {
				_, nodes := GetOrderer()
				orderernode = &nodes[0]
			})

			Context("should not trigger deployment restart if config overrides not updated", func() {
				var (
					oldPodName string
				)

				BeforeEach(func() {
					Eventually(orderernode.PodIsRunning).Should((Equal(true)))

					pods := orderernode.GetPods()
					if len(pods) > 0 {
						oldPodName = pods[0].Name
					}
				})

				It("does not restart the orderer node pod", func() {
					Eventually(orderernode.PodIsRunning).Should((Equal(true)))

					Eventually(func() bool {
						pods := orderernode.GetPods()
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

			Context("should trigger deployment restart if config overrides is updated", func() {
				var (
					oldPodName string
				)

				BeforeEach(func() {
					Eventually(orderernode.PodIsRunning).Should((Equal(true)))
					pods := orderernode.GetPods()
					Expect(len(pods)).To(Equal(1))
					oldPodName = pods[0].Name

					configOverride := &config.Orderer{
						Orderer: v2.Orderer{
							FileLedger: v1.FileLedger{
								Location: "/temp1",
							},
						},
					}
					configBytes, err := json.Marshal(configOverride)
					Expect(err).NotTo(HaveOccurred())
					orderernode.CR.Spec.ConfigOverride = &runtime.RawExtension{Raw: configBytes}

					bytes, err := json.Marshal(orderernode.CR)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibporderers").Name(orderernode.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})

				It("restarts the pod", func() {
					Eventually(orderernode.PodIsRunning).Should((Equal(false)))
					Eventually(orderernode.PodIsRunning).Should((Equal(true)))

					Eventually(func() bool {
						pods := orderernode.GetPods()
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

		Context("delete crs", func() {
			It("should delete IBPOrderer CR", func() {
				By("deleting the first instance of IBPOrderer CR", func() {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibporderers").Name(orderer.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})
			})
		})
	})
})

func GetOrderer() (*Orderer, []Orderer) {
	name := "ibporderer"
	configOverride := &config.Orderer{
		Orderer: v2.Orderer{
			General: v2.General{
				ListenPort: uint16(7052),
			},
		},
	}
	configBytes, err := json.Marshal(configOverride)
	Expect(err).NotTo(HaveOccurred())
	cr := &current.IBPOrderer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPOrdererSpec{
			License: current.License{
				Accept: true,
			},
			OrdererType:       "etcdraft",
			SystemChannelName: "testchainid",
			OrgName:           "orderermsp",
			MSPID:             "orderermsp",
			ImagePullSecrets:  []string{"regcred"},
			GenesisProfile:    "Initial",
			Domain:            integration.TestAutomation1IngressDomain,
			Images: &current.OrdererImages{
				GRPCWebImage:     integration.GrpcwebImage,
				GRPCWebTag:       integration.GrpcwebTag,
				OrdererImage:     integration.OrdererImage,
				OrdererTag:       integration.OrdererTag,
				OrdererInitImage: integration.InitImage,
				OrdererInitTag:   integration.InitTag,
			},
			ClusterSecret: []*current.SecretSpec{
				&current.SecretSpec{
					MSP: testMSPSpec,
				},
			},
			Resources: &current.OrdererResources{
				Orderer: &corev1.ResourceRequirements{
					Requests: defaultRequestsOrderer,
					Limits:   defaultLimitsOrderer,
				},
				GRPCProxy: &corev1.ResourceRequirements{
					Requests: defaultRequestsProxy,
					Limits:   defaultLimitsProxy,
				},
			},
			ConfigOverride: &runtime.RawExtension{Raw: configBytes},
			DisableNodeOU:  pointer.Bool(true),
			FabricVersion:  integration.FabricVersion24 + "-1",
		},
	}
	cr.Name = name

	nodes := []Orderer{
		Orderer{
			Name:     name + "node1",
			CR:       cr.DeepCopy(),
			NodeName: fmt.Sprintf("%s%s%d", name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name + "node1",
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}

	nodes[0].CR.ObjectMeta.Name = name + "node1"

	return &Orderer{
		Name:     name,
		CR:       cr,
		NodeName: fmt.Sprintf("%s-%s%d", name, baseorderer.NODE, 1),
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}, nodes
}

func GetOrderer2() (*Orderer, []Orderer) {
	name := "ibporderer2"
	cr := &current.IBPOrderer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPOrdererSpec{
			License: current.License{
				Accept: true,
			},
			OrdererType:       "etcdraft",
			ClusterSize:       3,
			SystemChannelName: "channel1",
			OrgName:           "orderermsp",
			MSPID:             "orderermsp",
			ImagePullSecrets:  []string{"regcred"},
			Domain:            integration.TestAutomation1IngressDomain,
			GenesisProfile:    "Initial",
			Images: &current.OrdererImages{
				GRPCWebImage:     integration.GrpcwebImage,
				GRPCWebTag:       integration.GrpcwebTag,
				OrdererImage:     integration.OrdererImage,
				OrdererTag:       integration.OrdererTag,
				OrdererInitImage: integration.InitImage,
				OrdererInitTag:   integration.InitTag,
			},
			ClusterSecret: []*current.SecretSpec{
				&current.SecretSpec{
					MSP: testMSPSpec,
				},
				&current.SecretSpec{
					MSP: testMSPSpec,
				},
				&current.SecretSpec{
					MSP: testMSPSpec,
				},
			},
			Zone:   "select",
			Region: "select",
			Resources: &current.OrdererResources{
				Orderer: &corev1.ResourceRequirements{
					Requests: defaultRequestsOrderer,
					Limits:   defaultLimitsOrderer,
				},
				GRPCProxy: &corev1.ResourceRequirements{
					Requests: defaultRequestsProxy,
					Limits:   defaultLimitsProxy,
				},
			},
			DisableNodeOU: pointer.Bool(true),
			FabricVersion: integration.FabricVersion + "-1",
		},
	}
	cr.Name = name

	nodes := []Orderer{
		Orderer{
			Name:     name + "node1",
			CR:       cr.DeepCopy(),
			NodeName: fmt.Sprintf("%s%s%d", name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name + "node1",
				Namespace: namespace,
				Client:    kclient,
			},
		},
		Orderer{
			Name:     name + "node2",
			CR:       cr.DeepCopy(),
			NodeName: fmt.Sprintf("%s%s%d", name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name + "node2",
				Namespace: namespace,
				Client:    kclient,
			},
		},
		Orderer{
			Name:     name + "node3",
			CR:       cr.DeepCopy(),
			NodeName: fmt.Sprintf("%s%s%d", name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name + "node3",
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}

	nodes[0].CR.ObjectMeta.Name = name + "node1"
	nodes[1].CR.ObjectMeta.Name = name + "node2"
	nodes[2].CR.ObjectMeta.Name = name + "node3"

	return &Orderer{
		Name:     name,
		CR:       cr,
		NodeName: fmt.Sprintf("%s-%s%d", name, baseorderer.NODE, 1),
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}, nodes
}

func GetOrderer3() (*Orderer, []Orderer) {
	name := "ibporderer3"
	cr := &current.IBPOrderer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPOrdererSpec{
			License: current.License{
				Accept: true,
			},
			OrdererType:       "etcdraft",
			ClusterSize:       1,
			SystemChannelName: "channel1",
			OrgName:           "ordererorg",
			MSPID:             "orderermsp",
			ImagePullSecrets:  []string{"regcred"},
			Domain:            integration.TestAutomation1IngressDomain,
			GenesisProfile:    "Initial",
			Images: &current.OrdererImages{
				GRPCWebImage:     integration.GrpcwebImage,
				GRPCWebTag:       integration.GrpcwebTag,
				OrdererImage:     integration.OrdererImage,
				OrdererTag:       integration.OrdererTag,
				OrdererInitImage: integration.InitImage,
				OrdererInitTag:   integration.InitTag,
			},
			Secret: &current.SecretSpec{
				MSP: testMSPSpec,
			},
			ClusterLocation: []current.IBPOrdererClusterLocation{
				current.IBPOrdererClusterLocation{
					Zone:   "dal1",
					Region: "us-south1",
				},
				current.IBPOrdererClusterLocation{
					Zone:   "dal2",
					Region: "us-south2",
				},
			},
			DisableNodeOU: pointer.Bool(true),
			FabricVersion: integration.FabricVersion + "-1",
		},
	}
	cr.Name = name

	nodes := []Orderer{
		Orderer{
			Name:     name + "node1",
			CR:       cr.DeepCopy(),
			NodeName: fmt.Sprintf("%s%s%d", name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name + "node1",
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}

	nodes[0].CR.ObjectMeta.Name = name + "node1"

	return &Orderer{
		Name:     name,
		CR:       cr,
		NodeName: fmt.Sprintf("%s-%s%d", name, baseorderer.NODE, 1),
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}, nodes
}

func GetOrderer4() (*Orderer, []Orderer) {
	name := "ibporderer4"
	cr := &current.IBPOrderer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPOrdererSpec{
			License: current.License{
				Accept: true,
			},
			OrdererType:       "etcdraft",
			ClusterSize:       3,
			SystemChannelName: "channel1",
			OrgName:           "orderermsp",
			MSPID:             "orderermsp",
			ImagePullSecrets:  []string{"regcred"},
			Domain:            integration.TestAutomation1IngressDomain,
			GenesisProfile:    "Initial",
			Images: &current.OrdererImages{
				GRPCWebImage:     integration.GrpcwebImage,
				GRPCWebTag:       integration.GrpcwebTag,
				OrdererImage:     integration.OrdererImage,
				OrdererTag:       integration.OrdererTag,
				OrdererInitImage: integration.InitImage,
				OrdererInitTag:   integration.InitTag,
			},
			ClusterSecret: []*current.SecretSpec{
				&current.SecretSpec{
					MSP: testMSPSpec,
				},
				&current.SecretSpec{
					MSP: testMSPSpec,
				},
				&current.SecretSpec{
					MSP: testMSPSpec,
				},
			},
			Zone:   "select",
			Region: "select",
			Resources: &current.OrdererResources{
				Orderer: &corev1.ResourceRequirements{
					Requests: defaultRequestsOrderer,
					Limits:   defaultLimitsOrderer,
				},
				GRPCProxy: &corev1.ResourceRequirements{
					Requests: defaultRequestsProxy,
					Limits:   defaultLimitsProxy,
				},
			},
			DisableNodeOU: pointer.Bool(true),
			FabricVersion: integration.FabricVersion + "-1",
		},
	}
	cr.Name = name

	nodes := []Orderer{
		Orderer{
			Name:     name + "node1",
			CR:       cr.DeepCopy(),
			NodeName: fmt.Sprintf("%s%s%d", name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name + "node1",
				Namespace: namespace,
				Client:    kclient,
			},
		},
		Orderer{
			Name:     name + "node2",
			CR:       cr.DeepCopy(),
			NodeName: fmt.Sprintf("%s%s%d", name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name + "node2",
				Namespace: namespace,
				Client:    kclient,
			},
		},
		Orderer{
			Name:     name + "node3",
			CR:       cr.DeepCopy(),
			NodeName: fmt.Sprintf("%s%s%d", name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name + "node3",
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}

	nodes[0].CR.ObjectMeta.Name = name + "node1"
	nodes[1].CR.ObjectMeta.Name = name + "node2"
	nodes[2].CR.ObjectMeta.Name = name + "node3"

	return &Orderer{
		Name:     name,
		CR:       cr,
		NodeName: fmt.Sprintf("%s-%s%d", name, baseorderer.NODE, 1),
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}, nodes
}

func GetOrderer5() (*Orderer, []Orderer) {
	name := "ibporderer5"
	cr := &current.IBPOrderer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPOrdererSpec{
			License: current.License{
				Accept: true,
			},
			OrdererType:       "etcdraft",
			SystemChannelName: "testchainid",
			UseChannelLess:    pointer.Bool(true),
			OrgName:           "orderermsp",
			MSPID:             "orderermsp",
			ImagePullSecrets:  []string{"regcred"},
			GenesisProfile:    "Initial",
			Domain:            integration.TestAutomation1IngressDomain,
			Images: &current.OrdererImages{
				GRPCWebImage:     integration.GrpcwebImage,
				GRPCWebTag:       integration.GrpcwebTag,
				OrdererImage:     integration.OrdererImage,
				OrdererTag:       integration.Orderer24Tag,
				OrdererInitImage: integration.InitImage,
				OrdererInitTag:   integration.InitTag,
			},
			ClusterSecret: []*current.SecretSpec{
				&current.SecretSpec{
					MSP: testMSPSpec,
				},
			},
			Resources: &current.OrdererResources{
				Orderer: &corev1.ResourceRequirements{
					Requests: defaultRequestsOrderer,
					Limits:   defaultLimitsOrderer,
				},
				GRPCProxy: &corev1.ResourceRequirements{
					Requests: defaultRequestsProxy,
					Limits:   defaultLimitsProxy,
				},
			},
			DisableNodeOU: pointer.Bool(true),
			FabricVersion: integration.FabricVersion24 + "-1",
		},
	}
	cr.Name = name

	nodes := []Orderer{
		Orderer{
			Name:     name + "node1",
			CR:       cr.DeepCopy(),
			NodeName: fmt.Sprintf("%s%s%d", name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      name + "node1",
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}

	nodes[0].CR.ObjectMeta.Name = name + "node1"

	return &Orderer{
		Name:     name,
		CR:       cr,
		NodeName: fmt.Sprintf("%s-%s%d", name, baseorderer.NODE, 1),
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}, nodes
}

type Orderer struct {
	Name     string
	CR       *current.IBPOrderer
	NodeName string
	integration.NativeResourcePoller
}

func (orderer *Orderer) pollForCRStatus() current.IBPCRStatusType {
	crStatus := &current.IBPOrderer{}

	result := ibpCRClient.Get().Namespace(namespace).Resource("ibporderers").Name(orderer.Name).Do(context.TODO())
	result.Into(crStatus)

	return crStatus.Status.Type
}

func (orderer *Orderer) allInitSecretsExist() bool {
	prefix := "ecert-" + orderer.NodeName
	name := prefix + "-admincerts"
	_, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false
	}

	name = prefix + "-cacerts"
	_, err = kclient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false
	}

	name = prefix + "-signcert"
	_, err = kclient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false
	}

	name = prefix + "-keystore"
	_, err = kclient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false
	}

	prefix = "tls-" + orderer.NodeName
	name = prefix + "-cacerts"
	_, err = kclient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false
	}

	name = prefix + "-signcert"
	_, err = kclient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false
	}

	name = prefix + "-keystore"
	_, err = kclient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false
	}

	return true
}

func (o *Orderer) DeploymentExists() bool {
	dep, err := kclient.AppsV1().Deployments(namespace).Get(context.TODO(), o.NodeName, metav1.GetOptions{})
	if err == nil && dep != nil {
		return true
	}

	return false
}
