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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	dep "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/serviceaccount"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Container names
const (
	INIT        = "init"
	PEER        = "peer"
	DIND        = "dind"
	PROXY       = "proxy"
	FLUENTD     = "chaincode-logs"
	COUCHDB     = "couchdb"
	COUCHDBINIT = "couchdbinit"
	CCLAUNCHER  = "chaincode-launcher"
	HSMCLIENT   = "hsm-client"
)

type CoreConfig interface {
	UsingPKCS11() bool
}

func (o *Override) Deployment(object v1.Object, deployment *appsv1.Deployment, action resources.Action) error {
	instance := object.(*current.IBPPeer)
	switch action {
	case resources.Create:
		return o.CreateDeployment(instance, deployment)
	case resources.Update:
		return o.UpdateDeployment(instance, deployment)
	}

	return nil
}

func (o *Override) CreateDeployment(instance *current.IBPPeer, k8sDep *appsv1.Deployment) error {
	var err error
	name := instance.GetName()

	if !instance.Spec.License.Accept {
		return errors.New("user must accept license before continuing")
	}

	mspID := instance.Spec.MSPID
	if mspID == "" {
		return errors.New("failed to provide MSP ID for peer")
	}

	deployment := dep.New(k8sDep)
	initContainer, err := deployment.GetContainer(INIT)
	if err != nil {
		return errors.New("init container not found in deployment spec")
	}
	peerContainer, err := deployment.GetContainer(PEER)
	if err != nil {
		return errors.New("peer container not found in deployment spec")
	}
	grpcwebContainer, err := deployment.GetContainer(PROXY)
	if err != nil {
		return errors.New("grpc container not found in deployment spec")
	}

	stateDB := instance.Spec.StateDb
	if instance.UsingCouchDB() {
		if !deployment.ContainerExists(COUCHDB) { // If coucdb container exists, don't need to create it again
			stateDB = "CouchDB"
			err = o.CreateCouchDBContainers(instance, deployment)
			if err != nil {
				return err
			}
		}
	} else if instance.Spec.UsingLevelDB() {
		stateDB = "goleveldb"

		peerContainer.AppendVolumeMountWithSubPathIfMissing("db-data", "/data/peer/ledgersData/stateLeveldb/", "data")
		initContainer.AppendVolumeMountWithSubPathIfMissing("db-data", "/data/peer/ledgersData/stateLeveldb/", "data")

		deployment.UpdateContainer(peerContainer)
		deployment.UpdateInitContainer(initContainer)
	} else {
		return errors.New("unsupported StateDB type")
	}

	err = o.CommonDeploymentOverrides(instance, deployment)
	if err != nil {
		return err
	}

	// At this point we know init, peer, and proxy containers exists.
	// Can use MustGetContainer to avoid handling error
	peerContainer = deployment.MustGetContainer(PEER)
	grpcwebContainer = deployment.MustGetContainer(PROXY)

	deployment.SetImagePullSecrets(instance.Spec.ImagePullSecrets)
	deployment.SetServiceAccountName(serviceaccount.GetName(name))
	deployment.SetAffinity(o.GetAffinity(instance))

	peerContainer.AppendEnvIfMissing("CORE_PEER_ID", instance.Name)
	peerContainer.AppendEnvIfMissing("CORE_PEER_LOCALMSPID", mspID)

	claimName := instance.Name + "-statedb-pvc"
	if instance.Spec.CustomNames.PVC.StateDB != "" {
		claimName = instance.Spec.CustomNames.PVC.StateDB
	}
	deployment.AppendPVCVolumeIfMissing("db-data", claimName)

	peerContainer.AppendEnvIfMissing("CORE_LEDGER_STATE_STATEDATABASE", stateDB)

	claimName = instance.Name + "-pvc"
	if instance.Spec.CustomNames.PVC.Peer != "" {
		claimName = instance.Spec.CustomNames.PVC.Peer
	}
	deployment.AppendPVCVolumeIfMissing("fabric-peer-0", claimName)

	deployment.AppendConfigMapVolumeIfMissing("fluentd-config", instance.Name+"-fluentd")

	ecertintercertSecret := fmt.Sprintf("ecert-%s-intercerts", instance.Name)
	tlsintercertSecret := fmt.Sprintf("tls-%s-intercerts", instance.Name)
	secretName := fmt.Sprintf("tls-%s-cacerts", instance.Name)
	// Check if intermediate ecerts exists
	if util.IntermediateSecretExists(o.Client, instance.Namespace, ecertintercertSecret) {
		peerContainer.AppendVolumeMountIfMissing("ecert-intercerts", "/certs/msp/intermediatecerts")
		deployment.AppendSecretVolumeIfMissing("ecert-intercerts", ecertintercertSecret)
	}

	// Check if intermediate tlscerts exists
	if util.IntermediateSecretExists(o.Client, instance.Namespace, tlsintercertSecret) {
		peerContainer.AppendVolumeMountIfMissing("tls-intercerts", "/certs/msp/tlsintermediatecerts")
		deployment.AppendSecretVolumeIfMissing("tls-intercerts", tlsintercertSecret)
	}

	tlsCACertsSecret, err := o.GetTLSCACertsSecret(instance, secretName)
	if err != nil {
		return err
	}

	var certsData string
	count := 0
	for key, _ := range tlsCACertsSecret.Data {
		v := fmt.Sprintf("/certs/msp/tlscacerts/%s", key)
		if count == 0 {
			certsData = certsData + v
		} else {
			certsData = certsData + " " + v
		}
		count++
	}
	peerContainer.AppendEnvIfMissingOverrideIfPresent("CORE_OPERATIONS_TLS_CLIENTROOTCAS_FILES", certsData)
	peerContainer.AppendEnvIfMissingOverrideIfPresent("CORE_PEER_TLS_ROOTCERT_FILE", certsData)
	grpcwebContainer.AppendEnvIfMissingOverrideIfPresent("SERVER_TLS_CLIENT_CA_FILES", certsData)
	peerContainer.AppendEnvIfMissingOverrideIfPresent("CORE_PEER_TLS_ROOTCERT_FILE", certsData)

	// Check if intermediate tlscerts exists
	if util.IntermediateSecretExists(o.Client, instance.Namespace, tlsintercertSecret) {
		secretName := fmt.Sprintf("tls-%s-intercerts", instance.Name)
		tlsCAInterCertsSecret, err := o.GetTLSCACertsSecret(instance, secretName)
		if err != nil {
			return err
		}

		var certsData string
		count := 0
		for key, _ := range tlsCAInterCertsSecret.Data {
			v := fmt.Sprintf("/certs/msp/tlsintermediatecerts/%s", key)
			if count == 0 {
				certsData = certsData + v
			} else {
				certsData = certsData + " " + v
			}
			count++
		}
		peerContainer.AppendEnvIfMissingOverrideIfPresent("CORE_PEER_TLS_ROOTCERT_FILE", certsData)
	}

	if o.AdminSecretExists(instance) {
		deployment.AppendSecretVolumeIfMissing("ecert-admincerts", fmt.Sprintf("ecert-%s-admincerts", instance.Name))
		peerContainer.AppendVolumeMountIfMissing("ecert-admincerts", "/certs/msp/admincerts")
	}

	co, err := instance.GetConfigOverride()
	if err != nil {
		return errors.Wrap(err, "failed to get configoverride")
	}

	configOverride := co.(CoreConfig)
	if !configOverride.UsingPKCS11() {
		deployment.AppendSecretVolumeIfMissing("ecert-keystore", fmt.Sprintf("ecert-%s-keystore", instance.Name))
		peerContainer.AppendVolumeMountIfMissing("ecert-keystore", "/certs/msp/keystore")
	}

	deployment.AppendSecretVolumeIfMissing("ecert-cacerts", fmt.Sprintf("ecert-%s-cacerts", instance.Name))
	deployment.AppendSecretVolumeIfMissing("ecert-signcert", fmt.Sprintf("ecert-%s-signcert", instance.Name))
	deployment.AppendSecretVolumeIfMissing("tls-cacerts", fmt.Sprintf("tls-%s-cacerts", instance.Name))
	deployment.AppendSecretVolumeIfMissing("tls-keystore", fmt.Sprintf("tls-%s-keystore", instance.Name))
	deployment.AppendSecretVolumeIfMissing("tls-signcert", fmt.Sprintf("tls-%s-signcert", instance.Name))

	if o.OrdererCACertsSecretExists(instance) {
		deployment.AppendSecretVolumeIfMissing("orderercacerts", fmt.Sprintf("%s-orderercacerts", instance.Name))
		peerContainer.AppendVolumeMountIfMissing("orderercacerts", "/orderer/certs")
	}

	deployment.AppendConfigMapVolumeIfMissing("peer-config", instance.Name+"-config")

	secret := &corev1.Secret{}
	err = o.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: instance.GetName() + "-secret", Namespace: instance.GetNamespace()},
		secret,
	)
	if err == nil {
		peerContainer.AppendEnvIfMissing("RESTART_OLD_RESOURCEVER", secret.ObjectMeta.ResourceVersion)
	}

	deployment.UpdateContainer(grpcwebContainer)

	if instance.UsingHSMProxy() {
		peerContainer.AppendEnvIfMissing("PKCS11_PROXY_SOCKET", instance.Spec.HSM.PKCS11Endpoint)
	} else if instance.IsHSMEnabled() {
		deployment.AppendVolumeIfMissing(corev1.Volume{
			Name: "shared",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		})

		hsmConfig, err := config.ReadHSMConfig(o.Client, instance)
		if err != nil {
			return err
		}

		hsmSettings(instance, hsmConfig, peerContainer, deployment)

		deployment.UpdateContainer(peerContainer)
	}

	if version.GetMajorReleaseVersion(instance.Spec.FabricVersion) == version.V2 {
		err = o.V2Deployment(instance, deployment)
		if err != nil {
			return errors.Wrap(err, "failed during V2 peer deployment overrides")
		}
		peerVersion := version.String(instance.Spec.FabricVersion)
		if peerVersion.EqualWithoutTag(version.V2_4_1) || peerVersion.EqualWithoutTag(version.V2_5_1) || peerVersion.GreaterThan(version.V2_4_1) {
			err = o.V24Deployment(instance, deployment)
			if err != nil {
				return errors.Wrap(err, "failed during V24/V25 peer deployment overrides")
			}
		}
	} else {
		err = o.V1Deployment(instance, deployment)
		if err != nil {
			return errors.Wrap(err, "failed during V1 peer deployment overrides")
		}
	}

	return nil
}

