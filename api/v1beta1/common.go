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

package v1beta1

import (
	"errors"
	"fmt"

	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

// Component is a custom type that enumerates all the components (containers)
type Component string

const (
	INIT       Component = "INIT"
	CA         Component = "CA"
	ORDERER    Component = "ORDERER"
	PEER       Component = "PEER"
	GRPCPROXY  Component = "GRPCPROXY"
	FLUENTD    Component = "FLUENTD"
	DIND       Component = "DIND"
	COUCHDB    Component = "COUCHDB"
	CCLAUNCHER Component = "CCLAUNCHER"
	ENROLLER   Component = "ENROLLER"
	HSMDAEMON  Component = "HSMDAEMON"
)

func (crn *CRN) String() string {
	return fmt.Sprintf("crn:%s:%s:%s:%s:%s:%s:%s:%s:%s",
		crn.Version, crn.CName, crn.CType, crn.Servicename, crn.Location, crn.AccountID, crn.InstanceID, crn.ResourceType, crn.ResourceID)
}

func (catls *CATLS) GetBytes() ([]byte, error) {
	return util.Base64ToBytes(catls.CACert)
}

func (e *Enrollment) GetCATLSBytes() ([]byte, error) {
	if e.CATLS != nil {
		return e.CATLS.GetBytes()
	}
	return nil, errors.New("no CA TLS certificate set")
}
