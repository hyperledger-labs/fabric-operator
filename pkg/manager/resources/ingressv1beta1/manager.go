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

package ingressv1beta1

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"

	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/pkg/errors"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("ingress_manager")

type Manager struct {
	Client      k8sclient.Client
	Scheme      *runtime.Scheme
	IngressFile string
	Suffix      string

	LabelsFunc   func(v1.Object) map[string]string
	OverrideFunc func(v1.Object, *networkingv1beta1.Ingress, resources.Action) error

	routeName string
	Name      string
}

func (m *Manager) Reconcile(instance v1.Object, update bool) error {
	name := instance.GetName()
	if m.Suffix != "" {
		name = fmt.Sprintf("%s-%s", instance.GetName(), m.Suffix)
	}
	ingress := &networkingv1beta1.Ingress{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, ingress)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Creating ingressv1beta1 '%s'", name))
			ingress, err := m.GetIngressBasedOnCRFromFile(instance)
			if err != nil {
				return err
			}
			err = m.Client.Create(context.TODO(), ingress, k8sclient.CreateOption{Owner: instance, Scheme: m.Scheme})
			if err != nil {
				return err
			}

			err = m.UpdateIngressClassName(name, instance)
			if err != nil {
				log.Error(err, "Error updating ingress class name")
				return err
			}

			return nil
		}
		return err
	}

	if update {
		if m.OverrideFunc != nil {
			log.Info(fmt.Sprintf("Updating ingressv1beta1 '%s'", name))
			err := m.OverrideFunc(instance, ingress, resources.Update)
			if err != nil {
				return err
			}

			err = m.Client.Update(context.TODO(), ingress, k8sclient.UpdateOption{Owner: instance, Scheme: m.Scheme})
			if err != nil {
				return err
			}

			err = m.UpdateIngressClassName(name, instance)
			if err != nil {
				log.Error(err, "Error updating ingress class name")
				return err
			}

			return nil
		}
	}

	// TODO: If needed, update logic for servie goes here

	return nil
}

func (m *Manager) Exists(instance v1.Object) bool {
	if instance == nil {
		return false // Instance has not been reconciled yet
	}

	name := instance.GetName()
	if m.Suffix != "" {
		name = fmt.Sprintf("%s-%s", instance.GetName(), m.Suffix)
	}

	ingress := &networkingv1beta1.Ingress{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, ingress)

	return err == nil
}

func (m *Manager) Delete(instance v1.Object) error {
	ingress, err := m.Get(instance)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	if ingress == nil {
		return nil
	}

	err = m.Client.Delete(context.TODO(), ingress)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (m *Manager) Get(instance v1.Object) (client.Object, error) {
	if instance == nil {
		return nil, nil // Instance has not been reconciled yet
	}

	name := m.GetName(instance)
	ingress := &networkingv1beta1.Ingress{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, ingress)
	if err != nil {
		return nil, err
	}

	return ingress, nil
}

func (m *Manager) GetName(instance v1.Object) string {
	if m.Name != "" {
		return fmt.Sprintf("%s-%s", instance.GetName(), m.Name)
	}
	return instance.GetName()
}

func (m *Manager) GetIngressBasedOnCRFromFile(instance v1.Object) (*networkingv1beta1.Ingress, error) {
	ingress, err := util.GetIngressv1beta1FromFile(m.IngressFile)
	if err != nil {
		log.Error(err, fmt.Sprintf("Error reading ingress ingressv1beta1 file: %s", m.IngressFile))
		return nil, err
	}

	return m.BasedOnCR(instance, ingress)
}

func (m *Manager) BasedOnCR(instance v1.Object, ingress *networkingv1beta1.Ingress) (*networkingv1beta1.Ingress, error) {
	if m.OverrideFunc != nil {
		err := m.OverrideFunc(instance, ingress, resources.Create)
		if err != nil {
			return nil, errors.Wrap(err, "failed during ingressv1beta1 override")
		}
	}

	ingress.Name = instance.GetName()
	if m.Suffix != "" {
		ingress.Name = fmt.Sprintf("%s-%s", instance.GetName(), m.Suffix)
	}

	ingress.Namespace = instance.GetNamespace()
	ingress.Labels = m.LabelsFunc(instance)

	return ingress, nil
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

func (m *Manager) UpdateIngressClassName(name string, instance v1.Object) error {
	ingress := &networkingv1beta1.Ingress{}

	// We have to wait for ingress to be available
	// as it fails if this function is called immediately after creation
	log.Info("Waiting for ingressv1beta1 resource to be ready", "ingress", name)

	ingressPollTimeout := 10 * time.Second

	if pollTime := os.Getenv("INGRESS_RESOURCE_POLL_TIMEOUT"); pollTime != "" {
		d, err := time.ParseDuration(pollTime)
		if err != nil {
			return err
		}

		ingressPollTimeout = d
	}

	var errGet error
	err := wait.Poll(500*time.Millisecond, ingressPollTimeout, func() (bool, error) {
		errGet = m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, ingress)
		if errGet != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return err
	}

	ingressClass := ingress.ObjectMeta.Annotations["kubernetes.io/ingress.class"]
	if ingressClass != "" {
		ingress.Spec.IngressClassName = &ingressClass
	}

	log.Info("Updating ingress classname in the ingress resource spec", "ingress", name, "ingressClassName", ingressClass)
	err = m.Client.Update(context.TODO(), ingress, k8sclient.UpdateOption{Owner: instance, Scheme: m.Scheme})
	if err != nil {
		return err
	}

	return nil
}
