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
			KeyStore:   "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ2FYb2MwNkxoWmliYjFsSEUKU0ZaY2NSeThmcWUySjROQW1rdEtXZEpFZVBxaFJBTkNBQVJ4UGVOKy94WHRLeTdXNGlZajUxQ29LQ2NmZ2Y4NApnMDBkamEzSStNeHNLSDZncVNQUGpXbThvUi9sYnZhbW9jay84bURoRi9yZTd3SU5qWkpGeG80aAotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==",
			SignCerts:  "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNUVENDQWZPZ0F3SUJBZ0lVTUw4NVhXVVJLZURqV1ZjelNWZ0ZoWDdtWlFjd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEl3TVRFek1ESXdNVGN3TUZvWERUSTFNVEV5T1RJd01qSXdNRm93WFRFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdZMnhwWlc1ME1RNHdEQVlEVlFRREV3VmhaRzFwYmpCWk1CTUdCeXFHClNNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJIRTk0MzcvRmUwckx0YmlKaVBuVUtnb0p4K0IvemlEVFIyTnJjajQKekd3b2ZxQ3BJOCtOYWJ5aEgrVnU5cWFoeVQveVlPRVgrdDd2QWcyTmtrWEdqaUdqZ1lVd2dZSXdEZ1lEVlIwUApBUUgvQkFRREFnZUFNQXdHQTFVZEV3RUIvd1FDTUFBd0hRWURWUjBPQkJZRUZNSGxPTGthZTFSbFRaZ1BNQ0ZQCkxKai80MHBzTUI4R0ExVWRJd1FZTUJhQUZNeTZicUR5Q1p1UThEeTBQWkhtVUNJTDRzNmlNQ0lHQTFVZEVRUWIKTUJtQ0YxTmhZV1J6TFUxaFkwSnZiMnN0VUhKdkxteHZZMkZzTUFvR0NDcUdTTTQ5QkFNQ0EwZ0FNRVVDSVFERAowY1Z6aEJFcGo1aFhYVXQzQSsxQVZOc2IyZDgxNVpZSVVVTG0xQXZ5T1FJZ1d1eldoVzQ5QUNWSG8zWkhNRE1vCmU5d3FRbUpTNDB2UGJtMEtOVUVkdURjPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==",
			CACerts:    []string{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNGakNDQWIyZ0F3SUJBZ0lVS2dNc2pwYlFSNlRHUUs3QVBhMEZmUVZxT1pvd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEl3TVRFek1ESXdNVFV3TUZvWERUTTFNVEV5TnpJd01UVXdNRm93YURFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1Sa3dGd1lEVlFRREV4Qm1ZV0p5YVdNdFkyRXRjMlZ5CmRtVnlNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUUrb2lXeWdGNWpLY081cWtzaG8zN3lzRSsKdXYxMEF5WWZrUGxVWXlBVkJOeGtlSGN1RUlWSmY5LzZRL2x2S2NvUyt6cFp2dlFiSTEzT1pSTDNMK25IZXFORgpNRU13RGdZRFZSMFBBUUgvQkFRREFnRUdNQklHQTFVZEV3RUIvd1FJTUFZQkFmOENBUUV3SFFZRFZSME9CQllFCkZNeTZicUR5Q1p1UThEeTBQWkhtVUNJTDRzNmlNQW9HQ0NxR1NNNDlCQU1DQTBjQU1FUUNJQmdSTXNqN3Azc1YKMHNieEQxa2t0amloVEpHVFJBWlZRQXVyY0hhRVVENFVBaUFoN0o4U2ZPQTc5VjN4RDdvaExFcmVpZHVnZnhIbAozWWxZS0g3MG9qQXhRZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"},
			AdminCerts: []string{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNUVENDQWZPZ0F3SUJBZ0lVTUw4NVhXVVJLZURqV1ZjelNWZ0ZoWDdtWlFjd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEl3TVRFek1ESXdNVGN3TUZvWERUSTFNVEV5T1RJd01qSXdNRm93WFRFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdZMnhwWlc1ME1RNHdEQVlEVlFRREV3VmhaRzFwYmpCWk1CTUdCeXFHClNNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJIRTk0MzcvRmUwckx0YmlKaVBuVUtnb0p4K0IvemlEVFIyTnJjajQKekd3b2ZxQ3BJOCtOYWJ5aEgrVnU5cWFoeVQveVlPRVgrdDd2QWcyTmtrWEdqaUdqZ1lVd2dZSXdEZ1lEVlIwUApBUUgvQkFRREFnZUFNQXdHQTFVZEV3RUIvd1FDTUFBd0hRWURWUjBPQkJZRUZNSGxPTGthZTFSbFRaZ1BNQ0ZQCkxKai80MHBzTUI4R0ExVWRJd1FZTUJhQUZNeTZicUR5Q1p1UThEeTBQWkhtVUNJTDRzNmlNQ0lHQTFVZEVRUWIKTUJtQ0YxTmhZV1J6TFUxaFkwSnZiMnN0VUhKdkxteHZZMkZzTUFvR0NDcUdTTTQ5QkFNQ0EwZ0FNRVVDSVFERAowY1Z6aEJFcGo1aFhYVXQzQSsxQVZOc2IyZDgxNVpZSVVVTG0xQXZ5T1FJZ1d1eldoVzQ5QUNWSG8zWkhNRE1vCmU5d3FRbUpTNDB2UGJtMEtOVUVkdURjPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="},
		},
		TLS: &current.MSP{
			KeyStore:  "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZzZuNit4cDJod1hrTzBrWHUKbUFiY2Z3aGNUcllDOEQ4SDJFNUZPUmNpMFBTaFJBTkNBQVFCMDBTNDhwbGlmd2tIN1RucGtZUTQrd1hJQ1piSwpnL1Z0U3ZoVUQyOC93dkd4VXdBZXBwSVZCRElCUUZBaE9xZ1F5SkpBQTZWbTVyd2RKaG1aR3M5SQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==",
			SignCerts: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNiRENDQWhLZ0F3SUJBZ0lVT3RnTGwwR0orSjU2T1llcXI3UFI1ckhKakhNd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEl3TVRFek1ESXdNVGt3TUZvWERUSTFNVEV5T1RJd01qUXdNRm93WFRFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdZMnhwWlc1ME1RNHdEQVlEVlFRREV3VmhaRzFwYmpCWk1CTUdCeXFHClNNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJBSFRSTGp5bVdKL0NRZnRPZW1SaERqN0JjZ0psc3FEOVcxSytGUVAKYnovQzhiRlRBQjZta2hVRU1nRkFVQ0U2cUJESWtrQURwV2JtdkIwbUdaa2F6MGlqZ2FRd2dhRXdEZ1lEVlIwUApBUUgvQkFRREFnT29NQjBHQTFVZEpRUVdNQlFHQ0NzR0FRVUZCd01CQmdnckJnRUZCUWNEQWpBTUJnTlZIUk1CCkFmOEVBakFBTUIwR0ExVWREZ1FXQkJTOTY4MUFxUEZ1dndHNUZsVFROS0J2Z2FKdk56QWZCZ05WSFNNRUdEQVcKZ0JUTXVtNmc4Z21ia1BBOHREMlI1bEFpQytMT29qQWlCZ05WSFJFRUd6QVpnaGRUWVdGa2N5MU5ZV05DYjI5cgpMVkJ5Ynk1c2IyTmhiREFLQmdncWhrak9QUVFEQWdOSUFEQkZBaUVBK0RzckZlUkxEQXJ1eVNxVWJmc2hVWkFCCmhMNXpqZ2k2ckpFZzFtQW1iSFVDSUUwSjFQOUlxVFZHMU54UjdEQ1lBdVZkbmJ4eWJHWkUyMDA5eDl3Y0pudksKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=",
			CACerts:   []string{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNGakNDQWIyZ0F3SUJBZ0lVS2dNc2pwYlFSNlRHUUs3QVBhMEZmUVZxT1pvd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEl3TVRFek1ESXdNVFV3TUZvWERUTTFNVEV5TnpJd01UVXdNRm93YURFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1Sa3dGd1lEVlFRREV4Qm1ZV0p5YVdNdFkyRXRjMlZ5CmRtVnlNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUUrb2lXeWdGNWpLY081cWtzaG8zN3lzRSsKdXYxMEF5WWZrUGxVWXlBVkJOeGtlSGN1RUlWSmY5LzZRL2x2S2NvUyt6cFp2dlFiSTEzT1pSTDNMK25IZXFORgpNRU13RGdZRFZSMFBBUUgvQkFRREFnRUdNQklHQTFVZEV3RUIvd1FJTUFZQkFmOENBUUV3SFFZRFZSME9CQllFCkZNeTZicUR5Q1p1UThEeTBQWkhtVUNJTDRzNmlNQW9HQ0NxR1NNNDlCQU1DQTBjQU1FUUNJQmdSTXNqN3Azc1YKMHNieEQxa2t0amloVEpHVFJBWlZRQXVyY0hhRVVENFVBaUFoN0o4U2ZPQTc5VjN4RDdvaExFcmVpZHVnZnhIbAozWWxZS0g3MG9qQXhRZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"},
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
			DisableNodeOU:  &current.BoolTrue,
			FabricVersion:  integration.FabricVersion + "-1",
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
			DisableNodeOU: &current.BoolTrue,
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
			DisableNodeOU: &current.BoolTrue,
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
			DisableNodeOU: &current.BoolTrue,
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
			UseChannelLess:    &current.BoolTrue,
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
			DisableNodeOU: &current.BoolTrue,
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
