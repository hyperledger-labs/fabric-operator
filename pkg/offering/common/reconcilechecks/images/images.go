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

package images

import (
	"encoding/json"
	"fmt"
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("image_checks")

//go:generate counterfeiter -o mocks/instance.go -fake-name Instance . Instance

// Instance is an instance of an IBP custom resource
type Instance interface {
	GetArch() []string
	GetRegistryURL() string
	GetFabricVersion() string
	SetFabricVersion(string)
	ImagesSet() bool
}

//go:generate counterfeiter -o mocks/update.go -fake-name Update . Update

// Update defines update events we are interested in
type Update interface {
	ImagesUpdated() bool
	FabricVersionUpdated() bool
}

// Image handles checks and defaults on versions of images
type Image struct {
	Versions           *deployer.Versions
	DefaultRegistryURL string
	DefaultArch        string
}

// SetDefaults sets defaults on instance based on fabric version
func (i *Image) SetDefaults(instance Instance) error {
	if !strings.Contains(instance.GetFabricVersion(), "-") {
		return fmt.Errorf("fabric version format '%s' is not valid, must pass hyphenated version (e.g. 2.2.1-1)", instance.GetFabricVersion())
	}

	arch := i.DefaultArch
	if len(instance.GetArch()) > 0 {
		arch = instance.GetArch()[0]
	}

	registryURL := i.DefaultRegistryURL
	if instance.GetRegistryURL() != "" {
		registryURL = instance.GetRegistryURL()
	}

	// Add '/' at the end if not present in registry URL
	if registryURL != "" && !strings.HasSuffix(registryURL, "/") {
		registryURL = registryURL + "/"
	}

	switch instance.(type) {
	case *current.IBPCA:
		return setDefaultCAImages(instance.(*current.IBPCA), arch, registryURL, i.Versions.CA)
	case *current.IBPPeer:
		return setDefaultPeerImages(instance.(*current.IBPPeer), arch, registryURL, i.Versions.Peer)
	case *current.IBPOrderer:
		return setDefaultOrdererImages(instance.(*current.IBPOrderer), arch, registryURL, i.Versions.Orderer)
	}

	return nil
}

// UpdateRequired process update events to determine if images needed to be updated.
func (i *Image) UpdateRequired(update Update) bool {
	if update.ImagesUpdated() {
		return false
	}

	// If neither fabric version nor images updated or both fabric version and images updated, return since no changes
	// made or required
	if !update.ImagesUpdated() && !update.FabricVersionUpdated() {
		return false
	}

	if update.FabricVersionUpdated() {
		return true
	}

	return false
}

func normalizeFabricVersion(fabricVersion string, versions interface{}) string {
	switch versions.(type) {
	case map[string]deployer.VersionCA:
		if !strings.Contains(fabricVersion, "-") {
			for version, config := range versions.(map[string]deployer.VersionCA) {
				if strings.HasPrefix(version, fabricVersion) && config.Default {
					return version
				}
			}
		}
	case map[string]deployer.VersionPeer:
		if !strings.Contains(fabricVersion, "-") {
			for version, config := range versions.(map[string]deployer.VersionPeer) {
				if strings.HasPrefix(version, fabricVersion) && config.Default {
					return version
				}
			}
		}
	case map[string]deployer.VersionOrderer:
		if !strings.Contains(fabricVersion, "-") {
			for version, config := range versions.(map[string]deployer.VersionOrderer) {
				if strings.HasPrefix(version, fabricVersion) && config.Default {
					return version
				}
			}
		}
	}

	return fabricVersion
}

func setDefaultCAImages(instance *current.IBPCA, arch, registryURL string, versions map[string]deployer.VersionCA) error {
	fabricVersion := instance.Spec.FabricVersion
	log.Info(fmt.Sprintf("Using default images for instance '%s' for fabric version '%s'", instance.GetName(), fabricVersion))

	version, found := versions[fabricVersion]
	if !found {
		return fmt.Errorf("no default CA images defined for fabric version '%s'", fabricVersion)
	}

	version.Image.Override(nil, registryURL, arch)
	specVersions := &current.CAImages{}
	versionBytes, err := json.Marshal(version.Image)
	if err != nil {
		return err
	}
	err = json.Unmarshal(versionBytes, specVersions)
	if err != nil {
		return err
	}
	instance.Spec.Images = specVersions

	return nil
}

func setDefaultPeerImages(instance *current.IBPPeer, arch, registryURL string, versions map[string]deployer.VersionPeer) error {
	fabricVersion := instance.Spec.FabricVersion
	log.Info(fmt.Sprintf("Using default images for instance '%s' for fabric version '%s'", instance.GetName(), fabricVersion))

	version, found := versions[fabricVersion]
	if !found {
		return fmt.Errorf("no default Peer images defined for fabric version '%s'", fabricVersion)
	}

	version.Image.Override(nil, registryURL, arch)
	specVersions := &current.PeerImages{}
	versionBytes, err := json.Marshal(version.Image)
	if err != nil {
		return err
	}
	err = json.Unmarshal(versionBytes, specVersions)
	if err != nil {
		return err
	}
	instance.Spec.Images = specVersions

	return nil
}

func setDefaultOrdererImages(instance *current.IBPOrderer, arch, registryURL string, versions map[string]deployer.VersionOrderer) error {
	fabricVersion := instance.Spec.FabricVersion
	log.Info(fmt.Sprintf("Using default images for instance '%s' for fabric version '%s'", instance.GetName(), fabricVersion))

	version, found := versions[fabricVersion]
	if !found {
		return fmt.Errorf("no default Orderer images defined for fabric version '%s'", fabricVersion)
	}

	version.Image.Override(nil, registryURL, arch)

	specVersions := &current.OrdererImages{}
	versionBytes, err := json.Marshal(version.Image)
	if err != nil {
		return err
	}
	err = json.Unmarshal(versionBytes, specVersions)
	if err != nil {
		return err
	}
	instance.Spec.Images = specVersions

	return nil
}
