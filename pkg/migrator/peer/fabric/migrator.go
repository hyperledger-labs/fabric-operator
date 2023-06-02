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

package fabric

import (
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("peer_fabric_migrator")

type Version string

const (
	V2 Version = "V2"
)

//go:generate counterfeiter -o mocks/migrator.go -fake-name Migrator . Migrator
type Migrator interface {
	MigrationNeeded(metav1.Object) bool
	UpgradeDBs(metav1.Object, config.DBMigrationTimeouts) error
	UpdateConfig(metav1.Object, string) error
	SetChaincodeLauncherResourceOnCR(metav1.Object) error
}

func V2Migrate(instance metav1.Object, migrator Migrator, version string, timeouts config.DBMigrationTimeouts) error {
	if !migrator.MigrationNeeded(instance) {
		log.Info("Migration not needed, skipping migration")
		return nil
	}
	log.Info("Migration is needed, starting migration")

	if err := migrator.SetChaincodeLauncherResourceOnCR(instance); err != nil {
		return errors.Wrap(err, "failed to update chaincode launcher resources on CR")
	}

	if err := migrator.UpdateConfig(instance, version); err != nil {
		return errors.Wrap(err, "failed to update config")
	}

	if err := migrator.UpgradeDBs(instance, timeouts); err != nil {
		return errors.Wrap(err, "failed to upgrade peer's dbs")
	}

	return nil
}

func V24Migrate(instance metav1.Object, migrator Migrator, version string, timeouts config.DBMigrationTimeouts) error {
	if err := migrator.UpdateConfig(instance, version); err != nil {
		return errors.Wrap(err, "failed to update v2.4.x configs")
	}
	return nil
}

func V25Migrate(instance metav1.Object, migrator Migrator, version string, timeouts config.DBMigrationTimeouts) error {
	if err := migrator.UpdateConfig(instance, version); err != nil {
		return errors.Wrap(err, "failed to update v2.5.x configs")
	}
	return nil
}
