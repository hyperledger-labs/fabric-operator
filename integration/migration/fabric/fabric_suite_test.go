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

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestFabric(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fabric Suite")
}

const (
	defaultConfigs        = "../../../defaultconfig"
	defaultPeerDef        = "../../../definitions/peer"
	defaultCADef          = "../../../definitions/ca"
	defaultOrdererDef     = "../../../definitions/orderer"
	defaultConsoleDef     = "../../../definitions/console"
	FabricBinaryVersion   = "2.2.3"
	FabricCABinaryVersion = "1.5.1"
	domain                = "vcap.me"
)

var (
	namespaceSuffix = "migration"

	namespace   string
	kclient     *kubernetes.Clientset
	ibpCRClient *ibpclient.IBPClient
	testFailed  bool
	wd          string // Working directory of test
)

var (
	err error

	org1ca  *helper.CA
	caHost  string
	tlsCert []byte
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(300 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	wd, err = os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	fmt.Fprintf(GinkgoWriter, "Working directory: %s\n", wd)

	cfg := &integration.Config{
		OperatorServiceAccount: "../../../config/rbac/service_account.yaml",
		OperatorRole:           "../../../config/rbac/role.yaml",
		OperatorRoleBinding:    "../../../config/rbac/role_binding.yaml",
		OperatorDeployment:     "../../../testdata/deploy/operator.yaml",
		OrdererSecret:          "../../../testdata/deploy/orderer/secret.yaml",
		PeerSecret:             "../../../testdata/deploy/peer/secret.yaml",
		ConsoleTLSSecret:       "../../../testdata/deploy/console/tlssecret.yaml",
	}

	namespace, kclient, ibpCRClient, err = integration.Setup(GinkgoWriter, cfg, namespaceSuffix, "../../..")
	Expect(err).NotTo(HaveOccurred())

	downloadBinaries()
	startCA()
	registerAndEnrollIDs()
})

var _ = AfterSuite(func() {

	if strings.ToLower(os.Getenv("SAVE_TEST")) == "true" {
		return
	}

	integration.Cleanup(GinkgoWriter, kclient, namespace)
})

func startCA() {
	By("starting CA pod", func() {
		org1ca = Org1CA()
		helper.CreateCA(ibpCRClient, org1ca.CR)

		Eventually(org1ca.PodIsRunning).Should((Equal(true)))
	})

	profile, err := org1ca.ConnectionProfile()
	Expect(err).NotTo(HaveOccurred())

	tlsCert, err = util.Base64ToBytes(profile.TLS.Cert)
	Expect(err).NotTo(HaveOccurred())

	By("performing CA health check", func() {
		Eventually(func() bool {
			url := fmt.Sprintf("https://%s/cainfo", org1ca.Address())
			fmt.Fprintf(GinkgoWriter, "Waiting for CA health check to pass for '%s' at url: %s\n", org1ca.Name, url)
			return org1ca.HealthCheck(url, tlsCert)
		}).Should(Equal(true))
	})

	org1ca.TLSToFile(tlsCert)

	caURL, err := url.Parse(profile.Endpoints.API)
	Expect(err).NotTo(HaveOccurred())
	caHost = strings.Split(caURL.Host, ":")[0]
}

func registerAndEnrollIDs() {
	By("enrolling ca admin", func() {
		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, org1ca.Name, "org1ca-admin"))
		sess, err := helper.StartSession(
			org1ca.Enroll("admin", "adminpw"),
			"Enroll CA Admin",
		)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))
	})

	By("registering peer identity", func() {
		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, org1ca.Name, "org1ca-admin"))
		sess, err := helper.StartSession(
			org1ca.Register(peerUsername, "peerpw", "peer"),
			"Register User",
		)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))
	})

	By("registering orderer identity", func() {
		os.Setenv("FABRIC_CA_CLIENT_HOME", filepath.Join(wd, org1ca.Name, "org1ca-admin"))
		sess, err := helper.StartSession(
			org1ca.Register(ordererUsername, "ordererpw", "orderer"),
			"Register User",
		)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))
	})
}

func downloadBinaries() {
	os.Setenv("FABRIC_VERSION", FabricBinaryVersion)
	os.Setenv("FABRIC_CA_VERSION", FabricCABinaryVersion)
	sess, err := helper.StartSession(
		helper.GetCommand(helper.AbsPath(wd, "../../../scripts/download_binaries.sh")),
		"Download Binaries",
	)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
}

func Org1CA() *helper.CA {
	cr := &current.IBPCA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "org1ca",
			Namespace: namespace,
		},
		Spec: current.IBPCASpec{
			License: current.License{
				Accept: true,
			},
			ImagePullSecrets: []string{"regcred"},
			Images: &current.CAImages{
				CAImage:     integration.CaImage,
				CATag:       integration.CaTag,
				CAInitImage: integration.InitImage,
				CAInitTag:   integration.InitTag,
			},
			Resources: &current.CAResources{
				CA: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("50m"),
						corev1.ResourceMemory:           resource.MustParse("100M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("50m"),
						corev1.ResourceMemory:           resource.MustParse("100M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					},
				},
			},
			Zone:          "select",
			Region:        "select",
			Domain:        domain,
			FabricVersion: integration.FabricCAVersion,
		},
	}

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
