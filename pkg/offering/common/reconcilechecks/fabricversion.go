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

package reconcilechecks

import (
	"errors"
	"fmt"

	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common/reconcilechecks/images"
	"github.com/IBM-Blockchain/fabric-operator/version"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("reconcile_checks")

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

// FabricVersionHelper is a helper function meant to be consumed by the different controllers to handle
// events on fabric version and images in specs
func FabricVersionHelper(instance Instance, versions *deployer.Versions, update Update) (bool, error) {
	image := &images.Image{
		Versions: versions,
		// DefaultRegistryURL: "hyperledger", // changing default for OSS
		DefaultArch: "amd64",
	}

	fv := &images.FabricVersion{
		Versions: versions,
	}

	return FabricVersion(instance, update, image, fv)
}

// Image defines the contract with the image checks
//
//go:generate counterfeiter -o mocks/image.go -fake-name Image . Image
type Image interface {
	UpdateRequired(images.Update) bool
	SetDefaults(images.Instance) error
}

// Version defines the contract with the version checks
//
//go:generate counterfeiter -o mocks/version.go -fake-name Version . Version
type Version interface {
	Normalize(images.FabricVersionInstance) string
	Validate(images.FabricVersionInstance) error
}

// FabricVersion is a lower-level call that requires all dependencies to be injected to handle
// events on fabric version and images in specs. It returns back two values, the first return
// value indicates if a spec change has been made. The second return value returns an error.
func FabricVersion(instance Instance, update Update, image Image, fv Version) (bool, error) {
	var requeue bool

	fabricVersion := instance.GetFabricVersion()
	if fabricVersion == "" {
		return false, errors.New("fabric version is not set")
	}

	// If fabric version is changed EXCEPT during migration, or images section is blank, then
	// lookup default images associated with fabric version and update images in instance's spec
	if update.FabricVersionUpdated() {

		// If fabric version update is triggered by migration of operator, then no changes required
		if version.IsMigratedFabricVersion(instance.GetFabricVersion()) {
			return false, nil
		}

		log.Info(fmt.Sprintf("Images to be updated, fabric version changed, new fabric version is '%s'", fabricVersion))
	}

	if !instance.ImagesSet() {
		log.Info(fmt.Sprintf("Images missing, setting to default images based on fabric version '%s'", fabricVersion))
	}

	// If images set, need to do further processing to determine if images need to be updated (overriden) based on events
	// detected on fabric version
	if instance.ImagesSet() {
		required := image.UpdateRequired(update)

		if !required {
			return false, nil
		}
	}

	// Normalize version to x.x.x-x
	normalizedVersion := fv.Normalize(instance)
	if instance.GetFabricVersion() != normalizedVersion {
		instance.SetFabricVersion(normalizedVersion)
		requeue = true
	}

	if err := fv.Validate(instance); err != nil {
		return false, err
	}

	if err := image.SetDefaults(instance); err != nil {
		return false, err
	}
	requeue = true

	return requeue, nil
}
