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

package deployment

import (
	"fmt"

	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func New(deployment *v1.Deployment) *Deployment {
	return &Deployment{
		Deployment: deployment,
	}
}

type Deployment struct {
	*v1.Deployment
}

func (d *Deployment) RemoveContainer(name string) {
	for i, c := range d.Deployment.Spec.Template.Spec.Containers {
		if c.Name == name {
			if i == len(d.Deployment.Spec.Template.Spec.Containers)-1 {
				d.Deployment.Spec.Template.Spec.Containers = d.Deployment.Spec.Template.Spec.Containers[:len(d.Deployment.Spec.Template.Spec.Containers)-1]
				return
			}

			d.Deployment.Spec.Template.Spec.Containers = append(
				d.Deployment.Spec.Template.Spec.Containers[:i],
				d.Deployment.Spec.Template.Spec.Containers[i+1:]...)
			return
		}
	}
}

func (d *Deployment) UpdateContainer(update container.Container) {
	for i, c := range d.Deployment.Spec.Template.Spec.Containers {
		if c.Name == update.Name {
			d.Deployment.Spec.Template.Spec.Containers[i] = *update.Container
			return
		}
	}
}

func (d *Deployment) UpdateInitContainer(update container.Container) {
	for i, c := range d.Deployment.Spec.Template.Spec.InitContainers {
		if c.Name == update.Name {
			d.Deployment.Spec.Template.Spec.InitContainers[i] = *update.Container
			return
		}
	}
}

func (d *Deployment) AddContainer(add container.Container) {
	d.Deployment.Spec.Template.Spec.Containers = util.AppendContainerIfMissing(d.Deployment.Spec.Template.Spec.Containers, *add.Container)
}

func (d *Deployment) AddInitContainer(add container.Container) {
	d.Deployment.Spec.Template.Spec.InitContainers = util.AppendContainerIfMissing(d.Deployment.Spec.Template.Spec.InitContainers, *add.Container)
}

func (d *Deployment) ContainerNames() []string {
	names := []string{}
	for _, c := range d.Deployment.Spec.Template.Spec.Containers {
		names = append(names, c.Name)
	}
	for _, c := range d.Deployment.Spec.Template.Spec.InitContainers {
		names = append(names, c.Name)
	}
	return names
}

func (d *Deployment) GetContainers() map[string]container.Container {
	return container.LoadFromDeployment(d.Deployment)
}

func (d *Deployment) MustGetContainer(name string) container.Container {
	cont, _ := d.GetContainer(name)
	return cont
}

func (d *Deployment) GetContainer(name string) (cont container.Container, err error) {
	for i, c := range d.Deployment.Spec.Template.Spec.Containers {
		if c.Name == name {
			cont = container.Container{Container: &d.Deployment.Spec.Template.Spec.Containers[i]}
			return
		}
	}
	for i, c := range d.Deployment.Spec.Template.Spec.InitContainers {
		if c.Name == name {
			cont = container.Container{Container: &d.Deployment.Spec.Template.Spec.InitContainers[i]}
			return
		}
	}
	return cont, fmt.Errorf("container '%s' not found", name)
}

func (d *Deployment) ContainerExists(name string) bool {
	_, found := d.GetContainers()[name]
	return found
}

func (d *Deployment) SetServiceAccountName(name string) {
	d.Deployment.Spec.Template.Spec.ServiceAccountName = name
}

func (d *Deployment) SetImagePullSecrets(pullSecrets []string) {
	if len(pullSecrets) > 0 {
		d.Deployment.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{}

		for _, pullSecret := range pullSecrets {
			imagePullSecret := corev1.LocalObjectReference{
				Name: pullSecret,
			}
			d.Deployment.Spec.Template.Spec.ImagePullSecrets = util.AppendImagePullSecretIfMissing(d.Deployment.Spec.Template.Spec.ImagePullSecrets, imagePullSecret)
		}
	}
}

func (d *Deployment) AppendPullSecret(imagePullSecret corev1.LocalObjectReference) {
	d.Deployment.Spec.Template.Spec.ImagePullSecrets = util.AppendImagePullSecretIfMissing(d.Deployment.Spec.Template.Spec.ImagePullSecrets, imagePullSecret)
}

func (d *Deployment) AppendVolumeIfMissing(volume corev1.Volume) {
	d.Deployment.Spec.Template.Spec.Volumes = util.AppendVolumeIfMissing(d.Deployment.Spec.Template.Spec.Volumes, volume)
}

func (d *Deployment) AppendPVCVolumeIfMissing(name, claimName string) {
	volume := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	}
	d.AppendVolumeIfMissing(volume)
}

func (d *Deployment) AppendConfigMapVolumeIfMissing(name, localObjReferenceName string) {
	volume := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: localObjReferenceName,
				},
			},
		},
	}
	d.AppendVolumeIfMissing(volume)
}

func (d *Deployment) AppendSecretVolumeIfMissing(name, secretName string) {
	volume := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
	d.AppendVolumeIfMissing(volume)
}

func (d *Deployment) AppendEmptyDirVolumeIfMissing(name string, storageMedium corev1.StorageMedium) {
	volume := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: storageMedium,
			},
		},
	}
	d.AppendVolumeIfMissing(volume)
}

func (d *Deployment) AppendHostPathVolumeIfMissing(name, hostPath string, hostPathType corev1.HostPathType) {
	volume := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: hostPath,
				Type: &hostPathType,
			},
		},
	}
	d.AppendVolumeIfMissing(volume)
}

func (d *Deployment) SetAffinity(affinity *corev1.Affinity) {
	d.Deployment.Spec.Template.Spec.Affinity = affinity
}

func (d *Deployment) SetReplicas(replicas *int32) {
	d.Deployment.Spec.Replicas = replicas
}

func (d *Deployment) SetStrategy(strategyType appsv1.DeploymentStrategyType) {
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
	}
	d.Deployment.Spec.Strategy = strategy
}

// UpdateSecurityContextForAllContainers updates the security context for all containers defined
// in the deployment
func (d *Deployment) UpdateSecurityContextForAllContainers(sc container.SecurityContext) {
	for i := range d.Spec.Template.Spec.InitContainers {
		container.UpdateSecurityContext(&d.Spec.Template.Spec.InitContainers[i], sc)
	}

	for i := range d.Spec.Template.Spec.Containers {
		container.UpdateSecurityContext(&d.Spec.Template.Spec.Containers[i], sc)
	}
}
