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

package override

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	cav1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	dep "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/serviceaccount"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Container names
const (
	INIT      = "init"
	CA        = "ca"
	HSMCLIENT = "hsm-client"
)

func (o *Override) Deployment(object v1.Object, deployment *appsv1.Deployment, action resources.Action) error {
	instance := object.(*current.IBPCA)
	switch action {
	case resources.Create:
		return o.CreateDeployment(instance, deployment)
	case resources.Update:
		return o.UpdateDeployment(instance, deployment)
	}

	return nil
}

func (o *Override) CreateDeployment(instance *current.IBPCA, k8sDep *appsv1.Deployment) error {
	var err error

	if !instance.Spec.License.Accept {
		return errors.New("user must accept license before continuing")
	}

	deployment := dep.New(k8sDep)

	name := instance.GetName()
	deployment.Spec.Template.Spec.ServiceAccountName = serviceaccount.GetName(name)
	err = o.CommonDeployment(instance, deployment)
	if err != nil {
		return err
	}

	caCont, err := deployment.GetContainer(CA)
	if err != nil {
		return errors.New("ca container not found in deployment spec")
	}
	initCont, err := deployment.GetContainer(INIT)
	if err != nil {
		return errors.New("init container not found in deployment spec")
	}

	deployment.SetImagePullSecrets(instance.Spec.ImagePullSecrets)

	if !o.IsPostgres(instance) {
		claimName := instance.Name + "-pvc"
		if instance.Spec.CustomNames.PVC.CA != "" {
			claimName = instance.Spec.CustomNames.PVC.CA
		}
		deployment.AppendPVCVolumeIfMissing("fabric-ca", claimName)

		initCont.AppendVolumeMountWithSubPathIfMissing("fabric-ca", "/data", "fabric-ca-server")
		caCont.AppendVolumeMountWithSubPathIfMissing("fabric-ca", "/data", "fabric-ca-server")
	} else {
		initCont.AppendVolumeMountIfMissing("shared", "/data")
		caCont.AppendVolumeMountIfMissing("shared", "/data")
	}

	deployment.AppendSecretVolumeIfMissing("ca-crypto", instance.Name+"-ca-crypto")
	deployment.AppendSecretVolumeIfMissing("tlsca-crypto", instance.Name+"-tlsca-crypto")
	deployment.AppendConfigMapVolumeIfMissing("ca-config", instance.Name+"-ca-config")
	deployment.AppendConfigMapVolumeIfMissing("tlsca-config", instance.Name+"-tlsca-config")
	deployment.SetAffinity(o.GetAffinity(instance))

	if instance.UsingHSMProxy() {
		caCont.AppendEnvIfMissing("PKCS11_PROXY_SOCKET", instance.Spec.HSM.PKCS11Endpoint)
	} else if instance.IsHSMEnabled() {
		hsmConfig, err := config.ReadHSMConfig(o.Client, instance)
		if err != nil {
			return errors.Wrapf(err, "failed to apply hsm settings to '%s' deployment", instance.GetName())
		}

		hsmSettings(instance, hsmConfig, caCont, deployment)
	}

	return nil
}

func (o *Override) UpdateDeployment(instance *current.IBPCA, k8sDep *appsv1.Deployment) error {
	deployment := dep.New(k8sDep)
	err := o.CommonDeployment(instance, deployment)
	if err != nil {
		return err
	}

	if instance.UsingHSMProxy() {
		caCont := deployment.MustGetContainer(CA)
		caCont.UpdateEnv("PKCS11_PROXY_SOCKET", instance.Spec.HSM.PKCS11Endpoint)
		deployment.UpdateContainer(caCont)
	} else if instance.IsHSMEnabled() {
		hsmInitCont := deployment.MustGetContainer(HSMCLIENT)
		image := instance.Spec.Images
		if image != nil {
			hsmInitCont.SetImage(image.HSMImage, image.HSMTag)
		}
	}

	return nil
}