func (o *Override) V1Deployment(instance *current.IBPPeer, deployment *dep.Deployment) error {
	initContainer := deployment.MustGetContainer(INIT)

	// NOTE: The container doesn't like when these bash commands are listed as separate strings in the command field
	// which is why the command has been formatted into a single string.
	//
	// This command checks the permissions, owner, and group of /data/ and runs chmod/chown on required dirs if they
	// have yet to be set to the default values (775, 1000, and 1000 respectively).
	//
	// cmdFormat is a format string that configured with the list of directories when used.
	cmdFormat := "DEFAULT_PERM=775 && DEFAULT_USER=7051 && DEFAULT_GROUP=1000 "
	cmdFormat += `&& PERM=$(stat -c "%%a" /data/) && USER=$(stat -c "%%u" /data/) && GROUP=$(stat -c "%%g" /data/) ` // %% is used to escape the percent symbol
	cmdFormat += `&& if [ ${PERM} != ${DEFAULT_PERM} ] || [ ${USER} != ${DEFAULT_USER} ] || [ ${GROUP} != ${DEFAULT_GROUP} ]; `
	cmdFormat += `then chmod -R ${DEFAULT_PERM} %[1]s && chown -R -H ${DEFAULT_USER}:${DEFAULT_GROUP} %[1]s; fi`

	// NOTE: There are two chmod & chown calls for /data/ and /data/peer/... because
	// those are two separate pvc mounts, so we were running the command for both the locations.
	if instance.UsingCouchDB() {
		directories := "/data/"
		cmd := fmt.Sprintf(cmdFormat, directories)
		initContainer.SetCommand([]string{
			"bash",
			"-c",
			cmd,
		})
	} else {
		directories := "/{data/,data/peer/ledgersData/stateLeveldb}"
		cmd := fmt.Sprintf(cmdFormat, directories)
		initContainer.SetCommand([]string{
			"bash",
			"-c",
			cmd,
		})
	}

	fluentdContainer, err := deployment.GetContainer(FLUENTD)
	if err != nil {
		return errors.New("fluentD container not found in deployment")
	}

	dindContainer, err := deployment.GetContainer(DIND)
	if err != nil {
		return errors.New("dind container not found in deployment")
	}

	dindargs := instance.Spec.DindArgs
	if dindargs == nil {
		dindargs = []string{"--log-driver", "fluentd", "--log-opt", "fluentd-address=localhost:9880", "--mtu", "1400"}
	}
	dindContainer.SetArgs(dindargs)

	image := instance.Spec.Images
	if image != nil {
		dindContainer.SetImage(image.DindImage, image.DindTag)
		fluentdContainer.SetImage(image.FluentdImage, image.FluentdTag)
	}

	resourcesRequest := instance.Spec.Resources
	if resourcesRequest != nil {
		if resourcesRequest.DinD != nil {
			err = dindContainer.UpdateResources(resourcesRequest.DinD)
			if err != nil {
				return errors.Wrap(err, "resource update for dind failed")
			}
		}

		if resourcesRequest.FluentD != nil {
			err = fluentdContainer.UpdateResources(resourcesRequest.FluentD)
			if err != nil {
				return errors.Wrap(err, "resource update for fluentd failed")
			}
		}
	}

	peerContainer := deployment.MustGetContainer(PEER)
	// env vars only required for 1.x peer
	peerContainer.AppendEnvIfMissing("CORE_VM_ENDPOINT", "localhost:2375")
	peerContainer.AppendEnvIfMissing("CORE_CHAINCODE_GOLANG_RUNTIME", "golangruntime:latest")
	peerContainer.AppendEnvIfMissing("CORE_CHAINCODE_CAR_RUNTIME", "carruntime:latest")
	peerContainer.AppendEnvIfMissing("CORE_CHAINCODE_JAVA_RUNTIME", "javaruntime:latest")
	peerContainer.AppendEnvIfMissing("CORE_CHAINCODE_NODE_RUNTIME", "noderuntime:latest")
	peerContainer.AppendEnvIfMissing("CORE_CHAINCODE_BUILDER", "builder:latest")
	peerContainer.AppendEnvIfMissing("CORE_CHAINCODE_GOLANG_DYNAMICLINK", "true")
	peerContainer.AppendEnvIfMissing("CORE_VM_DOCKER_ATTACHSTDOUT", "false")

	deployment.UpdateInitContainer(initContainer)
	deployment.UpdateContainer(fluentdContainer)
	deployment.UpdateContainer(dindContainer)
	deployment.UpdateContainer(peerContainer)
	return nil
}

