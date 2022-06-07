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

package global

import (
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	ibpdep "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	ibpjob "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/job"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConfigSetter sets values on all resources created by operator
type ConfigSetter struct {
	Config config.Globals
}

type resource interface {
	UpdateSecurityContextForAllContainers(sc container.SecurityContext)
}

// Apply applies all global configurations
func (cs *ConfigSetter) Apply(obj runtime.Object) {
	cs.UpdateSecurityContextForAllContainers(obj)
}

// UpdateSecurityContextForAllContainers updates the security context for all containers defined on
// resource object
func (cs *ConfigSetter) UpdateSecurityContextForAllContainers(obj runtime.Object) {
	if cs.Config.SecurityContext == nil {
		return
	}

	var resource resource
	switch obj.(type) {
	case *appsv1.Deployment:
		resource = ibpdep.New(obj.(*appsv1.Deployment))
	case *batchv1.Job:
		resource = ibpjob.NewWithDefaults(obj.(*batchv1.Job))
	default:
		return
	}

	resource.UpdateSecurityContextForAllContainers(*cs.Config.SecurityContext)
}