func (o *Override) CommonDeployment(instance *current.IBPCA, deployment *dep.Deployment) error {
	caCont := deployment.MustGetContainer(CA)
	initCont := deployment.MustGetContainer(INIT)

	if instance.Spec.CAResourcesSet() {
		err := caCont.UpdateResources(instance.Spec.Resources.CA)
		if err != nil {
			return errors.Wrap(err, "update resources for ca failed")
		}
	}

	if instance.Spec.InitResourcesSet() {
		err := initCont.UpdateResources(instance.Spec.Resources.Init)
		if err != nil {
			return errors.Wrap(err, "update resources for init failed")
		}
	}

	image := instance.Spec.Images
	if image != nil {
		caCont.SetImage(image.CAImage, image.CATag)
		initCont.SetImage(image.CAInitImage, image.CAInitTag)
	}

	if o.IsPostgres(instance) {
		deployment.SetStrategy(appsv1.RollingUpdateDeploymentStrategyType)
	}

	// TODO: Find a clean way to check for valid config other than the nested if/else statements
	if instance.Spec.Replicas != nil {
		if *instance.Spec.Replicas > 1 {
			err := o.ValidateConfigOverride(instance.Spec.ConfigOverride)
			if err != nil {
				return err
			}
		}

		deployment.SetReplicas(instance.Spec.Replicas)
	}

	// set seccompProfile to RuntimeDefault
	common.SetPodSecurityContext(caCont)

	return nil
}

func (o *Override) ValidateConfigOverride(configOverride *current.ConfigOverride) error {
	var byteArray *[]byte
	if configOverride == nil {
		return errors.New("Failed to provide override configuration to support greater than 1 replicas")
	}

	if configOverride.CA != nil {
		err := o.ValidateServerConfig(&configOverride.CA.Raw, "CA")
		if err != nil {
			return err
		}
	} else { // if it is nil call with empty bytearray
		err := o.ValidateServerConfig(byteArray, "CA")
		if err != nil {
			return err
		}
	}

	if configOverride.TLSCA != nil {
		err := o.ValidateServerConfig(&configOverride.TLSCA.Raw, "TLSCA")
		if err != nil {
			return err
		}
	} else { // if it is nil call with empty bytearray
		err := o.ValidateServerConfig(byteArray, "TLSCA")
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Override) ValidateServerConfig(byteArray *[]byte, configType string) error {
	if byteArray == nil {
		return errors.New(fmt.Sprintf("Failed to provide database configuration for %s to support greater than 1 replicas", configType))
	}

	overrides := &cav1.ServerConfig{}
	err := json.Unmarshal(*byteArray, overrides)
	if err != nil {
		return err
	}

	if overrides.DB != nil {
		if overrides.DB.Type != "postgres" {
			return errors.New(fmt.Sprintf("DB Type in %s config override should be `postgres` to allow replicas > 1", configType))
		}

		if overrides.DB.Datasource == "" {
			return errors.New(fmt.Sprintf("Datasource in %s config override should not be empty to allow replicas > 1", configType))
		}
	}

	return nil
}

func hsmInitContainer(instance *current.IBPCA, hsmConfig *config.HSMConfig) *container.Container {
	hsmLibraryPath := hsmConfig.Library.FilePath
	hsmLibraryName := filepath.Base(hsmLibraryPath)

	f := false
	user := int64(0)
	mountPath := "/shared"
	cont := &container.Container{
		Container: &corev1.Container{
			Name:            HSMCLIENT,
			Image:           fmt.Sprintf("%s:%s", instance.Spec.Images.HSMImage, instance.Spec.Images.HSMTag),
			ImagePullPolicy: corev1.PullAlways,
			Command: []string{
				"sh",
				"-c",
				fmt.Sprintf("mkdir -p %s/hsm && dst=\"%s/hsm/%s\" && echo \"Copying %s to ${dst}\" && mkdir -p $(dirname $dst) && cp -r %s $dst", mountPath, mountPath, hsmLibraryName, hsmLibraryPath, hsmLibraryPath),
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:    &user,
				RunAsNonRoot: &f,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "shared",
					MountPath: mountPath,
				},
			},
			Resources: instance.GetResource("init"),
		},
	}

	return cont
}

