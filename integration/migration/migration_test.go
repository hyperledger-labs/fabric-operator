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

package migration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	cainit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/mocks"
	ordererinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer"
	peerinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	allmigrator "github.com/IBM-Blockchain/fabric-operator/pkg/migrator"
	"github.com/IBM-Blockchain/fabric-operator/pkg/migrator/initsecret"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
	"github.com/IBM-Blockchain/fabric-operator/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func GetLabels(instance v1.Object) map[string]string {
	return map[string]string{
		"app": "peermigraton",
	}
}

func RandomNodePort() int32 {
	rand.Seed(time.Now().UnixNano())
	min := 30000
	max := 32767
	return int32(rand.Intn(max-min+1) + min)
}

// TODO api versioning/migration logic will be updated
var _ = PDescribe("migrating", func() {
	Context("ca", func() {
		var (
			migrator          *allmigrator.Migrator
			instance          *current.IBPCA
			httpNodePort      int32
			operationNodePort int32
		)

		BeforeEach(func() {
			logf.SetLogger(zap.New())

			defaultConfigs := "../../defaultconfig"
			of, err := offering.GetType("K8S")
			Expect(err).To(BeNil())

			operatorCfg := &config.Config{
				CAInitConfig: &cainit.Config{
					CADefaultConfigPath:    filepath.Join(defaultConfigs, "ca/ca.yaml"),
					TLSCADefaultConfigPath: filepath.Join(defaultConfigs, "ca/tlsca.yaml"),
					SharedPath:             "/shared",
				},
				PeerInitConfig: &peerinit.Config{
					OUFile: filepath.Join(defaultConfigs, "peer/ouconfig.yaml"),
				},
				OrdererInitConfig: &ordererinit.Config{
					OrdererFile:  filepath.Join(defaultConfigs, "orderer/orderer.yaml"),
					ConfigTxFile: filepath.Join(defaultConfigs, "orderer/configtx.yaml"),
					OUFile:       filepath.Join(defaultConfigs, "orderer/ouconfig.yaml"),
				},
				Offering: of,
			}

			migrator = allmigrator.New(mgr, operatorCfg, namespace)

			consoleinstance := &current.IBPConsole{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "consolemigration0",
					Namespace: namespace,
				},
				Spec: current.IBPConsoleSpec{
					NetworkInfo: &current.NetworkInfo{
						Domain: "domain",
					},
				},
				Status: current.IBPConsoleStatus{
					CRStatus: current.CRStatus{
						Status:  current.True,
						Version: version.V213,
					},
				},
			}
			err = client.Create(context.TODO(), consoleinstance)
			Expect(err).NotTo(HaveOccurred())

			err = client.UpdateStatus(context.TODO(), consoleinstance)
			Expect(err).NotTo(HaveOccurred())

			instance = &current.IBPCA{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "camigration",
					Namespace: namespace,
				},
				Spec: current.IBPCASpec{
					FabricVersion: integration.FabricCAVersion,
				},
				Status: current.IBPCAStatus{
					CRStatus: current.CRStatus{
						Status: current.True,
					},
				},
			}
			err = client.Create(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())
			err = client.UpdateStatus(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())

			operationNodePort = RandomNodePort()
			httpNodePort = RandomNodePort()
			service := &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							Name:     "http",
							Port:     int32(7054),
							NodePort: httpNodePort,
						},
						corev1.ServicePort{
							Name:     "operations",
							Port:     int32(9443),
							NodePort: operationNodePort,
						},
					},
				},
			}
			service.Name = "camigration-service"
			service.Namespace = namespace

			httpNodePort, operationNodePort = CreateServiceWithRetry(service, 3)
			pathType := networkingv1.PathTypeImplementationSpecific
			ingress := &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						networkingv1.IngressRule{
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										networkingv1.HTTPIngressPath{
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "camigration-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 443,
													},
												},
											},
											Path:     "/",
											PathType: &pathType,
										},
									},
								},
							},
						},
					},
				},
			}
			ingress.Name = "camigration"
			ingress.Namespace = namespace

			err = client.Create(context.TODO(), ingress)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-ca", instance.Name),
					Namespace: instance.Namespace,
				},
			}
			err = client.Create(context.TODO(), cm)
			Expect(err).NotTo(HaveOccurred())

			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-overrides", instance.Name),
					Namespace: instance.Namespace,
				},
			}
			err = client.Create(context.TODO(), cm)
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-ca", instance.Name),
					Namespace: instance.Namespace,
				},
			}
			err = client.Create(context.TODO(), secret)
			Expect(err).NotTo(HaveOccurred())

			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-tlsca", instance.Name),
					Namespace: instance.Namespace,
				},
			}
			err = client.Create(context.TODO(), secret)
			Expect(err).NotTo(HaveOccurred())
		})

		It("migrates ca resources", func() {
			err := migrator.Migrate()
			Expect(err).NotTo(HaveOccurred())

			By("creating a secret with state of current resources before migration", func() {
				var secret *corev1.Secret
				var err error

				Eventually(func() bool {
					secret, err = kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "camigration-oldstate", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))

				Expect(secret.Data["camigration-service"]).NotTo(Equal(""))
				Expect(secret.Data["camigration-cm-ca"]).NotTo(Equal(""))
				Expect(secret.Data["camigration-cm-overrides"]).NotTo(Equal(""))
				Expect(secret.Data["camigration-secret-ca"]).NotTo(Equal(""))
				Expect(secret.Data["camigration-secret-tlsca"]).NotTo(Equal(""))
			})

			By("creating a new service with no 'service' in name and same nodeport", func() {
				var service *corev1.Service
				var err error

				Eventually(func() bool {
					service, err = kclient.CoreV1().Services(namespace).Get(context.TODO(), "camigration", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))

				Expect(service.Spec.Ports[0].NodePort).To(Equal(httpNodePort))
				Expect(service.Spec.Ports[1].NodePort).To(Equal(operationNodePort))

				_, err = kclient.CoreV1().Services(namespace).Get(context.TODO(), "camigration-service", metav1.GetOptions{})
				Expect(err).To(HaveOccurred())
			})

			By("creating a new ingress with no dashes and same servicename", func() {
				var ingress *networkingv1.Ingress
				var err error

				Eventually(func() bool {
					ingress, err = kclient.NetworkingV1().Ingresses(namespace).Get(context.TODO(), "camigration", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))

				Expect(ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name).To(Equal("camigration"))
			})
		})
	})

	Context("console", func() {
		var (
			migrator          *allmigrator.Migrator
			instance          *current.IBPConsole
			httpNodePort      int32
			operationNodePort int32
		)

		BeforeEach(func() {
			logf.SetLogger(zap.New())

			defaultConfigs := "../../defaultconfig"
			of, err := offering.GetType("K8S")

			operatorCfg := &config.Config{
				CAInitConfig: &cainit.Config{
					CADefaultConfigPath:    filepath.Join(defaultConfigs, "ca/ca.yaml"),
					TLSCADefaultConfigPath: filepath.Join(defaultConfigs, "ca/tlsca.yaml"),
					SharedPath:             "/shared",
				},
				PeerInitConfig: &peerinit.Config{
					OUFile: filepath.Join(defaultConfigs, "peer/ouconfig.yaml"),
				},
				OrdererInitConfig: &ordererinit.Config{
					OrdererFile:  filepath.Join(defaultConfigs, "orderer/orderer.yaml"),
					ConfigTxFile: filepath.Join(defaultConfigs, "orderer/configtx.yaml"),
					OUFile:       filepath.Join(defaultConfigs, "orderer/ouconfig.yaml"),
				},
				Offering: of,
			}

			migrator = allmigrator.New(mgr, operatorCfg, namespace)

			instance = &current.IBPConsole{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "consolemigration",
					Namespace: namespace,
				},
				Spec: current.IBPConsoleSpec{
					NetworkInfo: &current.NetworkInfo{},
				},
				Status: current.IBPConsoleStatus{
					CRStatus: current.CRStatus{
						Status: current.True,
					},
				},
			}
			err = client.Create(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())
			err = client.UpdateStatus(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())

			operationNodePort = RandomNodePort()
			httpNodePort = RandomNodePort()
			service := &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							Name:     "http",
							Port:     int32(7054),
							NodePort: httpNodePort,
						},
						corev1.ServicePort{
							Name:     "operations",
							Port:     int32(9443),
							NodePort: operationNodePort,
						},
					},
				},
			}
			service.Name = "consolemigration-service"
			service.Namespace = namespace

			httpNodePort, operationNodePort = CreateServiceWithRetry(service, 3)

			ingress := &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						networkingv1.IngressRule{
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										networkingv1.HTTPIngressPath{
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "consolemigration-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 443,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			ingress.Name = "consolemigration"
			ingress.Namespace = namespace

			err = client.Create(context.TODO(), ingress)
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-console-pw", instance.Name),
					Namespace: instance.Namespace,
				},
			}
			err = client.Create(context.TODO(), secret)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-configmap", instance.Name),
					Namespace: instance.Namespace,
				},
			}
			err = client.Create(context.TODO(), cm)
			Expect(err).NotTo(HaveOccurred())

			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-deployer-template", instance.Name),
					Namespace: instance.Namespace,
				},
			}
			err = client.Create(context.TODO(), cm)
			Expect(err).NotTo(HaveOccurred())

			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-template-configmap", instance.Name),
					Namespace: instance.Namespace,
				},
			}
			err = client.Create(context.TODO(), cm)
			Expect(err).NotTo(HaveOccurred())

			n := types.NamespacedName{
				Name:      cm.GetName(),
				Namespace: cm.GetNamespace(),
			}

			err = wait.Poll(500*time.Millisecond, 30*time.Second, func() (bool, error) {
				err := client.Get(context.TODO(), n, cm)
				if err == nil {
					return true, nil
				}
				return false, nil
			})
			Expect(err).NotTo(HaveOccurred())

		})

		It("migrates console resources", func() {
			err := migrator.Migrate()
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				_, err := kclient.CoreV1().Services(namespace).Get(context.TODO(), "consolemigration-service", metav1.GetOptions{})
				if err != nil {
					return false
				}
				return true
			}).Should(Equal(false))

			By("creating a secret with state of current resources before migration", func() {
				var secret *corev1.Secret
				var err error

				Eventually(func() bool {
					secret, err = kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "consolemigration-oldstate", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))

				Expect(secret.Data["consolemigration-service"]).NotTo(Equal(""))
				Expect(secret.Data["consolemigration-cm"]).NotTo(Equal(""))
				Expect(secret.Data["consolemigration-cm-deployer"]).NotTo(Equal(""))
				Expect(secret.Data["consolemigration-cm-template"]).NotTo(Equal(""))
				Expect(secret.Data["consolemigration-secret-pw"]).NotTo(Equal(""))
			})

			By("creating a new service with 'service' and same nodeport", func() {
				var service *corev1.Service
				var err error

				Eventually(func() bool {
					service, err = kclient.CoreV1().Services(namespace).Get(context.TODO(), "consolemigration", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))

				Expect(service.Spec.Ports[0].NodePort).To(Equal(httpNodePort))
				Expect(service.Spec.Ports[1].NodePort).To(Equal(operationNodePort))
			})

			By("creating a new ingress with no dashes and same servicename", func() {
				var ingress *networkingv1.Ingress
				var err error

				Eventually(func() bool {
					ingress, err = kclient.NetworkingV1().Ingresses(namespace).Get(context.TODO(), "consolemigration", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))

				Expect(ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name).To(Equal("consolemigration"))
			})
		})
	})

	Context("peer", func() {
		var (
			migrator             *allmigrator.Migrator
			instance             *current.IBPPeer
			mspSecret            *initsecret.Secret
			peerApiNodePort      int32
			operationNodePort    int32
			grpcwebDebugNodePort int32
			grpcwebNodePort      int32
		)

		BeforeEach(func() {
			logf.SetLogger(zap.New())
			mockValidator := &mocks.CryptoValidator{}
			mockValidator.CheckEcertCryptoReturns(errors.New("not found"))

			defaultConfigs := "../../defaultconfig"
			of, err := offering.GetType("K8S")

			operatorCfg := &config.Config{
				CAInitConfig: &cainit.Config{
					CADefaultConfigPath:    filepath.Join(defaultConfigs, "ca/ca.yaml"),
					TLSCADefaultConfigPath: filepath.Join(defaultConfigs, "ca/tlsca.yaml"),
					SharedPath:             "/shared",
				},
				PeerInitConfig: &peerinit.Config{
					CorePeerFile: filepath.Join(defaultConfigs, "peer/core.yaml"),
					OUFile:       filepath.Join(defaultConfigs, "peer/ouconfig.yaml"),
				},
				OrdererInitConfig: &ordererinit.Config{
					OrdererFile:  filepath.Join(defaultConfigs, "orderer/orderer.yaml"),
					ConfigTxFile: filepath.Join(defaultConfigs, "orderer/configtx.yaml"),
					OUFile:       filepath.Join(defaultConfigs, "orderer/ouconfig.yaml"),
				},
				Offering: of,
			}

			migrator = allmigrator.New(mgr, operatorCfg, namespace)

			instance = &current.IBPPeer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "peermigration",
					Namespace: namespace,
				},
				Spec: current.IBPPeerSpec{
					Domain:           "127.0.0.1",
					ImagePullSecrets: []string{"pullSecret"},
					Images: &current.PeerImages{
						CouchDBImage: integration.CouchdbImage,
						CouchDBTag:   integration.CouchdbTag,
					},
				},
				Status: current.IBPPeerStatus{
					CRStatus: current.CRStatus{
						Status: current.True,
					},
				},
			}
			err = client.Create(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())
			err = client.UpdateStatus(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())

			peerApiNodePort = RandomNodePort()
			operationNodePort = RandomNodePort()
			grpcwebDebugNodePort = RandomNodePort()
			grpcwebNodePort = RandomNodePort()
			service := &corev1.Service{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							Name:     "peer-api",
							Port:     int32(7051),
							NodePort: peerApiNodePort,
						},
						corev1.ServicePort{
							Name:     "operations",
							Port:     int32(9443),
							NodePort: operationNodePort,
						},
						corev1.ServicePort{
							Name:     "grpcweb-debug",
							Port:     int32(8080),
							NodePort: grpcwebDebugNodePort,
						},
						corev1.ServicePort{
							Name:     "grpcweb",
							Port:     int32(7443),
							NodePort: grpcwebNodePort,
						},
					},
				},
			}
			service.Name = "peermigration-service"
			service.Namespace = namespace

			peerApiNodePort, operationNodePort, grpcwebDebugNodePort, grpcwebNodePort = CreatePeerServiceWithRetry(service, 3)

			ingress := &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						networkingv1.IngressRule{
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										networkingv1.HTTPIngressPath{
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "peermigration-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 443,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			ingress.Name = "peermigration"
			ingress.Namespace = namespace

			err = client.Create(context.TODO(), ingress)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-fluentd-configmap", instance.Name),
					Namespace: instance.Namespace,
				},
			}
			err = client.Create(context.TODO(), cm)
			Expect(err).NotTo(HaveOccurred())

			secretBytes, err := ioutil.ReadFile("../../testdata/migration/secret.json")
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{
				Data: map[string][]byte{"secret.json": secretBytes},
			}
			secret.Name = "peermigration-msp-secret"
			secret.Namespace = namespace

			err = client.Create(context.TODO(), secret)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Name:      secret.Name,
					Namespace: namespace,
				}

				secret := &corev1.Secret{}
				err := client.Get(context.TODO(), namespacedName, secret)
				if err != nil {
					return false
				}
				return true
			}).Should(Equal(true))

			mspSecret = &initsecret.Secret{}
			err = json.Unmarshal(secretBytes, mspSecret)
			Expect(err).NotTo(HaveOccurred())
		})

		It("migrates old MSP secret to new secrets", func() {
			err := migrator.Migrate()
			Expect(err).NotTo(HaveOccurred())

			By("creating a secret with state of current resources before migration", func() {
				var secret *corev1.Secret
				var err error

				Eventually(func() bool {
					secret, err = kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "peermigration-oldstate", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))

				Expect(secret.Data["peermigration-service"]).NotTo(Equal(""))
				Expect(secret.Data["peermigration-cm-fluentd"]).NotTo(Equal(""))
			})

			By("creating ecert ca certs secret", func() {
				Eventually(func() bool {
					_, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-peermigration-cacerts", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("creating ecert keystore secret", func() {
				Eventually(func() bool {
					_, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-peermigration-keystore", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("creating ecert signcert secret", func() {
				Eventually(func() bool {
					_, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-peermigration-signcert", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("creating ecert admin cert secret", func() {
				Eventually(func() bool {
					_, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "ecert-peermigration-admincerts", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("creating tls ca certs secret", func() {
				Eventually(func() bool {
					_, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-peermigration-cacerts", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("creating tls keystore certs secret", func() {
				Eventually(func() bool {
					_, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-peermigration-keystore", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("creating tls signcert secret", func() {
				Eventually(func() bool {
					_, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), "tls-peermigration-signcert", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("creating a new service with no 'service' and same nodeport", func() {
				var service *corev1.Service
				var err error

				Eventually(func() bool {
					service, err = kclient.CoreV1().Services(namespace).Get(context.TODO(), "peermigration", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))

				Expect(service.Spec.Ports[0].NodePort).To(Equal(peerApiNodePort))
				Expect(service.Spec.Ports[1].NodePort).To(Equal(operationNodePort))
				Expect(service.Spec.Ports[2].NodePort).To(Equal(grpcwebDebugNodePort))
				Expect(service.Spec.Ports[3].NodePort).To(Equal(grpcwebNodePort))

				_, err = kclient.CoreV1().Services(namespace).Get(context.TODO(), "peermigration-service", metav1.GetOptions{})
				Expect(err).To(HaveOccurred())
			})

			By("creating a new ingress with no dashes and same servicename", func() {
				var ingress *networkingv1.Ingress
				var err error

				Eventually(func() bool {
					ingress, err = kclient.NetworkingV1().Ingresses(namespace).Get(context.TODO(), "peermigration", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))

				Expect(ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name).To(Equal("peermigration"))
			})
		})
	})

	Context("orderer", func() {
		var (
			migrator  *allmigrator.Migrator
			instance  *current.IBPOrderer
			mspSecret *initsecret.Secret
		)

		BeforeEach(func() {
			logf.SetLogger(zap.New())

			defaultConfigs := "../../defaultconfig"
			of, err := offering.GetType("K8S")

			operatorCfg := &config.Config{
				CAInitConfig: &cainit.Config{
					CADefaultConfigPath:    filepath.Join(defaultConfigs, "ca/ca.yaml"),
					TLSCADefaultConfigPath: filepath.Join(defaultConfigs, "ca/tlsca.yaml"),
					SharedPath:             "/shared",
				},
				PeerInitConfig: &peerinit.Config{
					OUFile: filepath.Join(defaultConfigs, "peer/ouconfig.yaml"),
				},
				OrdererInitConfig: &ordererinit.Config{
					OrdererFile:  filepath.Join(defaultConfigs, "orderer/orderer.yaml"),
					ConfigTxFile: filepath.Join(defaultConfigs, "orderer/configtx.yaml"),
					OUFile:       filepath.Join(defaultConfigs, "orderer/ouconfig.yaml"),
				},
				Offering: of,
			}

			mockValidator := &mocks.CryptoValidator{}
			mockValidator.CheckEcertCryptoReturns(errors.New("not found"))

			migrator = allmigrator.New(mgr, operatorCfg, namespace)

			instance = &current.IBPOrderer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "orderer-migration",
					Namespace: namespace,
				},
				Spec: current.IBPOrdererSpec{
					Domain: "orderer.url",
				},
				Status: current.IBPOrdererStatus{
					CRStatus: current.CRStatus{
						Status: current.True,
					},
				},
			}
			err = client.Create(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())
			err = client.UpdateStatus(context.TODO(), instance)
			Expect(err).NotTo(HaveOccurred())

			secretBytes, err := ioutil.ReadFile("../../testdata/migration/secret.json")
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{
				Data: map[string][]byte{"secret.json": secretBytes},
			}
			secret.Name = "orderer-migration-secret"
			secret.Namespace = namespace

			err = client.Create(context.TODO(), secret)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				namespacedName := types.NamespacedName{
					Name:      secret.Name,
					Namespace: namespace,
				}

				secret := &corev1.Secret{}
				err := client.Get(context.TODO(), namespacedName, secret)
				if err != nil {
					return false
				}
				return true
			}).Should(Equal(true))

			mspSecret = &initsecret.Secret{}
			err = json.Unmarshal(secretBytes, mspSecret)
			Expect(err).NotTo(HaveOccurred())

			configmap := &corev1.ConfigMap{}
			configmap.Name = fmt.Sprintf("%s-env-configmap", instance.GetName())
			configmap.Namespace = namespace

			err = client.Create(context.TODO(), configmap)
			Expect(err).NotTo(HaveOccurred())

			ingress := &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						networkingv1.IngressRule{
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										networkingv1.HTTPIngressPath{
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "camigration-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 443,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			ingress.Name = "orderer-migration"
			ingress.Namespace = namespace

			err = client.Create(context.TODO(), ingress)
			Expect(err).NotTo(HaveOccurred())

			n := types.NamespacedName{
				Name:      ingress.GetName(),
				Namespace: ingress.GetNamespace(),
			}

			err = wait.Poll(500*time.Millisecond, 30*time.Second, func() (bool, error) {
				err := client.Get(context.TODO(), n, ingress)
				if err == nil {
					return true, nil
				}
				return false, nil
			})
			Expect(err).NotTo(HaveOccurred())

		})

		It("generates the configmap", func() {
			err := migrator.Migrate()
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				_, err := kclient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "orderer-migrationnode1-config", metav1.GetOptions{})
				if err != nil {
					return true
				}
				return false
			}).Should(Equal(true))
		})
	})
})

func CreateServiceWithRetry(service *corev1.Service, retryNumber int) (int32, int32) {
	err := client.Create(context.TODO(), service)
	if err != nil {
		if retryNumber == 0 {
			Expect(err).NotTo(HaveOccurred())
		}
		if strings.Contains(err.Error(), "provided port is already allocated") {
			fmt.Fprintf(GinkgoWriter, "encountered port error: %s, trying again\n", err)
			for i, _ := range service.Spec.Ports {
				service.Spec.Ports[i].NodePort = RandomNodePort()
			}
			CreateServiceWithRetry(service, retryNumber-1)
		}
	}

	return service.Spec.Ports[0].NodePort, service.Spec.Ports[1].NodePort
}

func CreatePeerServiceWithRetry(service *corev1.Service, retryNumber int) (int32, int32, int32, int32) {
	err := client.Create(context.TODO(), service)
	if err != nil {
		if retryNumber == 0 {
			Expect(err).NotTo(HaveOccurred())
		}
		if strings.Contains(err.Error(), "provided port is already allocated") {
			fmt.Fprintf(GinkgoWriter, "encountered port error: %s, trying again\n", err)
			for i, _ := range service.Spec.Ports {
				service.Spec.Ports[i].NodePort = RandomNodePort()
			}
			CreatePeerServiceWithRetry(service, retryNumber-1)
		}
	}

	return service.Spec.Ports[0].NodePort, service.Spec.Ports[1].NodePort, service.Spec.Ports[2].NodePort, service.Spec.Ports[3].NodePort
}