func (o *Override) V2Deployment(instance *current.IBPPeer, deployment *dep.Deployment) error {

	initContainer := deployment.MustGetContainer(INIT)
	peerContainer := deployment.MustGetContainer(PEER)

	// NOTE: The container doesn't like when these bash commands are listed as separate strings in the command field
	// which is why the command has been formatted into a single string.
	//
	// This command checks the permissions, owner, and group of /data/ and runs chmod/chown on required dirs if they
	// have yet to be set to the default values (775, 1000, and 1000 respectively).
	//
	// cmdFormat is a format string that configured with the list of directories when used.
	cmdFormat := "DEFAULT_PERM=775 && DEFAULT_USER=7051 && DEFAULT_GROUP=1000 "
	cmdFormat += `&& PERM=$(stat -c "%%a" /data/) && USER=$(stat -c "%%u" /data/) && GROUP=$(stat -c "%%g" /data/) ` // %% is used to escape the percent symbol
	cmdFormat += `&& if [ ${PERM} != ${DEFAULT_PERM} ] || [ ${USER} != ${DEFAULT_USER} ] || [ ${GROUP} != ${DEFAULT_GROUP} ]; `
	cmdFormat += `then chmod -R ${DEFAULT_PERM} %[1]s && chown -R -H ${DEFAULT_USER}:${DEFAULT_GROUP} %[1]s; fi`

	// NOTE: There are multiple chmod & chown calls for /data/ and /data/peer/... and /cclauncher because
	// those are separate pvc mounts, so we were running the command for all the locations
	dirs := []string{"data/"}
	if !instance.UsingCouchDB() {
		dirs = append(dirs, "data/peer/ledgersData/stateLeveldb")
	}
	if instance.UsingCCLauncherImage() {
		dirs = append(dirs, "cclauncher/")
	}

	var directories string
	if len(dirs) > 1 {
		directories = fmt.Sprintf("/{%s}", strings.Join(dirs, ","))
	} else {
		directories = "/data/"
	}

	initContainer.SetCommand([]string{
		"bash",
		"-c",
		fmt.Sprintf(cmdFormat, directories),
	})

	if instance.UsingCCLauncherImage() {
		err := o.CreateCCLauncherContainer(instance, deployment)
		if err != nil {
			return errors.Wrap(err, "failed to create chaincode launcher container")
		}

		volumeMountName := fmt.Sprintf("%s-cclauncher", instance.GetName())
		initContainer.AppendVolumeMountIfMissing(volumeMountName, "/cclauncher")
		peerContainer.AppendVolumeMountIfMissing(volumeMountName, "/cclauncher")

		peerContainer.AppendEnvIfMissing("IBP_BUILDER_SHARED_DIR", "/cclauncher")
		peerContainer.AppendEnvIfMissing("IBP_BUILDER_ENDPOINT", "127.0.0.1:11111")
		peerContainer.AppendEnvIfMissing("PEER_NAME", instance.GetName())

		// Overriding keepalive flags for peers to fix connection issues with VPC clusters
		peerContainer.AppendEnvIfMissing("CORE_PEER_KEEPALIVE_MININTERVAL", "25s")
		peerContainer.AppendEnvIfMissing("CORE_PEER_KEEPALIVE_CLIENT_INTERVAL", "30s")
		peerContainer.AppendEnvIfMissing("CORE_PEER_KEEPALIVE_DELIVERYCLIENT_INTERVAL", "30s")

		// Will delete these envs if found, these are not required for v2
		peerContainer.DeleteEnv("CORE_VM_ENDPOINT")
		peerContainer.DeleteEnv("CORE_CHAINCODE_GOLANG_RUNTIME")
		peerContainer.DeleteEnv("CORE_CHAINCODE_CAR_RUNTIME")
		peerContainer.DeleteEnv("CORE_CHAINCODE_JAVA_RUNTIME")
		peerContainer.DeleteEnv("CORE_CHAINCODE_NODE_RUNTIME")
		peerContainer.DeleteEnv("CORE_CHAINCODE_BUILDER")
		peerContainer.DeleteEnv("CORE_CHAINCODE_GOLANG_DYNAMICLINK")
		peerContainer.DeleteEnv("CORE_VM_DOCKER_ATTACHSTDOUT")

		deployment.AppendEmptyDirVolumeIfMissing(fmt.Sprintf("%s-cclauncher", instance.Name), corev1.StorageMediumMemory)
	}

	// Append a k/v JSON substitution map to the peer env.
	if instance.Spec.ChaincodeBuilderConfig != nil {
		configJSON, err := json.Marshal(instance.Spec.ChaincodeBuilderConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal chaincode builder config '%s',", instance.Spec.ChaincodeBuilderConfig)
		}
		peerContainer.AppendEnvIfMissing("CHAINCODE_AS_A_SERVICE_BUILDER_CONFIG", string(configJSON))
	}

	deployment.UpdateInitContainer(initContainer)
	deployment.UpdateContainer(peerContainer)
	deployment.RemoveContainer(FLUENTD)
	deployment.RemoveContainer(DIND)
	return nil
}

