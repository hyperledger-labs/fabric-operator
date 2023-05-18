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

package configtx

import (
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric-protos-go/orderer/etcdraft"
	"github.com/hyperledger/fabric/common/viperutil"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// +k8s:openapi-gen=true
func GetGenesisDefaults() *TopLevel {
	return &TopLevel{
		Profiles: map[string]*Profile{
			"Initial": {
				Orderer: &Orderer{
					Organizations: []*Organization{},
					OrdererType:   "etcdraft",
					Addresses:     []string{},
					BatchTimeout:  2 * time.Second,
					BatchSize: BatchSize{
						MaxMessageCount:   500,
						AbsoluteMaxBytes:  10 * 1024 * 1024,
						PreferredMaxBytes: 2 * 1024 * 1024,
					},
					EtcdRaft: &etcdraft.ConfigMetadata{
						Consenters: []*etcdraft.Consenter{},
						Options: &etcdraft.Options{
							TickInterval:         "500ms",
							ElectionTick:         10,
							HeartbeatTick:        1,
							MaxInflightBlocks:    5,
							SnapshotIntervalSize: 20 * 1024 * 1024, // 20 MB
						},
					},
					Capabilities: map[string]bool{
						"V1_4_2": true,
					},
					Policies: map[string]*Policy{
						"Readers": {
							Type: "ImplicitMeta",
							Rule: "ANY Readers",
						},
						"Writers": {
							Type: "ImplicitMeta",
							Rule: "ANY Writers",
						},
						"Admins": {
							Type: "ImplicitMeta",
							Rule: "ANY Admins",
						},
						"BlockValidation": {
							Type: "ImplicitMeta",
							Rule: "ANY Writers",
						},
					},
				},

				Consortiums: map[string]*Consortium{
					"SampleConsortium": {},
				},
				Capabilities: map[string]bool{
					"V1_4_3": true,
				},
				Policies: map[string]*Policy{
					"Readers": {
						Type: "ImplicitMeta",
						Rule: "ANY Readers",
					},
					"Writers": {
						Type: "ImplicitMeta",
						Rule: "ANY Writers",
					},
					"Admins": {
						Type: "ImplicitMeta",
						Rule: "MAJORITY Admins",
					},
				},
			},
		},
	}
}

func LoadTopLevelConfig(configFile string) (*TopLevel, error) {
	config := viper.New()
	configDir, err := filepath.Abs(filepath.Dir(configFile))
	if err != nil {
		return nil, errors.Wrap(err, "error getting absolute path")
	}
	config.AddConfigPath(configDir)
	config.SetConfigName("configtx")

	err = config.ReadInConfig()
	if err != nil {
		return nil, errors.Wrap(err, "error reading configuration")
	}

	var uconf TopLevel
	err = viperutil.EnhancedExactUnmarshal(config, &uconf)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling config into struct")
	}

	return &uconf, nil
}

type ConfigTx struct {
	Config *TopLevel
}

func New() *ConfigTx {
	c := &ConfigTx{
		Config: GetGenesisDefaults(),
	}

	return c
}

func (c *ConfigTx) GetProfile(name string) (*Profile, error) {
	p, found := c.Config.Profiles[name]
	if !found {
		return nil, errors.Errorf("profile '%s' does not exist", name)
	}

	err := c.CompleteProfileInitialization(p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (c *ConfigTx) CompleteProfileInitialization(p *Profile) error {
	if p.Orderer != nil {
		return c.CompleteOrdererInitialization(p.Orderer)
	}

	return nil
}

func (c *ConfigTx) CompleteOrdererInitialization(ord *Orderer) error {
	// Additional, consensus type-dependent initialization goes here
	// Also using this to panic on unknown orderer type.
	switch ord.OrdererType {
	case ConsensusTypeSolo:
		// nothing to be done here
	case ConsensusTypeKafka:
		// nothing to be done here
	case ConsensusTypeEtcdRaft:
		if _, err := time.ParseDuration(ord.EtcdRaft.Options.TickInterval); err != nil {
			return errors.Errorf("Etcdraft TickInterval (%s) must be in time duration format", ord.EtcdRaft.Options.TickInterval)
		}

		// validate the specified members for Options
		if ord.EtcdRaft.Options.ElectionTick <= ord.EtcdRaft.Options.HeartbeatTick {
			return errors.Errorf("election tick must be greater than heartbeat tick")
		}

		for _, c := range ord.EtcdRaft.GetConsenters() {
			if c.Host == "" {
				return errors.Errorf("consenter info in %s configuration did not specify host", ConsensusTypeEtcdRaft)
			}
			if c.Port == 0 {
				return errors.Errorf("consenter info in %s configuration did not specify port", ConsensusTypeEtcdRaft)
			}
			if c.ClientTlsCert == nil {
				return errors.Errorf("consenter info in %s configuration did not specify client TLS cert", ConsensusTypeEtcdRaft)
			}
			if c.ServerTlsCert == nil {
				return errors.Errorf("consenter info in %s configuration did not specify server TLS cert", ConsensusTypeEtcdRaft)
			}
		}
	default:
		return errors.Errorf("unknown orderer type: %s", ord.OrdererType)
	}

	return nil
}
