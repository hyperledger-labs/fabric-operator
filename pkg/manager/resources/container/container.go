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

package container

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type SecurityContext struct {
	Privileged               *bool
	RunAsNonRoot             *bool
	RunAsUser                *int64
	AllowPrivilegeEscalation *bool
}

func New(container *corev1.Container) *Container {
	return &Container{container}
}

func LoadFromDeployment(deployment *appsv1.Deployment) map[string]Container {
	containers := map[string]Container{}
	for i, c := range deployment.Spec.Template.Spec.Containers {
		containers[c.Name] = Container{&deployment.Spec.Template.Spec.Containers[i]}
	}
	for i, c := range deployment.Spec.Template.Spec.InitContainers {
		containers[c.Name] = Container{&deployment.Spec.Template.Spec.InitContainers[i]}
	}
	return containers
}

func LoadFromFile(file string) (*Container, error) {
	container, err := util.GetContainerFromFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read container file")
	}
	return &Container{container}, nil
}

type Container struct {
	*corev1.Container
}

func (c *Container) DeleteEnv(name string) {
	newEnvs := []corev1.EnvVar{}
	for _, env := range c.Env {
		if env.Name == name {
			continue
		}
		newEnvs = append(newEnvs, env)
	}

	c.Env = newEnvs
}

func (c *Container) UpdateEnv(name, value string) {
	var updated bool

	newEnvs := []corev1.EnvVar{}
	for _, env := range c.Env {
		if env.Name == name {
			env.Value = value
			updated = true
		}
		newEnvs = append(newEnvs, env)
	}

	if updated {
		c.Env = newEnvs
	} else {
		c.Env = append(newEnvs, corev1.EnvVar{Name: name, Value: value})
	}
}

func (c *Container) AppendEnvStructIfMissing(envVar corev1.EnvVar) {
	c.Env = util.AppendEnvIfMissing(c.Env, envVar)
}

func (c *Container) AppendEnvIfMissing(name, value string) {
	envVar := corev1.EnvVar{
		Name:  name,
		Value: value,
	}
	c.Env = util.AppendEnvIfMissing(c.Env, envVar)
}

func (c *Container) AppendEnvIfMissingOverrideIfPresent(name, value string) {
	envVar := corev1.EnvVar{
		Name:  name,
		Value: value,
	}
	c.Env = util.AppendEnvIfMissingOverrideIfPresent(c.Env, envVar)
}

func (c *Container) SetImage(img, tag string) {
	if img != "" {
		if tag != "" {
			c.Container.Image = image.Format(img, tag)
		} else {
			c.Container.Image = img + ":latest"
		}
	}
}

func (c *Container) AppendVolumeMountStructIfMissing(volumeMount corev1.VolumeMount) {
	c.VolumeMounts = util.AppendVolumeMountIfMissing(c.VolumeMounts, volumeMount)
}

func (c *Container) AppendVolumeMountIfMissing(name, mountPath string) {
	volumeMount := corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}
	c.VolumeMounts = util.AppendVolumeMountIfMissing(c.VolumeMounts, volumeMount)
}

func (c *Container) AppendVolumeMountWithSubPathIfMissing(name, mountPath, subPath string) {
	volumeMount := corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		SubPath:   subPath,
	}
	c.VolumeMounts = util.AppendVolumeMountWithSubPathIfMissing(c.VolumeMounts, volumeMount)
}

func (c *Container) AppendVolumeMountWithSubPath(name, mountPath, subPath string) {
	volumeMount := corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		SubPath:   subPath,
	}
	c.VolumeMounts = append(c.VolumeMounts, volumeMount)
}

func (c *Container) SetVolumeMounts(volumeMounts []corev1.VolumeMount) {
	c.VolumeMounts = volumeMounts
}

func (c *Container) UpdateResources(new *corev1.ResourceRequirements) error {
	current := &c.Resources
	update, err := util.GetResourcePatch(current, new)
	if err != nil {
		return errors.Wrap(err, "failed to get resource patch")
	}

	c.Resources = *update
	return nil
}

func (c *Container) SetCommand(command []string) {
	c.Command = command
}

func (c *Container) SetArgs(args []string) {
	c.Args = args
}

func (c *Container) AppendConfigMapFromSourceIfMissing(name string) {
	envFrom := corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
		},
	}
	c.EnvFrom = util.AppendConfigMapFromSourceIfMissing(c.Container.EnvFrom, envFrom)
}

func (c *Container) AppendEnvVarValueFromIfMissing(name string, valueFrom *corev1.EnvVarSource) {
	envVar := corev1.EnvVar{
		Name:      name,
		ValueFrom: valueFrom,
	}
	c.Env = util.AppendEnvIfMissing(c.Container.Env, envVar)
}

func (c *Container) GetEnvs(reqs []string) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}

	for _, env := range c.Env {
		for _, req := range reqs {
			if env.Name == req {
				envVars = append(envVars, env)
			}
		}
	}

	return envVars
}

// UpdateSecurityContext will update the security context of the container
func (c *Container) UpdateSecurityContext(sc SecurityContext) {
	UpdateSecurityContext(c.Container, sc)
}

// UpdateSecurityContext will update the security context of passed in container
func UpdateSecurityContext(c *corev1.Container, sc SecurityContext) {
	if c.SecurityContext == nil {
		c.SecurityContext = &corev1.SecurityContext{}
	}

	c.SecurityContext.Privileged = sc.Privileged
	c.SecurityContext.RunAsNonRoot = sc.RunAsNonRoot
	c.SecurityContext.RunAsUser = sc.RunAsUser
	c.SecurityContext.AllowPrivilegeEscalation = sc.AllowPrivilegeEscalation
}

func (c *Container) SetReadinessProbe(probe *corev1.Probe) {
	c.ReadinessProbe = probe
}

func (c *Container) SetLivenessProbe(probe *corev1.Probe) {
	c.LivenessProbe = probe
}

func (c *Container) SetStartupProbe(probe *corev1.Probe) {
	c.StartupProbe = probe
}
