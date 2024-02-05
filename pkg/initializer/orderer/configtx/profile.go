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
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric/common/channelconfig"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/common/policydsl"

	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/orderer/etcdraft"
	utils "github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
)

const (
	ordererAdminsPolicyName = "/Channel/Orderer/Admins"
)

func (p *Profile) AddOrdererAddress(address string) {
	p.Orderer.Addresses = append(p.Orderer.Addresses, address)
}

func (p *Profile) SetOrdererType(ordererType string) {
	p.Orderer.OrdererType = ordererType
}

func (p *Profile) SetCapabilitiesForOrderer(capabilities map[string]bool) {
	p.Orderer.Capabilities = capabilities
}

func (p *Profile) AddRaftConsentingNode(consenter *etcdraft.Consenter) error {
	if strings.ToLower(p.Orderer.OrdererType) != "etcdraft" {
		return errors.New("can only add raft consenting node if orderer type is 'etcdraft'")
	}
	p.Orderer.EtcdRaft.Consenters = append(p.Orderer.EtcdRaft.Consenters, consenter)
	return nil
}

func (p *Profile) AddConsortium(name string, consortium *Consortium) error {
	for _, org := range consortium.Organizations {
		err := ValidateOrg(org)
		if err != nil {
			return err
		}
	}
	p.Consortiums[name] = consortium
	return nil
}

func (p *Profile) AddOrgToConsortium(name string, org *Organization) error {
	err := ValidateOrg(org)
	if err != nil {
		return err
	}
	p.Consortiums[name].Organizations = append(p.Consortiums[name].Organizations, org)
	return nil
}

func (p *Profile) AddOrgToOrderer(org *Organization) error {
	err := ValidateOrg(org)
	if err != nil {
		return err
	}
	p.Orderer.Organizations = append(p.Orderer.Organizations, org)
	return nil
}

func (p *Profile) SetMaxChannel(max uint64) {
	p.Orderer.MaxChannels = max
}

func (p *Profile) SetChannelPolicy(policies map[string]*Policy) {
	p.Policies = policies
}

func (p *Profile) GenerateBlock(channelID string, mspConfigs map[string]*msp.MSPConfig) ([]byte, error) {
	if p.Orderer == nil {
		return nil, errors.Errorf("refusing to generate block which is missing orderer section")
	}

	if p.Consortiums == nil {
		return nil, errors.New("Genesis block does not contain a consortiums group definition.  This block cannot be used for orderer bootstrap.")
	}

	cg, err := p.NewChannelConfigGroup(mspConfigs)
	if err != nil {
		return nil, err
	}

	genesisBlock := p.Block(channelID, cg)
	gBlockBytes, err := utils.Marshal(genesisBlock)
	if err != nil {
		return nil, errors.Wrap(err, "error marshalling genesis block")
	}

	return gBlockBytes, nil

}

func (p *Profile) Block(channelID string, channelGroup *cb.ConfigGroup) *cb.Block {
	payloadChannelHeader := utils.MakeChannelHeader(cb.HeaderType_CONFIG, int32(1), channelID, 0)
	payloadSignatureHeader := utils.MakeSignatureHeader(nil, utils.CreateNonceOrPanic())
	utils.SetTxID(payloadChannelHeader, payloadSignatureHeader)
	payloadHeader := utils.MakePayloadHeader(payloadChannelHeader, payloadSignatureHeader)
	payload := &cb.Payload{Header: payloadHeader, Data: utils.MarshalOrPanic(&cb.ConfigEnvelope{Config: &cb.Config{ChannelGroup: channelGroup}})}
	envelope := &cb.Envelope{Payload: utils.MarshalOrPanic(payload), Signature: nil}

	block := utils.NewBlock(0, nil)
	block.Data = &cb.BlockData{Data: [][]byte{utils.MarshalOrPanic(envelope)}}
	block.Header.DataHash, _ = utils.BlockDataHash(block.Data)
	block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = utils.MarshalOrPanic(&cb.Metadata{
		Value: utils.MarshalOrPanic(&cb.LastConfig{Index: 0}),
	})
	return block
}

