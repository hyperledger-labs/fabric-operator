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

package console_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/IBM-Blockchain/fabric-operator/integration"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

func TestConsole(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Console Suite")
}

var (
	namespace   string
	kclient     *kubernetes.Clientset
	ibpCRClient *ibpclient.IBPClient
	testFailed  bool
)

var _ = BeforeSuite(func() {
	var err error

	cfg := &integration.Config{
		OperatorServiceAccount: "../../config/rbac/service_account.yaml",
		OperatorRole:           "../../config/rbac/role.yaml",
		OperatorRoleBinding:    "../../config/rbac/role_binding.yaml",
		OperatorDeployment:     "../../testdata/deploy/operator.yaml",
		OrdererSecret:          "../../testdata/deploy/orderer/secret.yaml",
		PeerSecret:             "../../testdata/deploy/peer/secret.yaml",
		ConsoleTLSSecret:       "../../testdata/deploy/console/tlssecret.yaml",
	}

	namespace, kclient, ibpCRClient, err = integration.Setup(GinkgoWriter, cfg, "console", "")
	Expect(err).NotTo(HaveOccurred())

	console = GetConsole()
	result := ibpCRClient.Post().Namespace(namespace).Resource("ibpconsoles").Body(console.CR).Do(context.TODO())
	err = result.Error()
	if !k8serrors.IsAlreadyExists(err) {
		Expect(result.Error()).NotTo(HaveOccurred())
	}

	// Disabled as it consumes too many resources on the GHA executor to reliably launch console1
	//console2 = GetConsole2()
	//result = ibpCRClient.Post().Namespace(namespace).Resource("ibpconsoles").Body(console2.CR).Do(context.TODO())
	//err = result.Error()
	//if !k8serrors.IsAlreadyExists(err) {
	//	Expect(err).NotTo(HaveOccurred())
	//}

	console3 = GetConsole3()
	result = ibpCRClient.Post().Namespace(namespace).Resource("ibpconsoles").Body(console3.CR).Do(context.TODO())
	err = result.Error()
	if !k8serrors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}
})

var _ = AfterSuite(func() {

	if strings.ToLower(os.Getenv("SAVE_TEST")) == "true" {
		return
	}
	err := integration.Cleanup(GinkgoWriter, kclient, namespace)
	Expect(err).NotTo(HaveOccurred())
})
