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

package peer_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v1"
	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v2"
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"
)

const (
	adminCert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNUakNDQWZXZ0F3SUJBZ0lVWHdiSXdVeXBmZE1WU1dZU24zWFBFcFZhd2tRd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRFeE1qRTJNRGd3TUZvWERUSXdNVEV4TVRFMk1UTXdNRm93WHpFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdZMnhwWlc1ME1SQXdEZ1lEVlFRREV3ZHZjbVJsY21WeU1Ga3dFd1lICktvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUV2S2RXNytpVVYxbVB3N0J3S2FESkNYVmpha2dqTDhwWCtWaHcKaENLSkNLeXE4Vis4U29tK1AyYzBXdExxbytFU1dVWENKNFJiN0pyOWIzZVc2SmplaHFPQmhUQ0JnakFPQmdOVgpIUThCQWY4RUJBTUNCNEF3REFZRFZSMFRBUUgvQkFJd0FEQWRCZ05WSFE0RUZnUVVSUW4yemN2b3hUUE1rV1JPClZxaG9DL293YXZnd0h3WURWUjBqQkJnd0ZvQVVTUU9ZL0Z5YnNXcTlIWEo3c296aUFyLzhtQkV3SWdZRFZSMFIKQkJzd0dZSVhVMkZoWkhNdFRXRmpRbTl2YXkxUWNtOHViRzlqWVd3d0NnWUlLb1pJemowRUF3SURSd0F3UkFJZwpCNEZmM1dUOWYxcWRjaXBUUzJ6dXFWVDl5WUc1S0dYWmpTN0cyaHZrd0JJQ0lHMXRHb0FkZzNoVWk2TkpyblFJClhaRXFOcWlJWmhPL2hPRmd1emE0VUpZaQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	signCert  = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNUekNDQWZXZ0F3SUJBZ0lVQWNnakVkOHBkOE43Vjg0YmFleG4yQzU0dWtzd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRFeE1qRTRNell3TUZvWERUSTBNVEV4TURFNE5ERXdNRm93WHpFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdZMnhwWlc1ME1SQXdEZ1lEVlFRREV3ZHZjbVJsY21WeU1Ga3dFd1lICktvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUU2NFJwK1pvVnYyaTg0cE5KUUFNUHJpenJmZVlNT2Y0UnZ1eHkKNHZOUU1Pd3JEemlIZkFLTnZmdUJlbDhpQ2dndHRXM2paZTVkSEFZaFVIS2Ryb3FodmFPQmhUQ0JnakFPQmdOVgpIUThCQWY4RUJBTUNCNEF3REFZRFZSMFRBUUgvQkFJd0FEQWRCZ05WSFE0RUZnUVVWakl3Y1YwYXRNZmZWV1E5CnhtenpXVG9uYmlJd0h3WURWUjBqQkJnd0ZvQVVTUU9ZL0Z5YnNXcTlIWEo3c296aUFyLzhtQkV3SWdZRFZSMFIKQkJzd0dZSVhVMkZoWkhNdFRXRmpRbTl2YXkxUWNtOHViRzlqWVd3d0NnWUlLb1pJemowRUF3SURTQUF3UlFJaApBUGE4Y3VjL3QvOW45ZDZlSHZoUWdialNBK1k2dytERW1ka2RpdnJHaGE5RUFpQXdTZStlVGdsQWJYQVNoTnhwCkJpR0Rjc2IwZ1pmRmhQd1pIN1VnQW1IQjN3PT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
	certKey   = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ1p2VWRsUVZ6QlVSc3I2STMKZEVvd0ZlVGkvVkNLZVZqMmFwN2x3QWNYSzJLaFJBTkNBQVRyaEduNW1oVy9hTHppazBsQUF3K3VMT3Q5NWd3NQovaEcrN0hMaTgxQXc3Q3NQT0lkOEFvMjkrNEY2WHlJS0NDMjFiZU5sN2wwY0JpRlFjcDJ1aXFHOQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg=="
	caCert    = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNGakNDQWIyZ0F3SUJBZ0lVZi84bk94M2NqM1htVzNDSUo1L0Q1ejRRcUVvd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRBek1ERTNNamd3TUZvWERUTTBNVEF5TmpFM01qZ3dNRm93YURFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1Sa3dGd1lEVlFRREV4Qm1ZV0p5YVdNdFkyRXRjMlZ5CmRtVnlNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUVSbzNmbUc2UHkyUHd6cUMwNnFWZDlFOFgKZ044eldqZzFMb3lnMmsxdkQ4MXY1dENRRytCTVozSUJGQnI2VTRhc0tZTUREakd6TElERmdUUTRjVDd1VktORgpNRU13RGdZRFZSMFBBUUgvQkFRREFnRUdNQklHQTFVZEV3RUIvd1FJTUFZQkFmOENBUUV3SFFZRFZSME9CQllFCkZFa0RtUHhjbTdGcXZSMXllN0tNNGdLLy9KZ1JNQW9HQ0NxR1NNNDlCQU1DQTBjQU1FUUNJRC92QVFVSEh2SWwKQWZZLzM5UWdEU2ltTWpMZnhPTG44NllyR1EvWHpkQVpBaUFpUmlyZmlMdzVGbXBpRDhtYmlmRjV4bzdFUzdqNApaUWQyT0FUNCt5OWE0Zz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
)

