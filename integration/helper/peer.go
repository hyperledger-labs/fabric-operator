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

package helper

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"sigs.k8s.io/yaml"
)

func CreatePeer(crClient *ibpclient.IBPClient, peer *current.IBPPeer) error {
	result := crClient.Post().Namespace(peer.Namespace).Resource("ibppeers").Body(peer).Do(context.TODO())
	err := result.Error()
	if !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

type Peer struct {
	Domain     string
	Name       string
	Namespace  string
	WorkingDir string

	CR       *current.IBPPeer
	CRClient *ibpclient.IBPClient
	KClient  *kubernetes.Clientset

	integration.NativeResourcePoller
}

func (p *Peer) PollForCRStatus() current.IBPCRStatusType {
	crStatus := &current.IBPPeer{}

	result := p.CRClient.Get().Namespace(p.Namespace).Resource("ibppeers").Name(p.Name).Do(context.TODO())
	// Not handling this as this is integration test
	_ = result.Into(crStatus)

	return crStatus.Status.Type
}

func (p *Peer) TLSToFile(cert []byte) error {
	err := os.MkdirAll(filepath.Dir(p.TLSPath()), 0750)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(p.TLSPath(), cert, 0600)
}

func (p *Peer) TLSPath() string {
	return filepath.Join(p.WorkingDir, p.Name, "tls-cert.pem")
}

func (p *Peer) ConnectionProfile() (*current.CAConnectionProfile, error) {
	cm, err := p.KClient.CoreV1().ConfigMaps(p.Namespace).Get(context.TODO(), fmt.Sprintf("%s-connection-profile", p.CR.Name), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	data := cm.BinaryData["profile.json"]

	profile := &current.CAConnectionProfile{}
	err = yaml.Unmarshal(data, profile)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (p *Peer) JobWithPrefixFound(prefix, namespace string) bool {
	jobs, err := p.KClient.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false
	}

	for _, job := range jobs.Items {
		if strings.HasPrefix(job.GetName(), prefix) {
			return true
		}
	}

	return false
}
