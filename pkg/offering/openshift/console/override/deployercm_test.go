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

package override_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/console/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("Openshift Console Deployer Config Map Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPConsole
		cm        *corev1.ConfigMap
	)

	BeforeEach(func() {
		var err error
		overrider = &override.Override{}
		instance = &current.IBPConsole{
			Spec: current.IBPConsoleSpec{
				ImagePullSecrets: []string{"pullsecret"},
				ConnectionString: "connectionString1",
				Storage: &current.ConsoleStorage{
					Console: &current.StorageSpec{
						Class: "sc1",
					},
				},
				NetworkInfo: &current.NetworkInfo{
					Domain: "domain1",
				},
				Versions: &current.Versions{
					CA: map[string]current.VersionCA{
						"1.4.6-1": current.VersionCA{
							Default: true,
							Version: "1.4.6-1",
							Image: current.CAImages{
								CAInitImage: "ca-init-image",
								CAInitTag:   "1.4.6",
								CAImage:     "ca-image",
								CATag:       "1.4.6",
							},
						},
					},
					Peer: map[string]current.VersionPeer{
						"1.4.6-1": current.VersionPeer{
							Default: true,
							Version: "1.4.6-1",
							Image: current.PeerImages{
								PeerInitImage:   "peer-init-image",
								PeerInitTag:     "1.4.6",
								PeerImage:       "peer-image",
								PeerTag:         "1.4.6",
								DindImage:       "dind-iamge",
								DindTag:         "1.4.6",
								GRPCWebImage:    "grpcweb-image",
								GRPCWebTag:      "1.4.6",
								FluentdImage:    "fluentd-image",
								FluentdTag:      "1.4.6",
								CouchDBImage:    "couchdb-image",
								CouchDBTag:      "1.4.6",
								CCLauncherImage: "cclauncer-image",
								CCLauncherTag:   "1.4.6",
							},
						},
					},
					Orderer: map[string]current.VersionOrderer{
						"1.4.6-1": current.VersionOrderer{
							Default: true,
							Version: "1.4.6-1",
							Image: current.OrdererImages{
								OrdererInitImage: "orderer-init-image",
								OrdererInitTag:   "1.4.6",
								OrdererImage:     "orderer-image",
								OrdererTag:       "1.4.6",
								GRPCWebImage:     "grpcweb-image",
								GRPCWebTag:       "1.4.6",
							},
						},
					},
				},
				CRN: &current.CRN{
					CName:       "cname",
					CType:       "ctype",
					Location:    "location1",
					Servicename: "Servicename1",
					Version:     "version1",
					AccountID:   "id123",
				},
				Deployer: &current.Deployer{
					ConnectionString: "connectionstring2",
				},
			},
		}
		cm, err = util.GetConfigMapFromFile("../../../../../testdata/deployercm/deployer-configmap.yaml")
		Expect(err).NotTo(HaveOccurred())
	})

	Context("create", func() {
		It("return an error if no image pull secret provided", func() {
			instance.Spec.ImagePullSecrets = nil
			err := overrider.DeployerCM(instance, cm, resources.Create, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no image pull secret provided"))
		})

		It("return an error if no domain provided", func() {
			instance.Spec.NetworkInfo.Domain = ""
			err := overrider.DeployerCM(instance, cm, resources.Create, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("domain not provided"))
		})

		It("overrides values based on spec", func() {
			err := overrider.DeployerCM(instance, cm, resources.Create, nil)
			Expect(err).NotTo(HaveOccurred())

			config := &deployer.Config{}

			err = yaml.Unmarshal([]byte(cm.Data["settings.yaml"]), config)
			Expect(err).NotTo(HaveOccurred())

			By("setting cluster type", func() {
				Expect(config.ClusterType).To(Equal(offering.OPENSHIFT.String()))
			})

			By("setting service type", func() {
				Expect(config.ServiceConfig.Type).To(Equal(corev1.ServiceTypeClusterIP))
			})

			By("setting domain", func() {
				Expect(config.Domain).To(Equal(instance.Spec.NetworkInfo.Domain))
			})

			By("setting image pull secret", func() {
				Expect(config.ImagePullSecrets).To(Equal(instance.Spec.ImagePullSecrets))
			})

			By("setting connection string", func() {
				Expect(config.Database.ConnectionURL).To(Equal(instance.Spec.Deployer.ConnectionString))
			})

			By("setting versions", func() {
				expectedVersions := &current.Versions{
					CA: map[string]current.VersionCA{
						"1.4.6-1": current.VersionCA{
							Default: true,
							Version: "1.4.6-1",
							Image: current.CAImages{
								CAInitImage: "ca-init-image",
								CAInitTag:   "1.4.6",
								CAImage:     "ca-image",
								CATag:       "1.4.6",
							},
						},
					},
					Peer: map[string]current.VersionPeer{
						"1.4.6-1": current.VersionPeer{
							Default: true,
							Version: "1.4.6-1",
							Image: current.PeerImages{
								PeerInitImage:   "peer-init-image",
								PeerInitTag:     "1.4.6",
								PeerImage:       "peer-image",
								PeerTag:         "1.4.6",
								DindImage:       "dind-iamge",
								DindTag:         "1.4.6",
								GRPCWebImage:    "grpcweb-image",
								GRPCWebTag:      "1.4.6",
								FluentdImage:    "fluentd-image",
								FluentdTag:      "1.4.6",
								CouchDBImage:    "couchdb-image",
								CouchDBTag:      "1.4.6",
								CCLauncherImage: "cclauncer-image",
								CCLauncherTag:   "1.4.6",
							},
						},
					},
					Orderer: map[string]current.VersionOrderer{
						"1.4.6-1": current.VersionOrderer{
							Default: true,
							Version: "1.4.6-1",
							Image: current.OrdererImages{
								OrdererInitImage: "orderer-init-image",
								OrdererInitTag:   "1.4.6",
								OrdererImage:     "orderer-image",
								OrdererTag:       "1.4.6",
								GRPCWebImage:     "grpcweb-image",
								GRPCWebTag:       "1.4.6",
							},
						},
					},
				}

				typeConvertedVersions := &current.Versions{}
				util.ConvertSpec(config.Versions, typeConvertedVersions)
				Expect(typeConvertedVersions).To(Equal(expectedVersions))
			})

			By("setting storage class name", func() {
				Expect(config.Defaults.Storage.CA.CA.Class).To(Equal(instance.Spec.Storage.Console.Class))
				Expect(config.Defaults.Storage.Peer.Peer.Class).To(Equal(instance.Spec.Storage.Console.Class))
				Expect(config.Defaults.Storage.Peer.StateDB.Class).To(Equal(instance.Spec.Storage.Console.Class))
				Expect(config.Defaults.Storage.Orderer.Orderer.Class).To(Equal(instance.Spec.Storage.Console.Class))
			})

			By("setting CRN", func() {
				crn := &current.CRN{
					CName:       instance.Spec.CRN.CName,
					CType:       instance.Spec.CRN.CType,
					Location:    instance.Spec.CRN.Location,
					Servicename: instance.Spec.CRN.Servicename,
					Version:     instance.Spec.CRN.Version,
					AccountID:   instance.Spec.CRN.AccountID,
				}
				Expect(config.CRN).To(Equal(crn))
			})
		})
	})

	Context("update", func() {
		It("return an error if no image pull secret provided", func() {
			instance.Spec.ImagePullSecrets = nil
			err := overrider.DeployerCM(instance, cm, resources.Update, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no image pull secret provided"))
		})
	})
})
