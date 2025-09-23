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
	"net/url"
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
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	"k8s.io/client-go/kubernetes"
)

func TestOrderer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Orderer Suite")
}

const (
	ccTarFile = "gocc.tar.gz"

	FabricBinaryVersion   = "2.5.9"
	FabricCABinaryVersion = "1.5.12"

	ordererUsername = "orderer"

	IBPCAS      = "ibpcas"
	IBPORDERERS = "ibporderers"

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

	org1ca  *helper.CA
	orderer *helper.Orderer
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(480 * time.Second)
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

	namespace, kclient, ibpCRClient, err = integration.Setup(GinkgoWriter, cfg, "orderer-actions", pathToRoot)
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

	caURL, err := url.Parse(profile.Endpoints.API)
	Expect(err).NotTo(HaveOccurred())
	caHost = strings.Split(caURL.Host, ":")[0]

	By("enrolling ca admin", func() {
		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, org1ca.Name, "org1ca-admin"))
		sess, err := helper.StartSession(org1ca.Enroll("admin", "adminpw"), "Enroll CA Admin")
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))
	})

	By("registering orderer identity", func() {
		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, org1ca.Name, "org1ca-admin"))
		sess, err := helper.StartSession(org1ca.Register(ordererUsername, "ordererpw", "orderer"), "Register Orderer Identity")
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))

		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, org1ca.Name, "org1ca-admin"))
		sess, err = helper.StartSession(org1ca.Register("orderer2", "ordererpw2", "orderer"), "Register Orderer Identity")
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))
	})

	By("starting Orderer pod", func() {
		orderer = GetOrderer(profile.TLS.Cert, caHost)
		err = helper.CreateOrderer(ibpCRClient, orderer.CR)
		Expect(err).NotTo(HaveOccurred())
	})

	Eventually(orderer.Nodes[0].PodIsRunning).Should((Equal(true)))
}

func downloadBinaries() {
	os.Setenv("FABRIC_VERSION", FabricBinaryVersion)
	os.Setenv("FABRIC_CA_VERSION", FabricCABinaryVersion)
	sess, err := helper.StartSession(
		helper.GetCommand(helper.AbsPath(wd, pathToRoot+"scripts/download_binaries.sh")),
		"Download Binaries",
	)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
}

func cleanupFiles() {
	os.RemoveAll(filepath.Join(wd, Org1CA().Name))
	os.RemoveAll(filepath.Join(wd, GetOrderer("", "").Nodes[0].Name))
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

func GetOrderer(tlsCert, caHost string) *helper.Orderer {
	cr, err := helper.OrdererCR(namespace, domain, ordererUsername, tlsCert, caHost)
	Expect(err).NotTo(HaveOccurred())

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
		helper.Orderer{
			Name:      cr.Name + "node2",
			Namespace: namespace,
			CR:        cr.DeepCopy(),
			NodeName:  fmt.Sprintf("%s%s%d", cr.Name, baseorderer.NODE, 2),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      cr.Name + "node2",
				Namespace: namespace,
				Client:    kclient,
			},
		},
		helper.Orderer{
			Name:      cr.Name + "node3",
			Namespace: namespace,
			CR:        cr.DeepCopy(),
			NodeName:  fmt.Sprintf("%s%s%d", cr.Name, baseorderer.NODE, 3),
			NativeResourcePoller: integration.NativeResourcePoller{
				Name:      cr.Name + "node3",
				Namespace: namespace,
				Client:    kclient,
			},
		},
	}

	nodes[0].CR.ObjectMeta.Name = cr.Name + "node1"
	nodes[1].CR.ObjectMeta.Name = cr.Name + "node2"
	nodes[2].CR.ObjectMeta.Name = cr.Name + "node3"

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
		Nodes:    nodes,
		CRClient: ibpCRClient,
	}
}