type CoreConfig interface {
	ToBytes() ([]byte, error)
}

var (
	defaultRequestsPeer = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("10m"),
		corev1.ResourceMemory:           resource.MustParse("20M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
	}

	defaultLimitsPeer = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("100m"),
		corev1.ResourceMemory:           resource.MustParse("200M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
	}

	defaultRequestsFluentd = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("10m"),
		corev1.ResourceMemory:           resource.MustParse("20M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
	}

	defaultLimitsFluentd = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("100m"),
		corev1.ResourceMemory:           resource.MustParse("200M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
	}

	defaultRequestsCouchdb = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("20m"),
		corev1.ResourceMemory:           resource.MustParse("40M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
	}

	defaultLimitsCouchdb = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("200m"),
		corev1.ResourceMemory:           resource.MustParse("400M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
	}

	defaultRequestsDind = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("10m"),
		corev1.ResourceMemory:           resource.MustParse("20M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
	}

	defaultLimitsDind = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("100m"),
		corev1.ResourceMemory:           resource.MustParse("200M"),
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
			KeyStore:   certKey,
			SignCerts:  signCert,
			CACerts:    []string{caCert},
			AdminCerts: []string{adminCert},
		},
		TLS: &current.MSP{
			KeyStore:  certKey,
			SignCerts: signCert,
			CACerts:   []string{caCert},
		},
	}
)

var (
	peer  *Peer
	peer2 *Peer
	peer3 *Peer
)

