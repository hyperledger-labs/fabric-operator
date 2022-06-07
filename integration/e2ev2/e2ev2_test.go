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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/gomega"

	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/onsi/gomega/gexec"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	org1peer  *helper.Peer
	orderer   *helper.Orderer
	peeradmin *PeerAdmin
)

type PeerAdmin struct {
	Envs []string
}

func NewPeerAdminSession(org1peer *helper.Peer, tlsRootCertPath string, address string) *PeerAdmin {
	peerHome := filepath.Join(wd, org1peer.Name)

	CopyFile("./config/core.yaml", filepath.Join(peerHome, "core.yaml"))
	CopyFile("./config.yaml", filepath.Join(peerHome, peerAdminUsername, "/msp/config.yaml"))

	envs := []string{
		fmt.Sprintf("FABRIC_CFG_PATH=%s", peerHome),
		fmt.Sprintf("CORE_PEER_TLS_ENABLED=%s", "true"),
		fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", org1peer.CR.Spec.MSPID),
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=%s", tlsRootCertPath),
		fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=%s", filepath.Join(wd, org1peer.Name, peerAdminUsername, "msp")),
		fmt.Sprintf("CORE_PEER_ADDRESS=%s", address),
	}

	envs = append(envs, os.Environ()...)

	return &PeerAdmin{
		Envs: envs,
	}
}

type Chaincode struct {
	Path       string
	Lang       string
	Label      string
	OutputFile string
}

func (p *PeerAdmin) PackageChaincode(c Chaincode) {
	args := []string{
		"lifecycle", "chaincode", "package",
		"--path", c.Path,
		"--lang", c.Lang,
		"--label", c.Label,
		c.OutputFile,
	}

	cmd := helper.GetCommand(helper.AbsPath(wd, "bin/peer"), args...)
	cmd.Env = p.Envs

	sess, err := helper.StartSession(cmd, "Package Chaincode")
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
}

func (p *PeerAdmin) InstallChaincode(packageFile string) {
	args := []string{
		"lifecycle", "chaincode", "install",
		packageFile,
	}

	cmd := helper.GetCommand(helper.AbsPath(wd, "bin/peer"), args...)
	cmd.Env = p.Envs
	cmd.Env = append(cmd.Env, fmt.Sprintf("FABRIC_LOGGING_SPEC=%s", "debug"))

	sess, err := helper.StartSession(cmd, "Install Chaincode")
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
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

func Org1Peer(tlsCert, caHost, adminCert string) *helper.Peer {
	cr, err := helper.Org1PeerCR(namespace, domain, peerUsername, tlsCert, caHost, adminCert)
	Expect(err).NotTo(HaveOccurred())

	return &helper.Peer{
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

func ClearOperatorConfig() {
	err := kclient.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), "operator-config", *metav1.NewDeleteOptions(0))
	if !k8serrors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}
}

func GetBackup(certType, name string) *common.Backup {
	backupSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("%s-crypto-backup", name), metav1.GetOptions{})
	if err != nil {
		Expect(k8serrors.IsNotFound(err)).To(Equal(true))
		return &common.Backup{}
	}

	backup := &common.Backup{}
	key := fmt.Sprintf("%s-backup.json", certType)
	err = json.Unmarshal(backupSecret.Data[key], backup)
	Expect(err).NotTo(HaveOccurred())

	return backup
}

func TLSSignCert(name string) []byte {
	secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-signcert", name), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	return secret.Data["cert.pem"]
}

func TLSKeystore(name string) []byte {
	secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("tls-%s-keystore", name), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	return secret.Data["key.pem"]
}

func EcertSignCert(name string) []byte {
	secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-signcert", name), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	return secret.Data["cert.pem"]
}

func EcertKeystore(name string) []byte {
	secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-keystore", name), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	return secret.Data["key.pem"]
}

func EcertCACert(name string) []byte {
	secret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("ecert-%s-cacerts", name), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	return secret.Data["cacert-0.pem"]
}
