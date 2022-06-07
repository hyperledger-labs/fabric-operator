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

package manager

import (
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/configmap"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/ingress"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/ingressv1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/orderernode"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/pv"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/pvc"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/role"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/rolebinding"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/route"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/service"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/serviceaccount"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Manager struct {
	Client k8sclient.Client
	Scheme *runtime.Scheme
}

func New(client k8sclient.Client, scheme *runtime.Scheme) *Manager {
	return &Manager{
		Client: client,
		Scheme: scheme,
	}
}

func (m *Manager) CreateDeploymentManager(name string, oFunc func(v1.Object, *appsv1.Deployment, resources.Action) error, labelsFunc func(v1.Object) map[string]string, deploymentFile string) *deployment.Manager {
	return &deployment.Manager{
		Client:         m.Client,
		Scheme:         m.Scheme,
		DeploymentFile: deploymentFile,
		LabelsFunc:     labelsFunc,
		Name:           name,
		OverrideFunc:   oFunc,
	}
}

func (m *Manager) CreateServiceManager(name string, oFunc func(v1.Object, *corev1.Service, resources.Action) error, labelsFunc func(v1.Object) map[string]string, serviceFile string) *service.Manager {
	return &service.Manager{
		Client:       m.Client,
		Scheme:       m.Scheme,
		ServiceFile:  serviceFile,
		LabelsFunc:   labelsFunc,
		Name:         name,
		OverrideFunc: oFunc,
	}
}

func (m *Manager) CreatePVCManager(name string, oFunc func(v1.Object, *corev1.PersistentVolumeClaim, resources.Action) error, labelsFunc func(v1.Object) map[string]string, pvcFile string) resources.Manager {
	return &pvc.Manager{
		Client:       m.Client,
		Scheme:       m.Scheme,
		PVCFile:      pvcFile,
		Name:         name,
		LabelsFunc:   labelsFunc,
		OverrideFunc: oFunc,
	}
}

func (m *Manager) CreatePVManager(name string, oFunc func(v1.Object, *corev1.PersistentVolume, resources.Action) error, labelsFunc func(v1.Object) map[string]string) resources.Manager {
	return &pv.Manager{
		Client:       m.Client,
		Scheme:       m.Scheme,
		Name:         name,
		LabelsFunc:   labelsFunc,
		OverrideFunc: oFunc,
	}
}

func (m *Manager) CreateConfigMapManager(name string, oFunc func(v1.Object, *corev1.ConfigMap, resources.Action, map[string]interface{}) error, labelsFunc func(v1.Object) map[string]string, file string, options map[string]interface{}) resources.Manager {
	return &configmap.Manager{
		Client:        m.Client,
		Scheme:        m.Scheme,
		ConfigMapFile: file,
		Name:          name,
		LabelsFunc:    labelsFunc,
		OverrideFunc:  oFunc,
		Options:       options,
	}
}

func (m *Manager) CreateRoleManager(name string, oFunc func(v1.Object, *rbacv1.Role, resources.Action) error, labelsFunc func(v1.Object) map[string]string, file string) resources.Manager {
	return &role.Manager{
		Client:       m.Client,
		Scheme:       m.Scheme,
		RoleFile:     file,
		Name:         name,
		LabelsFunc:   labelsFunc,
		OverrideFunc: oFunc,
	}
}

func (m *Manager) CreateRoleBindingManager(name string, oFunc func(v1.Object, *rbacv1.RoleBinding, resources.Action) error, labelsFunc func(v1.Object) map[string]string, file string) resources.Manager {
	return &rolebinding.Manager{
		Client:          m.Client,
		Scheme:          m.Scheme,
		RoleBindingFile: file,
		Name:            name,
		LabelsFunc:      labelsFunc,
		OverrideFunc:    oFunc,
	}
}

func (m *Manager) CreateServiceAccountManager(name string, oFunc func(v1.Object, *corev1.ServiceAccount, resources.Action) error, labelsFunc func(v1.Object) map[string]string, file string) resources.Manager {
	return &serviceaccount.Manager{
		Client:             m.Client,
		Scheme:             m.Scheme,
		ServiceAccountFile: file,
		Name:               name,
		LabelsFunc:         labelsFunc,
		OverrideFunc:       oFunc,
	}
}

func (m *Manager) CreateRouteManager(name string, oFunc func(v1.Object, *routev1.Route, resources.Action) error, labelsFunc func(v1.Object) map[string]string, file string) resources.Manager {
	return &route.Manager{
		Client:       m.Client,
		Scheme:       m.Scheme,
		RouteFile:    file,
		Name:         name,
		LabelsFunc:   labelsFunc,
		OverrideFunc: oFunc,
	}
}

func (m *Manager) CreateIngressManager(suffix string, oFunc func(v1.Object, *networkingv1.Ingress, resources.Action) error, labelsFunc func(v1.Object) map[string]string, file string) resources.Manager {
	return &ingress.Manager{
		Client:       m.Client,
		Scheme:       m.Scheme,
		IngressFile:  file,
		Suffix:       suffix,
		LabelsFunc:   labelsFunc,
		OverrideFunc: oFunc,
	}
}

func (m *Manager) CreateIngressv1beta1Manager(suffix string, oFunc func(v1.Object, *networkingv1beta1.Ingress, resources.Action) error, labelsFunc func(v1.Object) map[string]string, file string) resources.Manager {
	return &ingressv1beta1.Manager{
		Client:       m.Client,
		Scheme:       m.Scheme,
		IngressFile:  file,
		Suffix:       suffix,
		LabelsFunc:   labelsFunc,
		OverrideFunc: oFunc,
	}
}

func (m *Manager) CreateOrderernodeManager(suffix string, oFunc func(v1.Object, *current.IBPOrderer, resources.Action) error, labelsFunc func(v1.Object) map[string]string, file string) resources.Manager {
	return &orderernode.Manager{
		Client:          m.Client,
		Scheme:          m.Scheme,
		OrdererNodeFile: file,
		LabelsFunc:      labelsFunc,
		OverrideFunc:    oFunc,
	}
}
