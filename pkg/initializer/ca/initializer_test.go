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
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Initializing the CA before start up", func() {
	var (
		init *initializer.Initializer
		ca   *mocks.IBPCA
	)

	BeforeEach(func() {
		ca = &mocks.IBPCA{}
		init = &initializer.Initializer{}
	})

	Context("create", func() {
		It("returns an error if unable to override server config", func() {
			msg := "failed to override"
			ca.OverrideServerConfigReturns(errors.New(msg))
			_, err := init.Create(nil, &v1.ServerConfig{}, ca)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if unable to write config", func() {
			msg := "failed to write config"
			ca.ParseCryptoReturns(nil, errors.New(msg))
			_, err := init.Create(nil, &v1.ServerConfig{}, ca)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if unable to write config", func() {
			msg := "failed to parse crypto"
			ca.WriteConfigReturns(errors.New(msg))
			_, err := init.Create(nil, &v1.ServerConfig{}, ca)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if unable to init", func() {
			msg := "failed to init"
			ca.InitReturns(errors.New(msg))
			_, err := init.Create(nil, &v1.ServerConfig{}, ca)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if unable to parse ca block", func() {
			msg := "failed to parse ca block"
			ca.ParseCABlockReturns(nil, errors.New(msg))
			_, err := init.Create(nil, &v1.ServerConfig{}, ca)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if unable to remove home directory", func() {
			msg := "failed to remove home directory"
			ca.RemoveHomeDirReturns(errors.New(msg))
			_, err := init.Create(nil, &v1.ServerConfig{}, ca)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns a response containing server config and map contains all crypto material", func() {
			result, err := init.Create(nil, &v1.ServerConfig{}, ca)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(Equal(nil))
		})
	})

	Context("update", func() {
		It("returns an error if unable to override server config", func() {
			msg := "failed to override"
			ca.OverrideServerConfigReturns(errors.New(msg))
			_, err := init.Update(nil, &v1.ServerConfig{}, ca)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if unable to parse crypto", func() {
			msg := "failed to parse crypto"
			ca.ParseCryptoReturns(nil, errors.New(msg))
			_, err := init.Update(nil, &v1.ServerConfig{}, ca)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns a response containing server config and map contains all crypto material", func() {
			result, err := init.Update(nil, &v1.ServerConfig{}, ca)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(Equal(nil))
		})
	})
})
