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

package operatorerrors_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("operator errors", func() {
	var (
		operatorErr *operatorerrors.OperatorError
		log         logr.Logger
	)

	BeforeEach(func() {
		operatorErr = operatorerrors.New(operatorerrors.InvalidDeploymentCreateRequest, "operator error occurred")
		log = logf.Log.WithName("test")
	})

	Context("breaking error", func() {
		It("returns nil if breaking error detected", func() {
			err := operatorerrors.IsBreakingError(operatorErr, "operator error", log)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns err if errors is not an operator error", func() {
			err := operatorerrors.IsBreakingError(errors.New("non-operator error"), "not an operator error", log)
			Expect(err).To(HaveOccurred())
		})
	})
})
