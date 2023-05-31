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

package ibppeer

import (
	"reflect"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commoninit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
)

type Update struct {
	specUpdated           bool
	overridesUpdated      bool
	dindArgsUpdated       bool
	tlsCertUpdated        bool
	ecertUpdated          bool
	peerTagUpdated        bool
	restartNeeded         bool
	ecertReenrollNeeded   bool
	tlsReenrollNeeded     bool
	ecertNewKeyReenroll   bool
	tlscertNewKeyReenroll bool
	migrateToV2           bool
	migrateToV24          bool
	migrateToV25          bool
	mspUpdated            bool
	ecertEnroll           bool
	tlscertEnroll         bool
	upgradedbs            bool
	tlsCertCreated        bool
	ecertCreated          bool
	nodeOUUpdated         bool
	imagesUpdated         bool
	fabricVersionUpdated  bool
	// update GetUpdateStackWithTrues when new fields are added
}

func (u *Update) SpecUpdated() bool {
	return u.specUpdated
}

func (u *Update) ConfigOverridesUpdated() bool {
	return u.overridesUpdated
}

func (u *Update) DindArgsUpdated() bool {
	return u.dindArgsUpdated
}

func (u *Update) TLSCertUpdated() bool {
	return u.tlsCertUpdated
}

func (u *Update) EcertUpdated() bool {
	return u.ecertUpdated
}

func (u *Update) PeerTagUpdated() bool {
	return u.peerTagUpdated
}

func (u *Update) CertificateUpdated() bool {
	return u.tlsCertUpdated || u.ecertUpdated
}

func (u *Update) GetUpdatedCertType() commoninit.SecretType {
	if u.tlsCertUpdated {
		return commoninit.TLS
	} else if u.ecertUpdated {
		return commoninit.ECERT
	}
	return ""
}

func (u *Update) RestartNeeded() bool {
	return u.restartNeeded
}

func (u *Update) EcertReenrollNeeded() bool {
	return u.ecertReenrollNeeded
}

func (u *Update) TLSReenrollNeeded() bool {
	return u.tlsReenrollNeeded
}

func (u *Update) EcertNewKeyReenroll() bool {
	return u.ecertNewKeyReenroll
}

func (u *Update) TLScertNewKeyReenroll() bool {
	return u.tlscertNewKeyReenroll
}

func (u *Update) MigrateToV2() bool {
	return u.migrateToV2
}

func (u *Update) MigrateToV24() bool {
	return u.migrateToV24
}

func (u *Update) MigrateToV25() bool {
	return u.migrateToV25
}

func (u *Update) UpgradeDBs() bool {
	return u.upgradedbs
}

func (u *Update) EcertEnroll() bool {
	return u.ecertEnroll
}

func (u *Update) TLSCertEnroll() bool {
	return u.tlscertEnroll
}

func (u *Update) SetDindArgsUpdated(updated bool) {
	u.dindArgsUpdated = updated
}

func (u *Update) MSPUpdated() bool {
	return u.mspUpdated
}

func (u *Update) TLSCertCreated() bool {
	return u.tlsCertCreated
}

func (u *Update) EcertCreated() bool {
	return u.ecertCreated
}

func (u *Update) CertificateCreated() bool {
	return u.tlsCertCreated || u.ecertCreated
}

func (u *Update) GetCreatedCertType() commoninit.SecretType {
	if u.tlsCertCreated {
		return commoninit.TLS
	} else if u.ecertCreated {
		return commoninit.ECERT
	}
	return ""
}

func (u *Update) CryptoBackupNeeded() bool {
	return u.ecertEnroll ||
		u.tlscertEnroll ||
		u.ecertReenrollNeeded ||
		u.tlsReenrollNeeded ||
		u.ecertNewKeyReenroll ||
		u.tlscertNewKeyReenroll ||
		u.mspUpdated
}

func (u *Update) NodeOUUpdated() bool {
	return u.nodeOUUpdated
}

// ImagesUpdated returns true if images updated
func (u *Update) ImagesUpdated() bool {
	return u.imagesUpdated
}

// FabricVersionUpdated returns true if fabric version updated
func (u *Update) FabricVersionUpdated() bool {
	return u.fabricVersionUpdated
}

func (u *Update) Needed() bool {
	return u.specUpdated ||
		u.overridesUpdated ||
		u.dindArgsUpdated ||
		u.tlsCertUpdated ||
		u.ecertUpdated ||
		u.peerTagUpdated ||
		u.restartNeeded ||
		u.ecertReenrollNeeded ||
		u.tlsReenrollNeeded ||
		u.ecertNewKeyReenroll ||
		u.tlscertNewKeyReenroll ||
		u.migrateToV2 ||
		u.migrateToV24 ||
		u.migrateToV25 ||
		u.mspUpdated ||
		u.ecertEnroll ||
		u.upgradedbs ||
		u.nodeOUUpdated ||
		u.imagesUpdated ||
		u.fabricVersionUpdated
}

func (u *Update) GetUpdateStackWithTrues() string {
	stack := ""

	if u.specUpdated {
		stack += "specUpdated "
	}
	if u.overridesUpdated {
		stack += "overridesUpdated "
	}
	if u.dindArgsUpdated {
		stack += "dindArgsUpdated "
	}
	if u.tlsCertUpdated {
		stack += "tlsCertUpdated "
	}
	if u.ecertUpdated {
		stack += "ecertUpdated "
	}
	if u.peerTagUpdated {
		stack += "peerTagUpdated "
	}
	if u.restartNeeded {
		stack += "restartNeeded "
	}
	if u.ecertReenrollNeeded {
		stack += "ecertReenrollNeeded"
	}
	if u.tlsReenrollNeeded {
		stack += "tlsReenrollNeeded"
	}
	if u.migrateToV2 {
		stack += "migrateToV2 "
	}
	if u.migrateToV24 {
		stack += "migrateToV24 "
	}
	if u.migrateToV25 {
		stack += "migrateToV25 "
	}
	if u.mspUpdated {
		stack += "mspUpdated "
	}
	if u.ecertEnroll {
		stack += "ecertEnroll "
	}
	if u.tlscertEnroll {
		stack += "tlscertEnroll "
	}
	if u.upgradedbs {
		stack += "upgradedbs "
	}
	if u.tlsCertCreated {
		stack += "tlsCertCreated "
	}
	if u.ecertCreated {
		stack += "ecertCreated "
	}
	if u.nodeOUUpdated {
		stack += "nodeOUUpdated "
	}
	if u.imagesUpdated {
		stack += "imagesUpdated "
	}
	if u.fabricVersionUpdated {
		stack += "fabricVersionUpdated "
	}

	if len(stack) == 0 {
		stack = "emptystack "
	}

	return stack
}

func imagesUpdated(old, new *current.IBPPeer) bool {
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

func fabricVersionUpdated(old, new *current.IBPPeer) bool {
	return old.Spec.FabricVersion != new.Spec.FabricVersion
}
