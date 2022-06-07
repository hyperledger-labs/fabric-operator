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

package resources

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Action string

const (
	Create  Action = "CREATE"
	Update  Action = "UPDATE"
	Restart Action = "RESTART"
)

//go:generate counterfeiter -o mocks/resource_manager.go -fake-name ResourceManager . Manager

type Manager interface {
	Reconcile(v1.Object, bool) error
	CheckState(v1.Object) error
	RestoreState(v1.Object) error
	Exists(v1.Object) bool
	Get(v1.Object) (client.Object, error)
	Delete(v1.Object) error
	GetName(v1.Object) string
	SetCustomName(string)
}
