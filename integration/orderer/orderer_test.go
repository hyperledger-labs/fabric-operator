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
			KeyStore:   "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ2hkZDZuMTBkcGl2VlFRWXAKMzdJTFdUb056aEFHUWtQZnhEaTFHc2FESHBlaFJBTkNBQVNpMHRkZW9xa2lDODRJME8yWE1haXVXZHlqTWZwMQplQ1JPUnQrSFpHUWVleHhScWs1QlhzcEl1dEFnMVQxNGxSOEJWemR6Qm13NXVMQmNKY0RtbHdVVwotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==",
			SignCerts:  "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNERENDQWJPZ0F3SUJBZ0lSQUxWR3BXbTlTYTRGVzVaajF2aUZqU0V3Q2dZSUtvWkl6ajBFQXdJd2FURUwKTUFrR0ExVUVCaE1DVlZNeEV6QVJCZ05WQkFnVENrTmhiR2xtYjNKdWFXRXhGakFVQmdOVkJBY1REVk5oYmlCRwpjbUZ1WTJselkyOHhGREFTQmdOVkJBb1RDMlY0WVcxd2JHVXVZMjl0TVJjd0ZRWURWUVFERXc1allTNWxlR0Z0CmNHeGxMbU52YlRBZUZ3MHlOVEV5TVRjd05UUTBNREJhRncwek5URXlNVFV3TlRRME1EQmFNRmd4Q3pBSkJnTlYKQkFZVEFsVlRNUk13RVFZRFZRUUlFd3BEWVd4cFptOXlibWxoTVJZd0ZBWURWUVFIRXcxVFlXNGdSbkpoYm1OcApjMk52TVJ3d0dnWURWUVFERXhOdmNtUmxjbVZ5TG1WNFlXMXdiR1V1WTI5dE1Ga3dFd1lIS29aSXpqMENBUVlJCktvWkl6ajBEQVFjRFFnQUVvdExYWHFLcElndk9DTkR0bHpHb3JsbmNvekg2ZFhna1RrYmZoMlJrSG5zY1VhcE8KUVY3S1NMclFJTlU5ZUpVZkFWYzNjd1pzT2Jpd1hDWEE1cGNGRnFOTk1Fc3dEZ1lEVlIwUEFRSC9CQVFEQWdlQQpNQXdHQTFVZEV3RUIvd1FDTUFBd0t3WURWUjBqQkNRd0lvQWdVU2tjQlVKd1dKL2tPRmVhL3ZSUjlJUitwaGZmCmxrb3YxemRXQWZCOFBHd3dDZ1lJS29aSXpqMEVBd0lEUndBd1JBSWdYKzBWRzNhSlBTTXUrelZpWDlJRmluYkcKSVJjSU5FL3Rhd09LdUFlR21wZ0NJRkxackFVYlMvVXd0OHNPUHdXZTFCbTJYTW8rOXpCQ28zek5ma1BJZ2ZnQwotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
			CACerts:    []string{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNQakNDQWVTZ0F3SUJBZ0lSQUpCRFVzQWROYjlISGRERmpPbk91MW93Q2dZSUtvWkl6ajBFQXdJd2FURUwKTUFrR0ExVUVCaE1DVlZNeEV6QVJCZ05WQkFnVENrTmhiR2xtYjNKdWFXRXhGakFVQmdOVkJBY1REVk5oYmlCRwpjbUZ1WTJselkyOHhGREFTQmdOVkJBb1RDMlY0WVcxd2JHVXVZMjl0TVJjd0ZRWURWUVFERXc1allTNWxlR0Z0CmNHeGxMbU52YlRBZUZ3MHlOVEV5TVRjd05UUTBNREJhRncwek5URXlNVFV3TlRRME1EQmFNR2t4Q3pBSkJnTlYKQkFZVEFsVlRNUk13RVFZRFZRUUlFd3BEWVd4cFptOXlibWxoTVJZd0ZBWURWUVFIRXcxVFlXNGdSbkpoYm1OcApjMk52TVJRd0VnWURWUVFLRXd0bGVHRnRjR3hsTG1OdmJURVhNQlVHQTFVRUF4TU9ZMkV1WlhoaGJYQnNaUzVqCmIyMHdXVEFUQmdjcWhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUmpodFlodTVpOTRrcmtCaDBkRWg0aGNUMjIKVHhmZ3RnWjBkYUYyUE4wdUh1emhhMTZKcWVpckEyRUpKRndzQ0RTejUxTEd3ZEY1SFRzSHVwMzFrYm1sbzIwdwphekFPQmdOVkhROEJBZjhFQkFNQ0FhWXdIUVlEVlIwbEJCWXdGQVlJS3dZQkJRVUhBd0lHQ0NzR0FRVUZCd01CCk1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0tRWURWUjBPQkNJRUlGRXBIQVZDY0ZpZjVEaFhtdjcwVWZTRWZxWVgKMzVaS0w5YzNWZ0h3ZkR4c01Bb0dDQ3FHU000OUJBTUNBMGdBTUVVQ0lRQ1NOL0I2UXVqOFJualNTS2JNL2YwbwpzN0g2NkV2aDYySnozMEc1R0tOaGhBSWdRVkNSZ3N0SkRLU1h2NHFIdFpONjZ1Qm5nNGtUT3BiRVYwc2EyTm9BCnQrdz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="},
			AdminCerts: []string{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNDakNDQWJDZ0F3SUJBZ0lRY3JCU01RNE5WaG5GNk0xR2E0WWFEVEFLQmdncWhrak9QUVFEQWpCcE1Rc3cKQ1FZRFZRUUdFd0pWVXpFVE1CRUdBMVVFQ0JNS1EyRnNhV1p2Y201cFlURVdNQlFHQTFVRUJ4TU5VMkZ1SUVaeQpZVzVqYVhOamJ6RVVNQklHQTFVRUNoTUxaWGhoYlhCc1pTNWpiMjB4RnpBVkJnTlZCQU1URG1OaExtVjRZVzF3CmJHVXVZMjl0TUI0WERUSTFNVEl4TnpBMU5EUXdNRm9YRFRNMU1USXhOVEExTkRRd01Gb3dWakVMTUFrR0ExVUUKQmhNQ1ZWTXhFekFSQmdOVkJBZ1RDa05oYkdsbWIzSnVhV0V4RmpBVUJnTlZCQWNURFZOaGJpQkdjbUZ1WTJsegpZMjh4R2pBWUJnTlZCQU1NRVVGa2JXbHVRR1Y0WVcxd2JHVXVZMjl0TUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJCnpqMERBUWNEUWdBRXN0QlZiQUNmNEtIaTZtVktaaWp4ZnNidkNnd3BOL3VLdENFWTZVUVNEOUxhVzkvYVhLRjAKUXBrclo3enR3dUZPL2c0Z2paeFZFd1BFTG9lNlQ0OWxrNk5OTUVzd0RnWURWUjBQQVFIL0JBUURBZ2VBTUF3RwpBMVVkRXdFQi93UUNNQUF3S3dZRFZSMGpCQ1F3SW9BZ1VTa2NCVUp3V0ova09GZWEvdlJSOUlSK3BoZmZsa292CjF6ZFdBZkI4UEd3d0NnWUlLb1pJemowRUF3SURTQUF3UlFJaEFQMmxNK3c5aHpTbUx4MzRGR1l2RUZyd2JIWHcKemlySnh4WVcxN3dvUVN4d0FpQUR0ajBsRUFqejh0RVlSTS9WRytNQ2RHUmc0Tzc0a1VDOWJGeXVpelJubUE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="},
		},
		TLS: &current.MSP{
			KeyStore:  "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ2hkZDZuMTBkcGl2VlFRWXAKMzdJTFdUb056aEFHUWtQZnhEaTFHc2FESHBlaFJBTkNBQVNpMHRkZW9xa2lDODRJME8yWE1haXVXZHlqTWZwMQplQ1JPUnQrSFpHUWVleHhScWs1QlhzcEl1dEFnMVQxNGxSOEJWemR6Qm13NXVMQmNKY0RtbHdVVwotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==",
			SignCerts: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNERENDQWJPZ0F3SUJBZ0lSQUxWR3BXbTlTYTRGVzVaajF2aUZqU0V3Q2dZSUtvWkl6ajBFQXdJd2FURUwKTUFrR0ExVUVCaE1DVlZNeEV6QVJCZ05WQkFnVENrTmhiR2xtYjNKdWFXRXhGakFVQmdOVkJBY1REVk5oYmlCRwpjbUZ1WTJselkyOHhGREFTQmdOVkJBb1RDMlY0WVcxd2JHVXVZMjl0TVJjd0ZRWURWUVFERXc1allTNWxlR0Z0CmNHeGxMbU52YlRBZUZ3MHlOVEV5TVRjd05UUTBNREJhRncwek5URXlNVFV3TlRRME1EQmFNRmd4Q3pBSkJnTlYKQkFZVEFsVlRNUk13RVFZRFZRUUlFd3BEWVd4cFptOXlibWxoTVJZd0ZBWURWUVFIRXcxVFlXNGdSbkpoYm1OcApjMk52TVJ3d0dnWURWUVFERXhOdmNtUmxjbVZ5TG1WNFlXMXdiR1V1WTI5dE1Ga3dFd1lIS29aSXpqMENBUVlJCktvWkl6ajBEQVFjRFFnQUVvdExYWHFLcElndk9DTkR0bHpHb3JsbmNvekg2ZFhna1RrYmZoMlJrSG5zY1VhcE8KUVY3S1NMclFJTlU5ZUpVZkFWYzNjd1pzT2Jpd1hDWEE1cGNGRnFOTk1Fc3dEZ1lEVlIwUEFRSC9CQVFEQWdlQQpNQXdHQTFVZEV3RUIvd1FDTUFBd0t3WURWUjBqQkNRd0lvQWdVU2tjQlVKd1dKL2tPRmVhL3ZSUjlJUitwaGZmCmxrb3YxemRXQWZCOFBHd3dDZ1lJS29aSXpqMEVBd0lEUndBd1JBSWdYKzBWRzNhSlBTTXUrelZpWDlJRmluYkcKSVJjSU5FL3Rhd09LdUFlR21wZ0NJRkxackFVYlMvVXd0OHNPUHdXZTFCbTJYTW8rOXpCQ28zek5ma1BJZ2ZnQwotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
			CACerts:   []string{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNQakNDQWVTZ0F3SUJBZ0lSQUpCRFVzQWROYjlISGRERmpPbk91MW93Q2dZSUtvWkl6ajBFQXdJd2FURUwKTUFrR0ExVUVCaE1DVlZNeEV6QVJCZ05WQkFnVENrTmhiR2xtYjNKdWFXRXhGakFVQmdOVkJBY1REVk5oYmlCRwpjbUZ1WTJselkyOHhGREFTQmdOVkJBb1RDMlY0WVcxd2JHVXVZMjl0TVJjd0ZRWURWUVFERXc1allTNWxlR0Z0CmNHeGxMbU52YlRBZUZ3MHlOVEV5TVRjd05UUTBNREJhRncwek5URXlNVFV3TlRRME1EQmFNR2t4Q3pBSkJnTlYKQkFZVEFsVlRNUk13RVFZRFZRUUlFd3BEWVd4cFptOXlibWxoTVJZd0ZBWURWUVFIRXcxVFlXNGdSbkpoYm1OcApjMk52TVJRd0VnWURWUVFLRXd0bGVHRnRjR3hsTG1OdmJURVhNQlVHQTFVRUF4TU9ZMkV1WlhoaGJYQnNaUzVqCmIyMHdXVEFUQmdjcWhrak9QUUlCQmdncWhrak9QUU1CQndOQ0FBUmpodFlodTVpOTRrcmtCaDBkRWg0aGNUMjIKVHhmZ3RnWjBkYUYyUE4wdUh1emhhMTZKcWVpckEyRUpKRndzQ0RTejUxTEd3ZEY1SFRzSHVwMzFrYm1sbzIwdwphekFPQmdOVkhROEJBZjhFQkFNQ0FhWXdIUVlEVlIwbEJCWXdGQVlJS3dZQkJRVUhBd0lHQ0NzR0FRVUZCd01CCk1BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0tRWURWUjBPQkNJRUlGRXBIQVZDY0ZpZjVEaFhtdjcwVWZTRWZxWVgKMzVaS0w5YzNWZ0h3ZkR4c01Bb0dDQ3FHU000OUJBTUNBMGdBTUVVQ0lRQ1NOL0I2UXVqOFJualNTS2JNL2YwbwpzN0g2NkV2aDYySnozMEc1R0tOaGhBSWdRVkNSZ3N0SkRLU1h2NHFIdFpONjZ1Qm5nNGtUT3BiRVYwc2EyTm9BCnQrdz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="},
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
		if CurrentGinkgoTestDescription().Failed {
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
					for _, node := range orderer1nodes {
						Eventually(node.pollForCRStatus).Should((Equal(current.Precreated)))
					}
					// TODO flake
					// Eventually(orderer.pollForCRStatus).Should((Equal(current.Deploying)))
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
