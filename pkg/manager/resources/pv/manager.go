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

package pv

import (
	"context"
	"fmt"

	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("pv_manager")

type Manager struct {
	Client k8sclient.Client
	Scheme *runtime.Scheme
	Name   string

	LabelsFunc   func(v1.Object) map[string]string
	OverrideFunc func(v1.Object, *corev1.PersistentVolume, resources.Action) error
}

func (m *Manager) GetName(instance v1.Object) string {
	return fmt.Sprintf("%s-%s", instance.GetNamespace(), instance.GetName())
}

func (m *Manager) Reconcile(instance v1.Object, update bool) error {
	name := m.GetName(instance)
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, &corev1.PersistentVolume{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Creating pv '%s'", name))
			pv, err := m.GetPVFromTemplate(instance)
			if err != nil {
				return err
			}

			err = m.Client.Create(context.TODO(), pv, k8sclient.CreateOption{Owner: instance, Scheme: m.Scheme})
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

func (m *Manager) GetPVFromTemplate(instance v1.Object) (*corev1.PersistentVolume, error) {
	pvc := &corev1.PersistentVolume{
		ObjectMeta: v1.ObjectMeta{
			Name:      m.GetName(instance),
			Namespace: instance.GetNamespace(),
			Labels:    m.LabelsFunc(instance),
		},
	}

	return m.BasedOnCR(instance, pvc)
}

func (m *Manager) BasedOnCR(instance v1.Object, pvc *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	if m.OverrideFunc != nil {
		err := m.OverrideFunc(instance, pvc, resources.Create)
		if err != nil {
			return nil, operatorerrors.New(operatorerrors.InvalidPVCCreateRequest, err.Error())
		}
	}

	return pvc, nil
}

func (m *Manager) Get(instance v1.Object) (client.Object, error) {
	if instance == nil {
		return nil, nil // Instance has not been reconciled yet
	}

	name := m.GetName(instance)
	pvc := &corev1.PersistentVolume{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, pvc)
	if err != nil {
		return nil, err
	}

	return pvc, nil
}

func (m *Manager) Exists(instance v1.Object) bool {
	_, err := m.Get(instance)
	return err == nil
}

func (m *Manager) Delete(instance v1.Object) error {
	pvc, err := m.Get(instance)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	if pvc == nil {
		return nil
	}

	err = m.Client.Delete(context.TODO(), pvc)
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
