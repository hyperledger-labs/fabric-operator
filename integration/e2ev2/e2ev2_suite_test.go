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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"

	"k8s.io/client-go/kubernetes"
)

func TestE2ev2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2ev2 Suite")
}

const (
	ccTarFile = "gocc.tar.gz"

	FabricBinaryVersion   = "2.2.3"
	FabricCABinaryVersion = "1.5.1"

	peerAdminUsername = "peer-admin"
	peerUsername      = "peer"
	ordererUsername   = "orderer"
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
		OperatorServiceAccount: "../../config/rbac/service_account.yaml",
		OperatorRole:           "../../config/rbac/role.yaml",
		OperatorRoleBinding:    "../../config/rbac/role_binding.yaml",
		OperatorDeployment:     "../../testdata/deploy/operator.yaml",
		OrdererSecret:          "../../testdata/deploy/orderer/secret.yaml",
		PeerSecret:             "../../testdata/deploy/peer/secret.yaml",
		ConsoleTLSSecret:       "../../testdata/deploy/console/tlssecret.yaml",
	}

	namespace, kclient, ibpCRClient, err = integration.Setup(GinkgoWriter, cfg, "e2ev2", "")
	Expect(err).NotTo(HaveOccurred())

	downloadBinaries()

	SetupConsole()
	CreateNetwork()
})

var _ = AfterSuite(func() {

	if strings.ToLower(os.Getenv("SAVE_TEST")) == "true" {
		return
	}

	integration.Cleanup(GinkgoWriter, kclient, namespace)
})

func SetupConsole() {
	console = GetConsole()
	CreateConsole(console)
}

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

	By("registering peer identity", func() {
		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, org1ca.Name, "org1ca-admin"))
		sess, err := helper.StartSession(org1ca.Register(peerUsername, "peerpw", "peer"), "Register User")
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))

		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, org1ca.Name, "org1ca-admin"))
		sess, err = helper.StartSession(org1ca.Register("peer2", "peerpw2", "peer"), "Register User")
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))
	})

	By("registering and enrolling peer admin", func() {
		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, org1ca.Name, "org1ca-admin"))
		sess, err := helper.StartSession(org1ca.Register(peerAdminUsername, "peer-adminpw", "admin"), "Register Peer Admin")
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))

		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, "org1peer", peerAdminUsername))
		sess, err = helper.StartSession(org1ca.Enroll(peerAdminUsername, "peer-adminpw"), "Enroll Peer Admin")
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))

		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, "org1peer", peerAdminUsername+"2"))
		sess, err = helper.StartSession(org1ca.Enroll(peerAdminUsername, "peer-adminpw"), "Enroll Second Peer Admin")
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

	adminCertBytes, err := ioutil.ReadFile(
		filepath.Join(
			wd,
			"org1peer",
			peerAdminUsername,
			"msp",
			"signcerts",
			"cert.pem",
		),
	)
	Expect(err).NotTo(HaveOccurred())
	adminCertB64 := base64.StdEncoding.EncodeToString(adminCertBytes)
	tlsCert := base64.StdEncoding.EncodeToString(tlsBytes)

	By("starting Peer pod", func() {
		org1peer = Org1Peer(tlsCert, caHost, adminCertB64)
		err = helper.CreatePeer(ibpCRClient, org1peer.CR)
		Expect(err).NotTo(HaveOccurred())
	})

	By("starting Orderer pod", func() {
		orderer = GetOrderer(tlsCert, caHost)
		err = helper.CreateOrderer(ibpCRClient, orderer.CR)
		Expect(err).NotTo(HaveOccurred())
	})

	Eventually(org1peer.PodIsRunning).Should((Equal(true)))
	Eventually(orderer.Nodes[0].PodIsRunning).Should((Equal(true)))
}

func downloadBinaries() {
	os.Setenv("FABRIC_VERSION", FabricBinaryVersion)
	os.Setenv("FABRIC_CA_VERSION", FabricCABinaryVersion)
	sess, err := helper.StartSession(
		helper.GetCommand(helper.AbsPath(wd, "../../scripts/download_binaries.sh")),
		"Download Binaries",
	)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
}

func cleanupFiles() {
	os.RemoveAll(filepath.Join(wd, Org1CA().Name))
	os.RemoveAll(filepath.Join(wd, Org1Peer("", "", "").Name))
	os.RemoveAll(filepath.Join(wd, GetOrderer("", "").Nodes[0].Name))
	os.RemoveAll(filepath.Join(wd, ccTarFile))
}

func CopyFile(from string, to string) {
	bytes, err := ioutil.ReadFile(from)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(to, bytes, 0644)
	Expect(err).NotTo(HaveOccurred())
}
