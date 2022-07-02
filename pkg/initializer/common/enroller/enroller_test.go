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
	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller/mocks"
)

var _ = Describe("Enroller", func() {
	var (
		mockCryptoEnroller *mocks.CryptoEnroller
		testEnroller       *enroller.Enroller
	)

	BeforeEach(func() {
		mockCryptoEnroller = &mocks.CryptoEnroller{}
		testEnroller = &enroller.Enroller{
			Enroller: mockCryptoEnroller,
		}
	})

	Context("get crypto", func() {
		BeforeEach(func() {
			mockCryptoEnroller.GetEnrollmentRequestReturns(&current.Enrollment{})
			mockCryptoEnroller.EnrollReturns(&config.Response{}, nil)
		})

		It("returns response", func() {
			resp, err := testEnroller.GetCrypto()
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})

		It("returns error if enroll fails", func() {
			mockCryptoEnroller.EnrollReturns(nil, errors.New("enroll failed"))

			resp, err := testEnroller.GetCrypto()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("enroll failed")))
			Expect(resp).To(BeNil())
		})
	})

	Context("ping CA", func() {
		It("returns true if ca reachable", func() {
			err := testEnroller.PingCA()
			Expect(err).To(BeNil())
		})

		It("returns true if ca reachable", func() {
			mockCryptoEnroller.PingCAReturns(errors.New("ping failed"))

			err := testEnroller.PingCA()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("ping failed")))
		})
	})

	Context("validate", func() {
		var req *current.Enrollment

		BeforeEach(func() {
			req = &current.Enrollment{
				CAHost:       "host",
				CAPort:       "1234",
				EnrollID:     "id",
				EnrollSecret: "secret",
				CATLS: &current.CATLS{
					CACert: "cacert",
				},
			}
			mockCryptoEnroller.GetEnrollmentRequestReturns(req)
		})

		It("successfull validation returns no error", func() {
			err := testEnroller.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error if missing CA host", func() {
			req.CAHost = ""

			err := testEnroller.Validate()
			Expect(err).To(MatchError("unable to enroll, CA host not specified"))
		})

		It("returns error if missing CA port", func() {
			req.CAPort = ""

			err := testEnroller.Validate()
			Expect(err).To(MatchError("unable to enroll, CA port not specified"))
		})

		It("returns error if missing enrollment ID", func() {
			req.EnrollID = ""

			err := testEnroller.Validate()
			Expect(err).To(MatchError("unable to enroll, enrollment ID not specified"))
		})

		It("returns error if missing enrollment secret", func() {
			req.EnrollSecret = ""

			err := testEnroller.Validate()
			Expect(err).To(MatchError("unable to enroll, enrollment secret not specified"))
		})

		It("returns error if missing CA TLS cert", func() {
			req.CATLS.CACert = ""

			err := testEnroller.Validate()
			Expect(err).To(MatchError("unable to enroll, CA TLS certificate not specified"))
		})
	})
})
