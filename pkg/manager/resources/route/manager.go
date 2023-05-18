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

package route

import (
	"context"
	"fmt"

	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("route_manager")

type Manager struct {
	Client    k8sclient.Client
	Scheme    *runtime.Scheme
	RouteFile string
	Name      string

	LabelsFunc   func(v1.Object) map[string]string
	OverrideFunc func(v1.Object, *routev1.Route, resources.Action) error
}

func (m *Manager) GetName(instance v1.Object) string {
	if m.Name != "" {
		return GetName(instance, m.Name)
	}
	return GetName(instance)
}

func (m *Manager) Reconcile(instance v1.Object, update bool) error {
	name := m.GetName(instance)
	route := &routev1.Route{
		TypeMeta: v1.TypeMeta{
			APIVersion: "route.openshift.io/v1",
			Kind:       "Route",
		},
	}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, route)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Creating route '%s'", name))
			route, err := m.GetRouteBasedOnCRFromFile(instance)
			if err != nil {
				return err
			}
			route.TypeMeta.APIVersion = "route.openshift.io/v1"
			route.TypeMeta.Kind = "Route"

			err = m.Client.Create(context.TODO(), route, k8sclient.CreateOption{Owner: instance, Scheme: m.Scheme})
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// TODO: If needed, update logic for route goes here

	return nil
}

func (m *Manager) GetRouteBasedOnCRFromFile(instance v1.Object) (*routev1.Route, error) {
	route, err := util.GetRouteFromFile(m.RouteFile)
	if err != nil {
		log.Error(err, fmt.Sprintf("Error reading route configuration file: %s", m.RouteFile))
		return nil, err
	}

	route.Name = m.GetName(instance)
	route.Namespace = instance.GetNamespace()
	route.Labels = m.LabelsFunc(instance)

	return m.BasedOnCR(instance, route)
}

func (m *Manager) BasedOnCR(instance v1.Object, route *routev1.Route) (*routev1.Route, error) {
	if m.OverrideFunc != nil {
		err := m.OverrideFunc(instance, route, resources.Create)
		if err != nil {
			return nil, errors.Wrap(err, "failed during route override")
		}
	}

	return route, nil
}

func (m *Manager) Get(instance v1.Object) (client.Object, error) {
	if instance == nil {
		return nil, nil // Instance has not been reconciled yet
	}

	name := m.GetName(instance)
	route := &routev1.Route{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, route)
	if err != nil {
		return nil, err
	}

	return route, nil
}

func (m *Manager) Exists(instance v1.Object) bool {
	_, err := m.Get(instance)

	return err == nil
}

func (m *Manager) Delete(instance v1.Object) error {
	route, err := m.Get(instance)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	if route == nil {
		return nil
	}

	err = m.Client.Delete(context.TODO(), route)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (m *Manager) CheckState(instance v1.Object) error {
	// NO-OP
	return nil
}

func (m *Manager) RestoreState(instance v1.Object) error {
	// NO-OP
	return nil
}

func (m *Manager) SetCustomName(name string) {
	// NO-OP
}

func GetName(instance v1.Object, suffix ...string) string {
	if len(suffix) != 0 {
		if suffix[0] != "" {
			return fmt.Sprintf("%s-%s", instance.GetName(), suffix[0])
		}
	}
	return instance.GetName()
}