func (p *Profile) NewChannelConfigGroup(mspConfigs map[string]*msp.MSPConfig) (*cb.ConfigGroup, error) {
	channelGroup := utils.NewConfigGroup()
	if len(p.Policies) == 0 {
		addImplicitMetaPolicyDefaults(channelGroup)
	}

	err := addPolicies(channelGroup, p.Policies, channelconfig.AdminsPolicyKey)
	if err != nil {
		return nil, errors.Wrapf(err, "error adding policies to channel group")
	}

	addValue(channelGroup, channelconfig.HashingAlgorithmValue(), channelconfig.AdminsPolicyKey)
	addValue(channelGroup, channelconfig.BlockDataHashingStructureValue(), channelconfig.AdminsPolicyKey)
	if p.Orderer != nil && len(p.Orderer.Addresses) > 0 {
		addValue(channelGroup, channelconfig.OrdererAddressesValue(p.Orderer.Addresses), ordererAdminsPolicyName)
	}

	if p.Consortium != "" {
		addValue(channelGroup, channelconfig.ConsortiumValue(p.Consortium), channelconfig.AdminsPolicyKey)
	}

	if len(p.Capabilities) > 0 {
		addValue(channelGroup, channelconfig.CapabilitiesValue(p.Capabilities), channelconfig.AdminsPolicyKey)
	}

	if p.Orderer != nil {
		channelGroup.Groups[channelconfig.OrdererGroupKey], err = p.NewOrdererGroup(p.Orderer, mspConfigs)
		if err != nil {
			return nil, errors.Wrap(err, "could not create orderer group")
		}
	}

	if p.Application != nil {
		channelGroup.Groups[channelconfig.ApplicationGroupKey], err = NewApplicationGroup(p.Application)
		if err != nil {
			return nil, errors.Wrap(err, "could not create application group")
		}
	}

	if p.Consortiums != nil {
		channelGroup.Groups[channelconfig.ConsortiumsGroupKey], err = NewConsortiumsGroup(p.Consortiums)
		if err != nil {
			return nil, errors.Wrap(err, "could not create consortiums group")
		}
	}

	channelGroup.ModPolicy = channelconfig.AdminsPolicyKey
	return channelGroup, nil
}

func (p *Profile) NewOrdererGroup(conf *Orderer, mspConfigs map[string]*msp.MSPConfig) (*cb.ConfigGroup, error) {
	ordererGroup := utils.NewConfigGroup()
	if len(conf.Policies) == 0 {
		addImplicitMetaPolicyDefaults(ordererGroup)
	} else {
		if err := addPolicies(ordererGroup, conf.Policies, channelconfig.AdminsPolicyKey); err != nil {
			return nil, errors.Wrapf(err, "error adding policies to orderer group")
		}
	}
	ordererGroup.Policies[BlockValidationPolicyKey] = &cb.ConfigPolicy{
		Policy:    policies.ImplicitMetaAnyPolicy(channelconfig.WritersPolicyKey).Value(),
		ModPolicy: channelconfig.AdminsPolicyKey,
	}
	addValue(ordererGroup, channelconfig.BatchSizeValue(
		conf.BatchSize.MaxMessageCount,
		conf.BatchSize.AbsoluteMaxBytes,
		conf.BatchSize.PreferredMaxBytes,
	), channelconfig.AdminsPolicyKey)
	addValue(ordererGroup, channelconfig.BatchTimeoutValue(conf.BatchTimeout.String()), channelconfig.AdminsPolicyKey)
	addValue(ordererGroup, channelconfig.ChannelRestrictionsValue(conf.MaxChannels), channelconfig.AdminsPolicyKey)

	if len(conf.Capabilities) > 0 {
		addValue(ordererGroup, channelconfig.CapabilitiesValue(conf.Capabilities), channelconfig.AdminsPolicyKey)
	}

	var consensusMetadata []byte
	switch conf.OrdererType {
	case ConsensusTypeSolo:
		// nothing to be done here
	case ConsensusTypeKafka:
		// nothing to be done here
	case ConsensusTypeEtcdRaft:
		cm, err := proto.Marshal(p.Orderer.EtcdRaft)
		if err != nil {
			return nil, err
		}
		consensusMetadata = cm
	default:
		return nil, errors.Errorf("unknown orderer type: %s", conf.OrdererType)
	}

	addValue(ordererGroup, channelconfig.ConsensusTypeValue(conf.OrdererType, consensusMetadata), channelconfig.AdminsPolicyKey)

	for _, org := range conf.Organizations {
		var err error
		ordererGroup.Groups[org.Name], err = NewOrdererOrgGroup(org, mspConfigs[org.Name])
		if err != nil {
			return nil, errors.Wrap(err, "failed to create orderer org")
		}
	}

	ordererGroup.ModPolicy = channelconfig.AdminsPolicyKey
	return ordererGroup, nil
}

func ValidateOrg(org *Organization) error {
	if org.MSPType == "" {
		return errors.Errorf("failed to provide msp type for org '%s'", org.Name)
	}

	if org.AdminPrincipal == "" {
		return errors.Errorf("failed to provide admin principal")
	}

	return nil
}

