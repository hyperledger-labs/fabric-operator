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

package ca_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	"k8s.io/client-go/kubernetes"
)

func TestCa(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ca Suite")
}

const (
	ccTarFile = "gocc.tar.gz"

	FabricBinaryVersion   = "2.2.3"
	FabricCABinaryVersion = "1.5.1"

	IBPCAS = "ibpcas"

	pathToRoot = "../../../"
)

var (
	wd          string // Working directory of test
	namespace   string
	domain      string
	kclient     *kubernetes.Clientset
	ibpCRClient *ibpclient.IBPClient
	colorIndex  uint
	testFailed  bool
	caHost      string
	tlsBytes    []byte

	org1ca *helper.CA
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(420 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	var err error
	domain = os.Getenv("DOMAIN")
	if domain == "" {
		domain = integration.TestAutomation1IngressDomain
	}

	wd, err = os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	fmt.Fprintf(GinkgoWriter, "Working directory: %s\n", wd)

	cleanupFiles()

	cfg := &integration.Config{
		OperatorServiceAccount: "../../../config/rbac/service_account.yaml",
		OperatorRole:           "../../../config/rbac/role.yaml",
		OperatorRoleBinding:    "../../../config/rbac/role_binding.yaml",
		OperatorDeployment:     "../../../testdata/deploy/operator.yaml",
		OrdererSecret:          "../../../testdata/deploy/orderer/secret.yaml",
		PeerSecret:             "../../../testdata/deploy/peer/secret.yaml",
		ConsoleTLSSecret:       "../../../testdata/deploy/console/tlssecret.yaml",
	}

	namespace, kclient, ibpCRClient, err = integration.Setup(GinkgoWriter, cfg, "ca-actions", pathToRoot)
	Expect(err).NotTo(HaveOccurred())

	downloadBinaries()

	CreateNetwork()
})

var _ = AfterSuite(func() {

	if strings.ToLower(os.Getenv("SAVE_TEST")) == "true" {
		return
	}

	integration.Cleanup(GinkgoWriter, kclient, namespace)
	cleanupFiles()
})

func CreateNetwork() {
	By("starting CA pod", func() {
		org1ca = Org1CA()
		helper.CreateCA(ibpCRClient, org1ca.CR)

		Eventually(org1ca.PodIsRunning).Should((Equal(true)))
	})

	profile, err := org1ca.ConnectionProfile()
	Expect(err).NotTo(HaveOccurred())

	tlsBytes, err = util.Base64ToBytes(profile.TLS.Cert)
	Expect(err).NotTo(HaveOccurred())

	By("performing CA health check", func() {
		Eventually(func() bool {
			url := fmt.Sprintf("https://%s/cainfo", org1ca.Address())
			fmt.Fprintf(GinkgoWriter, "Waiting for CA health check to pass for '%s' at url: %s\n", org1ca.Name, url)
			return org1ca.HealthCheck(url, tlsBytes)
		}).Should(Equal(true))
	})

	org1ca.TLSToFile(tlsBytes)
}

func downloadBinaries() {
	os.Setenv("FABRIC_VERSION", FabricBinaryVersion)
	os.Setenv("FABRIC_CA_VERSION", FabricCABinaryVersion)
	path := pathToRoot + "scripts/download_binaries.sh"
	sess, err := helper.StartSession(
		helper.GetCommand(helper.AbsPath(wd, path)),
		"Download Binaries",
	)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
}

func cleanupFiles() {
	os.RemoveAll(filepath.Join(wd, Org1CA().Name))
	os.RemoveAll(filepath.Join(wd, ccTarFile))
}

func Org1CA() *helper.CA {
	cr := helper.Org1CACR(namespace, domain)

	return &helper.CA{
		Domain:     domain,
		Name:       cr.Name,
		Namespace:  namespace,
		WorkingDir: wd,
		CR:         cr,
		CRClient:   ibpCRClient,
		KClient:    kclient,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      cr.Name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}
