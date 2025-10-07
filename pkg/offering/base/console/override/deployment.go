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
	"net/url"

	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	defaultconsole "github.com/IBM-Blockchain/fabric-operator/defaultconfig/console"
	deployerimgs "github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	dep "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/serviceaccount"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Container names
const (
	INIT          = "init"
	CONSOLE       = "optools"
	DEPLOYER      = "deployer"
	CONFIGTXLATOR = "configtxlator"
	COUCHDB       = "couchdb"
)

func (o *Override) Deployment(object v1.Object, deployment *appsv1.Deployment, action resources.Action) error {
	instance := object.(*current.IBPConsole)
	switch action {
	case resources.Create:
		return o.CreateDeployment(instance, deployment)
	case resources.Update:
		return o.UpdateDeployment(instance, deployment)
	}

	return nil
}

func (o *Override) CreateDeployment(instance *current.IBPConsole, k8sDep *appsv1.Deployment) error {
	deployment := dep.New(k8sDep)

	name := instance.GetName()
	deployment.SetServiceAccountName(serviceaccount.GetName(name))

	// Make sure containers exist
	console, err := deployment.GetContainer(CONSOLE)
	if err != nil {
		return errors.New("console container not found in deployment spec")
	}
	_, err = deployment.GetContainer(INIT)
	if err != nil {
		return errors.New("init container not found in deployment spec")
	}
	_, err = deployment.GetContainer(DEPLOYER)
	if err != nil {
		return errors.New("deployer container not found in deployment spec")
	}
	_, err = deployment.GetContainer(CONFIGTXLATOR)
	if err != nil {
		return errors.New("configtxlator container not found in deployment spec")
	}

	if !instance.Spec.UsingRemoteDB() {
		couchdb := o.CreateCouchdbContainer()

		couchdb.AppendVolumeMountWithSubPathIfMissing("couchdb", "/opt/couchdb/data", "data")
		deployment.AddContainer(couchdb)
	}

	err = o.CommonDeployment(instance, deployment)
	if err != nil {
		return err
	}

	deployment.SetImagePullSecrets(instance.Spec.ImagePullSecrets)

	console.AppendConfigMapFromSourceIfMissing(name)

	passwordSecretName := instance.Spec.PasswordSecretName
	valueFrom := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: passwordSecretName,
			},
			Key: "password",
		},
	}
	console.AppendEnvVarValueFromIfMissing("DEFAULT_USER_PASSWORD_INITIAL", valueFrom)

	tlsSecretName := instance.Spec.TLSSecretName
	if tlsSecretName != "" {
		console.AppendVolumeMountIfMissing("tls-certs", "/certs/tls")
		deployment.AppendSecretVolumeIfMissing("tls-certs", tlsSecretName)
	} else {
		// TODO: generate and create the TLS Secret here itself
	}

	if !instance.Spec.UsingRemoteDB() {
		deployment.AppendPVCVolumeIfMissing("couchdb", instance.Name+"-pvc")
	}

	deployment.AppendConfigMapVolumeIfMissing("deployer-template", name+"-deployer")
	deployment.AppendConfigMapVolumeIfMissing("template", name+"-console")
	deployment.SetAffinity(o.GetAffinity(instance))

	return nil
}

func (o *Override) UpdateDeployment(instance *current.IBPConsole, k8sDep *appsv1.Deployment) error {
	deployment := dep.New(k8sDep)
	return o.CommonDeployment(instance, deployment)
}

