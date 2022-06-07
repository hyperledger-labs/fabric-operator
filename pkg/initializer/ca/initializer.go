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

package initializer

import (
	"github.com/hyperledger/fabric-ca/lib"
	"k8s.io/apimachinery/pkg/runtime"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("ca_initializer")

type Config struct {
	SharedPath              string `json:"sharedPath"`
	CADefaultConfigPath     string `json:"cadefaultconfigpath"`
	TLSCADefaultConfigPath  string `json:"tlscadefaultconfigpath"`
	CAOverrideConfigPath    string `json:"caoverrideconfigpath"`
	TLSCAOverrideConfigPath string `json:"tlscaoverrideconfigpath"`
	DeploymentFile          string
	PVCFile                 string
	ServiceFile             string
	RoleFile                string
	ServiceAccountFile      string
	RoleBindingFile         string
	ConfigMapFile           string
	IngressFile             string
	Ingressv1beta1File      string
	RouteFile               string
}

type ConfigOptions struct {
	DefaultPath  string `json:"defaultpath"`
	OverridePath string `json:"overridepath"`
}

type Response struct {
	Config    *v1.ServerConfig
	CryptoMap map[string][]byte
}

//go:generate counterfeiter -o mocks/ibpca.go -fake-name IBPCA . IBPCA

type IBPCA interface {
	OverrideServerConfig(newConfig *v1.ServerConfig) (err error)
	ViperUnmarshal(configFile string) (*lib.ServerConfig, error)
	ParseCrypto() (map[string][]byte, error)
	ParseCABlock() (map[string][]byte, error)
	GetServerConfig() *v1.ServerConfig
	WriteConfig() (err error)
	RemoveHomeDir() error
	IsBeingUpdated()
	ConfigToBytes() ([]byte, error)
	GetHomeDir() string
	Init() (err error)
	SetMountPaths()
	GetType() config.Type
}

type Initializer struct {
	Timeouts HSMInitJobTimeouts
	Client   k8sclient.Client
	Scheme   *runtime.Scheme
}

func (i *Initializer) Create(instance *current.IBPCA, overrides *v1.ServerConfig, ca IBPCA) (*Response, error) {
	type Create interface {
		Create(instance *current.IBPCA, overrides *v1.ServerConfig, ca IBPCA) (*Response, error)
	}

	var initializer Create
	if instance.IsHSMEnabledForType(ca.GetType()) {
		if instance.UsingHSMProxy() {
			// If Using HSM Proxy, currently sticking with old way of initialization which is within the operator process
			// and not a kuberenetes job
			initializer = &SW{}
		} else {
			hsmConfig, err := commonconfig.ReadHSMConfig(i.Client, instance)
			if err != nil {
				return nil, err
			}

			if hsmConfig.Daemon != nil {
				initializer = &HSMDaemon{Client: i.Client, Timeouts: i.Timeouts, Config: hsmConfig}
			} else {
				initializer = &HSM{Client: i.Client, Timeouts: i.Timeouts, Config: hsmConfig}
			}
		}
	} else {
		initializer = &SW{}
	}

	return initializer.Create(instance, overrides, ca)
}

func (i *Initializer) Update(instance *current.IBPCA, overrides *v1.ServerConfig, ca IBPCA) (*Response, error) {
	ca.IsBeingUpdated()

	err := ca.OverrideServerConfig(overrides)
	if err != nil {
		return nil, err
	}

	crypto, err := ca.ParseCrypto()
	if err != nil {
		return nil, err
	}

	ca.SetMountPaths()

	return &Response{
		Config:    ca.GetServerConfig(),
		CryptoMap: crypto,
	}, nil
}