func (o *Override) V24Deployment(instance *current.IBPPeer, deployment *dep.Deployment) error {
	if instance.UsingCCLauncherImage() {
		launcherContainer := deployment.MustGetContainer(CCLAUNCHER)

		launcherContainer.LivenessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS
		launcherContainer.ReadinessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS
		deployment.UpdateContainer(launcherContainer)
	}
	return nil
}

func (o *Override) V2DeploymentUpdate(instance *current.IBPPeer, deployment *dep.Deployment) error {
	peerContainer, err := deployment.GetContainer(PEER)
	if err != nil {
		return err
	}
	peerContainer.AppendEnvIfMissing("PEER_NAME", instance.GetName())

	// For V2Deployments using chaincode-as-a-service and external builders, there is no need to include
	// or modify the chaincode launcher sidecar.
	if !instance.UsingCCLauncherImage() {
		if err := o.V2Deployment(instance, deployment); err != nil {
			return err
		}
		return nil
	}

	// V2DeploymentUpdate will be triggered when migrating from v1 to v2 peer, during this update we might
	// have to run initialization logic for a v2 fabric deployment. If the chaincode launcher container is
	// not found, we try to initialize the deployment based on v2 deployment to add chaincode launcher
	// before continuing with the remaining update logic. Not ideal, but until a bigger refactor is performed
	// this is the least intrusive way to handle this.
	ccLauncherContainer, err := deployment.GetContainer(CCLAUNCHER)
	if err != nil {
		if err := o.V2Deployment(instance, deployment); err != nil {
			return err
		}
		return nil
	}

	ccLauncherContainer = deployment.MustGetContainer(CCLAUNCHER)
	images := instance.Spec.Images
	if images != nil {
		if images.CCLauncherImage != "" && images.CCLauncherTag != "" {
			ccLauncherContainer.SetImage(images.CCLauncherImage, images.CCLauncherTag)
		}

		ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent(
			"FILETRANSFERIMAGE", image.Format(instance.Spec.Images.PeerInitImage, instance.Spec.Images.PeerInitTag),
		)
		ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent(
			"BUILDERIMAGE", image.Format(instance.Spec.Images.BuilderImage, instance.Spec.Images.BuilderTag),
		)
		ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent(
			"GOENVIMAGE", image.Format(instance.Spec.Images.GoEnvImage, instance.Spec.Images.GoEnvTag),
		)
		ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent(
			"JAVAENVIMAGE", image.Format(instance.Spec.Images.JavaEnvImage, instance.Spec.Images.JavaEnvTag),
		)
		ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent(
			"NODEENVIMAGE", image.Format(instance.Spec.Images.NodeEnvImage, instance.Spec.Images.NodeEnvTag),
		)
		ccLauncherContainer.AppendEnvIfMissing("CORE_PEER_LOCALMSPID", instance.Spec.MSPID)
	}

	resourcesRequest := instance.Spec.Resources
	if resourcesRequest != nil {
		if resourcesRequest.CCLauncher != nil {
			err := ccLauncherContainer.UpdateResources(resourcesRequest.CCLauncher)
			if err != nil {
				return errors.Wrap(err, "resource update for cclauncher failed")
			}
		}
	}

	return nil
}

