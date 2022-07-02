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

package reenroller_test

import (
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate/reenroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate/reenroller/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReenroller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reenroller Suite")
}

var (
	err error

	testReenroller *reenroller.Reenroller
	config         *current.Enrollment
	mockIdentity   *mocks.Identity

	server       *httptest.Server
	serverCert   string
	serverURL    string
	serverUrlObj *url.URL
)

var _ = BeforeSuite(func() {
	// Start a local HTTP server
	server = httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		Expect(req.URL.String()).To(Equal("/cainfo"))
		return
	}))

	serverURL = server.URL
	rawCert := server.Certificate().Raw
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rawCert})
	serverCert = string(util.BytesToBase64(pemCert))

	urlObj, err := url.Parse(serverURL)
	Expect(err).NotTo(HaveOccurred())
	serverUrlObj = urlObj

	// Generate temporary key for reenroll test
	keystorePath := filepath.Join(homeDir, "msp", "keystore")
	err = os.MkdirAll(keystorePath, 0755)
	Expect(err).NotTo(HaveOccurred())

	key, err := util.Base64ToBytes(testkey)
	Expect(err).NotTo(HaveOccurred())
	err = ioutil.WriteFile(filepath.Join(keystorePath, "key.pem"), key, 0755)
})

var _ = AfterSuite(func() {
	// Close the server when test finishes
	server.Close()

	err = os.RemoveAll(homeDir)
	Expect(err).NotTo(HaveOccurred())
})
