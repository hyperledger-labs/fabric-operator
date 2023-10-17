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

package enroller_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/hyperledger/fabric-ca/lib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnroller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Enroller Suite")
}

var (
	server      *httptest.Server
	fabCaClient *enroller.FabCAClient
)

var _ = BeforeSuite(func() {
	server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		Expect(req.URL.String()).To(Equal("/cainfo"))
		// Send response to be tested
		rw.Write([]byte(`OK`))
	}))

	fabCaClient = &enroller.FabCAClient{
		Client: &lib.Client{
			Config: &lib.ClientConfig{
				URL: server.URL,
			},
		},
	}
})

var _ = AfterSuite(func() {
	server.Close()
})

//go:generate counterfeiter -o mocks/client.go -fake-name Client ../../../k8s/controllerclient Client