// NewOrdererOrgGroup returns an orderer org component of the channel configuration.  It defines the crypto material for the
// organization (its MSP).  It sets the mod_policy of all elements to "Admins".
func NewOrdererOrgGroup(conf *Organization, mspConfig *msp.MSPConfig) (*cb.ConfigGroup, error) {
	ordererOrgGroup := utils.NewConfigGroup()
	if len(conf.Policies) == 0 {
		addSignaturePolicyDefaults(ordererOrgGroup, conf.ID, conf.AdminPrincipal != AdminRoleAdminPrincipal)
	} else {
		if err := addPolicies(ordererOrgGroup, conf.Policies, channelconfig.AdminsPolicyKey); err != nil {
			return nil, errors.Wrapf(err, "error adding policies to orderer org group '%s'", conf.Name)
		}
	}

	addValue(ordererOrgGroup, channelconfig.MSPValue(mspConfig), channelconfig.AdminsPolicyKey)

	ordererOrgGroup.ModPolicy = channelconfig.AdminsPolicyKey

	if len(conf.OrdererEndpoints) > 0 {
		addValue(ordererOrgGroup, channelconfig.EndpointsValue(conf.OrdererEndpoints), channelconfig.AdminsPolicyKey)
	}

	return ordererOrgGroup, nil
}

func addValue(cg *cb.ConfigGroup, value channelconfig.ConfigValue, modPolicy string) {
	cg.Values[value.Key()] = &cb.ConfigValue{
		Value:     utils.MarshalOrPanic(value.Value()),
		ModPolicy: modPolicy,
	}
}

func addPolicy(cg *cb.ConfigGroup, policy policies.ConfigPolicy, modPolicy string) {
	cg.Policies[policy.Key()] = &cb.ConfigPolicy{
		Policy:    policy.Value(),
		ModPolicy: modPolicy,
	}
}

func addPolicies(cg *cb.ConfigGroup, policyMap map[string]*Policy, modPolicy string) error {
	for policyName, policy := range policyMap {
		switch policy.Type {
		case ImplicitMetaPolicyType:
			imp, err := policies.ImplicitMetaFromString(policy.Rule)
			if err != nil {
				return errors.Wrapf(err, "invalid implicit meta policy rule '%s'", policy.Rule)
			}
			cg.Policies[policyName] = &cb.ConfigPolicy{
				ModPolicy: modPolicy,
				Policy: &cb.Policy{
					Type:  int32(cb.Policy_IMPLICIT_META),
					Value: utils.MarshalOrPanic(imp),
				},
			}
		case SignaturePolicyType:
			sp, err := policydsl.FromString(policy.Rule)
			if err != nil {
				return errors.Wrapf(err, "invalid signature policy rule '%s'", policy.Rule)
			}
			cg.Policies[policyName] = &cb.ConfigPolicy{
				ModPolicy: modPolicy,
				Policy: &cb.Policy{
					Type:  int32(cb.Policy_SIGNATURE),
					Value: utils.MarshalOrPanic(sp),
				},
			}
		default:
			return errors.Errorf("unknown policy type: %s", policy.Type)
		}
	}
	return nil
}

func addImplicitMetaPolicyDefaults(cg *cb.ConfigGroup) {
	addPolicy(cg, policies.ImplicitMetaMajorityPolicy(channelconfig.AdminsPolicyKey), channelconfig.AdminsPolicyKey)
	addPolicy(cg, policies.ImplicitMetaAnyPolicy(channelconfig.ReadersPolicyKey), channelconfig.AdminsPolicyKey)
	addPolicy(cg, policies.ImplicitMetaAnyPolicy(channelconfig.WritersPolicyKey), channelconfig.AdminsPolicyKey)
}

func addSignaturePolicyDefaults(cg *cb.ConfigGroup, mspID string, devMode bool) {
	if devMode {
		addPolicy(cg, policies.SignaturePolicy(channelconfig.AdminsPolicyKey, policydsl.SignedByMspMember(mspID)), channelconfig.AdminsPolicyKey)
	} else {
		addPolicy(cg, policies.SignaturePolicy(channelconfig.AdminsPolicyKey, policydsl.SignedByMspAdmin(mspID)), channelconfig.AdminsPolicyKey)
	}
	addPolicy(cg, policies.SignaturePolicy(channelconfig.ReadersPolicyKey, policydsl.SignedByMspMember(mspID)), channelconfig.AdminsPolicyKey)
	addPolicy(cg, policies.SignaturePolicy(channelconfig.WritersPolicyKey, policydsl.SignedByMspMember(mspID)), channelconfig.AdminsPolicyKey)
}
