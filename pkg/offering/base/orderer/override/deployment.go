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
	"context"
	"errors"
	"fmt"
	"path/filepath"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
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
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("orderer_deployment_override")

type OrdererConfig interface {
	MergeWith(interface{}, bool) error
	ToBytes() ([]byte, error)
	UsingPKCS11() bool
	SetPKCS11Defaults(bool)
	GetBCCSPSection() *commonapi.BCCSP
	SetDefaultKeyStore()
}

// Container names
const (
	INIT      = "init"
	ORDERER   = "orderer"
	PROXY     = "proxy"
	HSMCLIENT = "hsm-client"
)

func (o *Override) Deployment(object v1.Object, deployment *appsv1.Deployment, action resources.Action) error {
	instance := object.(*current.IBPOrderer)
	switch action {
	case resources.Create:
		return o.CreateDeployment(instance, deployment)
	case resources.Update:
		return o.UpdateDeployment(instance, deployment)
	}

	return nil
}

func (o *Override) CreateDeployment(instance *current.IBPOrderer, k8sDep *appsv1.Deployment) error {
	var err error

	if !instance.Spec.License.Accept {
		return errors.New("user must accept license before continuing")
	}

	ordererType := instance.Spec.OrdererType
	if ordererType == "" {
		return errors.New("Orderer Type not provided")
	}

	systemChannelName := instance.Spec.SystemChannelName
	if systemChannelName == "" {
		return errors.New("System Channel Name not provided")
	}

	ordererOrgName := instance.Spec.OrgName
	if ordererOrgName == "" {
		return errors.New("Orderer Org Name not provided")
	}

	externalAddress := instance.Spec.ExternalAddress
	if externalAddress == "" {
		return errors.New("External Address not set")
	}

	deployment := dep.New(k8sDep)
	deployment.SetServiceAccountName(serviceaccount.GetName(instance.GetName()))

	orderer, err := deployment.GetContainer(ORDERER)
	if err != nil {
		return errors.New("orderer container not found in deployment spec")
	}
	grpcWeb, err := deployment.GetContainer(PROXY)
	if err != nil {
		return errors.New("proxy container not found in deployment spec")
	}
	_, err = deployment.GetContainer(INIT)
	if err != nil {
		return errors.New("init container not found in deployment spec")
	}

	err = o.CommonDeploymentOverrides(instance, deployment)
	if err != nil {
		return err
	}

	deployment.SetImagePullSecrets(instance.Spec.ImagePullSecrets)

	orderer.AppendConfigMapFromSourceIfMissing(instance.Name + "-env")

	claimName := instance.Name + "-pvc"
	if instance.Spec.CustomNames.PVC.Orderer != "" {
		claimName = instance.Spec.CustomNames.PVC.Orderer
	}
	deployment.AppendPVCVolumeIfMissing("orderer-data", claimName)

	grpcWeb.AppendEnvIfMissing("EXTERNAL_ADDRESS", externalAddress)

	deployment.SetAffinity(o.GetAffinity(instance))

	if o.AdminSecretExists(instance) {
		deployment.AppendSecretVolumeIfMissing("ecert-admincerts", fmt.Sprintf("ecert-%s-admincerts", instance.Name))
		orderer.AppendVolumeMountIfMissing("ecert-admincerts", "/certs/msp/admincerts")
	}

	deployment.AppendSecretVolumeIfMissing("ecert-cacerts", fmt.Sprintf("ecert-%s-cacerts", instance.Name))

	co, err := instance.GetConfigOverride()
	if err != nil {
		return err
	}

	configOverride := co.(OrdererConfig)
	if !configOverride.UsingPKCS11() {
		deployment.AppendSecretVolumeIfMissing("ecert-keystore", fmt.Sprintf("ecert-%s-keystore", instance.Name))
		orderer.AppendVolumeMountIfMissing("ecert-keystore", "/certs/msp/keystore")
	}

	deployment.AppendSecretVolumeIfMissing("ecert-signcert", fmt.Sprintf("ecert-%s-signcert", instance.Name))

	secretName := fmt.Sprintf("tls-%s-cacerts", instance.Name)
	ecertintercertSecret := fmt.Sprintf("ecert-%s-intercerts", instance.Name)
	tlsintercertSecret := fmt.Sprintf("tls-%s-intercerts", instance.Name)
	// Check if intermediate ecerts exists
	if util.IntermediateSecretExists(o.Client, instance.Namespace, ecertintercertSecret) {
		// Mount intermediate ecert
		orderer.AppendVolumeMountIfMissing("ecert-intercerts", "/certs/msp/intermediatecerts")
		deployment.AppendSecretVolumeIfMissing("ecert-intercerts", ecertintercertSecret)
	}

	// Check if intermediate tlscerts exists
	if util.IntermediateSecretExists(o.Client, instance.Namespace, tlsintercertSecret) {
		// Mount intermediate tls certs
		orderer.AppendVolumeMountIfMissing("tls-intercerts", "/certs/msp/tlsintermediatecerts")
		deployment.AppendSecretVolumeIfMissing("tls-intercerts", tlsintercertSecret)
	}

	deployment.AppendSecretVolumeIfMissing("tls-cacerts", secretName)
	deployment.AppendSecretVolumeIfMissing("tls-keystore", fmt.Sprintf("tls-%s-keystore", instance.Name))
	deployment.AppendSecretVolumeIfMissing("tls-signcert", fmt.Sprintf("tls-%s-signcert", instance.Name))
	deployment.AppendConfigMapVolumeIfMissing("orderer-config", instance.Name+"-config")

	if !instance.Spec.IsUsingChannelLess() {
		deployment.AppendSecretVolumeIfMissing("orderer-genesis", fmt.Sprintf("%s-genesis", instance.Name))
		orderer.AppendVolumeMountIfMissing("orderer-genesis", "/certs/genesis")
	}

	secret := &corev1.Secret{}
	err = o.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: instance.GetName() + "-secret", Namespace: instance.GetNamespace()},
		secret,
	)
	if err == nil {
		orderer.AppendEnvIfMissing("RESTART_OLD_RESOURCEVER", secret.ObjectMeta.ResourceVersion)
	}

	deployment.UpdateContainer(orderer)
	if instance.UsingHSMProxy() {
		orderer.AppendEnvIfMissing("PKCS11_PROXY_SOCKET", instance.Spec.HSM.PKCS11Endpoint)
	} else if instance.IsHSMEnabled() {
		deployment.AppendVolumeIfMissing(corev1.Volume{
			Name: "shared",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		})

		orderer.AppendVolumeMountWithSubPathIfMissing("shared", "/hsm/lib", "hsm")

		hsmConfig, err := config.ReadHSMConfig(o.Client, instance)
		if err != nil {
			return err
		}

		hsmSettings(instance, hsmConfig, orderer, deployment)
		deployment.UpdateContainer(orderer)
	}

	return nil
}

