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

package cryptogen

import (
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	common "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/version"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate counterfeiter -o mocks/instance.go -fake-name Instance . Instance

type Instance interface {
	runtime.Object
	metav1.Object
	IsHSMEnabled() bool
	UsingHSMProxy() bool
	GetConfigOverride() (interface{}, error)
}

//go:generate counterfeiter -o mocks/config.go -fake-name Config . Config

type Config interface {
	SetDefaultKeyStore()
	SetPKCS11Defaults(bool)
	GetBCCSPSection() *common.BCCSP
}

func InitBCCSP(instance Instance) *common.BCCSP {
	if !instance.IsHSMEnabled() {
		return nil
	}

	co, err := instance.GetConfigOverride()
	if err != nil {
		return nil
	}

	configOverride, ok := co.(Config)
	if !ok {
		return nil
	}

	if instance.IsHSMEnabled() {
		configOverride.SetPKCS11Defaults(instance.UsingHSMProxy())

		switch i := instance.(type) {
		case *current.IBPPeer:
			// If peer is older than 1.4.7 than we need to set msp/keystore path
			// even when using PKCS11 (HSM) other wise fabric peer refuses to start
			peerTag := strings.Split(i.Spec.Images.PeerTag, "-")[0]
			if version.String(peerTag).LessThan(version.V1_4_7) {
				configOverride.SetDefaultKeyStore()
			}
		}
	}

	return configOverride.GetBCCSPSection()
}
