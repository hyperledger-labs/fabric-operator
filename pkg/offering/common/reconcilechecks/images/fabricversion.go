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

package images

import (
	"fmt"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
)

// FabricVersion handles validation on fabric version
type FabricVersion struct {
	Versions *deployer.Versions
}

//go:generate counterfeiter -o mocks/fabricversion.go -fake-name FabricVersionInstance . FabricVersionInstance

// FabricVersionInstance defines the contract expected from instances
type FabricVersionInstance interface {
	GetFabricVersion() string
}

// Normalize normalizes the fabric version to x.x.x-x
func (fv *FabricVersion) Normalize(instance FabricVersionInstance) string {
	var v interface{}

	switch instance.(type) {
	case *current.IBPCA:
		v = fv.Versions.CA
	case *current.IBPPeer:
		v = fv.Versions.Peer
	case *current.IBPOrderer:
		v = fv.Versions.Orderer
	}

	return normalizeFabricVersion(instance.GetFabricVersion(), v)
}

// Validate will interate through the keys in versions map and check to
// see if versions is present (valid)
func (fv *FabricVersion) Validate(instance FabricVersionInstance) error {
	fabricVersion := instance.GetFabricVersion()

	switch instance.(type) {
	case *current.IBPCA:
		_, found := fv.Versions.CA[fabricVersion]
		if !found {
			return fmt.Errorf("fabric version '%s' is not supported for CA", fabricVersion)
		}
	case *current.IBPPeer:
		_, found := fv.Versions.Peer[fabricVersion]
		if !found {
			return fmt.Errorf("fabric version '%s' is not supported for Peer", fabricVersion)
		}
	case *current.IBPOrderer:
		_, found := fv.Versions.Orderer[fabricVersion]
		if !found {
			return fmt.Errorf("fabric version '%s' is not supported for Orderer", fabricVersion)
		}
	}

	return nil
}