func (o *Override) UpdateDeployment(instance *current.IBPOrderer, k8sDep *appsv1.Deployment) error {
	deployment := dep.New(k8sDep)
	err := o.CommonDeploymentOverrides(instance, deployment)
	if err != nil {
		return err
	}

	if instance.UsingHSMProxy() {
		orderer := deployment.MustGetContainer(ORDERER)
		orderer.UpdateEnv("PKCS11_PROXY_SOCKET", instance.Spec.HSM.PKCS11Endpoint)
		deployment.UpdateContainer(orderer)
	} else if instance.IsHSMEnabled() {
		hsmInitCont := deployment.MustGetContainer(HSMCLIENT)
		image := instance.Spec.Images
		if image != nil {
			hsmInitCont.SetImage(image.HSMImage, image.HSMTag)
		}
	}

	return nil
}

func (o *Override) CommonDeploymentOverrides(instance *current.IBPOrderer, deployment *dep.Deployment) error {
	orderer := deployment.MustGetContainer(ORDERER)
	grpcProxy := deployment.MustGetContainer(PROXY)
	initCont := deployment.MustGetContainer(INIT)

	if instance.Spec.Replicas != nil {
		if *instance.Spec.Replicas > 1 {
			return errors.New("replicas > 1 not allowed in IBPOrderer")
		}
		deployment.SetReplicas(instance.Spec.Replicas)
	}

	resourcesRequest := instance.Spec.Resources
	if resourcesRequest != nil {
		if resourcesRequest.Init != nil {
			err := initCont.UpdateResources(resourcesRequest.Init)
			if err != nil {
				return err
			}
		}
		if resourcesRequest.Orderer != nil {
			err := orderer.UpdateResources(resourcesRequest.Orderer)
			if err != nil {
				return err
			}
		}
		if resourcesRequest.GRPCProxy != nil {
			err := grpcProxy.UpdateResources(resourcesRequest.GRPCProxy)
			if err != nil {
				return err
			}
		}
	}

	image := instance.Spec.Images
	if image != nil {
		orderer.SetImage(image.OrdererImage, image.OrdererTag)
		initCont.SetImage(image.OrdererInitImage, image.OrdererInitTag)
		grpcProxy.SetImage(image.GRPCWebImage, image.GRPCWebTag)
	}

	if o.Config != nil && o.Config.Operator.Orderer.DisableProbes == "true" {
		log.Info("Env var IBPOPERATOR_ORDERER_DISABLEPROBES set to 'true', disabling orderer container probes")
		orderer.SetLivenessProbe(nil)
		orderer.SetReadinessProbe(nil)
		orderer.SetStartupProbe(nil)
	}

	// Overriding keepalive default serverMinInterval to 25s to make this work on VPC clusters
	orderer.AppendEnvIfMissing("ORDERER_GENERAL_KEEPALIVE_SERVERMININTERVAL", "25s")

	deployment.UpdateContainer(orderer)
	deployment.UpdateContainer(grpcProxy)
	deployment.UpdateInitContainer(initCont)

	return nil
}

