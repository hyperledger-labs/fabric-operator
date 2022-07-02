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

package k8sorderer_test

import (
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	cmocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	ordererinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer"
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/mocks"
	k8sorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/orderer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("K8S Orderer", func() {
	var (
		orderer        *k8sorderer.Orderer
		instance       *current.IBPOrderer
		mockKubeClient *cmocks.Client
		cfg            *config.Config
		update         *mocks.Update
	)

	BeforeEach(func() {
		mockKubeClient = &cmocks.Client{}
		update = &mocks.Update{}
		instance = &current.IBPOrderer{
			Spec: current.IBPOrdererSpec{
				License: current.License{
					Accept: true,
				},
				OrdererType:       "etcdraft",
				SystemChannelName: "testchainid",
				OrgName:           "orderermsp",
				MSPID:             "orderermsp",
				ImagePullSecrets:  []string{"regcred"},
				ClusterSecret:     []*current.SecretSpec{},
				Secret:            &current.SecretSpec{},
				GenesisBlock:      "GenesisBlock",
				Images:            &current.OrdererImages{},
			},
		}
		instance.Kind = "IBPOrderer"

		cfg = &config.Config{
			OrdererInitConfig: &ordererinit.Config{
				ConfigTxFile: "../../../../defaultconfig/orderer/configtx.yaml",
				OUFile:       "../../../../defaultconfig/orderer/ouconfig.yaml",
			},
		}

		orderer = &k8sorderer.Orderer{
			Orderer: &baseorderer.Orderer{
				Client: mockKubeClient,
				Scheme: &runtime.Scheme{},
				Config: cfg,
			},
		}
	})

	Context("Reconciles", func() {
		It("reconciles IBPOrderer", func() {
			_, err := orderer.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
