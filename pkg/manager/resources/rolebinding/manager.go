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

package rolebinding

import (
	"context"
	"fmt"

	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/role"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/serviceaccount"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("rolebinding_manager")

type Manager struct {
	Client          k8sclient.Client
	Scheme          *runtime.Scheme
	RoleBindingFile string
	Name            string

	LabelsFunc   func(v1.Object) map[string]string
	OverrideFunc func(v1.Object, *rbacv1.RoleBinding, resources.Action) error
}

func (m *Manager) GetName(instance v1.Object) string {
	if m.Name != "" {
		return fmt.Sprintf("%s-%s-rolebinding", instance.GetName(), m.Name)
	}
	return fmt.Sprintf("%s-rolebinding", instance.GetName())
}

func (m *Manager) Reconcile(instance v1.Object, update bool) error {
	name := m.GetName(instance)
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, &rbacv1.RoleBinding{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Creating role binding '%s'", name))
			roleBinding, err := m.GetRoleBindingBasedOnCRFromFile(instance)
			if err != nil {
				return err
			}

			err = m.Client.Create(context.TODO(), roleBinding, k8sclient.CreateOption{Owner: instance, Scheme: m.Scheme})
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// TODO: If needed, update logic for servie goes here

	return nil
}

func (m *Manager) GetRoleBindingBasedOnCRFromFile(instance v1.Object) (*rbacv1.RoleBinding, error) {
	roleBinding, err := util.GetRoleBindingFromFile(m.RoleBindingFile)
	if err != nil {
		log.Error(err, fmt.Sprintf("Error reading role binding configuration file: %s", m.RoleBindingFile))
		return nil, err
	}

	name := m.GetName(instance)
	roleBinding.Name = name
	roleBinding.RoleRef.Name = role.GetName(instance.GetName())
	roleBinding.Namespace = instance.GetNamespace()
	roleBinding.Labels = m.LabelsFunc(instance)
	roleBinding.Subjects[0].Name = serviceaccount.GetName(instance.GetName())
	roleBinding.Subjects[0].Namespace = instance.GetNamespace()

	return m.BasedOnCR(instance, roleBinding)
}

func (m *Manager) BasedOnCR(instance v1.Object, roleBinding *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
	if m.OverrideFunc != nil {
		err := m.OverrideFunc(instance, roleBinding, resources.Create)
		if err != nil {
			return nil, operatorerrors.New(operatorerrors.InvalidRoleBindingCreateRequest, err.Error())
		}
	}

	return roleBinding, nil
}

func (m *Manager) Get(instance v1.Object) (client.Object, error) {
	if instance == nil {
		return nil, nil // Instance has not been reconciled yet
	}

	name := m.GetName(instance)
	rb := &rbacv1.RoleBinding{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, rb)
	if err != nil {
		return nil, err
	}

	return rb, nil
}

func (m *Manager) Exists(instance v1.Object) bool {
	_, err := m.Get(instance)

	return err == nil
}

func (m *Manager) Delete(instance v1.Object) error {
	rb, err := m.Get(instance)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	if rb == nil {
		return nil
	}

	err = m.Client.Delete(context.TODO(), rb)
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
