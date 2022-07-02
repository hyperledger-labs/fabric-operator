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

package fabric_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	"github.com/IBM-Blockchain/fabric-operator/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	peerUsername = "peer"
)

var _ = Describe("Fabric peer migration", func() {
	var (
		peer *helper.Peer
	)

	BeforeEach(func() {
		peer = GetPeer()
		err := helper.CreatePeer(ibpCRClient, peer.CR)
		Expect(err).NotTo(HaveOccurred())

		By("starting peer pod", func() {
			Eventually(peer.PodIsRunning).Should((Equal(true)))
		})
	})

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("migration from v1.4.x to v2.x peer", func() {
		BeforeEach(func() {
			result := ibpCRClient.
				Get().
				Namespace(namespace).
				Resource("ibppeers").
				Name(peer.Name).
				Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			ibppeer := &current.IBPPeer{}
			result.Into(ibppeer)

			ibppeer.Spec.Images.PeerTag = integration.PeerTag
			ibppeer.Spec.FabricVersion = version.V2_2_5

			bytes, err := json.Marshal(ibppeer)
			Expect(err).NotTo(HaveOccurred())

			// Update the peer's CR spec
			result = ibpCRClient.
				Put().
				Namespace(namespace).
				Resource("ibppeers").
				Name(ibppeer.Name).
				Body(bytes).
				Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())
		})

		It("migrates", func() {
			By("starting migration job", func() {
				Eventually(func() bool {
					dbmigrationJobName, err := helper.GetJobID(kclient, namespace, fmt.Sprintf("%s-dbmigration", peer.CR.Name))
					if err != nil {
						return false
					}

					_, err = kclient.BatchV1().Jobs(namespace).
						Get(context.TODO(), dbmigrationJobName, metav1.GetOptions{})
					if err != nil {
						return false
					}
					return true
				}).Should(Equal(true))
			})

			By("starting peer pod", func() {
				Eventually(func() int {
					deps := peer.DeploymentList()
					dep := deps.Items[0]
					return len(dep.Spec.Template.Spec.Containers)
				}).Should(Equal(4))
				Eventually(peer.PodIsRunning).Should((Equal(true)))
			})

			By("adding chaincode launcher container and removing dind", func() {
				deps := peer.DeploymentList()
				dep := deps.Items[0]

				containerNames := []string{}
				for _, cont := range dep.Spec.Template.Spec.Containers {
					containerNames = append(containerNames, cont.Name)
				}

				Expect(containerNames).To(ContainElement("chaincode-launcher"))
				Expect(containerNames).NotTo(ContainElement("dind"))
			})
		})
	})
})

// TODO:OSS
func GetPeer() *helper.Peer {
	name := "ibppeer1"
	cr := &current.IBPPeer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IBPPeer",
			APIVersion: "ibp.com/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: current.IBPPeerSpec{
			License: current.License{
				Accept: true,
			},
			MSPID:            "test-peer-mspid",
			Region:           "select",
			Zone:             "select",
			ImagePullSecrets: []string{"regcred"},
			Images: &current.PeerImages{
				// TODO: OSS
				CouchDBImage: "ghcr.io/ibm-blockchain/couchdb",
				CouchDBTag:   "2.3.1-20210826-amd64",
				// do not change dind tag, it is used for loading dind faster
				DindImage:     "ghcr.io/ibm-blockchain/dind",
				DindTag:       "noimages-amd64",
				FluentdImage:  "ghcr.io/ibm-blockchain/fluentd",
				FluentdTag:    "1.0.0-20210826-amd64",
				GRPCWebImage:  "ghcr.io/ibm-blockchain/grpcweb",
				GRPCWebTag:    "1.0.0-20210826-amd64",
				PeerImage:     "ghcr.io/ibm-blockchain/peer",
				PeerTag:       "1.4.12-20210826-amd64",
				PeerInitImage: "ghcr.io/ibm-blockchain/init",
				PeerInitTag:   "1.0.0-20210826-amd64",
				EnrollerImage: "ghcr.io/ibm-blockchain/enroller",
				EnrollerTag:   "1.0.0-20210826-amd64",
			},
			Domain: domain,
			Secret: &current.SecretSpec{
				Enrollment: &current.EnrollmentSpec{
					Component: &current.Enrollment{
						CAHost: caHost,
						CAPort: "443",
						CAName: "ca",
						CATLS: &current.CATLS{
							CACert: base64.StdEncoding.EncodeToString(tlsCert),
						},
						EnrollID:     peerUsername,
						EnrollSecret: "peerpw",
					},
					TLS: &current.Enrollment{
						CAHost: caHost,
						CAPort: "443",
						CAName: "tlsca",
						CATLS: &current.CATLS{
							CACert: base64.StdEncoding.EncodeToString(tlsCert),
						},
						EnrollID:     peerUsername,
						EnrollSecret: "peerpw",
					},
				},
			},
			FabricVersion: "1.4.12",
		},
	}

	return &helper.Peer{
		Domain:    domain,
		Name:      cr.Name,
		Namespace: namespace,
		CR:        cr,
		CRClient:  ibpCRClient,
		KClient:   kclient,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      cr.Name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}
