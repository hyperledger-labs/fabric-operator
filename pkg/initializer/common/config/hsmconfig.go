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
	"context"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

var log = logf.Log.WithName("config")

// Client defines the contract to get resources from clusters
type Client interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object) error
}

// ReadHSMConfig reads hsm configuration from 'ibm-hlfsupport-hsm-config', and key 'ibm-hlfsupport-hsm-config.yaml'
// from data
func ReadHSMConfig(client Client, instance metav1.Object) (*HSMConfig, error) {
	// NOTE: This is hard-coded because this name should never be different
	name := "ibm-hlfsupport-hsm-config"

	cm := &corev1.ConfigMap{}
	err := client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      name,
			Namespace: instance.GetNamespace(),
		},
		cm,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get hsm config 'ibm-hlfsupport-hsm-config'")
	}

	hsmConfig := &HSMConfig{}
	err = yaml.Unmarshal([]byte(cm.Data["ibm-hlfsupport-hsm-config.yaml"]), hsmConfig)
	if err != nil {
		return nil, err
	}

	return hsmConfig, nil
}

// HSMConfig defines the configuration parameters for HSMs
type HSMConfig struct {
	Type       string          `json:"type,omitempty"`
	Version    string          `json:"version,omitempty"`
	Library    Library         `json:"library"`
	MountPaths []MountPath     `json:"mountpaths"`
	Envs       []corev1.EnvVar `json:"envs,omitempty"`
	Daemon     *Daemon         `json:"daemon,omitempty"`
}

// Library represents the configuration for an HSM library
type Library struct {
	FilePath           string `json:"filepath"`
	Image              string `json:"image"`
	AutoUpdateDisabled bool   `json:"autoUpdateDisabled,omitempty"`
	Auth               *Auth  `json:"auth,omitempty"`
}

// BuildPullSecret builds the string secret into the type expected by kubernetes
func (h *HSMConfig) BuildPullSecret() corev1.LocalObjectReference {
	if h.Library.Auth != nil {
		return h.Library.Auth.BuildPullSecret()
	}
	return corev1.LocalObjectReference{}
}

// GetVolumes builds the volume spec into the type expected by kubernetes, by default
// the volume source is empty dir with memory as the storage medium
func (h *HSMConfig) GetVolumes() []corev1.Volume {
	volumes := []corev1.Volume{}
	for _, mount := range h.MountPaths {
		// Skip building volume if using PVC, the PVC is known to the caller of method.
		// The caller will build the proper PVC volume by setting the appropriate claim name.
		if !mount.UsePVC {
			volumes = append(volumes, mount.BuildVolume())
		}
	}
	return volumes
}

// GetVolumeMounts builds the volume mount spec into the type expected by kubernetes
func (h *HSMConfig) GetVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}
	for _, mount := range h.MountPaths {
		// Skip building volume mount if using PVC, the PVC is known to the caller of method.
		// The caller will build the proper PVC volume mount with the mount path specified
		// in the HSM config
		if !mount.UsePVC {
			volumeMounts = append(volumeMounts, mount.BuildVolumeMount())
		}
	}
	return volumeMounts
}

// GetEnvs builds the env var spec into the type expected by kubernetes
func (h *HSMConfig) GetEnvs() []corev1.EnvVar {
	return h.Envs
}

// Auth represents the authentication methods that are supported
type Auth struct {
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
	// UserID          string `json:"userid,omitempty"`
	// Password        string `json:"password,omitempty"`
}

// BuildPullSecret builds the pull secret string into the type expected by kubernetes
func (a *Auth) BuildPullSecret() corev1.LocalObjectReference {
	return corev1.LocalObjectReference{Name: a.ImagePullSecret}
}

// MountPath represent the configuration of volume mounts on a container
type MountPath struct {
	Name         string               `json:"name"`
	Secret       string               `json:"secret"`
	MountPath    string               `json:"mountpath"`
	UsePVC       bool                 `json:"usePVC"`
	SubPath      string               `json:"subpath,omitempty"`
	Paths        []Path               `json:"paths,omitempty"`
	VolumeSource *corev1.VolumeSource `json:"volumeSource,omitempty"`
}

type Path struct {
	Key  string `json:"key"`
	Path string `json:"path"`
}

// BuildVolumeMount builds the volume mount spec into the type expected by kubernetes
func (m *MountPath) BuildVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      m.Name,
		MountPath: m.MountPath,
		SubPath:   m.SubPath,
	}
}

// BuildVolume builds the volume spec into the type expected by kubernetes
func (m *MountPath) BuildVolume() corev1.Volume {
	// In our initial HSM implementation, we made secrets as the default volume source and only
	// allowed secrets based volumes. With the introducing of other HSM implementations (opencryptoki),
	// other volume types had to be introduced and are now directly allowed in the configuration.
	// However, to not break current users using older config this logic needs to persistent until
	// we can deprecate older configs.
	if m.VolumeSource == nil {
		m.VolumeSource = &corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: m.Secret,
			},
		}
	}

	// Setting key/path is only supported for secrets, this is due to the fact
	// that we made secrets as the default volume source and only allowed secrets based volumes.
	// For other volume source types, they should configured directly in the hsm config.
	for _, path := range m.Paths {
		m.VolumeSource.Secret.Items = append(m.VolumeSource.Secret.Items,
			corev1.KeyToPath{
				Key:  path.Key,
				Path: path.Path,
			},
		)
	}

	return corev1.Volume{
		Name:         m.Name,
		VolumeSource: *m.VolumeSource,
	}
}
