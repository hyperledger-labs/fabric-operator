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

	consolev1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/console/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func (o *Override) ConsoleCM(object v1.Object, cm *corev1.ConfigMap, action resources.Action, options map[string]interface{}) error {
	instance := object.(*current.IBPConsole)
	switch action {
	case resources.Create:
		return o.CreateConsoleCM(instance, cm, options)
	case resources.Update:
		return o.UpdateConsoleCM(instance, cm, options)
	}

	return nil
}

func (o *Override) CreateConsoleCM(instance *current.IBPConsole, cm *corev1.ConfigMap, options map[string]interface{}) error {
	return errors.New("no create console cm defined, this needs to implemented")
}

func (o *Override) UpdateConsoleCM(instance *current.IBPConsole, cm *corev1.ConfigMap, options map[string]interface{}) error {
	data := cm.Data["settings.yaml"]

	config := &consolev1.ConsoleSettingsConfig{}
	err := yaml.Unmarshal([]byte(data), config)
	if err != nil {
		return err
	}

	err = CommonConsoleCM(instance, config, options)
	if err != nil {
		return err
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	cm.Data["settings.yaml"] = string(bytes)
	return nil
}

func CommonConsoleCM(instance *current.IBPConsole, config *consolev1.ConsoleSettingsConfig, options map[string]interface{}) error {
	config.DynamicConfig = true
	config.IBMID = instance.Spec.IBMID
	config.IAMApiKey = instance.Spec.IAMApiKey
	config.SegmentWriteKey = instance.Spec.SegmentWriteKey
	config.TrustProxy = "loopback, linklocal, uniquelocal"

	if instance.Spec.Email != "" {
		config.Email = instance.Spec.Email
	}

	if instance.Spec.AuthScheme != "" {
		config.AuthScheme = instance.Spec.AuthScheme
	}

	if instance.Spec.AllowDefaultPassword {
		config.AllowDefaultPassword = true
	}

	if instance.Spec.ConfigtxlatorURL != "" {
		config.Configtxlator = instance.Spec.ConfigtxlatorURL
	}

	if instance.Spec.DeployerURL != "" {
		config.DeployerURL = instance.Spec.DeployerURL
	}

	if instance.Spec.DeployerTimeout != 0 {
		config.DeployerTimeout = instance.Spec.DeployerTimeout
	}

	if instance.Spec.Components != "" {
		config.DBCustomNames.Components = instance.Spec.Components
	}

	if instance.Spec.Sessions != "" {
		config.DBCustomNames.Sessions = instance.Spec.Sessions
	}

	if instance.Spec.System != "" {
		config.DBCustomNames.System = instance.Spec.System
	}

	if instance.Spec.SystemChannel != "" {
		config.SystemChannelID = instance.Spec.SystemChannel
	}

	// ensures a default value
	if instance.Spec.Proxying == nil {
		t := true
		instance.Spec.Proxying = &t
	}

	if *instance.Spec.Proxying {
		config.ProxyTLSReqs = "always"
	}

	if instance.Spec.FeatureFlags != nil {
		config.Featureflags = instance.Spec.FeatureFlags
	} else {
		config.Featureflags = &consolev1.FeatureFlags{
			ReadOnlyEnabled:         new(bool),
			ImportOnlyEnabled:       new(bool),
			CreateChannelEnabled:    true,
			RemotePeerConfigEnabled: true,
			TemplatesEnabled:        false,
			CapabilitiesEnabled:     true,
			HighAvailability:        true,
			EnableNodeOU:            true,
			HSMEnabled:              true,
			ScaleRaftNodesEnabled:   true,
			Lifecycle20Enabled:      true,
			Patch14to20Enabled:      true,
			MustgatherEnabled:       true,
			InfraImportOptions: &consolev1.InfraImportOptions{
				SupportedCAs:      []string{OPENSHIFT, K8S},
				SupportedOrderers: []string{OPENSHIFT, K8S},
				SupportedPeers:    []string{OPENSHIFT, K8S},
			},
		}
	}

	if instance.Spec.ClusterData != nil {
		config.ClusterData = instance.Spec.ClusterData
	} else {
		config.ClusterData = &consolev1.IBPConsoleClusterData{}
	}

	if config.ClusterData.Type == "" {
		config.ClusterData.Type = "paid"
	}

	crn := instance.Spec.CRN
	if crn != nil {
		config.CRN = &consolev1.CRN{
			Version:      crn.Version,
			CName:        crn.CName,
			CType:        crn.CType,
			Servicename:  crn.Servicename,
			Location:     crn.Location,
			AccountID:    crn.AccountID,
			InstanceID:   crn.InstanceID,
			ResourceType: crn.ResourceType,
			ResourceID:   crn.ResourceID,
		}
		config.CRNString = fmt.Sprintf("crn:%s:%s:%s:%s:%s:%s:%s:%s:%s",
			crn.Version, crn.CName, crn.CType, crn.Servicename, crn.Location, crn.AccountID, crn.InstanceID, crn.ResourceType, crn.ResourceID)
	}

	consoleOverrides, err := instance.Spec.GetOverridesConsole()
	if err != nil {
		return err
	}

	if consoleOverrides.ActivityTrackerConsolePath != "" {
		config.ActivityTrackerPath = consoleOverrides.ActivityTrackerConsolePath
	}

	// This field is to indicate if the new way of setting up HSM
	// with init sidecar is enabled
	if consoleOverrides.HSM != "" && consoleOverrides.HSM != "false" {
		config.HSM = "true"
	}

	if options != nil && options["username"] != nil && options["password"] != nil {
		config.DeployerURL = fmt.Sprintf("http://%s:%s@localhost:8080", options["username"].(string), options["password"].(string))
	}

	return nil
}