func (o *Override) GetAffinity(instance *current.IBPOrderer) *corev1.Affinity {
	arch := instance.Spec.Arch
	zone := instance.Spec.Zone
	region := instance.Spec.Region
	nodeSelectorTerms := common.GetNodeSelectorTerms(arch, zone, region)

	orgName := instance.Spec.OrgName
	podAntiAffinity := common.GetPodAntiAffinity(orgName)

	affinity := &corev1.Affinity{
		PodAntiAffinity: podAntiAffinity,
	}

	if len(nodeSelectorTerms[0].MatchExpressions) != 0 {
		affinity.NodeAffinity = &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: nodeSelectorTerms,
			},
		}
	}

	return affinity
}

func (o *Override) AdminSecretExists(instance *current.IBPOrderer) bool {
	secret := &corev1.Secret{}
	err := o.Client.Get(context.TODO(), types.NamespacedName{
		Name:      fmt.Sprintf("ecert-%s-admincerts", instance.Name),
		Namespace: instance.Namespace}, secret)
	if err != nil {
		return false
	}

	return true
}

func hsmInitContainer(instance *current.IBPOrderer, hsmConfig *config.HSMConfig) *container.Container {
	hsmLibraryPath := hsmConfig.Library.FilePath
	hsmLibraryName := filepath.Base(hsmLibraryPath)

	f := false
	user := int64(0)
	mountPath := "/shared"
	return &container.Container{
		Container: &corev1.Container{
			Name:            "hsm-client",
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
				corev1.VolumeMount{
					Name:      "shared",
					MountPath: mountPath,
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:              resource.MustParse("0.1"),
					corev1.ResourceMemory:           resource.MustParse("100Mi"),
					corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:              resource.MustParse("2"),
					corev1.ResourceMemory:           resource.MustParse("4Gi"),
					corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}

func hsmSettings(instance *current.IBPOrderer, hsmConfig *config.HSMConfig, ordererCont container.Container, dep *deployment.Deployment) {
	for _, v := range hsmConfig.GetVolumes() {
		dep.AppendVolumeIfMissing(v)
	}

	for _, vm := range hsmConfig.GetVolumeMounts() {
		ordererCont.AppendVolumeMountStructIfMissing(vm)
	}

	for _, env := range hsmConfig.GetEnvs() {
		ordererCont.AppendEnvStructIfMissing(env)
	}

	if hsmConfig.Library.Auth != nil {
		dep.Spec.Template.Spec.ImagePullSecrets = util.AppendPullSecretIfMissing(dep.Spec.Template.Spec.ImagePullSecrets, hsmConfig.Library.Auth.ImagePullSecret)
	}

	dep.AddInitContainer(*hsmInitContainer(instance, hsmConfig))

	// If daemon settings are configured in HSM config, create a sidecar that is running the daemon image
	if hsmConfig.Daemon != nil {
		hsmDaemonSettings(instance, hsmConfig, ordererCont, dep)
	}
}

func hsmDaemonSettings(instance *current.IBPOrderer, hsmConfig *config.HSMConfig, ordererCont container.Container, deployment *deployment.Deployment) {
	// Unable to launch daemon if not running priviledged moe
	t := true
	ordererCont.SecurityContext.Privileged = &t
	ordererCont.SecurityContext.AllowPrivilegeEscalation = &t

	// Update command in deployment to ensure that deamon is running before starting the ca
	ordererCont.Command = []string{
		"sh",
		"-c",
		fmt.Sprintf("%s && orderer", config.DAEMON_CHECK_CMD),
	}

	// This is the shared volume where the file 'pkcsslotd-luanched' is touched to let
	// other containers know that the daemon has successfully launched.
	ordererCont.AppendVolumeMountIfMissing("shared", "/shared")

	pvcVolumeName := "orderer-data"
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
		ordererCont.AppendVolumeMountStructIfMissing(*pvcMount)
		deployment.AppendPVCVolumeIfMissing(pvcVolumeName, instance.PVCName())
	}
}