func hsmSettings(instance *current.IBPCA, hsmConfig *config.HSMConfig, caCont container.Container, deployment *deployment.Deployment) {
	caCont.Command = []string{
		"sh",
		"-c",
		"mkdir -p /data/tlsca && cp /config/tlsca/fabric-ca-server-config.yaml /data/tlsca && mkdir -p /data/ca && cp /config/ca/fabric-ca-server-config.yaml /data/ca && fabric-ca-server start --home /data/ca",
	}

	// Add volumes from HSM config to deployment container
	for _, v := range hsmConfig.GetVolumes() {
		deployment.AppendVolumeIfMissing(v)
	}

	// Add volume mounts from HSM config to CA container
	for _, vm := range hsmConfig.GetVolumeMounts() {
		caCont.AppendVolumeMountStructIfMissing(vm)
	}

	// Add environment variables from HSM config to CA container
	for _, env := range hsmConfig.GetEnvs() {
		caCont.AppendEnvStructIfMissing(env)
	}

	caCont.AppendVolumeMountWithSubPathIfMissing("shared", "/hsm/lib", "hsm")

	// If a pull secret is required to pull HSM library image, update the deployment's image pull secrets
	if hsmConfig.Library.Auth != nil {
		deployment.Spec.Template.Spec.ImagePullSecrets = util.AppendPullSecretIfMissing(
			deployment.Spec.Template.Spec.ImagePullSecrets,
			hsmConfig.Library.Auth.ImagePullSecret,
		)
	}

	// Add HSM init container to deployment, the init container is responsible for copying over HSM
	// client library to the path expected by the CA
	deployment.AddInitContainer(*hsmInitContainer(instance, hsmConfig))

	// If daemon settings are configured in HSM config, create a sidecar that is running the daemon image
	if hsmConfig.Daemon != nil {
		hsmDaemonSettings(instance, hsmConfig, caCont, deployment)
	}
}

func hsmDaemonSettings(instance *current.IBPCA, hsmConfig *config.HSMConfig, caCont container.Container, deployment *deployment.Deployment) {
	// Unable to launch daemon if not running priviledged moe
	t := true
	caCont.SecurityContext.Privileged = &t
	caCont.SecurityContext.AllowPrivilegeEscalation = &t

	// Update command in deployment to ensure that deamon is running before starting the ca
	caCont.Command = []string{
		"sh",
		"-c",
		config.DAEMON_CHECK_CMD + " && mkdir -p /data/tlsca && cp /config/tlsca/fabric-ca-server-config.yaml /data/tlsca && mkdir -p /data/ca && cp /config/ca/fabric-ca-server-config.yaml /data/ca && fabric-ca-server start --home /data/ca",
	}

	// This is the shared volume where the file 'pkcsslotd-luanched' is touched to let
	// other containers know that the daemon has successfully launched.
	caCont.AppendVolumeMountIfMissing("shared", "/shared")

	pvcVolumeName := "fabric-ca"
	// Certain token information requires to be stored in persistent store, the administrator
	// responsible for configuring HSM sets the HSM config to point to the path where the PVC
	// needs to be mounted.
	var pvcMount *corev1.VolumeMount
	for _, vm := range hsmConfig.MountPaths {
		if vm.UsePVC {
			pvcMount = &corev1.VolumeMount{
				Name:      pvcVolumeName,
				MountPath: vm.MountPath,
			}
		}
	}

	// If a pull secret is required to pull daemon image, update the deployment's image pull secrets
	if hsmConfig.Daemon.Auth != nil {
		deployment.Spec.Template.Spec.ImagePullSecrets = util.AppendPullSecretIfMissing(
			deployment.Spec.Template.Spec.ImagePullSecrets,
			hsmConfig.Daemon.Auth.ImagePullSecret,
		)
	}

	// Add daemon container to the deployment
	config.AddDaemonContainer(hsmConfig, deployment, instance.GetResource(current.HSMDAEMON), pvcMount)

	// If a pvc mount has been configured in HSM config, set the volume mount on the ca container
	// and PVC volume to deployment if missing
	if pvcMount != nil {
		caCont.AppendVolumeMountStructIfMissing(*pvcMount)
		deployment.AppendPVCVolumeIfMissing(pvcVolumeName, instance.PVCName())
	}
}
