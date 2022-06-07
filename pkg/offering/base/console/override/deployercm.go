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
	"encoding/json"
	"errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/defaultconfig/console"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func (o *Override) DeployerCM(object v1.Object, cm *corev1.ConfigMap, action resources.Action, options map[string]interface{}) error {
	instance := object.(*current.IBPConsole)
	switch action {
	case resources.Create:
		return o.CreateDeployerCM(instance, cm, options)
	case resources.Update:
		return o.UpdateDeployerCM(instance, cm, options)
	}

	return nil
}

func (o *Override) CreateDeployerCM(instance *current.IBPConsole, cm *corev1.ConfigMap, options map[string]interface{}) error {
	return errors.New("no create deployer cm defined, this needs to implemented")
}

func (o *Override) UpdateDeployerCM(instance *current.IBPConsole, cm *corev1.ConfigMap, options map[string]interface{}) error {
	data := cm.Data["settings.yaml"]

	config := &deployer.Config{}
	err := yaml.Unmarshal([]byte(data), config)
	if err != nil {
		return err
	}

	err = CommonDeployerCM(instance, config, options)
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

func CommonDeployerCM(instance *current.IBPConsole, config *deployer.Config, options map[string]interface{}) error {
	if len(instance.Spec.ImagePullSecrets) == 0 {
		return errors.New("no image pull secret provided")
	}

	if instance.Spec.NetworkInfo == nil || instance.Spec.NetworkInfo.Domain == "" {
		return errors.New("no domain provided")
	}

	config.ImagePullSecrets = instance.Spec.ImagePullSecrets
	config.Domain = instance.Spec.NetworkInfo.Domain

	if instance.Spec.Deployer != nil {
		if instance.Spec.Deployer.CreateDB {
			config.Database.CreateDB = instance.Spec.Deployer.CreateDB
		}
		if instance.Spec.Deployer.ComponentsDB != "" {
			config.Database.Components.Name = instance.Spec.Deployer.ComponentsDB
		}
		if instance.Spec.Deployer.ConnectionString != "" {
			config.Database.ConnectionURL = instance.Spec.Deployer.ConnectionString
		}
	}

	registryURL := instance.Spec.RegistryURL
	arch := "amd64"
	if instance.Spec.Arch != nil && len(instance.Spec.Arch) > 0 {
		arch = instance.Spec.Arch[0]
	}

	if instance.Spec.UseTags != nil && instance.Spec.UseTags != config.UseTags {
		config.UseTags = instance.Spec.UseTags
	}

	requestedVersions := &deployer.Versions{}
	if instance.Spec.Versions != nil {
		// convert spec version to deployer config versions
		instanceVersionBytes, err := json.Marshal(instance.Spec.Versions)
		if err != nil {
			return err
		}
		err = json.Unmarshal(instanceVersionBytes, requestedVersions)
		if err != nil {
			return err
		}
	} else {
		// use default config versions
		requestedVersions = config.Versions
	}
	config.Versions.Override(requestedVersions, registryURL, arch)

	images := instance.Spec.Images
	if images == nil {
		images = &current.ConsoleImages{}
	}
	defaultimage := console.GetImages()

	// TODO:OSS what happens if defaultimage is empty
	mustgatherImage := image.GetImage(registryURL, defaultimage.MustgatherImage, images.MustgatherImage)
	mustgatherTag := image.GetTag(arch, defaultimage.MustgatherTag, images.MustgatherTag)

	config.OtherImages = &deployer.OtherImages{
		MustgatherImage: mustgatherImage,
		MustgatherTag:   mustgatherTag,
	}

	config.ServiceAccount = instance.GetName()

	storageClassName := ""
	if instance.Spec.Storage != nil && instance.Spec.Storage.Console != nil {
		storageClassName = instance.Spec.Storage.Console.Class
	}

	config.Defaults.Storage.CA.CA.Class = storageClassName
	config.Defaults.Storage.Peer.Peer.Class = storageClassName
	config.Defaults.Storage.Peer.StateDB.Class = storageClassName
	config.Defaults.Storage.Orderer.Orderer.Class = storageClassName

	crn := instance.Spec.CRN
	if crn != nil {
		config.CRN = &current.CRN{
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
	}

	// used for passing separate domains for optools and deployer
	if instance.Spec.Deployer != nil && instance.Spec.Deployer.Domain != "" {
		config.Domain = instance.Spec.Deployer.Domain
	}

	deployerOverrides, err := instance.Spec.GetOverridesDeployer()
	if err != nil {
		return err
	}
	if deployerOverrides != nil && deployerOverrides.Timeouts != nil {
		config.Timeouts = &deployer.Timeouts{}
		if deployerOverrides.Timeouts.APIServer != 0 {
			config.Timeouts.APIServer = deployerOverrides.Timeouts.APIServer
		}
		if deployerOverrides.Timeouts.Deployment != 0 {
			config.Timeouts.Deployment = deployerOverrides.Timeouts.Deployment
		}
	}

	if options != nil && options["username"] != nil && options["password"] != nil {
		config.Auth.Username = options["username"].(string)
		config.Auth.Password = options["password"].(string)
	}

	return nil
}
