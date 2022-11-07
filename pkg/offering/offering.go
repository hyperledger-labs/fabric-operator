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

package offering

import (
	"fmt"
	"strings"
)

type Type string

func (t Type) String() string {
	return string(t)
}

const (
	OPENSHIFT Type = "OPENSHIFT"
	K8S       Type = "K8S"
)

func GetType(oType string) (Type, error) {
	switch strings.ToLower(oType) {
	case "openshift":
		return OPENSHIFT, nil
	case "k8s":
		return K8S, nil
	}
	return "", fmt.Errorf("Cluster Type %s not supported", oType)
}
