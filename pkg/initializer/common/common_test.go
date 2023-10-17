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

package common_test

import (
	"encoding/base64"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Common", func() {
	var (
		mockValidator *mocks.CryptoValidator
		instance      *current.IBPPeer
	)

	BeforeEach(func() {
		mockValidator = &mocks.CryptoValidator{}

		instance = &current.IBPPeer{}
		instance.Name = "instance1"
	})

	Context("check crypto", func() {
		It("returns true, if missing a ecert crypto", func() {
			mockValidator.CheckEcertCryptoReturns(errors.New("not found"))
			err := common.CheckCrypto(mockValidator, instance, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing ecert crypto"))
		})

		It("returns true, if missing a tls crypto", func() {
			mockValidator.CheckTLSCryptoReturns(errors.New("not found"))
			err := common.CheckCrypto(mockValidator, instance, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing TLS crypto"))
		})

		It("returns true, if missing a tls crypto", func() {
			mockValidator.CheckClientAuthCryptoReturns(errors.New("not found"))
			err := common.CheckCrypto(mockValidator, instance, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing Client Auth crypto"))
		})

		It("returns false, if all crypto found and is in proper format", func() {
			err := common.CheckCrypto(mockValidator, instance, true)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("check if certificates are different", func() {
		var (
			currentCerts map[string][]byte
			base64cert   string
			base64cert2  string
		)

		BeforeEach(func() {
			base64cert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNvVENDQWtlZ0F3SUJBZ0lVTUwrYW4vS2QwRllaazhLTDRRMUQ2eHVJK08wd0NnWUlLb1pJemowRUF3SXcKV2pFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVFzd0NRWURWUVFERXdKallUQWVGdzB5Ck1EQTJNVGd5TVRRNU1EQmFGdzB5TVRBMk1UZ3lNVFUwTURCYU1HRXhDekFKQmdOVkJBWVRBbFZUTVJjd0ZRWUQKVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFVU1CSUdBMVVFQ2hNTFNIbHdaWEpzWldSblpYSXhEakFNQmdOVgpCQXNUQldGa2JXbHVNUk13RVFZRFZRUURFd3B3WldWeUxXRmtiV2x1TUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJCnpqMERBUWNEUWdBRVRDOXVtbDExU240UVlDQklPWnlUdGxXVHhFTy90R1Q0cGFNMXVYcXF0dlhkMWVSR1RSMVcKL0x2M0Y3K1k3M1cxZ0VqeEp0UkZaY0oxN3pOZUVHc2lYYU9CNHpDQjREQU9CZ05WSFE4QkFmOEVCQU1DQjRBdwpEQVlEVlIwVEFRSC9CQUl3QURBZEJnTlZIUTRFRmdRVVNsbVJ4a2JJMzNteHNLaEVtY1R6eVZYeHNkOHdId1lEClZSMGpCQmd3Rm9BVStKWU5rWFgyb0VUREdVbHl2OEdHcDk3YUM4RXdJZ1lEVlIwUkJCc3dHWUlYVTJGaFpITXQKVFdGalFtOXZheTFRY204dWJHOWpZV3d3WEFZSUtnTUVCUVlIQ0FFRVVIc2lZWFIwY25NaU9uc2lhR1l1UVdabQphV3hwWVhScGIyNGlPaUlpTENKb1ppNUZibkp2Ykd4dFpXNTBTVVFpT2lKd1pXVnlMV0ZrYldsdUlpd2lhR1l1ClZIbHdaU0k2SW1Ga2JXbHVJbjE5TUFvR0NDcUdTTTQ5QkFNQ0EwZ0FNRVVDSVFDZWRLazZPcVczR3JmdDZQWksKUHZwWUdla1c4NzdsUmgvOUtERHNWdlJKYlFJZ01aanRja2dBL2RTN0VjUXJ5VHl2cHB0TTdKWWJoZGRrZDdTcgp5TXl0b3c0PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
			base64cert2 = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNyRENDQWxPZ0F3SUJBZ0lVRUEwRGE1Ym5Eb1JzbWZLWGE4d0U5NkxNdTJBd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEl3TURZeE9ESXdORGt3TUZvWERUSXhNRFl4T0RJd05UUXdNRm93WURFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRU9NQXdHQTFVRUN4TUZZV1J0YVc0eEVqQVFCZ05WQkFNVENYQmxaWEpoWkcxcGJqQlpNQk1HCkJ5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEEwSUFCT0MwRG1vMm5MUUd4YzJRcnMyTlRUZ3hOdy9MTVluRWFheVQKQ0RKNldFVmlod2VPQ01WeTZ6MkVLVG81MHZsSm40aGd0VXhYR2xzb1AvN1YxZHdyMi9pamdlSXdnZDh3RGdZRApWUjBQQVFIL0JBUURBZ2VBTUF3R0ExVWRFd0VCL3dRQ01BQXdIUVlEVlIwT0JCWUVGRWVlSEZTUjladmMyeUxZCkZ3T1pkV0Iva0ozdU1COEdBMVVkSXdRWU1CYUFGSWd0eTR2U0VUZllCeDBTS1BPdExQQmZ0YTVxTUNJR0ExVWQKRVFRYk1CbUNGMU5oWVdSekxVMWhZMEp2YjJzdFVISnZMbXh2WTJGc01Gc0dDQ29EQkFVR0J3Z0JCRTk3SW1GMApkSEp6SWpwN0ltaG1Ma0ZtWm1sc2FXRjBhVzl1SWpvaUlpd2lhR1l1Ulc1eWIyeHNiV1Z1ZEVsRUlqb2ljR1ZsCmNtRmtiV2x1SWl3aWFHWXVWSGx3WlNJNkltRmtiV2x1SW4xOU1Bb0dDQ3FHU000OUJBTUNBMGNBTUVRQ0lGTFoKNnBCMWpDaWZIejRVTlZqd0p3RjlKUWZ2UCsxbFpJN0JydjFYdi9nUkFpQk0yMVg4N1N1V2tWaEdGRUpPOElnMQptMU9SNkZKSzBMUEN4SkU3bnlMdTRRPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="

			certBytes, err := base64.StdEncoding.DecodeString(base64cert)
			Expect(err).NotTo(HaveOccurred())

			cert2Bytes, err := base64.StdEncoding.DecodeString(base64cert2)
			Expect(err).NotTo(HaveOccurred())

			currentCerts = map[string][]byte{
				"cert1.pem": certBytes,
				"cert2.pem": cert2Bytes,
			}
		})

		It("returns false if list of certificates is equal", func() {
			newCerts := []string{base64cert2, base64cert}
			updated, err := common.CheckIfCertsDifferent(currentCerts, newCerts)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(Equal(false))
		})

		It("returns true if list of certificates is not equal", func() {
			base64cert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNyRENDQWxPZ0F3SUJBZ0lVVlM0WXQ3aFRUYnZFVWk4S1R0QWpEU0pHUG5jd0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEl3TURZeE9ESXdOREV3TUZvWERUSXhNRFl4T0RJd05EWXdNRm93WURFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRU9NQXdHQTFVRUN4TUZZV1J0YVc0eEVqQVFCZ05WQkFNVENYQmxaWEpoWkcxcGJqQlpNQk1HCkJ5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEEwSUFCSHBSRjJKRkhLZnVxNUR0bHArZDJGak0rZytacWJCY0FGN3QKQVpTM2VBL2JzRTNIcllLUWRaelNGSzhNUStGQnF5cFYrdEpDaldWMktZRFRvTGJvTk5DamdlSXdnZDh3RGdZRApWUjBQQVFIL0JBUURBZ2VBTUF3R0ExVWRFd0VCL3dRQ01BQXdIUVlEVlIwT0JCWUVGRWRRRHQwMDJSWGpwcXdnCmFjMTJuK3FlVHdTN01COEdBMVVkSXdRWU1CYUFGSWd0eTR2U0VUZllCeDBTS1BPdExQQmZ0YTVxTUNJR0ExVWQKRVFRYk1CbUNGMU5oWVdSekxVMWhZMEp2YjJzdFVISnZMbXh2WTJGc01Gc0dDQ29EQkFVR0J3Z0JCRTk3SW1GMApkSEp6SWpwN0ltaG1Ma0ZtWm1sc2FXRjBhVzl1SWpvaUlpd2lhR1l1Ulc1eWIyeHNiV1Z1ZEVsRUlqb2ljR1ZsCmNtRmtiV2x1SWl3aWFHWXVWSGx3WlNJNkltRmtiV2x1SW4xOU1Bb0dDQ3FHU000OUJBTUNBMGNBTUVRQ0lGZEQKODVFY2ErcTFralRmTGNLZlZhalVBb2I2OGtwUzUrM0ZraitsdUo1MUFpQTluZmRiZnMxYUpEV2VpUTdFOFdqLwpLOXgxRHUzY051Nno3Ym9leldlM1FRPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="

			newCerts := []string{base64cert2, base64cert}
			updated, err := common.CheckIfCertsDifferent(currentCerts, newCerts)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(Equal(true))
		})

		It("return true if list of certificates are different lengths", func() {
			newCerts := []string{base64cert}
			updated, err := common.CheckIfCertsDifferent(currentCerts, newCerts)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(Equal(true))
		})

		It("returns false if list of updated certificates is empty", func() {
			newCerts := []string{}
			updated, err := common.CheckIfCertsDifferent(currentCerts, newCerts)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated).To(Equal(false))
		})
	})

})
