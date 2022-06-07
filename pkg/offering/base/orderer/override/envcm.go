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

package override

import (
	"errors"
	"fmt"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (o *Override) EnvCM(object v1.Object, cm *corev1.ConfigMap, action resources.Action, options map[string]interface{}) error {
	instance := object.(*current.IBPOrderer)
	switch action {
	case resources.Create:
		return o.CreateEnvCM(instance, cm)
	case resources.Update:
		return o.UpdateEnvCM(instance, cm)
	}

	return nil
}

func (o *Override) CreateEnvCM(instance *current.IBPOrderer, cm *corev1.ConfigMap) error {
	genesisProfile := instance.Spec.GenesisProfile
	if genesisProfile == "" {
		genesisProfile = "Initial"
	}
	cm.Data["ORDERER_GENERAL_GENESISPROFILE"] = genesisProfile

	mspID := instance.Spec.MSPID
	if mspID == "" {
		return errors.New("failed to provide MSP ID for orderer")
	}
	cm.Data["ORDERER_GENERAL_LOCALMSPID"] = mspID

	if version.GetMajorReleaseVersion(instance.Spec.FabricVersion) == version.V2 {
		if instance.Spec.IsUsingChannelLess() {
			cm.Data["ORDERER_GENERAL_BOOTSTRAPMETHOD"] = "none"
		} else {
			cm.Data["ORDERER_GENERAL_BOOTSTRAPMETHOD"] = "file"
			cm.Data["ORDERER_GENERAL_BOOTSTRAPFILE"] = "/certs/genesis/orderer.block"
		}
	} else {
		cm.Data["ORDERER_GENERAL_GENESISMETHOD"] = "file"
		cm.Data["ORDERER_GENERAL_GENESISFILE"] = "/certs/genesis/orderer.block"
	}

	intermediateExists := util.IntermediateSecretExists(o.Client, instance.Namespace, fmt.Sprintf("ecert-%s-intercerts", instance.Name)) &&
		util.IntermediateSecretExists(o.Client, instance.Namespace, fmt.Sprintf("tls-%s-intercerts", instance.Name))
	intercertPath := "/certs/msp/tlsintermediatecerts/intercert-0.pem"
	if intermediateExists {
		cm.Data["ORDERER_GENERAL_TLS_ROOTCAS"] = intercertPath
		cm.Data["ORDERER_OPERATIONS_TLS_ROOTCAS"] = intercertPath
		cm.Data["ORDERER_OPERATIONS_TLS_CLIENTROOTCAS"] = intercertPath
		cm.Data["ORDERER_GENERAL_CLUSTER_ROOTCAS"] = intercertPath
	}
	// Add configs for 2.4.x
	// Add default cert location for admin service
	currentVer := version.String(instance.Spec.FabricVersion)
	if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.GreaterThan(version.V2_4_1) {
		// Enable Channel participation for 2.4.x orderers
		cm.Data["ORDERER_CHANNELPARTICIPATION_ENABLED"] = "true"

		cm.Data["ORDERER_ADMIN_TLS_ENABLED"] = "true"
		cm.Data["ORDERER_ADMIN_TLS_CERTIFICATE"] = "/certs/tls/signcerts/cert.pem"
		cm.Data["ORDERER_ADMIN_TLS_PRIVATEKEY"] = "/certs/tls/keystore/key.pem"
		cm.Data["ORDERER_ADMIN_TLS_CLIENTAUTHREQUIRED"] = "true"
		// override the default value 127.0.0.1:9443
		cm.Data["ORDERER_ADMIN_LISTENADDRESS"] = "0.0.0.0:9443"
		if intermediateExists {
			// override intermediate cert paths for root and clientroot cas
			cm.Data["ORDERER_ADMIN_TLS_ROOTCAS"] = intercertPath
			cm.Data["ORDERER_ADMIN_TLS_CLIENTROOTCAS"] = intercertPath
		} else {
			cm.Data["ORDERER_ADMIN_TLS_ROOTCAS"] = "/certs/msp/tlscacerts/cacert-0.pem"
			cm.Data["ORDERER_ADMIN_TLS_CLIENTROOTCAS"] = "/certs/msp/tlscacerts/cacert-0.pem"
		}
	}

	return nil
}

func (o *Override) UpdateEnvCM(instance *current.IBPOrderer, cm *corev1.ConfigMap) error {
	return nil
}
