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

package image

import (
	"fmt"
	"strings"
)

func Format(image, tag string) string {
	if !strings.HasPrefix(tag, "sha256:") {
		return fmt.Sprintf("%s:%s", image, tag)
	} else {
		return fmt.Sprintf("%s@%s", image, tag)
	}
}

func GetImage(registryURL, image, requestedImage string) string {
	// if requested image is passed use it
	// else fallback to using default image
	if requestedImage != "" {
		image = requestedImage
	}
	if image != "" {
		// if registry url is empty or set to `no-registry-url` return image as is
		if registryURL == "" || registryURL == "no-registry-url" || registryURL == "no-registry-url/" {
			// use the image as is
			return image
		}
		if !strings.Contains(image, registryURL) {
			// if image doesn't contain registy url pre-pend the same to image
			image = registryURL + image
		}
	}
	return image
}

func GetTag(arch, tag, requestedTag string) string {
	// if override is passed return it
	// else return default
	if requestedTag != "" {
		return requestedTag
	}

	return tag
}
