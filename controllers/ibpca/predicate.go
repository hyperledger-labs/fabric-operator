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

package ibpca

import (
	"reflect"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
)

// Update defines a list of elements that we detect spec updates on
type Update struct {
	specUpdated           bool
	caOverridesUpdated    bool
	tlscaOverridesUpdated bool
	restartNeeded         bool
	caCryptoUpdated       bool
	caCryptoCreated       bool
	renewTLSCert          bool
	imagesUpdated         bool
	fabricVersionUpdated  bool
	caTagUpdated          bool
	// update GetUpdateStackWithTrues when new fields are added
}

// SpecUpdated returns true if any fields in spec are updated
func (u *Update) SpecUpdated() bool {
	return u.specUpdated
}

// CAOverridesUpdated returns true if ca config overrides updated
func (u *Update) CAOverridesUpdated() bool {
	return u.caOverridesUpdated
}

// TLSCAOverridesUpdated returns true if TLS ca config overrides updated
func (u *Update) TLSCAOverridesUpdated() bool {
	return u.tlscaOverridesUpdated
}

// ConfigOverridesUpdated returns true if either ca or TLS ca overrides updated
func (u *Update) ConfigOverridesUpdated() bool {
	return u.caOverridesUpdated || u.tlscaOverridesUpdated
}

// RestartNeeded returns true if changes in spec require components to restart
func (u *Update) RestartNeeded() bool {
	return u.restartNeeded
}

// CACryptoUpdated returns true if crypto material updated
func (u *Update) CACryptoUpdated() bool {
	return u.caCryptoUpdated
}

// CACryptoCreated returns true if crypto material created
func (u *Update) CACryptoCreated() bool {
	return u.caCryptoCreated
}

// RenewTLSCert returns true if need to renew TLS cert
func (u *Update) RenewTLSCert() bool {
	return u.renewTLSCert
}

// ImagesUpdated returns true if images updated
func (u *Update) ImagesUpdated() bool {
	return u.imagesUpdated
}

// FabricVersionUpdated returns true if fabric version updated
func (u *Update) FabricVersionUpdated() bool {
	return u.fabricVersionUpdated
}

func (u *Update) CATagUpdated() bool {
	return u.caTagUpdated
}

// GetUpdateStackWithTrues is a helper method to print updates that have been detected
func (u *Update) GetUpdateStackWithTrues() string {
	stack := ""

	if u.specUpdated {
		stack += "specUpdated "
	}
	if u.caOverridesUpdated {
		stack += "caOverridesUpdated "
	}
	if u.tlscaOverridesUpdated {
		stack += "tlscaOverridesUpdated "
	}
	if u.restartNeeded {
		stack += "restartNeeded "
	}
	if u.caCryptoUpdated {
		stack += "caCryptoUpdated "
	}
	if u.caCryptoCreated {
		stack += "caCryptoCreated "
	}
	if u.renewTLSCert {
		stack += "renewTLSCert "
	}
	if u.imagesUpdated {
		stack += "imagesUpdated "
	}
	if u.fabricVersionUpdated {
		stack += "fabricVersionUpdated "
	}
	if u.caTagUpdated {
		stack += "caTagUpdated "
	}

	if len(stack) == 0 {
		stack = "emptystack "
	}

	return stack
}

func imagesUpdated(old, new *current.IBPCA) bool {
	if new.Spec.Images != nil {
		if old.Spec.Images == nil {
			return true
		}

		if old.Spec.Images != nil {
			return !reflect.DeepEqual(old.Spec.Images, new.Spec.Images)
		}
	}

	return false
}

func fabricVersionUpdated(old, new *current.IBPCA) bool {
	return old.Spec.FabricVersion != new.Spec.FabricVersion
}
