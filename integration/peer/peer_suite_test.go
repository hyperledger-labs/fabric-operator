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

package peer_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestPeer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Peer Suite")
}

const (
	FabricBinaryVersion   = "2.2.3"
	FabricCABinaryVersion = "1.5.1"
	peerAdminUsername     = "peer-admin"
	peerUsername          = "peer"
)

var (
	namespaceSuffix        = "peer"
	operatorDeploymentFile = "../../testdata/deploy/operator.yaml"

	namespace   string
	kclient     *kubernetes.Clientset
	ibpCRClient *ibpclient.IBPClient
	testFailed  bool
	wd          string
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(240 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	var err error

	wd, err = os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	fmt.Fprintf(GinkgoWriter, "Working directory: %s\n", wd)

	cfg := &integration.Config{
		OperatorDeployment:     operatorDeploymentFile,
		OperatorServiceAccount: "../../config/rbac/service_account.yaml",
		OperatorRole:           "../../config/rbac/role.yaml",
		OperatorRoleBinding:    "../../config/rbac/role_binding.yaml",
		OrdererSecret:          "../../testdata/deploy/orderer/secret.yaml",
		PeerSecret:             "../../testdata/deploy/peer/secret.yaml",
		ConsoleTLSSecret:       "../../testdata/deploy/console/tlssecret.yaml",
	}

	namespace, kclient, ibpCRClient, err = integration.Setup(GinkgoWriter, cfg, namespaceSuffix, "")
	Expect(err).NotTo(HaveOccurred())

	downloadBinaries()
})

var _ = AfterSuite(func() {

	if strings.ToLower(os.Getenv("SAVE_TEST")) == "true" {
		return
	}

	err := integration.Cleanup(GinkgoWriter, kclient, namespace)
	Expect(err).NotTo(HaveOccurred())
})

func CreatePeer(peer *Peer) {
	result := ibpCRClient.Post().Namespace(namespace).Resource("ibppeers").Body(peer.CR).Do(context.TODO())
	err := result.Error()
	if !k8serrors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

type Peer struct {
	Name string
	CR   *current.IBPPeer
	integration.NativeResourcePoller
}

func (peer *Peer) pollForCRStatus() current.IBPCRStatusType {
	crStatus := &current.IBPPeer{}

	result := ibpCRClient.Get().Namespace(namespace).Resource("ibppeers").Name(peer.Name).Do(context.TODO())
	result.Into(crStatus)

	return crStatus.Status.Type
}

func (peer *Peer) ingressExists() bool {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", peer.Name),
	}
	ingressList, err := kclient.NetworkingV1().Ingresses(namespace).List(context.TODO(), opts)
	if err != nil {
		return false
	}
	for _, ingress := range ingressList.Items {
		if strings.HasPrefix(ingress.Name, peer.Name) {
			return true
		}
	}

	return false
}

func (peer *Peer) getPVCStorageFromSpec(name string) string {
	pvc, err := kclient.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return ""
	}

	storage := pvc.Spec.Resources.Requests[corev1.ResourceStorage]

	return storage.String()
}

func (peer *Peer) checkAdminCertUpdate() string {
	secretName := fmt.Sprintf("%s-%s-%s", "ecert", peer.Name, "admincerts")
	sec, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	certBytes := sec.Data["admincert-0.pem"]
	str := base64.StdEncoding.EncodeToString(certBytes)
	return str
}

func downloadBinaries() {
	os.Setenv("FABRIC_VERSION", FabricBinaryVersion)
	os.Setenv("FABRIC_CA_VERSION", FabricCABinaryVersion)
	sess, err := helper.StartSession(helper.GetCommand(filepath.Join(wd, "../../scripts/download_binaries.sh")), "Download Binaries")
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
}
