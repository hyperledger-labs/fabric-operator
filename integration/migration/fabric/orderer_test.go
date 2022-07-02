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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ordererUsername = "orderer"
)

var _ = Describe("Fabric orderer migration", func() {
	var (
		node1 *helper.Orderer
	)

	BeforeEach(func() {
		orderer := GetOrderer()
		err := helper.CreateOrderer(ibpCRClient, orderer.CR)
		Expect(err).NotTo(HaveOccurred())

		node1 = &orderer.Nodes[0]

		By("starting orderer pod", func() {
			Eventually(node1.PodIsRunning).Should((Equal(true)))
		})
	})

	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("migration from v1.4.x to v2.x", func() {
		BeforeEach(func() {
			result := ibpCRClient.
				Get().
				Namespace(namespace).
				Resource("ibporderers").
				Name(node1.Name).
				Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			ibporderer := &current.IBPOrderer{}
			result.Into(ibporderer)

			ibporderer.Spec.Images.OrdererTag = integration.OrdererTag
			ibporderer.Spec.FabricVersion = integration.FabricVersion

			bytes, err := json.Marshal(ibporderer)
			Expect(err).NotTo(HaveOccurred())

			// Update the orderer's CR spec
			result = ibpCRClient.
				Put().
				Namespace(namespace).
				Resource("ibporderers").
				Name(node1.Name).
				Body(bytes).
				Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())
		})

		It("terminates pod", func() {
			Eventually(func() int {
				return len(node1.GetRunningPods())
			}).Should((Equal(0)))
		})

		It("restarts pod", func() {
			Eventually(node1.PodIsRunning).Should((Equal(true)))
		})
	})
})

func GetOrderer() *helper.Orderer {
	enrollment := &current.EnrollmentSpec{
		Component: &current.Enrollment{
			CAHost: caHost,
			CAPort: "443",
			CAName: "ca",
			CATLS: &current.CATLS{
				CACert: base64.StdEncoding.EncodeToString(tlsCert),
			},
			EnrollID:     ordererUsername,
			EnrollSecret: "ordererpw",
		},
		TLS: &current.Enrollment{
			CAHost: caHost,
			CAPort: "443",
			CAName: "tlsca",
			CATLS: &current.CATLS{
				CACert: base64.StdEncoding.EncodeToString(tlsCert),
			},
			EnrollID:     ordererUsername,
			EnrollSecret: "ordererpw",
		},
	}

	cr := &current.IBPOrderer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ibporderer1",
			Namespace: namespace,
		},
		Spec: current.IBPOrdererSpec{
			License: current.License{
				Accept: true,
			},
			ClusterSize:       1,
			OrdererType:       "etcdraft",
			SystemChannelName: "testchainid",
			OrgName:           "orderermsp",
			MSPID:             "orderermsp",
			ImagePullSecrets:  []string{"regcred"},
			GenesisProfile:    "Initial",
			Domain:            domain,
			Images: &current.OrdererImages{
				GRPCWebImage:     integration.GrpcwebImage,
				GRPCWebTag:       integration.GrpcwebTag,
				OrdererImage:     integration.OrdererImage,
				OrdererTag:       integration.Orderer14Tag,
				OrdererInitImage: integration.InitImage,
				OrdererInitTag:   integration.InitTag,
			},
			ClusterSecret: []*current.SecretSpec{
				&current.SecretSpec{
					Enrollment: enrollment,
				},
			},
			FabricVersion: "1.4.12",
		},
	}

	nodes := []helper.Orderer{
		helper.Orderer{
			Name:      cr.Name + "node1",
			Namespace: namespace,
			CR:        cr.DeepCopy(),
			NodeName:  fmt.Sprintf("%s%s%d", cr.Name, baseorderer.NODE, 1),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      cr.Name + "node1",
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}

	nodes[0].CR.ObjectMeta.Name = cr.Name + "node1"

	return &helper.Orderer{
		Name:      cr.Name,
		Namespace: namespace,
		CR:        cr,
		NodeName:  fmt.Sprintf("%s-%s%d", cr.Name, baseorderer.NODE, 1),
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      cr.Name,
			Namespace: namespace,
			Client:    kclient,
		},
		Nodes: nodes,
	}
}
