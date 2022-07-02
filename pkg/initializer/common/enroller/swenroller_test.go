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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hyperledger/fabric-ca/lib"
	"github.com/hyperledger/fabric-ca/lib/client/credential"
	"github.com/hyperledger/fabric-ca/lib/client/credential/x509"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller/mocks"
)

var _ = Describe("Software enroller", func() {
	var (
		e        *enroller.SWEnroller
		caClient *mocks.CAClient
	)

	BeforeEach(func() {
		caClient = &mocks.CAClient{}
		caClient.GetHomeDirReturns("../../../../testdata")

		creds := []credential.Credential{
			x509.NewCredential("", "", nil),
		}
		caClient.EnrollReturns(&lib.EnrollmentResponse{
			Identity: lib.NewIdentity(nil, "", creds),
		}, nil)
		caClient.GetEnrollmentRequestReturns(&current.Enrollment{
			CATLS: &current.CATLS{
				CACert: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNGakNDQWIyZ0F3SUJBZ0lVZi84bk94M2NqM1htVzNDSUo1L0Q1ejRRcUVvd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRBek1ERTNNamd3TUZvWERUTTBNVEF5TmpFM01qZ3dNRm93YURFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1Sa3dGd1lEVlFRREV4Qm1ZV0p5YVdNdFkyRXRjMlZ5CmRtVnlNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUVSbzNmbUc2UHkyUHd6cUMwNnFWZDlFOFgKZ044eldqZzFMb3lnMmsxdkQ4MXY1dENRRytCTVozSUJGQnI2VTRhc0tZTUREakd6TElERmdUUTRjVDd1VktORgpNRU13RGdZRFZSMFBBUUgvQkFRREFnRUdNQklHQTFVZEV3RUIvd1FJTUFZQkFmOENBUUV3SFFZRFZSME9CQllFCkZFa0RtUHhjbTdGcXZSMXllN0tNNGdLLy9KZ1JNQW9HQ0NxR1NNNDlCQU1DQTBjQU1FUUNJRC92QVFVSEh2SWwKQWZZLzM5UWdEU2ltTWpMZnhPTG44NllyR1EvWHpkQVpBaUFpUmlyZmlMdzVGbXBpRDhtYmlmRjV4bzdFUzdqNApaUWQyT0FUNCt5OWE0Zz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K",
			},
		})

		e = &enroller.SWEnroller{
			Client: caClient,
		}
	})

	Context("enroll", func() {
		It("returns no error on successfull enroll", func() {
			resp, err := e.Enroll()
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})
	})

	// TODO: Add more tests for error path testing
})