func (o *Override) V24DeploymentUpdate(instance *current.IBPPeer, deployment *dep.Deployment) error {
	if instance.UsingCCLauncherImage() {
		ccLauncherContainer, err := deployment.GetContainer(CCLAUNCHER)
		if err != nil {
			return err
		}
		ccLauncherContainer.LivenessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS
		ccLauncherContainer.ReadinessProbe.HTTPGet.Scheme = corev1.URISchemeHTTPS

		deployment.UpdateContainer(ccLauncherContainer)
	}
	return nil
}

func (o *Override) CreateCCLauncherContainer(instance *current.IBPPeer, deployment *dep.Deployment) error {
	ccLauncherContainer, err := container.LoadFromFile(o.DefaultCCLauncherFile)
	if err != nil {
		return errors.Wrap(err, "failed to read default chaincode launcher container file")
	}

	images := instance.Spec.Images
	if images == nil || images.CCLauncherImage == "" {
		return errors.New("no image specified for chaincode launcher")
	}

	ccLauncherContainer.SetImage(images.CCLauncherImage, images.CCLauncherTag)
	ccLauncherContainer.AppendEnvIfMissing("KUBE_NAMESPACE", instance.GetNamespace())
	ccLauncherContainer.AppendEnvIfMissing("SHARED_VOLUME_PATH", "/cclauncher")
	ccLauncherContainer.AppendEnvIfMissing("IMAGEPULLSECRETS", strings.Join(instance.Spec.ImagePullSecrets, " "))
	ccLauncherContainer.AppendEnvIfMissing("CORE_PEER_LOCALMSPID", instance.Spec.MSPID)

	valueFrom := &corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{
			FieldPath: "metadata.name",
		},
	}
	ccLauncherContainer.AppendEnvVarValueFromIfMissing("PEER_POD_NAME", valueFrom)

	valueFrom = &corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{
			FieldPath: "metadata.uid",
		},
	}
	ccLauncherContainer.AppendEnvVarValueFromIfMissing("PEER_POD_UID", valueFrom)

	ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent("FILETRANSFERIMAGE", image.Format(instance.Spec.Images.PeerInitImage, instance.Spec.Images.PeerInitTag))
	ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent("BUILDERIMAGE", image.Format(instance.Spec.Images.BuilderImage, instance.Spec.Images.BuilderTag))
	ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent("GOENVIMAGE", image.Format(instance.Spec.Images.GoEnvImage, instance.Spec.Images.GoEnvTag))
	ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent("JAVAENVIMAGE", image.Format(instance.Spec.Images.JavaEnvImage, instance.Spec.Images.JavaEnvTag))
	ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent("NODEENVIMAGE", image.Format(instance.Spec.Images.NodeEnvImage, instance.Spec.Images.NodeEnvTag))
	ccLauncherContainer.AppendEnvIfMissingOverrideIfPresent("PEER_ID", instance.GetName())
	ccLauncherContainer.AppendVolumeMountIfMissing(fmt.Sprintf("%s-cclauncher", instance.Name), "/cclauncher")

	resourcesRequest := instance.Spec.Resources
	if resourcesRequest != nil {
		if resourcesRequest.CCLauncher != nil {
			err = ccLauncherContainer.UpdateResources(resourcesRequest.CCLauncher)
			if err != nil {
				return errors.Wrap(err, "resource update for cclauncher failed")
			}
		}
	}

	deployment.AddContainer(*ccLauncherContainer)
	return nil
}

