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

package enroller

import (
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/cryptogen"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
)

//go:generate counterfeiter -o mocks/cryptoinstance.go -fake-name CryptoInstance . CryptoInstance

type CryptoInstance interface {
	runtime.Object
	Instance
	IsHSMEnabled() bool
	UsingHSMProxy() bool
	GetConfigOverride() (interface{}, error)
}

func Factory(enrollment *current.Enrollment, k8sClient k8sclient.Client, instance CryptoInstance, storagePath string, scheme *runtime.Scheme, bytes []byte, timeouts HSMEnrollJobTimeouts) (*Enroller, error) {
	caClient := NewFabCAClient(enrollment, storagePath, nil, bytes)
	certEnroller := New(NewSWEnroller(caClient))

	if instance.IsHSMEnabled() {
		switch instance.UsingHSMProxy() {
		case true:
			log.Info("Using HSM Proxy enroller")
			bccsp := cryptogen.InitBCCSP(instance)
			caClient = NewFabCAClient(enrollment, storagePath, bccsp, bytes)
			certEnroller = New(NewHSMProxyEnroller(caClient))
		case false:
			hsmConfig, err := config.ReadHSMConfig(k8sClient, instance)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read HSM config")
			}

			bccsp := cryptogen.InitBCCSP(instance)
			caClient = NewFabCAClient(enrollment, storagePath, bccsp, bytes)

			if hsmConfig.Daemon != nil {
				log.Info("Using HSM Daemon enroller")
				hsmDaemonEnroller := NewHSMDaemonEnroller(enrollment, instance, caClient, k8sClient, scheme, timeouts, hsmConfig)
				certEnroller = New(hsmDaemonEnroller)
			} else {
				log.Info("Using HSM enroller")
				hsmEnroller := NewHSMEnroller(enrollment, instance, caClient, k8sClient, scheme, timeouts, hsmConfig)
				certEnroller = New(hsmEnroller)
			}
		}
	} else {
		log.Info("Using SW enroller")
	}

	return certEnroller, nil
}