func (o *Override) CommonDeployment(instance *current.IBPConsole, deployment *dep.Deployment) error {
	init := deployment.MustGetContainer(INIT)
	console := deployment.MustGetContainer(CONSOLE)
	deployer := deployment.MustGetContainer(DEPLOYER)
	configtxlator := deployment.MustGetContainer(CONFIGTXLATOR)

	registryURL := instance.Spec.RegistryURL
	arch := "amd64"
	if instance.Spec.Arch != nil {
		arch = instance.Spec.Arch[0]
	}

	images := &deployerimgs.ConsoleImages{}
	if instance.Spec.Images != nil {
		// convert spec images to deployer config images
		instanceImgBytes, err := json.Marshal(instance.Spec.Images)
		if err != nil {
			return err
		}
		err = json.Unmarshal(instanceImgBytes, images)
		if err != nil {
			return err
		}
	}

	var consoleImage, consoleTag, initImage, initTag, deployerImage, deployerTag string
	var configtxlatorImage, configtxlatorTag, couchdbImage, couchdbTag string

	defaultimage := defaultconsole.GetImages()
	consoleImage = image.GetImage(registryURL, defaultimage.ConsoleImage, images.ConsoleImage)
	initImage = image.GetImage(registryURL, defaultimage.ConsoleInitImage, images.ConsoleInitImage)
	deployerImage = image.GetImage(registryURL, defaultimage.DeployerImage, images.DeployerImage)
	configtxlatorImage = image.GetImage(registryURL, defaultimage.ConfigtxlatorImage, images.ConfigtxlatorImage)

	if instance.UseTags() {
		consoleTag = image.GetTag(arch, defaultimage.ConsoleTag, images.ConsoleTag)
		initTag = image.GetTag(arch, defaultimage.ConsoleInitTag, images.ConsoleInitTag)
		deployerTag = image.GetTag(arch, defaultimage.DeployerTag, images.DeployerTag)
		configtxlatorTag = image.GetTag(arch, defaultimage.ConfigtxlatorTag, images.ConfigtxlatorTag)
	} else {
		consoleTag = image.GetTag(arch, defaultimage.ConsoleDigest, images.ConsoleDigest)
		initTag = image.GetTag(arch, defaultimage.ConsoleInitDigest, images.ConsoleInitDigest)
		deployerTag = image.GetTag(arch, defaultimage.DeployerDigest, images.DeployerDigest)
		configtxlatorTag = image.GetTag(arch, defaultimage.ConfigtxlatorDigest, images.ConfigtxlatorDigest)
	}
	init.SetImage(initImage, initTag)
	console.SetImage(consoleImage, consoleTag)
	deployer.SetImage(deployerImage, deployerTag)
	configtxlator.SetImage(configtxlatorImage, configtxlatorTag)

	resourcesRequest := instance.Spec.Resources
	if !instance.Spec.UsingRemoteDB() {
		couchdb := deployment.MustGetContainer(COUCHDB)
		common.SetPodSecurityContext(couchdb)
		if instance.Spec.ConnectionString != "" {
			connectionURL, err := url.Parse(instance.Spec.ConnectionString)
			if err != nil {
				return err
			}
			if connectionURL.Host == "localhost:5984" {
				if connectionURL.Scheme == "http" {
					couchdbUser := connectionURL.User.Username()
					couchdbPassword, set := connectionURL.User.Password()
					if set {
						couchdb.AppendEnvIfMissing("COUCHDB_USER", couchdbUser)
						couchdb.AppendEnvIfMissing("COUCHDB_PASSWORD", couchdbPassword)
						couchdb.AppendEnvIfMissing("SKIP_PERMISSIONS_UPDATE", "true")
					}
				}
			}
		}

		couchdbImage = image.GetImage(registryURL, defaultimage.CouchDBImage, images.CouchDBImage)
		if instance.Spec.UseTags == nil || *(instance.Spec.UseTags) == true {
			couchdbTag = image.GetTag(arch, defaultimage.CouchDBTag, images.CouchDBTag)
		} else {
			couchdbTag = image.GetTag(arch, defaultimage.CouchDBDigest, images.CouchDBDigest)

		}
		couchdb.SetImage(couchdbImage, couchdbTag)

		if resourcesRequest != nil {
			if resourcesRequest.CouchDB != nil {
				err := couchdb.UpdateResources(resourcesRequest.CouchDB)
				if err != nil {
					return errors.Wrap(err, "update resources for couchdb failed")
				}
			}
		}
	}

	if resourcesRequest != nil {
		if resourcesRequest.Console != nil {
			err := console.UpdateResources(resourcesRequest.Console)
			if err != nil {
				return errors.Wrap(err, "update resources for console failed")
			}
		}

		if resourcesRequest.Deployer != nil {
			err := deployer.UpdateResources(resourcesRequest.Deployer)
			if err != nil {
				return errors.Wrap(err, "update resources for deployer failed")
			}
		}

		if resourcesRequest.Configtxlator != nil {
			err := configtxlator.UpdateResources(resourcesRequest.Configtxlator)
			if err != nil {
				return errors.Wrap(err, "update resources for configtxlator failed")
			}
		}
	}

	if err := setReplicas(instance, deployment); err != nil {
		return err
	}
	setDeploymentStrategy(instance, deployment)
	setSpreadConstraints(instance, deployment)

	kubeconfigSecretName := instance.Spec.KubeconfigSecretName
	if kubeconfigSecretName != "" {
		deployer.AppendVolumeMountIfMissing("kubeconfig", "/kubeconfig/")
		deployment.AppendSecretVolumeIfMissing("kubeconfig", kubeconfigSecretName)
		deployer.AppendEnvIfMissing("KUBECONFIGPATH", "/kubeconfig/kubeconfig.yaml")
	}

	kubeconfigNamespace := instance.Spec.KubeconfigNamespace
	if kubeconfigNamespace != "" {
		deployer.AppendEnvIfMissing("DEPLOY_NAMESPACE", kubeconfigNamespace)
	} else {
		valueFrom := &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		}
		deployer.AppendEnvVarValueFromIfMissing("DEPLOY_NAMESPACE", valueFrom)
	}

	consoleOverrides, err := instance.Spec.GetOverridesConsole()
	if err != nil {
		return err
	}

	initCommand := ""
	if !instance.Spec.UsingRemoteDB() {
		initCommand = "chmod -R 775 /opt/couchdb/data/ && chown -R -H 5984:5984 /opt/couchdb/data/ && chmod -R 775 /certs/ && chown -R -H 1000:1000 /certs/"

		couchDBVolumeMount := corev1.VolumeMount{
			Name:      "couchdb",
			MountPath: "/opt/couchdb/data",
			SubPath:   "data",
		}

		certsVolumeMount := corev1.VolumeMount{
			Name:      "couchdb",
			MountPath: "/certs/",
			SubPath:   "tls",
		}
		init.SetVolumeMounts([]corev1.VolumeMount{couchDBVolumeMount, certsVolumeMount})
		console.AppendVolumeMountWithSubPathIfMissing("couchdb", "/certs/", "tls")
	}

	if consoleOverrides.ActivityTrackerConsolePath != "" {
		hostPath := "/var/log/at"
		if consoleOverrides.ActivityTrackerHostPath != "" {
			hostPath = consoleOverrides.ActivityTrackerHostPath
		}
		deployment.AppendHostPathVolumeIfMissing("activity", hostPath, corev1.HostPathDirectoryOrCreate)

		console.AppendVolumeMountWithSubPathIfMissing("activity", consoleOverrides.ActivityTrackerConsolePath, instance.Namespace)
		init.AppendVolumeMountWithSubPathIfMissing("activity", consoleOverrides.ActivityTrackerConsolePath, instance.Namespace)

		if initCommand != "" {
			initCommand += " && "
		}
		initCommand += "chmod -R 775 " + consoleOverrides.ActivityTrackerConsolePath + " && chown -R -H 1000:1000 " + consoleOverrides.ActivityTrackerConsolePath
	}

	if initCommand == "" {
		initCommand = "exit 0"
	}
	init.SetCommand([]string{"sh", "-c", initCommand})

	// set seccompProfile to RuntimeDefault
	common.SetPodSecurityContext(console)
	common.SetPodSecurityContext(deployer)
	common.SetPodSecurityContext(configtxlator)

	return nil
}