func (o *Override) UpdateDeployment(instance *current.IBPPeer, k8sDep *appsv1.Deployment) error {
	deployment := dep.New(k8sDep)
	err := o.CommonDeploymentOverrides(instance, deployment)
	if err != nil {
		return err
	}

	switch version.GetMajorReleaseVersion(instance.Spec.FabricVersion) {
	case version.V1:
		err = o.V1Deployment(instance, deployment)
		if err != nil {
			return errors.Wrap(err, "failed during V1 peer deployment overrides")
		}
	case version.V2:
		err := o.V2DeploymentUpdate(instance, deployment)
		if err != nil {
			return errors.Wrapf(err, "failed to update V2 fabric deployment for instance '%s'", instance.GetName())
		}
		peerVersion := version.String(instance.Spec.FabricVersion)
		if peerVersion.EqualWithoutTag(version.V2_4_1) || peerVersion.EqualWithoutTag(version.V2_5_1) || peerVersion.GreaterThan(version.V2_4_1) {
			err := o.V24DeploymentUpdate(instance, deployment)
			if err != nil {
				return errors.Wrapf(err, "failed to update V24/V25 fabric deployment for instance '%s'", instance.GetName())
			}
		}
	}

	if instance.UsingCouchDB() {
		couchdb := deployment.MustGetContainer(COUCHDB)

		image := instance.Spec.Images
		if image != nil {
			couchdb.SetImage(image.CouchDBImage, image.CouchDBTag)
		}

		couchdb.AppendEnvIfMissing("SKIP_PERMISSIONS_UPDATE", "true")
	}

	if instance.UsingHSMProxy() {
		peerContainer := deployment.MustGetContainer(PEER)
		peerContainer.UpdateEnv("PKCS11_PROXY_SOCKET", instance.Spec.HSM.PKCS11Endpoint)
		deployment.UpdateContainer(peerContainer)
	} else if instance.IsHSMEnabled() {
		hsmInitCont := deployment.MustGetContainer(HSMCLIENT)
		image := instance.Spec.Images
		if image != nil {
			hsmInitCont.SetImage(image.HSMImage, image.HSMTag)
		}
	}

	return nil
}

