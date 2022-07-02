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

package configtx_test

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/configtx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	defaultConfigTxFile = "../../../../testdata/init/orderer/configtx.yaml"
	defaultConfigTxDir  = "../../../../testdata/init/orderer"
)

var _ = Describe("configtx", func() {
	var (
		err      error
		configTx *configtx.ConfigTx
	)

	BeforeEach(func() {
		configTx = configtx.New()
	})

	It("returns an error if profile does not exist", func() {
		_, err = configTx.GetProfile("badprofile")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("profile 'badprofile' does not exist"))
	})

	It("loads top level config from file", func() {
		config, err := configtx.LoadTopLevelConfig(defaultConfigTxFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(config).NotTo(BeNil())
	})
})
