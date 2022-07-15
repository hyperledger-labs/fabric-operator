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

package e2ev2_test

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	console *Console
)

var _ = Describe("console", func() {
	BeforeEach(func() {
		Eventually(console.PodIsRunning).Should((Equal(true)))
	})

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("trigger actions", func() {
		var (
			podName    string
			ibpconsole *current.IBPConsole
		)

		BeforeEach(func() {
			Eventually(func() int {
				return len(console.GetPods())
			}).Should(Equal(1))

			podName = console.GetPods()[0].Name

			result := ibpCRClient.Get().Namespace(namespace).Resource("ibpconsoles").Name(console.Name).Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			ibpconsole = &current.IBPConsole{}
			result.Into(ibpconsole)
		})

		When("spec has restart flag set to true", func() {
			BeforeEach(func() {
				ibpconsole.Spec.Action.Restart = true
			})

			It("performs restart action", func() {
				bytes, err := json.Marshal(ibpconsole)
				Expect(err).NotTo(HaveOccurred())

				result := ibpCRClient.Put().Namespace(namespace).Resource("ibpconsoles").Name(console.Name).Body(bytes).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				Eventually(console.PodIsRunning).Should((Equal(true)))

				By("restarting console pod", func() {
					Eventually(func() bool {
						pods := console.GetPods()
						if len(pods) == 0 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName != podName {
							return true
						}

						return false
					}).Should(Equal(true))
				})

				By("setting restart flag back to false after restart", func() {
					Eventually(func() bool {
						result := ibpCRClient.Get().Namespace(namespace).Resource("ibpconsoles").Name(console.Name).Do(context.TODO())
						console := &current.IBPConsole{}
						result.Into(console)

						return console.Spec.Action.Restart
					}).Should(Equal(false))
				})
			})
		})
	})

})

func CreateConsole(console *Console) {
	result := ibpCRClient.Post().Namespace(namespace).Resource("ibpconsoles").Body(console.CR).Do(context.TODO())
	err := result.Error()
	if !k8serrors.IsAlreadyExists(err) {
		Expect(result.Error()).NotTo(HaveOccurred())
	}
}

func GetConsole() *Console {
	consolePort := randNum(30000, 32768)
	proxyPort := randNum(30000, 32768)

	useTagsFlag := true

	cr := &current.IBPConsole{
		TypeMeta: metav1.TypeMeta{
			Kind: "IBPConsole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ibpconsole",
			Namespace: namespace,
		},
		Spec: current.IBPConsoleSpec{
			License: current.License{
				Accept: true,
			},
			ConnectionString:   "http://localhost:5984",
			ServiceAccountName: "ibpconsole1",
			NetworkInfo: &current.NetworkInfo{
				Domain:      "test-domain",
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

	return &Console{
		Name: cr.Name,
		CR:   cr,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      cr.Name,
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

func randNum(min, max int) int32 {
	rand.Seed(time.Now().UnixNano())
	return int32(rand.Intn(max-min+1) + min)
}