func (o *Override) GetAffinity(instance *current.IBPConsole) *corev1.Affinity {
	arch := instance.Spec.Arch
	zone := instance.Spec.Zone
	region := instance.Spec.Region
	nodeSelectorTerms := common.GetNodeSelectorTerms(arch, zone, region)

	affinity := &corev1.Affinity{}

	if len(nodeSelectorTerms[0].MatchExpressions) != 0 {
		affinity.NodeAffinity = &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: nodeSelectorTerms,
			},
		}
	}

	return affinity
}

func (o *Override) CreateCouchdbContainer() container.Container {
	falsep := false
	truep := true
	portp := int64(5984)

	couchdb := &corev1.Container{
		Name:            "couchdb",
		Image:           "",
		ImagePullPolicy: "Always",
		Env: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  "LICENSE",
				Value: "accept",
			},
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged:               &falsep,
			AllowPrivilegeEscalation: &falsep,
			ReadOnlyRootFilesystem:   &falsep,
			RunAsNonRoot:             &truep,
			RunAsUser:                &portp,
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
				Add:  []corev1.Capability{"NET_BIND_SERVICE", "CHOWN", "DAC_OVERRIDE", "SETGID", "SETUID"},
			},
		},
		Ports: []corev1.ContainerPort{
			corev1.ContainerPort{
				Name:          "http",
				ContainerPort: 5984,
			},
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(5984),
				},
			},
			InitialDelaySeconds: 16,
			TimeoutSeconds:      5,
			FailureThreshold:    5,
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(5984),
				},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      5,
			FailureThreshold:    5,
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("500m"),
				corev1.ResourceMemory:           resource.MustParse("1000Mi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("500m"),
				corev1.ResourceMemory:           resource.MustParse("1000Mi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
			},
		},
	}

	return *container.New(couchdb)
}

func setReplicas(instance *current.IBPConsole, d *dep.Deployment) error {
	if instance.Spec.Replicas != nil {
		if !instance.Spec.UsingRemoteDB() && *instance.Spec.Replicas > 1 {
			return errors.New("replicas > 1 not allowed in IBPConsole")
		}

		d.SetReplicas(instance.Spec.Replicas)
	}

	return nil
}

func setDeploymentStrategy(instance *current.IBPConsole, d *dep.Deployment) {
	switch instance.Spec.UsingRemoteDB() {
	case false:
		d.Spec.Strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	case true:
		opts := intstr.FromString("25%")
		d.Spec.Strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxUnavailable: &opts,
				MaxSurge:       &opts,
			},
		}
	}
}

func setSpreadConstraints(instance *current.IBPConsole, d *dep.Deployment) {
	if instance.Spec.UsingRemoteDB() {
		d.Spec.Template.Spec.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{
			{
				MaxSkew:           1,
				TopologyKey:       "topology.kubernetes.io/zone",
				WhenUnsatisfiable: corev1.ScheduleAnyway,
				LabelSelector: &v1.LabelSelector{
					MatchLabels: map[string]string{
						"type": "ibpconsole",
					},
				},
			},
		}
	}
}
