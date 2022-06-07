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

package config

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	corev1 "k8s.io/api/core/v1"
)

const DAEMON_CHECK_CMD = "while true; do if [ -f /shared/daemon-launched ]; then break; fi; done"

// Resource defines the contract required for adding a daemon init containter on to a kubernetes resource
type Resource interface {
	AddContainer(add container.Container)
	AppendVolumeIfMissing(volume corev1.Volume)
	AppendPullSecret(imagePullSecret corev1.LocalObjectReference)
}

// AddDaemonContainer appends an init container responsible for launching HSM daemon
// as a background process within the processNamespace of the pod
func AddDaemonContainer(config *HSMConfig, resource Resource, contResource corev1.ResourceRequirements, pvcMount *corev1.VolumeMount) {
	t := true
	f := false

	// The daemon needs to be started by root user, otherwise, results in this error:
	// This daemon needs root privileges, but the effective user id is not 'root'.
	user := int64(0)

	cont := corev1.Container{
		Name:            "hsm-daemon",
		Image:           config.Daemon.Image,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                &user,
			RunAsNonRoot:             &f,
			Privileged:               &t,
			AllowPrivilegeEscalation: &t,
		},
		Resources: contResource,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "shared",
				MountPath: "/shared",
			},
		},
		Env: config.Daemon.Envs,
	}

	volumeMounts := config.GetVolumeMounts()
	if pvcMount != nil {
		volumeMounts = append(volumeMounts, *pvcMount)
	}

	cont.VolumeMounts = append(cont.VolumeMounts, volumeMounts...)
	if config.Daemon.Auth != nil {
		resource.AppendPullSecret(config.BuildPullSecret())
	}
	// if securityContext is passed in hsm config override the same
	if config.Daemon.SecurityContext != nil {
		if config.Daemon.SecurityContext.Privileged != nil {
			cont.SecurityContext.Privileged = config.Daemon.SecurityContext.Privileged
		}
		if config.Daemon.SecurityContext.RunAsNonRoot != nil {
			cont.SecurityContext.RunAsNonRoot = config.Daemon.SecurityContext.RunAsNonRoot
		}
		if config.Daemon.SecurityContext.RunAsUser != nil {
			cont.SecurityContext.RunAsUser = config.Daemon.SecurityContext.RunAsUser
		}
		if config.Daemon.SecurityContext.AllowPrivilegeEscalation != nil {
			cont.SecurityContext.AllowPrivilegeEscalation = config.Daemon.SecurityContext.AllowPrivilegeEscalation
		}
	}

	// if resources are passed in hsm config, override
	if config.Daemon.Resources != nil {
		cont.Resources = *config.Daemon.Resources
	}

	resource.AddContainer(container.Container{Container: &cont})
}

// Daemon represents that configuration for the HSM Daemon
type Daemon struct {
	Image           string                       `json:"image"`
	Envs            []corev1.EnvVar              `json:"envs,omitempty"`
	Auth            *Auth                        `json:"auth,omitempty"`
	SecurityContext *container.SecurityContext   `json:"securityContext,omitempty"`
	Resources       *corev1.ResourceRequirements `json:"daemon,omitempty"`
}

// GetEnvs returns environment variables
func (d *Daemon) GetEnvs() []corev1.EnvVar {
	return d.Envs
}

// BuildPullSecret builds the string secret into the type expected by kubernetes
func (d *Daemon) BuildPullSecret() corev1.LocalObjectReference {
	if d.Auth != nil {
		return d.Auth.BuildPullSecret()
	}
	return corev1.LocalObjectReference{}
}