func (o *Override) CommonDeploymentOverrides(instance *current.IBPPeer, deployment *dep.Deployment) error {
	initContainer := deployment.MustGetContainer(INIT)
	peerContainer := deployment.MustGetContainer(PEER)
	grpcContainer, err := deployment.GetContainer(PROXY)
	if err != nil {
		return errors.New("proxy container not found in deployment spec")
	}

	image := instance.Spec.Images
	if image != nil {
		initContainer.SetImage(image.PeerInitImage, image.PeerInitTag)
		peerContainer.SetImage(image.PeerImage, image.PeerTag)
		grpcContainer.SetImage(image.GRPCWebImage, image.GRPCWebTag)

		if instance.UsingCouchDB() {
			couchdb := deployment.MustGetContainer(COUCHDB)
			couchdb.SetImage(image.CouchDBImage, image.CouchDBTag)

			couchdbInitContainer := deployment.MustGetContainer(COUCHDBINIT)
			couchdbInitContainer.SetImage(image.PeerInitImage, image.PeerInitTag)
		}
	}

	resourcesRequest := instance.Spec.Resources
	if resourcesRequest != nil {
		if resourcesRequest.Peer != nil {
			err = peerContainer.UpdateResources(resourcesRequest.Peer)
			if err != nil {
				return errors.Wrap(err, "resource update for peer failed")
			}
		}

		if resourcesRequest.GRPCProxy != nil {
			err = grpcContainer.UpdateResources(resourcesRequest.GRPCProxy)
			if err != nil {
				return errors.Wrap(err, "resource update for grpcproxy failed")
			}
		}

		if resourcesRequest.Init != nil {
			err = initContainer.UpdateResources(resourcesRequest.Init)
			if err != nil {
				return errors.Wrap(err, "resource update for init failed")
			}
		}

		if instance.UsingCouchDB() {
			couchdb := deployment.MustGetContainer(COUCHDB)
			if resourcesRequest.CouchDB != nil {
				err = couchdb.UpdateResources(resourcesRequest.CouchDB)
				if err != nil {
					return errors.Wrap(err, "resource update for couchdb failed")
				}
			}

			couchdbinit := deployment.MustGetContainer(COUCHDBINIT)
			if resourcesRequest.Init != nil {
				err = couchdbinit.UpdateResources(resourcesRequest.Init)
				if err != nil {
					return errors.Wrap(err, "resource update for couchdb init failed")
				}
			}
		}
	}

	externalAddress := instance.Spec.PeerExternalEndpoint
	// Set external address to "do-not-set" in Peer CR spec to disable Service discovery
	if externalAddress != "" && externalAddress != "do-not-set" {
		peerContainer.AppendEnvIfMissing("CORE_PEER_GOSSIP_EXTERNALENDPOINT", externalAddress)
		peerContainer.AppendEnvIfMissing("CORE_PEER_GOSSIP_ENDPOINT", externalAddress)
		grpcContainer.AppendEnvIfMissing("EXTERNAL_ADDRESS", externalAddress)
	}

	if instance.Spec.Replicas != nil {
		if *instance.Spec.Replicas > 1 {
			return errors.New("replicas > 1 not allowed in IBPPeer")
		}
		deployment.SetReplicas(instance.Spec.Replicas)
	}

	deployment.UpdateContainer(peerContainer)
	deployment.UpdateContainer(grpcContainer)
	return nil
}

func (o *Override) CreateCouchDBContainers(instance *current.IBPPeer, deployment *dep.Deployment) error {
	couchdbUser := o.CouchdbUser
	if couchdbUser == "" {
		couchdbUser = util.GenerateRandomString(32)
	}

	couchdbPassword := o.CouchdbPassword
	if couchdbPassword == "" {
		couchdbPassword = util.GenerateRandomString(32)
	}

	couchdbContainer, err := container.LoadFromFile(o.DefaultCouchContainerFile)
	if err != nil {
		return errors.Wrap(err, "failed to read default couch container file")
	}

	couchdbInitContainer, err := container.LoadFromFile(o.DefaultCouchInitContainerFile)
	if err != nil {
		return errors.Wrap(err, "failed to read default couch init container file")
	}

	image := instance.Spec.Images
	if image != nil {
		couchdbContainer.SetImage(image.CouchDBImage, image.CouchDBTag)
		couchdbInitContainer.SetImage(image.PeerInitImage, image.PeerInitTag)
	}

	couchdbContainer.AppendEnvIfMissing("COUCHDB_USER", couchdbUser)
	couchdbContainer.AppendEnvIfMissing("COUCHDB_PASSWORD", couchdbPassword)
	couchdbContainer.AppendEnvIfMissing("SKIP_PERMISSIONS_UPDATE", "true")

	peerContainer := deployment.MustGetContainer(PEER)
	peerContainer.AppendEnvIfMissing("CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME", couchdbUser)
	peerContainer.AppendEnvIfMissing("CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD", couchdbPassword)
	peerContainer.AppendEnvIfMissing("CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS", "localhost:5984")
	peerContainer.AppendEnvIfMissing("CORE_LEDGER_STATE_COUCHDBCONFIG_MAXRETRIESONSTARTUP", "20")

	deployment.AddContainer(*couchdbContainer)
	deployment.AddInitContainer(*couchdbInitContainer)
	deployment.UpdateContainer(peerContainer)

	return nil
}

