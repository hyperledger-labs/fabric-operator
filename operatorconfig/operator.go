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

package operatorconfig

import (
	"context"

	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	cainit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	"github.com/vrischmann/envconfig"

	corev1 "k8s.io/api/core/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/yaml"
)

// Client defines interface for making GET calls to kubernetes API server
type Client interface {
	Get(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object) error
}

// LoadFromConfigMap will read config map and return back operator config built on top by
// updating config values based on environment variables.
func LoadFromConfigMap(nn k8sclient.ObjectKey, key string, client Client, operator *Operator) error {
	cm := &corev1.ConfigMap{}
	err := client.Get(context.TODO(), nn, cm)
	if k8sclient.IgnoreNotFound(err) != nil {
		return err
	}

	err = load([]byte(cm.Data[key]), operator)
	if err != nil {
		return err
	}

	return nil
}

func load(config []byte, operator *Operator) error {
	// If no config bytes passed, we can assume that a config file (config map) for operator
	// does not exist and we can skip the unmarshal step.
	if len(config) > 0 {
		if err := yaml.Unmarshal(config, operator); err != nil {
			return err
		}
	}

	opts := envconfig.Options{
		Prefix:      "IBPOPERATOR",
		AllOptional: true,
		LeaveNil:    true,
	}

	if err := envconfig.InitWithOptions(operator, opts); err != nil {
		return err
	}

	return nil
}

// Operator defines operator configuration parameters
type Operator struct {
	Orderer  Orderer            `json:"orderer" yaml:"orderer"`
	Peer     Peer               `json:"peer" yaml:"peer"`
	CA       CA                 `json:"ca" yaml:"ca"`
	Console  Console            `json:"console" yaml:"console"`
	Restart  Restart            `json:"restart" yaml:"restart"`
	Versions *deployer.Versions `json:"versions,omitempty" yaml:"versions,omitempty"`
	Globals  Globals            `json:"globals,omitempty" yaml:"globals,omitempty" envconfig:"optional"`
	Debug    Debug              `json:"debug" yaml:"debug"`
}

// CA defines configurable properties for CA custom resource
type CA struct {
	Timeouts CATimeouts `json:"timeouts" yaml:"timeouts"`
}

// CATimeouts defines timeouts properties that can be configured
type CATimeouts struct {
	HSMInitJob cainit.HSMInitJobTimeouts `json:"hsmInitJob" yaml:"hsmInitJob"`
}

type Orderer struct {
	Timeouts      OrdererTimeouts `json:"timeouts" yaml:"timeouts"`
	Renewals      OrdererRenewals `json:"renewals" yaml:"renewals"`
	DisableProbes string          `json:"disableProbes" yaml:"disableProbes"`
}

type OrdererTimeouts struct {
	SecretPoll common.Duration               `json:"secretPollTimeout" yaml:"secretPollTimeout"`
	EnrollJob  enroller.HSMEnrollJobTimeouts `json:"enrollJob" yaml:"enrollJob"`
}

type OrdererRenewals struct {
	DisableTLScert bool `json:"disableTLScert" yaml:"disableTLScert"`
}

type Peer struct {
	Timeouts PeerTimeouts `json:"timeouts" yaml:"timeouts"`
}

type PeerTimeouts struct {
	DBMigration DBMigrationTimeouts           `json:"dbMigration" yaml:"dbMigration"`
	EnrollJob   enroller.HSMEnrollJobTimeouts `json:"enrollJob" yaml:"enrollJob"`
}

type DBMigrationTimeouts struct {
	CouchDBStartUp common.Duration `json:"couchDBStartUp" yaml:"couchDbStartUp"`
	JobStart       common.Duration `json:"jobStart" yaml:"jobStart"`
	JobCompletion  common.Duration `json:"jobCompletion" yaml:"jobCompletion"`
	ReplicaChange  common.Duration `json:"replicaChange" yaml:"replicaChange"`
	PodDeletion    common.Duration `json:"podDeletion" yaml:"podDeletion"`
	PodStart       common.Duration `json:"podStart" yaml:"podStart"`
}

type Restart struct {
	WaitTime common.Duration `json:"waitTime" yaml:"waitTime"`
	Disable  DisableRestart  `json:"disable" yaml:"disable"`
	Timeout  common.Duration `json:"timeout" yaml:"timeout"`
}

type DisableRestart struct {
	Components bool `json:"components" yaml:"components"`
}

type Globals struct {
	SecurityContext         *container.SecurityContext `json:"securityContext,omitempty" yaml:"securityContext,omitempty"`
	AllowKubernetesEighteen string                     `json:"allowKubernetesEighteen,omitempty" yaml:"allowKubernetesEighteen,omitempty"`
}

type Debug struct {
	DisableDeploymentChecks string `json:"disableDeploymentChecks,omitempty" yaml:"disableDeploymentChecks,omitempty"`
}

type Console struct {
	ApplyNetworkPolicy string `json:"applyNetworkPolicy" yaml:"applyNetworkPolicy"`
}

// SetDefaults will set defaults as defined by to the operator configuration settings
func (o *Operator) SetDefaults() {
	*o = Operator{
		Orderer: Orderer{
			Timeouts: OrdererTimeouts{
				SecretPoll: common.MustParseDuration("30s"),
				EnrollJob: enroller.HSMEnrollJobTimeouts{
					JobStart:      common.MustParseDuration("90s"),
					JobCompletion: common.MustParseDuration("90s"),
				},
			},
		},
		Peer: Peer{
			Timeouts: PeerTimeouts{
				DBMigration: DBMigrationTimeouts{
					CouchDBStartUp: common.MustParseDuration("90s"),
					JobStart:       common.MustParseDuration("90s"),
					JobCompletion:  common.MustParseDuration("90s"),
					ReplicaChange:  common.MustParseDuration("90s"),
					PodDeletion:    common.MustParseDuration("90s"),
					PodStart:       common.MustParseDuration("90s"),
				},
				EnrollJob: enroller.HSMEnrollJobTimeouts{
					JobStart:      common.MustParseDuration("90s"),
					JobCompletion: common.MustParseDuration("90s"),
				},
			},
		},
		CA: CA{
			Timeouts: CATimeouts{
				HSMInitJob: cainit.HSMInitJobTimeouts{
					JobStart:      common.MustParseDuration("90s"),
					JobCompletion: common.MustParseDuration("90s"),
				},
			},
		},
		Restart: Restart{
			WaitTime: common.MustParseDuration("10m"),
			Timeout:  common.MustParseDuration("5m"),
		},
		Versions: getDefaultVersions(),
	}
}
