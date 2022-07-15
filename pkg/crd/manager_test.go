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

package crd_test

import (
	"errors"

	"github.com/IBM-Blockchain/fabric-operator/pkg/crd"
	"github.com/IBM-Blockchain/fabric-operator/pkg/crd/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("manager", func() {
	var mockClient *mocks.Client

	BeforeEach(func() {
		mockClient = &mocks.Client{}
	})

	Context("NewManager", func() {
		It("returns an error if it fails to load a file", func() {
			m, err := crd.NewManager(mockClient, "bad.yaml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such file or directory"))
			Expect(m).To(BeNil())
		})

		It("returns a manager", func() {
			m, err := crd.NewManager(mockClient, "../../config/crd/bases/ibp.com_ibpcas.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).NotTo(BeNil())
		})
	})

	Context("Create", func() {
		var (
			err     error
			manager *crd.Manager
		)

		BeforeEach(func() {
			manager, err = crd.NewManager(mockClient, "../../config/crd/bases/ibp.com_ibpcas.yaml")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error if it fails to create CRD", func() {
			mockClient.CreateCRDReturns(nil, errors.New("failed to create crd"))
			err = manager.Create()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create crd"))
		})

		It("returns no error on successful creation", func() {
			err = manager.Create()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
