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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"sigs.k8s.io/yaml"
)

func CreateCA(crClient *ibpclient.IBPClient, ca *current.IBPCA) error {
	result := crClient.Post().Namespace(ca.Namespace).Resource("ibpcas").Body(ca).Do(context.TODO())
	err := result.Error()
	if !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

type CA struct {
	Domain     string
	Name       string
	Namespace  string
	WorkingDir string

	CR       *current.IBPCA
	CRClient *ibpclient.IBPClient
	KClient  *kubernetes.Clientset

	integration.NativeResourcePoller
}

func (ca *CA) PollForCRStatus() current.IBPCRStatusType {
	crStatus := &current.IBPCA{}

	result := ca.CRClient.Get().Namespace(ca.Namespace).Resource("ibpcas").Name(ca.Name).Do(context.TODO())
	// Not handling this because - integration test
	_ = result.Into(crStatus)

	return crStatus.Status.Type
}

func (ca *CA) HealthCheck(url string, cert []byte) bool {
	rootCertPool := x509.NewCertPool()
	rootCertPool.AppendCertsFromPEM(cert)

	transport := http.DefaultTransport
	transport.(*http.Transport).TLSClientConfig = &tls.Config{
		RootCAs:    rootCertPool,
		MinVersion: tls.VersionTLS12, // TLS 1.2 recommended, TLS 1.3 (current latest version) encouraged
	}

	client := http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	_, err := client.Get(url)

	return err == nil
}

func (ca *CA) ConnectionProfile() (*current.CAConnectionProfile, error) {
	cm, err := ca.KClient.CoreV1().ConfigMaps(ca.Namespace).Get(context.TODO(), fmt.Sprintf("%s-connection-profile", ca.CR.Name), metav1.GetOptions{})
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

func (ca *CA) Address() string {
	return fmt.Sprintf("%s-%s-ca.%s", ca.Namespace, ca.Name, ca.Domain)
}

func (ca *CA) Register(name string, secret string, userType string) *exec.Cmd {
	url := fmt.Sprintf("https://%s", ca.Address())
	args := []string{
		"--tls.certfiles", ca.TLSPath(),
		"--id.name", name,
		"--id.secret", secret,
		"--id.type", userType,
		"-u", url,
		"-d",
	}
	return GetCommand(filepath.Join(ca.WorkingDir, "bin/fabric-ca-client register"), args...)
}

func (ca *CA) Enroll(name string, secret string) *exec.Cmd {
	url := fmt.Sprintf("https://%s:%s@%s", name, secret, ca.Address())
	args := []string{
		"--tls.certfiles", ca.TLSPath(),
		"-u", url,
		"-d",
	}
	return GetCommand(filepath.Join(ca.WorkingDir, "bin/fabric-ca-client enroll"), args...)
}

func (ca *CA) DeleteIdentity(name string) *exec.Cmd {
	url := fmt.Sprintf("https://%s", ca.Address())
	args := []string{
		name,
		"--tls.certfiles", ca.TLSPath(),
		"-u", url,
		"-d",
	}
	return GetCommand(filepath.Join(ca.WorkingDir, "bin/fabric-ca-client identity remove"), args...)
}

func (ca *CA) TLSToFile(cert []byte) error {
	err := os.MkdirAll(filepath.Dir(ca.TLSPath()), 0750)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(ca.TLSPath(), cert, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (ca *CA) TLSPath() string {
	return filepath.Join(ca.WorkingDir, ca.Name, "tls-cert.pem")
}

func (ca *CA) JobWithPrefixFound(prefix, namespace string) bool {
	jobs, err := ca.KClient.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
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