var _ = Describe("Interaction between IBP-Operator and Kubernetes cluster", func() {
	SetDefaultEventuallyTimeout(420 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	BeforeEach(func() {
		peer = GetPeer1()
		CreatePeer(peer)

		peer2 = GetPeer2()
		CreatePeer(peer2)

		peer3 = GetPeer3()
		CreatePeer(peer3)

		integration.ClearOperatorConfig(kclient, namespace)
	})

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("IBPPeer controller", func() {
		When("applying first instance of IBPPeer CR", func() {
			var (
				err error
				dep *appsv1.Deployment
			)

			It("creates a IBPPeer custom resource", func() {
				By("setting the CR status to deploying", func() {
					Eventually(peer.pollForCRStatus).Should((Equal(current.Deploying)))
				})

				By("creating pvcs", func() {
					Eventually(peer.PVCExists).Should((Equal(true)))
					Expect(peer.getPVCStorageFromSpec(fmt.Sprintf("%s-pvc", peer.Name))).To(Equal("150Mi"))
					Expect(peer.getPVCStorageFromSpec(fmt.Sprintf("%s-statedb-pvc", peer.Name))).To(Equal("1Gi"))
				})

				By("creating a service", func() {
					Eventually(peer.ServiceExists).Should((Equal(true)))
				})

				By("creating a configmap", func() {
					Eventually(peer.ConfigMapExists).Should((Equal(true)))
				})

				By("starting a ingress", func() {
					Eventually(peer.IngressExists).Should((Equal(true)))
				})

				By("creating a deployment", func() {
					Eventually(peer.DeploymentExists).Should((Equal(true)))
				})

				By("starting a pod", func() {
					Eventually(peer.PodIsRunning).Should((Equal(true)))
				})

				By("creating config map that contains spec", func() {
					Eventually(func() bool {
						_, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), peer.Name+"-spec", metav1.GetOptions{})
						if err != nil {
							return false
						}
						return true
					}).Should(Equal(true))
				})

				By("setting the CR status to deployed when pod is running", func() {
					Eventually(peer.pollForCRStatus).Should((Equal(current.Deployed)))
				})

				cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), peer.Name+"-config", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				coreBytes := cm.BinaryData["core.yaml"]
				core, err := config.ReadCoreFromBytes(coreBytes)
				Expect(err).NotTo(HaveOccurred())

				By("overriding peer section in core.yaml", func() {
					configOverride, err := peer.CR.GetConfigOverride()
					Expect(err).NotTo(HaveOccurred())
					bytes, err := configOverride.(CoreConfig).ToBytes()
					Expect(err).NotTo(HaveOccurred())
					coreConfig := &config.Core{}
					err = yaml.Unmarshal(bytes, coreConfig)
					Expect(err).NotTo(HaveOccurred())
					Expect(core.Peer.ID).To(Equal(coreConfig.Peer.ID))
					Expect(string(coreBytes)).To(ContainSubstring("chaincode"))
					Expect(string(coreBytes)).To(ContainSubstring("vm"))
					Expect(string(coreBytes)).To(ContainSubstring("ledger"))
					Expect(string(coreBytes)).To(ContainSubstring("operations"))
					Expect(string(coreBytes)).To(ContainSubstring("metrics"))
				})

				By("overriding chaincode section in core.yaml", func() {
					configOverride, err := peer.CR.GetConfigOverride()
					Expect(err).NotTo(HaveOccurred())
					bytes, err := configOverride.(CoreConfig).ToBytes()
					Expect(err).NotTo(HaveOccurred())
					coreConfig := &config.Core{}
					err = yaml.Unmarshal(bytes, coreConfig)
					Expect(err).NotTo(HaveOccurred())
					Expect(core.Chaincode.StartupTimeout).To(Equal(coreConfig.Chaincode.StartupTimeout))
					Expect(core.Chaincode.ExecuteTimeout).To(Equal(coreConfig.Chaincode.ExecuteTimeout))
					//TODO: Disable the test flake
					// Expect(core.Chaincode.InstallTimeout).To(Equal(coreConfig.Chaincode.InstallTimeout))
				})

				By("creating secrets contain DeliveryClient.AddressOverrides ca certs", func() {
					Expect(core.Peer.DeliveryClient.AddressOverrides[0].CACertsFile).To(Equal("/orderer/certs/cert0.pem"))
					Expect(core.Peer.DeliveryClient.AddressOverrides[1].CACertsFile).To(Equal("/orderer/certs/cert1.pem"))

					s, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), peer.Name+"-orderercacerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					data := s.Data
					Expect(len(data)).To(Equal(2))

					caCertBytes, err := base64.StdEncoding.DecodeString(caCert)
					Expect(err).NotTo(HaveOccurred())

					signCertBytes, err := base64.StdEncoding.DecodeString(signCert)
					Expect(err).NotTo(HaveOccurred())

					Expect(data["cert0.pem"]).To(Equal(caCertBytes))
					Expect(data["cert1.pem"]).To(Equal(signCertBytes))
				})

				By("overriding operations section in core.yaml", func() {
					configOverride, err := peer.CR.GetConfigOverride()
					Expect(err).NotTo(HaveOccurred())
					bytes, err := configOverride.(CoreConfig).ToBytes()
					Expect(err).NotTo(HaveOccurred())
					coreConfig := &config.Core{}
					err = yaml.Unmarshal(bytes, coreConfig)
					Expect(err).NotTo(HaveOccurred())
					Expect(core.Operations.ListenAddress).To(Equal(coreConfig.Operations.ListenAddress))
					Expect(core.Operations.TLS.Certificate).To(Equal(coreConfig.Operations.TLS.Certificate))
				})

				By("overriding metrics section in core.yaml", func() {
					configOverride, err := peer.CR.GetConfigOverride()
					Expect(err).NotTo(HaveOccurred())
					bytes, err := configOverride.(CoreConfig).ToBytes()
					Expect(err).NotTo(HaveOccurred())
					coreConfig := &config.Core{}
					err = yaml.Unmarshal(bytes, coreConfig)
					Expect(err).NotTo(HaveOccurred())
					Expect(core.Metrics.Statsd.Address).To(Equal(coreConfig.Metrics.Statsd.Address))
				})
			})

			// TODO: Test marked as pending until portworx issue is resolved, currently zone is
			// required to be passed for provisioning to work. Once portworx is working again, this
			// test should be reenabled
			PIt("should not find zone and region", func() {
				// Wait for new deployment before querying deployment for updates
				wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer.Name, metav1.GetOptions{})
					if dep != nil {
						if dep.Status.UpdatedReplicas >= 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
							return true, nil
						}
					}
					return false, nil
				})

				dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("checking zone", func() {
					Expect(peer.TestAffinityZone(dep)).Should((Equal(false)))
				})

				By("checking region", func() {
					Expect(peer.TestAffinityRegion(dep)).Should((Equal(false)))
				})
			})

			When("the custom resource is updated", func() {
				var (
					dep                        *appsv1.Deployment
					newResourceRequestsPeer    corev1.ResourceList
					newResourceLimitsPeer      corev1.ResourceList
					newResourceRequestsProxy   corev1.ResourceList
					newResourceLimitsProxy     corev1.ResourceList
					newResourceRequestsCouchdb corev1.ResourceList
					newResourceLimitsCouchdb   corev1.ResourceList
				)

				BeforeEach(func() {
					newResourceRequestsPeer = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("90m"),
						corev1.ResourceMemory:           resource.MustParse("180M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					}
					newResourceLimitsPeer = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("90m"),
						corev1.ResourceMemory:           resource.MustParse("180M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					}

					newResourceRequestsProxy = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("91m"),
						corev1.ResourceMemory:           resource.MustParse("181M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					}
					newResourceLimitsProxy = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("91m"),
						corev1.ResourceMemory:           resource.MustParse("181M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					}

					newResourceRequestsCouchdb = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("193m"),
						corev1.ResourceMemory:           resource.MustParse("383M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					}
					newResourceLimitsCouchdb = map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU:              resource.MustParse("193m"),
						corev1.ResourceMemory:           resource.MustParse("383M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					}

					peer.CR.Spec.Resources = &current.PeerResources{
						Peer: &corev1.ResourceRequirements{
							Requests: newResourceRequestsPeer,
							Limits:   newResourceLimitsPeer,
						},
						GRPCProxy: &corev1.ResourceRequirements{
							Requests: newResourceRequestsProxy,
							Limits:   newResourceLimitsProxy,
						},
						CouchDB: &corev1.ResourceRequirements{
							Requests: newResourceRequestsCouchdb,
							Limits:   newResourceLimitsCouchdb,
						},
					}

					startupTimeout, err := common.ParseDuration("200s")
					Expect(err).NotTo(HaveOccurred())

					configOverride := config.Core{
						Core: v2.Core{
							Peer: v2.Peer{
								ID: "new-peerid",
							},
							Chaincode: v2.Chaincode{
								StartupTimeout: startupTimeout,
							},
						},
					}

					configBytes, err := json.Marshal(configOverride)
					Expect(err).NotTo(HaveOccurred())

					peer.CR.Spec.ConfigOverride = &runtime.RawExtension{Raw: configBytes}

					Eventually(peer.DeploymentExists).Should((Equal(true)))
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
				})

				It("updates the instance of IBPPeer if resources and config overrides are updated in CR", func() {
					peerResources := dep.Spec.Template.Spec.Containers[0].Resources
					Expect(peerResources.Requests).To(Equal(defaultRequestsPeer))
					Expect(peerResources.Limits).To(Equal(defaultLimitsPeer))

					proxyResources := dep.Spec.Template.Spec.Containers[1].Resources
					Expect(proxyResources.Requests).To(Equal(defaultRequestsProxy))
					Expect(proxyResources.Limits).To(Equal(defaultLimitsProxy))

					couchDBResources := dep.Spec.Template.Spec.Containers[2].Resources
					Expect(couchDBResources.Requests).To(Equal(defaultRequestsCouchdb))
					Expect(couchDBResources.Limits).To(Equal(defaultLimitsCouchdb))

					bytes, err := json.Marshal(peer.CR)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibppeers").Name(peer.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					// Wait for new deployment before querying deployment for updates
					wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer.Name, metav1.GetOptions{})
						if dep != nil {
							if dep.Status.UpdatedReplicas >= 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
								if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() == newResourceRequestsProxy.Cpu().MilliValue() {
									return true, nil
								}
							}
						}
						return false, nil
					})

					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					updatedPeerResources := dep.Spec.Template.Spec.Containers[0].Resources
					Expect(updatedPeerResources.Requests).To(Equal(newResourceRequestsPeer))
					Expect(updatedPeerResources.Limits).To(Equal(newResourceLimitsPeer))

					updatedProxyResources := dep.Spec.Template.Spec.Containers[1].Resources
					Expect(updatedProxyResources.Requests).To(Equal(newResourceRequestsProxy))
					Expect(updatedProxyResources.Limits).To(Equal(newResourceLimitsProxy))

					updatedCouchDBResources := dep.Spec.Template.Spec.Containers[2].Resources
					Expect(updatedCouchDBResources.Requests).To(Equal(newResourceRequestsCouchdb))
					Expect(updatedCouchDBResources.Limits).To(Equal(newResourceLimitsCouchdb))

					By("updating the config map with new values from override", func() {
						core := &config.Core{}

						Eventually(func() string {
							cm, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), peer.Name+"-config", metav1.GetOptions{})
							Expect(err).NotTo(HaveOccurred())

							coreBytes := cm.BinaryData["core.yaml"]
							core, err = config.ReadCoreFromBytes(coreBytes)
							Expect(err).NotTo(HaveOccurred())

							return core.Peer.ID
						}).Should(Equal("new-peerid"))

						configOverride, err := peer.CR.GetConfigOverride()
						Expect(err).NotTo(HaveOccurred())

						bytes, err := configOverride.(CoreConfig).ToBytes()
						Expect(err).NotTo(HaveOccurred())

						coreConfig := &config.Core{}
						err = yaml.Unmarshal(bytes, coreConfig)
						Expect(err).NotTo(HaveOccurred())
						Expect(core.Chaincode.StartupTimeout).To(Equal(coreConfig.Chaincode.StartupTimeout))
					})
				})
			})

			When("a deployment managed by operator is manually edited", func() {
				var (
					err error
					dep *appsv1.Deployment
				)

				BeforeEach(func() {
					Eventually(func() bool {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer.Name, metav1.GetOptions{})
						if err == nil && dep != nil {
							return true
						}
						return false
					}).Should(Equal(true))
				})

				It("restores states", func() {
					origRequests := dep.Spec.Template.Spec.Containers[0].Resources.Requests

					dep.Spec.Template.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("107m"),
						corev1.ResourceMemory: resource.MustParse("207M"),
					}

					depBytes, err := json.Marshal(dep)
					Expect(err).NotTo(HaveOccurred())

					kclient.AppsV1().Deployments(namespace).Patch(context.TODO(), peer.Name, types.MergePatchType, depBytes, metav1.PatchOptions{})
					// Wait for new deployment before querying deployment for updates
					wait.Poll(500*time.Millisecond, 300*time.Second, func() (bool, error) {
						dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer.Name, metav1.GetOptions{})
						if dep != nil {
							if len(dep.Spec.Template.Spec.Containers) >= 1 {
								if dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() == origRequests.Cpu().MilliValue() {
									return true, nil
								}
							}
						}
						return false, nil
					})

					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					Expect(dep.Spec.Template.Spec.Containers[0].Resources.Requests).To(Equal(origRequests))
				})
			})

			When("admin certs are updated in peer spec", func() {
				It("updates the admin cert secret", func() {
					sec, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-ibppeer1-admincerts", metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					certBytes := sec.Data["admincert-0.pem"]
					certBase64 := base64.StdEncoding.EncodeToString(certBytes)
					Expect(certBase64).To(Equal(adminCert))

					peer.CR.Spec.Secret.MSP.Component.AdminCerts = []string{signCert}
					bytes, err := json.Marshal(peer.CR)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibppeers").Name(peer.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())

					Eventually(peer.checkAdminCertUpdate).Should(Equal(signCert))
				})
			})
		})

		When("applying the second instance of IBPPeer CR", func() {
			var (
				err error
				dep *appsv1.Deployment
			)

			It("creates a second IBPPeer custom resource", func() {
				By("starting a pod", func() {
					Eventually(peer2.PodIsRunning).Should((Equal(true)))
				})
			})

			// TODO: Test marked as pending until portworx issue is resolved, currently zone is
			// required to be passed for provisioning to work. Once portworx is working again, this
			// test should be reenabled
			PIt("should find zone and region", func() {
				// Wait for new deployment before querying deployment for updates
				wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
					dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer2.Name, metav1.GetOptions{})
					if dep != nil {
						if dep.Status.UpdatedReplicas >= 1 && dep.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
							return true, nil
						}
					}
					return false, nil
				})

				dep, err = kclient.AppsV1().Deployments(namespace).Get(context.TODO(), peer2.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("checking zone", func() {
					Expect(peer2.TestAffinityZone(dep)).To((Equal(true)))
				})

				By("checking region", func() {
					Expect(peer2.TestAffinityRegion(dep)).To((Equal(true)))
				})
			})
		})

		Context("operator pod restart", func() {
			var (
				oldPodName string
			)

			Context("should not trigger deployment restart if config overrides not updated", func() {
				BeforeEach(func() {
					Eventually(peer.PodIsRunning).Should((Equal(true)))

					Eventually(func() int { return len(peer.GetRunningPods()) }).Should(Equal(1))
					oldPodName = peer.GetRunningPods()[0].Name
				})

				It("does not restart the peer pod", func() {
					Eventually(peer.PodIsRunning).Should((Equal(true)))

					Eventually(func() bool {
						pods := peer.GetRunningPods()
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

			PContext("should trigger deployment restart if config overrides are updated", func() {
				BeforeEach(func() {
					Eventually(peer.PodIsRunning).Should((Equal(true)))
					Eventually(func() int {
						return len(peer.GetPods())
					}).Should(Equal(1))

					configOverride := config.Core{
						Core: v2.Core{
							Peer: v2.Peer{
								ID: "new-id",
							},
						},
					}

					configBytes, err := json.Marshal(configOverride)
					Expect(err).NotTo(HaveOccurred())

					peer.CR.Spec.ConfigOverride = &runtime.RawExtension{Raw: configBytes}

					bytes, err := json.Marshal(peer.CR)
					Expect(err).NotTo(HaveOccurred())

					result := ibpCRClient.Patch(types.MergePatchType).Namespace(namespace).Resource("ibppeers").Name(peer.Name).Body(bytes).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})

				It("restarts the peer pod", func() {
					Eventually(peer.PodIsRunning).Should((Equal(false)))
					Eventually(peer.PodIsRunning).Should((Equal(true)))

					Eventually(func() bool {
						pods := peer.GetPods()
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

		When("applying incorrectly configured third instance of IBPPeer CR", func() {
			It("should set the CR status to error", func() {
				Eventually(peer3.pollForCRStatus).Should((Equal(current.Error)))

				crStatus := &current.IBPPeer{}
				result := ibpCRClient.Get().Namespace(namespace).Resource("ibppeers").Name(peer3.Name).Do(context.TODO())
				result.Into(crStatus)

				Expect(crStatus.Status.Message).To(ContainSubstring("user must accept license before continuing"))
			})
		})

		Context("delete crs", func() {
			It("should delete IBPPeer CR", func() {
				By("deleting the first instance of IBPPeer CR", func() {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibppeers").Name(peer.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})

				By("deleting the second instance of IBPPeer CR", func() {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibppeers").Name(peer2.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})

				By("deleting the third instance of IBPPeer CR", func() {
					result := ibpCRClient.Delete().Namespace(namespace).Resource("ibppeers").Name(peer3.Name).Do(context.TODO())
					Expect(result.Error()).NotTo(HaveOccurred())
				})
			})
		})
	})
})

func GetPeer1() *Peer {
	startupTimeout, err := common.ParseDuration("200s")
	Expect(err).NotTo(HaveOccurred())
	executeTimeout, err := common.ParseDuration("20s")
	Expect(err).NotTo(HaveOccurred())
	installTimeout, err := common.ParseDuration("600s")
	Expect(err).NotTo(HaveOccurred())

	configOverride := config.Core{
		Core: v2.Core{
			Peer: v2.Peer{
				ID: "testPeerID",
				DeliveryClient: v1.DeliveryClient{
					AddressOverrides: []v1.AddressOverride{
						v1.AddressOverride{
							CACertsFile: caCert,
						},
						v1.AddressOverride{
							CACertsFile: signCert,
						},
					},
				},
			},
			Chaincode: v2.Chaincode{
				StartupTimeout: startupTimeout,
				ExecuteTimeout: executeTimeout,
				InstallTimeout: installTimeout,
			},
			Metrics: v1.Metrics{
				Statsd: v1.Statsd{
					Address: "127.0.0.1:9445",
				},
			},
			Operations: v1.Operations{
				ListenAddress: "127.0.0.1:9444",
				TLS: v1.OperationsTLS{
					Certificate: v1.File{
						File: "ops-tls-cert.pem",
					},
				},
			},
		},
	}

	configBytes, err := json.Marshal(configOverride)
	Expect(err).NotTo(HaveOccurred())

	name := "ibppeer1"
	cr := &current.IBPPeer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IBPPeer",
			APIVersion: "ibp.com/v1beta1",
		},
		Spec: current.IBPPeerSpec{
			License: current.License{
				Accept: true,
			},
			MSPID:            "test-peer-mspid",
			ImagePullSecrets: []string{"regcred"},
			Region:           "select",
			Zone:             "select",
			Images: &current.PeerImages{
				CouchDBImage:  integration.CouchdbImage,
				CouchDBTag:    integration.CouchdbTag,
				GRPCWebImage:  integration.GrpcwebImage,
				GRPCWebTag:    integration.GrpcwebTag,
				PeerImage:     integration.PeerImage,
				PeerTag:       integration.PeerTag,
				PeerInitImage: integration.InitImage,
				PeerInitTag:   integration.InitTag,
			},
			Domain: integration.TestAutomation1IngressDomain,
			Resources: &current.PeerResources{
				DinD: &corev1.ResourceRequirements{

					Requests: defaultRequestsDind,
					Limits:   defaultLimitsDind,
				},
				Peer: &corev1.ResourceRequirements{
					Requests: defaultRequestsPeer,
					Limits:   defaultLimitsPeer,
				},
				GRPCProxy: &corev1.ResourceRequirements{
					Requests: defaultRequestsProxy,
					Limits:   defaultLimitsProxy,
				},
				FluentD: &corev1.ResourceRequirements{
					Requests: defaultRequestsFluentd,
					Limits:   defaultLimitsFluentd,
				},
				CouchDB: &corev1.ResourceRequirements{
					Requests: defaultRequestsCouchdb,
					Limits:   defaultLimitsCouchdb,
				},
			},
			Storage: &current.PeerStorages{
				Peer: &current.StorageSpec{
					Size: "150Mi",
				},
				StateDB: &current.StorageSpec{
					Size: "1Gi",
				},
			},
			Ingress: current.Ingress{
				TlsSecretName: "tlssecret",
			},
			Secret: &current.SecretSpec{
				MSP: testMSPSpec,
			},
			ConfigOverride: &runtime.RawExtension{Raw: configBytes},
			DisableNodeOU:  pointer.Bool(true),
			FabricVersion:  integration.FabricVersion + "-1",
		},
	}
	cr.Name = name

	return &Peer{
		Name: name,
		CR:   cr,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}

func GetPeer2() *Peer {
	name := "ibppeer2"
	cr := &current.IBPPeer{
		Spec: current.IBPPeerSpec{
			License: current.License{
				Accept: true,
			},
			MSPID:            "test-peer2-mspid",
			StateDb:          "leveldb",
			Region:           "select",
			Zone:             "select",
			ImagePullSecrets: []string{"regcred"},
			Images: &current.PeerImages{
				CouchDBImage:  integration.CouchdbImage,
				CouchDBTag:    integration.CouchdbTag,
				GRPCWebImage:  integration.GrpcwebImage,
				GRPCWebTag:    integration.GrpcwebTag,
				PeerImage:     integration.PeerImage,
				PeerTag:       integration.PeerTag,
				PeerInitImage: integration.InitImage,
				PeerInitTag:   integration.InitTag,
			},
			Domain: integration.TestAutomation1IngressDomain,
			Storage: &current.PeerStorages{
				Peer: &current.StorageSpec{
					Size: "150Mi",
				},
				StateDB: &current.StorageSpec{
					Size: "1Gi",
				},
			},
			Ingress: current.Ingress{
				TlsSecretName: "tlssecret",
			},
			Secret: &current.SecretSpec{
				MSP: testMSPSpec,
			},
			DisableNodeOU: pointer.Bool(true),
			FabricVersion: integration.FabricVersion + "-1",
		},
	}
	cr.Name = name

	return &Peer{
		Name: name,
		CR:   cr,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}

func GetPeer3() *Peer {
	name := "ibppeer3"
	cr := &current.IBPPeer{
		Spec: current.IBPPeerSpec{
			Domain:        integration.TestAutomation1IngressDomain,
			FabricVersion: integration.FabricVersion + "-1",
		},
	}
	cr.Name = name

	return &Peer{
		Name: name,
		CR:   cr,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}
