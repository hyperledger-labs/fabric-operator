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

package fabric_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/migrator/peer/fabric"
	"github.com/IBM-Blockchain/fabric-operator/pkg/migrator/peer/fabric/mocks"
)

var _ = Describe("Peer migrator", func() {
	var (
		migrator *mocks.Migrator
		instance *current.IBPPeer
	)
	const FABRIC_V2 = "2.2.5-1"

	BeforeEach(func() {
		migrator = &mocks.Migrator{}
		migrator.MigrationNeededReturns(true)

		instance = &current.IBPPeer{}
	})

	Context("migrate to version", func() {
		Context("V2", func() {
			It("returns error on failure", func() {
				migrator.UpgradeDBsReturns(errors.New("failed to reset peer"))
				err := fabric.V2Migrate(instance, migrator, FABRIC_V2, config.DBMigrationTimeouts{})
				Expect(err).To(HaveOccurred())
				Expect(err).Should(MatchError(ContainSubstring("failed to reset peer")))
			})

			It("migrates", func() {
				err := fabric.V2Migrate(instance, migrator, FABRIC_V2, config.DBMigrationTimeouts{})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("V2 migration", func() {
		It("returns immediately when migration not needed", func() {
			migrator.MigrationNeededReturns(false)
			err := fabric.V2Migrate(instance, migrator, FABRIC_V2, config.DBMigrationTimeouts{})
			Expect(err).NotTo(HaveOccurred())
			Expect(migrator.UpdateConfigCallCount()).To(Equal(0))
			Expect(migrator.UpgradeDBsCallCount()).To(Equal(0))
		})

		It("returns an error if unable to update config", func() {
			migrator.UpdateConfigReturns(errors.New("failed to update config"))
			err := fabric.V2Migrate(instance, migrator, FABRIC_V2, config.DBMigrationTimeouts{})
			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(ContainSubstring("failed to update config")))
		})

		It("returns an error if unable to reset peer", func() {
			migrator.UpgradeDBsReturns(errors.New("failed to reset peer"))
			err := fabric.V2Migrate(instance, migrator, FABRIC_V2, config.DBMigrationTimeouts{})
			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(ContainSubstring("failed to reset peer")))
		})
	})
})