func (o *Override) GetAffinity(instance *current.IBPPeer) *corev1.Affinity {
	arch := instance.Spec.Arch
	zone := instance.Spec.Zone
	region := instance.Spec.Region
	nodeSelectorTerms := common.GetNodeSelectorTerms(arch, zone, region)

	orgName := instance.Spec.MSPID
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

func (o *Override) AdminSecretExists(instance *current.IBPPeer) bool {
	secret := &corev1.Secret{}
	err := o.Client.Get(context.TODO(), types.NamespacedName{
		Name:      fmt.Sprintf("ecert-%s-admincerts", instance.Name),
		Namespace: instance.Namespace}, secret)
	if err != nil {
		return false
	}

	return true
}

func (o *Override) OrdererCACertsSecretExists(instance *current.IBPPeer) bool {
	err := o.Client.Get(context.TODO(), types.NamespacedName{
		Name:      fmt.Sprintf("%s-orderercacerts", instance.Name),
		Namespace: instance.Namespace}, &corev1.Secret{})
	if err != nil {
		return false
	}

	return true
}

func (o *Override) GetTLSCACertsSecret(instance *current.IBPPeer, secretName string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := o.Client.Get(context.TODO(), types.NamespacedName{
		Name:      secretName,
		Namespace: instance.Namespace}, secret)
	if err != nil {
	}

	return secret, nil
}

func hsmInitContainer(instance *current.IBPPeer, hsmConfig *config.HSMConfig) *container.Container {
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
					corev1.ResourceCPU:    resource.MustParse("0.1"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			},
		},
	}
}

func hsmSettings(instance *current.IBPPeer, hsmConfig *config.HSMConfig, peerCont container.Container, deployment *deployment.Deployment) {
	for _, v := range hsmConfig.GetVolumes() {
		deployment.AppendVolumeIfMissing(v)
	}

	for _, vm := range hsmConfig.GetVolumeMounts() {
		peerCont.AppendVolumeMountStructIfMissing(vm)
	}

	for _, env := range hsmConfig.GetEnvs() {
		peerCont.AppendEnvStructIfMissing(env)
	}

	peerCont.AppendVolumeMountWithSubPathIfMissing("shared", "/hsm/lib", "hsm")

	if hsmConfig.Library.Auth != nil {
		deployment.Spec.Template.Spec.ImagePullSecrets = util.AppendPullSecretIfMissing(
			deployment.Spec.Template.Spec.ImagePullSecrets,
			hsmConfig.Library.Auth.ImagePullSecret,
		)
	}

	deployment.AddInitContainer(*hsmInitContainer(instance, hsmConfig))

	// If daemon settings are configured in HSM config, create a sidecar that is running the daemon image
	if hsmConfig.Daemon != nil {
		hsmDaemonSettings(instance, hsmConfig, peerCont, deployment)
	}
}

func hsmDaemonSettings(instance *current.IBPPeer, hsmConfig *config.HSMConfig, peerCont container.Container, deployment *deployment.Deployment) {
	// Unable to launch daemon if not running priviledged moe
	t := true
	peerCont.SecurityContext.Privileged = &t
	peerCont.SecurityContext.AllowPrivilegeEscalation = &t

	// Update command in deployment to ensure that deamon is running before starting the ca
	peerCont.Command = []string{
		"sh",
		"-c",
		fmt.Sprintf("%s && %s", config.DAEMON_CHECK_CMD, "peer node start"),
	}

	// This is the shared volume where the file 'pkcsslotd-luanched' is touched to let
	// other containers know that the daemon has successfully launched.
	peerCont.AppendVolumeMountIfMissing("shared", "/shared")

	pvcVolumeName := "fabric-peer-0"
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
		peerCont.AppendVolumeMountStructIfMissing(*pvcMount)
		deployment.AppendPVCVolumeIfMissing(pvcVolumeName, instance.PVCName())
	}
}
