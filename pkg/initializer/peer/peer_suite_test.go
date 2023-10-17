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

package initializer_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPeer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Peer Suite")
}

var (
	server *httptest.Server
)

var _ = BeforeSuite(func() {
	// Start a local HTTP server
	server = httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		Expect(req.URL.String()).To(Equal("/cainfo"))
		return
	}))
})

var _ = AfterSuite(func() {
	// Close the server when test finishes
	server.Close()
})
