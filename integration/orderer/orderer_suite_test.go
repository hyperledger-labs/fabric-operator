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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"k8s.io/client-go/kubernetes"
)

func TestOrderer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Orderer Suite")
}

const (
	FabricBinaryVersion   = "2.2.3"
	FabricCABinaryVersion = "1.5.1"
	ordererUsername       = "orderer"
	ordererPassword       = "orderer"
)

var (
	namespaceSuffix = "orderer"

	namespace   string
	kclient     *kubernetes.Clientset
	ibpCRClient *ibpclient.IBPClient
	testFailed  bool
	wd          string
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(300 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	var err error

	wd, err = os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	fmt.Fprintf(GinkgoWriter, "Working directory: %s\n", wd)

	cfg := &integration.Config{
		OperatorServiceAccount: "../../config/rbac/service_account.yaml",
		OperatorRole:           "../../config/rbac/role.yaml",
		OperatorRoleBinding:    "../../config/rbac/role_binding.yaml",
		OperatorDeployment:     "../../testdata/deploy/operator.yaml",
		OrdererSecret:          "../../testdata/deploy/orderer/secret.yaml",
		PeerSecret:             "../../testdata/deploy/peer/secret.yaml",
		ConsoleTLSSecret:       "../../testdata/deploy/console/tlssecret.yaml",
	}

	namespace, kclient, ibpCRClient, err = integration.Setup(GinkgoWriter, cfg, namespaceSuffix, "")
	Expect(err).NotTo(HaveOccurred())

})

var _ = AfterSuite(func() {

	if strings.ToLower(os.Getenv("SAVE_TEST")) == "true" {
		return
	}

	if strings.ToLower(os.Getenv("SAVE_TEST")) == "true" {
		return
	}

	err := integration.Cleanup(GinkgoWriter, kclient, namespace)
	Expect(err).NotTo(HaveOccurred())
})

func downloadBinaries() {
	os.Setenv("FABRIC_VERSION", FabricBinaryVersion)
	os.Setenv("FABRIC_CA_VERSION", FabricCABinaryVersion)
	sess, err := helper.StartSession(helper.GetCommand(filepath.Join(wd, "../../scripts/download_binaries.sh")), "Download Binaries")
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
}
