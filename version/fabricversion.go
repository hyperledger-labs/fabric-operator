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

package version

import (
	"strings"
)

const (
	// Fabric versions
	V1     = "1"
	V1_0_0 = "1.0.0"
	V1_4_0 = "1.4.0"
	V1_4_6 = "1.4.6"
	V1_4_7 = "1.4.7"
	V1_4_8 = "1.4.8"
	V1_4_9 = "1.4.9"
	V1_5_3 = "1.5.3"
	V2     = "2"
	V2_0_0 = "2.0.0"
	V2_0_1 = "2.0.1"
	V2_1_0 = "2.1.0"
	V2_1_1 = "2.1.1"
	V2_2_0 = "2.2.0"
	V2_2_1 = "2.2.1"
	V2_2_3 = "2.2.3"
	V2_2_4 = "2.2.4"
	V2_2_5 = "2.2.5"

	V2_4_1 = "2.4.1"
	V2_5_1 = "2.5.1"

	V1_4 = "V1.4"

	Unsupported = "unsupported"
)

// OldFabricVersionsLookup map contains old fabric versions keyed
// by image tag. Used to set the fabric version of migrated instances
// that don't have fabric version set in their specs.
// This should not contain newer fabric versions as instances with newer
// fabric versions should have fabric version set in their spec.
var OldFabricVersionsLookup = map[string]interface{}{
	"1.4.2":       nil,
	"1.4.3":       nil,
	"1.4.4":       nil,
	"1.4.5":       nil,
	"1.4.6":       nil,
	"V1.4":        nil,
	"unsupported": nil,
}

// GetFabricVersionFrom extracts fabric version from image tag in the format: <version>-<releasedate>-<arch>
func GetFabricVersionFrom(imageTag string) string {
	tagItems := strings.Split(imageTag, "-")
	if len(tagItems) != 3 {
		// Newer tags use sha256 digests, from which
		// versions cannot be extracted.
		return ""
	}

	fabVersion := tagItems[0]
	return fabVersion
}

// GetOldFabricVersionFrom is only to be used when we need to find the
// fabric version of a migrated instance where instance.Spec.FabricVersion
// was not set previously.
func GetOldFabricVersionFrom(imageTag string) string {
	version := GetFabricVersionFrom(imageTag)

	_, found := OldFabricVersionsLookup[version]
	if !found {
		return Unsupported
	}

	return version
}

// IsMigratedFabricVersion returns true if the given fabric version
// was set during migration to 2.5.2 or above
func IsMigratedFabricVersion(fabricVersion string) bool {
	_, found := OldFabricVersionsLookup[fabricVersion]
	return found
}
