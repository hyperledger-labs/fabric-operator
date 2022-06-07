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

package enroller

import (
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/hyperledger/fabric-ca/api"
	"github.com/hyperledger/fabric-ca/lib"
)

type HSMProxyCAClient interface {
	Init() error
	Enroll(*api.EnrollmentRequest) (*lib.EnrollmentResponse, error)
	GetEnrollmentRequest() *current.Enrollment
	GetHomeDir() string
	GetTLSCert() []byte
	PingCA(time.Duration) error
	SetHSMLibrary(string)
}

type HSMProxyEnroller struct {
	Client HSMProxyCAClient
	Req    *current.Enrollment
}

func NewHSMProxyEnroller(caClient HSMProxyCAClient) *HSMProxyEnroller {
	return &HSMProxyEnroller{
		Client: caClient,
	}
}

func (e *HSMProxyEnroller) GetEnrollmentRequest() *current.Enrollment {
	return e.Client.GetEnrollmentRequest()
}

func (e *HSMProxyEnroller) PingCA(timeout time.Duration) error {
	return e.Client.PingCA(timeout)
}

func (e *HSMProxyEnroller) Enroll() (*config.Response, error) {
	e.Client.SetHSMLibrary("/usr/local/lib/libpkcs11-proxy.so")
	return enroll(e.Client)
}
