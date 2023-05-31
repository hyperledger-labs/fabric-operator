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

package basepeer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/action"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate"
	commoninit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	peerconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/validator"
	controllerclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	jobv1 "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/job"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	"github.com/IBM-Blockchain/fabric-operator/pkg/migrator/peer/fabric"
	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/migrator/peer/fabric/v2"
	v25 "github.com/IBM-Blockchain/fabric-operator/pkg/migrator/peer/fabric/v25"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common/reconcilechecks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

var log = logf.Log.WithName("base_peer")

const (
	DefaultCouchContainer     = "./definitions/peer/couchdb.yaml"
	DefaultCouchInitContainer = "./definitions/peer/couchdb-init.yaml"

	defaultDeployment       = "./definitions/peer/deployment.yaml"
	defaultPVC              = "./definitions/peer/pvc.yaml"
	defaultCouchDBPVC       = "./definitions/peer/couchdb-pvc.yaml"
	defaultService          = "./definitions/peer/service.yaml"
	defaultRole             = "./definitions/peer/role.yaml"
	defaultServiceAccount   = "./definitions/peer/serviceaccount.yaml"
	defaultRoleBinding      = "./definitions/peer/rolebinding.yaml"
	defaultFluentdConfigMap = "./definitions/peer/fluentd-configmap.yaml"

	DaysToSecondsConversion = int64(24 * 60 * 60)
)

type Override interface {
	Deployment(v1.Object, *appsv1.Deployment, resources.Action) error
	Service(v1.Object, *corev1.Service, resources.Action) error
	PVC(v1.Object, *corev1.PersistentVolumeClaim, resources.Action) error
	StateDBPVC(v1.Object, *corev1.PersistentVolumeClaim, resources.Action) error
}

//go:generate counterfeiter -o mocks/deployment_manager.go -fake-name DeploymentManager . DeploymentManager

type DeploymentManager interface {
	resources.Manager
	CheckForSecretChange(v1.Object, string, func(string, *appsv1.Deployment) bool) error
	DeploymentStatus(v1.Object) (appsv1.DeploymentStatus, error)
	GetScheme() *runtime.Scheme
}

//go:generate counterfeiter -o mocks/initializer.go -fake-name InitializeIBPPeer . InitializeIBPPeer

type InitializeIBPPeer interface {
	GenerateOrdererCACertsSecret(instance *current.IBPPeer, certs map[string][]byte) error
	GenerateSecrets(prefix commoninit.SecretType, instance v1.Object, crypto *commonconfig.Response) error
	Create(initializer.CoreConfig, initializer.IBPPeer, string) (*initializer.Response, error)
	Update(initializer.CoreConfig, initializer.IBPPeer) (*initializer.Response, error)
	CheckIfAdminCertsUpdated(*current.IBPPeer) (bool, error)
	UpdateAdminSecret(*current.IBPPeer) error
	MissingCrypto(*current.IBPPeer) bool
	GetInitPeer(instance *current.IBPPeer, storagePath string) (*initializer.Peer, error)
	GetUpdatedPeer(instance *current.IBPPeer) (*initializer.Peer, error)
	GenerateSecretsFromResponse(instance *current.IBPPeer, cryptoResponse *commonconfig.CryptoResponse) error
	UpdateSecretsFromResponse(instance *current.IBPPeer, cryptoResponse *commonconfig.CryptoResponse) error
	GetCrypto(instance *current.IBPPeer) (*commonconfig.CryptoResponse, error)
	CoreConfigMap() *initializer.CoreConfigMap
}

//go:generate counterfeiter -o mocks/certificate_manager.go -fake-name CertificateManager . CertificateManager

type CertificateManager interface {
	CheckCertificatesForExpire(instance v1.Object, numSecondsBeforeExpire int64) (current.IBPCRStatusType, string, error)
	GetSignCert(string, string) ([]byte, error)
	GetDurationToNextRenewal(commoninit.SecretType, v1.Object, int64) (time.Duration, error)
	RenewCert(commoninit.SecretType, certificate.Instance, *current.EnrollmentSpec, *commonapi.BCCSP, string, bool, bool) error
}

//go:generate counterfeiter -o mocks/restart_manager.go -fake-name RestartManager . RestartManager

type RestartManager interface {
	ForAdminCertUpdate(instance v1.Object) error
	ForCertUpdate(certType commoninit.SecretType, instance v1.Object) error
	ForConfigOverride(instance v1.Object) error
	ForNodeOU(instance v1.Object) error
	ForRestartAction(instance v1.Object) error
	TriggerIfNeeded(instance restart.Instance) error
}

//go:generate counterfeiter -o mocks/update.go -fake-name Update . Update
type Update interface {
	SpecUpdated() bool
	ConfigOverridesUpdated() bool
	DindArgsUpdated() bool
	TLSCertUpdated() bool
	EcertUpdated() bool
	PeerTagUpdated() bool
	CertificateUpdated() bool
	SetDindArgsUpdated(updated bool)
	RestartNeeded() bool
	EcertReenrollNeeded() bool
	TLSReenrollNeeded() bool
	EcertNewKeyReenroll() bool
	TLScertNewKeyReenroll() bool
	MigrateToV2() bool
	MigrateToV24() bool
	MigrateToV25() bool
	UpgradeDBs() bool
	MSPUpdated() bool
	EcertEnroll() bool
	TLSCertEnroll() bool
	CertificateCreated() bool
	GetCreatedCertType() commoninit.SecretType
	CryptoBackupNeeded() bool
	NodeOUUpdated() bool
	FabricVersionUpdated() bool
	ImagesUpdated() bool
}

type IBPPeer interface {
	Initialize(instance *current.IBPPeer, update Update) error
	CheckStates(instance *current.IBPPeer) error
	PreReconcileChecks(instance *current.IBPPeer, update Update) (bool, error)
	ReconcileManagers(instance *current.IBPPeer, update Update) error
	Reconcile(instance *current.IBPPeer, update Update) (common.Result, error)
}

type CoreConfig interface {
	GetMaxNameLength() *int
	GetAddressOverrides() []peerconfig.AddressOverride
	GetBCCSPSection() *commonapi.BCCSP
	MergeWith(interface{}, bool) error
	SetPKCS11Defaults(bool)
	ToBytes() ([]byte, error)
	UsingPKCS11() bool
	SetBCCSPLibrary(string)
}

var _ IBPPeer = &Peer{}

type Peer struct {
	Client controllerclient.Client
	Scheme *runtime.Scheme
	Config *config.Config

	DeploymentManager       DeploymentManager
	ServiceManager          resources.Manager
	PVCManager              resources.Manager
	StateDBPVCManager       resources.Manager
	FluentDConfigMapManager resources.Manager
	RoleManager             resources.Manager
	RoleBindingManager      resources.Manager
	ServiceAccountManager   resources.Manager

	Override    Override
	Initializer InitializeIBPPeer

	CertificateManager CertificateManager
	RenewCertTimers    map[string]*time.Timer

	Restart RestartManager
}

func New(client controllerclient.Client, scheme *runtime.Scheme, config *config.Config, o Override) *Peer {
	p := &Peer{
		Client:   client,
		Scheme:   scheme,
		Config:   config,
		Override: o,
	}

	p.CreateManagers()

	validator := &validator.Validator{
		Client: client,
	}

	init := initializer.New(config.PeerInitConfig, scheme, client, p.GetLabels, validator, config.Operator.Peer.Timeouts.EnrollJob)
	p.Initializer = init

	p.CertificateManager = certificate.New(client, scheme)
	p.RenewCertTimers = make(map[string]*time.Timer)

	p.Restart = restart.New(client, config.Operator.Restart.WaitTime.Get(), config.Operator.Restart.Timeout.Get())

	return p
}

func (p *Peer) CreateManagers() {
	override := p.Override
	resourceManager := resourcemanager.New(p.Client, p.Scheme)
	peerConfig := p.Config.PeerInitConfig

	p.DeploymentManager = resourceManager.CreateDeploymentManager("", override.Deployment, p.GetLabels, peerConfig.DeploymentFile)
	p.PVCManager = resourceManager.CreatePVCManager("", override.PVC, p.GetLabels, peerConfig.PVCFile)
	p.StateDBPVCManager = resourceManager.CreatePVCManager("statedb", override.StateDBPVC, p.GetLabels, peerConfig.CouchDBPVCFile)
	p.FluentDConfigMapManager = resourceManager.CreateConfigMapManager("fluentd", nil, p.GetLabels, peerConfig.FluentdConfigMapFile, nil)
	p.RoleManager = resourceManager.CreateRoleManager("", nil, p.GetLabels, peerConfig.RoleFile)
	p.RoleBindingManager = resourceManager.CreateRoleBindingManager("", nil, p.GetLabels, peerConfig.RoleBindingFile)
	p.ServiceAccountManager = resourceManager.CreateServiceAccountManager("", nil, p.GetLabels, peerConfig.ServiceAccountFile)
	p.ServiceManager = resourceManager.CreateServiceManager("", override.Service, p.GetLabels, peerConfig.ServiceFile)
}

func (p *Peer) PreReconcileChecks(instance *current.IBPPeer, update Update) (bool, error) {
	var maxNameLength *int

	imagesUpdated, err := reconcilechecks.FabricVersionHelper(instance, p.Config.Operator.Versions, update)
	if err != nil {
		return false, errors.Wrap(err, "failed to during version and image checks")
	}

	co, err := instance.GetConfigOverride()
	if err != nil {
		return false, err
	}

	configOverride := co.(CoreConfig)
	maxNameLength = configOverride.GetMaxNameLength()

	err = util.ValidationChecks(instance.TypeMeta, instance.ObjectMeta, "IBPPeer", maxNameLength)
	if err != nil {
		return false, err
	}

	if instance.Spec.Action.Enroll.Ecert && instance.Spec.Action.Reenroll.Ecert {
		return false, errors.New("both enroll and renenroll action requested for ecert, must only select one")
	}

	if instance.Spec.Action.Enroll.TLSCert && instance.Spec.Action.Reenroll.TLSCert {
		return false, errors.New("both enroll and renenroll action requested for TLS cert, must only select one")
	}

	if instance.Spec.Action.Reenroll.Ecert && instance.Spec.Action.Reenroll.EcertNewKey {
		return false, errors.New("both reenroll and renenroll with new action requested for ecert, must only select one")
	}

	if instance.Spec.Action.Reenroll.TLSCert && instance.Spec.Action.Reenroll.TLSCertNewKey {
		return false, errors.New("both reenroll and renenroll with new action requested for TLS cert, must only select one")
	}

	if instance.Spec.HSMSet() {
		err = util.ValidateHSMProxyURL(instance.Spec.HSM.PKCS11Endpoint)
		if err != nil {
			return false, errors.Wrapf(err, "invalid HSM endpoint for peer instance '%s'", instance.GetName())
		}
	}

	hsmImageUpdated := p.ReconcileHSMImages(instance)

	if !instance.Spec.DomainSet() {
		return false, fmt.Errorf("domain not set for peer instance '%s'", instance.GetName())
	}

	zoneUpdated, err := p.SelectZone(instance)
	if err != nil {
		return false, err
	}

	regionUpdated, err := p.SelectRegion(instance)
	if err != nil {
		return false, err
	}

	var replicasUpdated bool
	if instance.Spec.Replicas == nil {
		replicas := int32(1)
		instance.Spec.Replicas = &replicas
		replicasUpdated = true
	}

	dbTypeUpdated := p.CheckDBType(instance)
	updated := dbTypeUpdated || zoneUpdated || regionUpdated || update.DindArgsUpdated() || hsmImageUpdated || replicasUpdated || imagesUpdated

	if updated {
		log.Info(fmt.Sprintf(
			"dbTypeUpdate %t, zoneUpdated %t, regionUpdated %t, dindArgsUpdated %t, hsmImageUpdated %t, replicasUpdated %t, imagesUpdated %t",
			dbTypeUpdated,
			zoneUpdated,
			regionUpdated,
			update.DindArgsUpdated(),
			hsmImageUpdated,
			replicasUpdated,
			imagesUpdated))
	}

	return updated, nil
}

func (p *Peer) SetVersion(instance *current.IBPPeer) (bool, error) {
	if instance.Status.Version == "" || !version.String(instance.Status.Version).Equal(version.Operator) {
		log.Info("Version of Operator: ", "version", version.Operator)
		log.Info("Version of CR: ", "version", instance.Status.Version)
		log.Info(fmt.Sprintf("Updating CR '%s' to version '%s'", instance.Name, version.Operator))

		instance.Status.Version = version.Operator
		err := p.Client.PatchStatus(context.TODO(), instance, nil, controllerclient.PatchOption{
			Resilient: &controllerclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPPeer{},
				Strategy: k8sclient.MergeFrom,
			},
		})
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (p *Peer) Initialize(instance *current.IBPPeer, update Update) error {
	var err error

	log.Info(fmt.Sprintf("Checking if peer '%s' needs initialization", instance.GetName()))

	// TODO: Add checks to determine if initialization is neeeded. Split this method into
	// two, one should handle initialization during the create event of a CR and the other
	// should update events

	// Service account is required by HSM init job
	if err := p.ReconcilePeerRBAC(instance); err != nil {
		return err
	}

	if instance.IsHSMEnabled() {
		// If HSM config not found, HSM proxy is being used
		if instance.UsingHSMProxy() {
			err = os.Setenv("PKCS11_PROXY_SOCKET", instance.Spec.HSM.PKCS11Endpoint)
			if err != nil {
				return err
			}
		} else {
			hsmConfig, err := commonconfig.ReadHSMConfig(p.Client, instance)
			if err != nil {
				return errors.New("using non-proxy HSM, but no HSM config defined as config map 'ibp-hsm-config'")
			}

			if hsmConfig.Daemon != nil {
				log.Info("Using daemon based HSM, creating pvc...")
				p.PVCManager.SetCustomName(instance.Spec.CustomNames.PVC.Peer)
				err = p.PVCManager.Reconcile(instance, update.SpecUpdated())
				if err != nil {
					return errors.Wrap(err, "failed PVC reconciliation")
				}
			}
		}
	}

	peerConfig := p.Config.PeerInitConfig.CorePeerFile
	if version.GetMajorReleaseVersion(instance.Spec.FabricVersion) == version.V2 {
		peerversion := version.String(instance.Spec.FabricVersion)
		peerConfig = p.Config.PeerInitConfig.CorePeerV2File
		if peerversion.EqualWithoutTag(version.V2_5_1) || peerversion.GreaterThan(version.V2_5_1) {
			peerConfig = p.Config.PeerInitConfig.CorePeerV25File
		}
	}

	if instance.UsingHSMProxy() {
		err = os.Setenv("PKCS11_PROXY_SOCKET", instance.Spec.HSM.PKCS11Endpoint)
		if err != nil {
			return err
		}
	}

	storagePath := p.GetInitStoragePath(instance)
	initPeer, err := p.Initializer.GetInitPeer(instance, storagePath)
	if err != nil {
		return err
	}
	initPeer.UsingHSMProxy = instance.UsingHSMProxy()
	initPeer.Config, err = initializer.GetCoreConfigFromFile(instance, peerConfig)
	if err != nil {
		return err
	}

	updated := update.ConfigOverridesUpdated() || update.NodeOUUpdated()
	if update.ConfigOverridesUpdated() {
		err = p.InitializeUpdateConfigOverride(instance, initPeer)
		if err != nil {
			return err
		}
		// Request deployment restart for config override update
		if err = p.Restart.ForConfigOverride(instance); err != nil {
			return err
		}
	}
	if update.NodeOUUpdated() {
		err = p.InitializeUpdateNodeOU(instance)
		if err != nil {
			return err
		}
		// Request deloyment restart for node OU update
		if err = p.Restart.ForNodeOU(instance); err != nil {
			return err
		}
	}
	if !updated {
		err = p.InitializeCreate(instance, initPeer)
		if err != nil {
			return err
		}
	}

	updateNeeded, err := p.Initializer.CheckIfAdminCertsUpdated(instance)
	if err != nil {
		return err
	}
	if updateNeeded {
		err = p.Initializer.UpdateAdminSecret(instance)
		if err != nil {
			return err
		}
		// Request deployment restart for admin cert updates
		if err = p.Restart.ForAdminCertUpdate(instance); err != nil {
			return err
		}
	}

	return nil
}

func (p *Peer) InitializeUpdateConfigOverride(instance *current.IBPPeer, initPeer *initializer.Peer) error {
	var err error

	if p.Initializer.MissingCrypto(instance) {
		// If crypto is missing, we should run the create logic
		err := p.InitializeCreate(instance, initPeer)
		if err != nil {
			return err
		}

		return nil
	}

	log.Info(fmt.Sprintf("Initialize peer '%s' during update config override", instance.GetName()))

	cm, err := initializer.GetCoreFromConfigMap(p.Client, instance)
	if err != nil {
		return err
	}

	initPeer.Config, err = initializer.GetCoreConfigFromBytes(instance, cm.BinaryData["core.yaml"])
	if err != nil {
		return err
	}

	co, err := instance.GetConfigOverride()
	if err != nil {
		return err
	}
	configOverrides := co.(CoreConfig)

	resp, err := p.Initializer.Update(configOverrides, initPeer)
	if err != nil {
		return err
	}

	if resp != nil {
		if resp.Config != nil {
			// Update core.yaml in config map
			err = p.Initializer.CoreConfigMap().CreateOrUpdate(instance, resp.Config)
			if err != nil {
				return err
			}
		}

		if len(resp.DeliveryClientCerts) > 0 {
			log.Info(fmt.Sprintf("Orderer CA Certs detected in DeliveryClient config, creating secret '%s-orderercacerts' with certs", instance.Name))
			err = p.Initializer.GenerateOrdererCACertsSecret(instance, resp.DeliveryClientCerts)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Peer) InitializeUpdateNodeOU(instance *current.IBPPeer) error {
	log.Info(fmt.Sprintf("Node OU updated with enabled: %t for peer '%s", !instance.Spec.NodeOUDisabled(), instance.GetName()))

	crypto, err := p.Initializer.GetCrypto(instance)
	if err != nil {
		return err
	}
	if !instance.Spec.NodeOUDisabled() {
		if err := crypto.VerifyCertOU("peer"); err != nil {
			return err
		}
	} else {
		// If nodeOUDisabled, admin certs are required
		if crypto.Enrollment.AdminCerts == nil {
			return errors.New("node OU disabled, admin certs are required but missing")
		}
	}

	// Update config.yaml in config map
	err = p.Initializer.CoreConfigMap().AddNodeOU(instance)
	if err != nil {
		return err
	}

	return nil
}

func (p *Peer) InitializeCreate(instance *current.IBPPeer, initPeer *initializer.Peer) error {
	var err error

	if p.ConfigExists(instance) {
		log.Info(fmt.Sprintf("Config '%s-config' exists, not reinitializing peer", instance.GetName()))
		return nil
	}

	log.Info(fmt.Sprintf("Initialize peer '%s' during create", instance.GetName()))

	storagePath := p.GetInitStoragePath(instance)

	co, err := instance.GetConfigOverride()
	if err != nil {
		return err
	}
	configOverrides := co.(CoreConfig)

	resp, err := p.Initializer.Create(configOverrides, initPeer, storagePath)
	if err != nil {
		return err
	}

	if resp != nil {
		if resp.Crypto != nil {
			if !instance.Spec.NodeOUDisabled() {
				if err := resp.Crypto.VerifyCertOU("peer"); err != nil {
					return err
				}
			}

			err = p.Initializer.GenerateSecretsFromResponse(instance, resp.Crypto)
			if err != nil {
				return err
			}
		}

		if len(resp.DeliveryClientCerts) > 0 {
			log.Info(fmt.Sprintf("Orderer CA Certs detected in DeliverClient config, creating secret '%s-orderercacerts' with certs", instance.Name))
			err = p.Initializer.GenerateOrdererCACertsSecret(instance, resp.DeliveryClientCerts)
			if err != nil {
				return err
			}
		}

		if resp.Config != nil {
			if instance.IsHSMEnabled() && !instance.UsingHSMProxy() {
				hsmConfig, err := commonconfig.ReadHSMConfig(p.Client, instance)
				if err != nil {
					return err
				}
				resp.Config.SetBCCSPLibrary(filepath.Join("/hsm/lib", filepath.Base(hsmConfig.Library.FilePath)))
			}

			err = p.Initializer.CoreConfigMap().CreateOrUpdate(instance, resp.Config)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Peer) Reconcile(instance *current.IBPPeer, update Update) (common.Result, error) {
	var err error
	var status *current.CRStatus

	versionSet, err := p.SetVersion(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, fmt.Sprintf("failed updating CR '%s' to version '%s'", instance.Name, version.Operator))
	}
	if versionSet {
		log.Info("Instance version updated, requeuing request...")
		return common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
		}, nil
	}

	instanceUpdated, err := p.PreReconcileChecks(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed pre reconcile checks")
	}

	// We do not have to wait for service to get the external endpoint
	// thus we call UpdateExternalEndpoint in reconcile before reconcile managers
	externalEndpointUpdated := p.UpdateExternalEndpoint(instance)

	if instanceUpdated || externalEndpointUpdated {
		log.Info(fmt.Sprintf("Updating instance after pre reconcile checks: %t, updating external endpoint: %t", instanceUpdated, externalEndpointUpdated))
		err = p.Client.Patch(context.TODO(), instance, nil, controllerclient.PatchOption{
			Resilient: &controllerclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPPeer{},
				Strategy: k8sclient.MergeFrom,
			},
		})
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update instance")
		}

		log.Info("Instance updated, requeuing request...")
		return common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
		}, nil
	}

	err = p.Initialize(instance, update)
	if err != nil {
		return common.Result{}, operatorerrors.Wrap(err, operatorerrors.PeerInitilizationFailed, "failed to initialize peer")
	}

	err = p.ReconcileManagers(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to reconcile managers")
	}

	err = p.UpdateConnectionProfile(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to create connection profile")
	}

	err = p.CheckStates(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check and restore state")
	}

	// custom product logic can be implemented here
	// No-Op atm
	status, result, err := p.CustomLogic(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to run custom offering logic")
	}
	if result != nil {
		log.Info(fmt.Sprintf("Finished reconciling '%s' with Custom Logic result", instance.GetName()))
		return *result, nil
	}

	if update.MSPUpdated() {
		err = p.UpdateMSPCertificates(instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update certificates passed in MSP spec")
		}
		// A successful update will trigger a tlsCertUpdated or ecertUpdated event, which will handle restarting deployment
	}

	if update.EcertUpdated() {
		log.Info("Ecert was updated")
		// Request deployment restart for tls cert update
		err = p.Restart.ForCertUpdate(commoninit.ECERT, instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	if update.TLSCertUpdated() {
		log.Info("TLS cert was updated")
		// Request deployment restart for ecert update
		err = p.Restart.ForCertUpdate(commoninit.TLS, instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	if err := p.HandleActions(instance, update); err != nil {
		return common.Result{}, err
	}

	if err := p.HandleRestart(instance, update); err != nil {
		return common.Result{}, err
	}

	return common.Result{
		Status: status,
	}, nil
}

func (p *Peer) ReconcileManagers(instance *current.IBPPeer, updated Update) error {
	var err error

	update := updated.SpecUpdated()

	p.PVCManager.SetCustomName(instance.Spec.CustomNames.PVC.Peer)
	err = p.PVCManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed PVC reconciliation")
	}

	p.StateDBPVCManager.SetCustomName(instance.Spec.CustomNames.PVC.StateDB)
	err = p.StateDBPVCManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed CouchDB PVC reconciliation")
	}

	err = p.ReconcileSecret(instance)
	if err != nil {
		return errors.Wrap(err, "failed Secret reconciliation")
	}

	err = p.ServiceManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Service reconciliation")
	}

	err = p.DeploymentManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Deployment reconciliation")
	}

	err = p.ReconcilePeerRBAC(instance)
	if err != nil {
		return errors.Wrap(err, "failed RBAC reconciliation")
	}

	err = p.FluentDConfigMapManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed FluentD ConfigMap reconciliation")
	}

	return nil
}

func (p *Peer) ReconcilePeerRBAC(instance *current.IBPPeer) error {
	var err error

	err = p.RoleManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	err = p.RoleBindingManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	err = p.ServiceAccountManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	return nil
}

// this function makes sure the deployment spec matches the expected state
func (p *Peer) CheckStates(instance *current.IBPPeer) error {
	if p.DeploymentManager.Exists(instance) {
		err := p.DeploymentManager.CheckState(instance)
		if err != nil {
			log.Error(err, "unexpected state")
			err = p.DeploymentManager.RestoreState(instance)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Peer) ReconcileSecret(instance *current.IBPPeer) error {
	name := instance.Spec.MSPSecret
	if name == "" {
		name = instance.Name + "-secret" // default value for secret, if none specified
	}

	secret := &corev1.Secret{}
	err := p.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.Namespace}, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Creating secret '%s'", name))
			createErr := p.CreateSecret(instance)
			if createErr != nil {
				return createErr
			}
			return nil
		}
		return err
	} else {
		log.Info(fmt.Sprintf("Updating secret '%s'", name))
		updateErr := p.UpdateSecret(instance, secret)
		if updateErr != nil {
			return updateErr
		}
	}

	return nil
}

func (p *Peer) CreateSecret(instance *current.IBPPeer) error {
	secret := &corev1.Secret{}
	secret.Name = instance.Spec.MSPSecret
	if secret.Name == "" {
		secret.Name = instance.Name + "-secret" // default value for secret, if none specified
	}
	secret.Namespace = instance.Namespace
	secret.Labels = p.GetLabels(instance)

	secretData := instance.Spec.Secret
	bytesData, err := json.Marshal(secretData)
	if err != nil {
		return err
	}
	secret.Data = make(map[string][]byte)
	secret.Data["secret.json"] = bytesData

	err = p.Client.Create(context.TODO(), secret, controllerclient.CreateOption{Owner: instance, Scheme: p.Scheme})
	if err != nil {
		return err
	}

	return nil
}

func (p *Peer) UpdateSecret(instance *current.IBPPeer, secret *corev1.Secret) error {
	secretData := instance.Spec.Secret
	bytesData, err := json.Marshal(secretData)
	if err != nil {
		return err
	}

	if secret.Data != nil && !bytes.Equal(secret.Data["secret.json"], bytesData) {
		secret.Data["secret.json"] = bytesData

		err = p.Client.Update(context.TODO(), secret, controllerclient.UpdateOption{Owner: instance, Scheme: p.Scheme})
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Peer) UpdateExternalEndpoint(instance *current.IBPPeer) bool {
	// Disable Service discovery
	if instance.Spec.PeerExternalEndpoint == "do-not-set" {
		return false
	}

	if instance.Spec.PeerExternalEndpoint == "" {
		instance.Spec.PeerExternalEndpoint = instance.Namespace + "-" + instance.Name + "-peer." + instance.Spec.Domain + ":443"
		return true
	}
	return false
}

func (p *Peer) SelectZone(instance *current.IBPPeer) (bool, error) {
	if instance.Spec.Zone == "select" {
		zone := util.GetZone(p.Client)
		instance.Spec.Zone = zone
		log.Info(fmt.Sprintf("Setting zone to '%s', and updating spec", zone))
		return true, nil
	}
	if instance.Spec.Zone != "" {
		err := util.ValidateZone(p.Client, instance.Spec.Zone)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (p *Peer) SelectRegion(instance *current.IBPPeer) (bool, error) {
	if instance.Spec.Region == "select" {
		region := util.GetRegion(p.Client)
		instance.Spec.Region = region
		log.Info(fmt.Sprintf("Setting region to '%s', and updating spec", region))
		return true, nil
	}
	if instance.Spec.Region != "" {
		err := util.ValidateRegion(p.Client, instance.Spec.Region)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (p *Peer) CheckDBType(instance *current.IBPPeer) bool {
	if instance.Spec.StateDb == "" {
		log.Info("Setting statedb type to 'CouchDB', and updating spec")
		instance.Spec.StateDb = "CouchDB"
		return true
	}

	return false
}

func (p *Peer) GetLabels(instance v1.Object) map[string]string {
	label := os.Getenv("OPERATOR_LABEL_PREFIX")
	if label == "" {
		label = "fabric"
	}

	i := instance.(*current.IBPPeer)
	return map[string]string{
		"app":                          instance.GetName(),
		"creator":                      label,
		"orgname":                      i.Spec.MSPID,
		"app.kubernetes.io/name":       label,
		"app.kubernetes.io/instance":   label + "peer",
		"app.kubernetes.io/managed-by": label + "-operator",
	}
}

func (p *Peer) UpdateConnectionProfile(instance *current.IBPPeer) error {
	var err error

	endpoints := p.GetEndpoints(instance)

	tlscert, err := common.GetTLSSignCertEncoded(p.Client, instance)
	if err != nil {
		return err
	}

	tlscacerts, err := common.GetTLSCACertEncoded(p.Client, instance)
	if err != nil {
		return err
	}

	tlsintercerts, err := common.GetTLSIntercertEncoded(p.Client, instance)
	if err != nil {
		return err
	}

	ecert, err := common.GetEcertSignCertEncoded(p.Client, instance)
	if err != nil {
		return err
	}

	cacert, err := common.GetEcertCACertEncoded(p.Client, instance)
	if err != nil {
		return err
	}

	admincerts, err := common.GetEcertAdmincertEncoded(p.Client, instance)
	if err != nil {
		return err
	}

	if len(tlsintercerts) > 0 {
		tlscacerts = tlsintercerts
	}

	err = p.UpdateConnectionProfileConfigmap(instance, *endpoints, tlscert, tlscacerts, ecert, cacert, admincerts)
	if err != nil {
		return err
	}

	return nil
}

func (p *Peer) UpdateConnectionProfileConfigmap(instance *current.IBPPeer, endpoints current.PeerEndpoints, tlscert string, tlscacerts []string, ecert string, cacert []string, admincerts []string) error {
	// TODO add ecert.intermediatecerts and ecert.admincerts
	// TODO add tls.cacerts
	// TODO get the whole PeerConnectionProfile object from caller??
	name := instance.Name + "-connection-profile"
	connectionProfile := &current.PeerConnectionProfile{
		Endpoints: endpoints,
		TLS: &current.MSP{
			SignCerts: tlscert,
			CACerts:   tlscacerts,
		},
		Component: &current.MSP{
			SignCerts:  ecert,
			CACerts:    cacert,
			AdminCerts: admincerts,
		},
	}

	bytes, err := json.Marshal(connectionProfile)
	if err != nil {
		return errors.Wrap(err, "failed to marshal connectionprofile")
	}
	cm := &corev1.ConfigMap{
		BinaryData: map[string][]byte{"profile.json": bytes},
	}
	cm.Name = name
	cm.Namespace = instance.Namespace
	cm.Labels = p.GetLabels(instance)

	nn := types.NamespacedName{
		Name:      name,
		Namespace: instance.GetNamespace(),
	}

	err = p.Client.Get(context.TODO(), nn, &corev1.ConfigMap{})
	if err == nil {
		log.Info(fmt.Sprintf("Updating connection profle configmap for %s", instance.Name))
		err = p.Client.Update(context.TODO(), cm, controllerclient.UpdateOption{
			Owner:  instance,
			Scheme: p.Scheme,
		})
		if err != nil {
			return errors.Wrap(err, "failed to update connection profile configmap")
		}
	} else {
		log.Info(fmt.Sprintf("Creating connection profle configmap for %s", instance.Name))
		err = p.Client.Create(context.TODO(), cm, controllerclient.CreateOption{
			Owner:  instance,
			Scheme: p.Scheme,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create connection profile configmap")
		}
	}
	return nil
}

func (p *Peer) GetEndpoints(instance *current.IBPPeer) *current.PeerEndpoints {
	endpoints := &current.PeerEndpoints{
		API:        "grpcs://" + instance.Namespace + "-" + instance.Name + "-peer." + instance.Spec.Domain + ":443",
		Operations: "https://" + instance.Namespace + "-" + instance.Name + "-operations." + instance.Spec.Domain + ":443",
		Grpcweb:    "https://" + instance.Namespace + "-" + instance.Name + "-grpcweb." + instance.Spec.Domain + ":443",
	}
	return endpoints
}

func (p *Peer) ConfigExists(instance *current.IBPPeer) bool {
	name := fmt.Sprintf("%s-config", instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: instance.Namespace,
	}

	cm := &corev1.ConfigMap{}
	err := p.Client.Get(context.TODO(), namespacedName, cm)
	if err != nil {
		return false
	}

	return true
}

func (p *Peer) CheckCSRHosts(instance *current.IBPPeer, hosts []string) bool {
	if instance.Spec.Secret != nil {
		if instance.Spec.Secret.Enrollment != nil {
			if instance.Spec.Secret.Enrollment.TLS == nil {
				instance.Spec.Secret.Enrollment.TLS = &current.Enrollment{}
			}
			if instance.Spec.Secret.Enrollment.TLS.CSR == nil {
				instance.Spec.Secret.Enrollment.TLS.CSR = &current.CSR{}
				instance.Spec.Secret.Enrollment.TLS.CSR.Hosts = hosts
				return true
			} else {
				originalLength := len(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts)
				for _, host := range instance.Spec.Secret.Enrollment.TLS.CSR.Hosts {
					hosts = util.AppendStringIfMissing(hosts, host)
				}
				instance.Spec.Secret.Enrollment.TLS.CSR.Hosts = hosts
				newLength := len(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts)
				return originalLength != newLength
			}
		}
	}
	return false
}

func (p *Peer) GetBCCSPSectionForInstance(instance *current.IBPPeer) (*commonapi.BCCSP, error) {
	var bccsp *commonapi.BCCSP
	if instance.IsHSMEnabled() {
		co, err := instance.GetConfigOverride()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get configoverride")
		}

		configOverride := co.(CoreConfig)
		configOverride.SetPKCS11Defaults(instance.UsingHSMProxy())
		bccsp = configOverride.GetBCCSPSection()
	}

	return bccsp, nil
}

func (p *Peer) GetInitStoragePath(instance *current.IBPPeer) string {
	if p.Config != nil && p.Config.PeerInitConfig != nil && p.Config.PeerInitConfig.StoragePath != "" {
		return filepath.Join(p.Config.PeerInitConfig.StoragePath, instance.GetName())
	}

	return filepath.Join("/", "peerinit", instance.GetName())
}

func (p *Peer) ReconcileFabricPeerMigrationV1_4(instance *current.IBPPeer) error {
	peerConfig, err := p.FabricPeerMigrationV1_4(instance)
	if err != nil {
		return errors.Wrap(err, "failed to migrate peer between fabric versions")
	}

	if peerConfig != nil {
		log.Info("Peer config updated during fabric peer migration, updating config map...")
		if err := p.Initializer.CoreConfigMap().CreateOrUpdate(instance, peerConfig); err != nil {
			return errors.Wrapf(err, "failed to create/update '%s' peer's config map", instance.GetName())
		}
	}

	return nil
}

// Moving to fabric version above 1.4.6 require that the `msp/keystore` value be removed
// from BCCSP section if configured to use PKCS11 (HSM). NOTE: This does not support
// migration across major release, will not cover migration peer from 1.4.x to 2.x
func (p *Peer) FabricPeerMigrationV1_4(instance *current.IBPPeer) (*peerconfig.Core, error) {
	if !instance.IsHSMEnabled() {
		return nil, nil
	}

	peerTag := instance.Spec.Images.PeerTag
	if !strings.Contains(peerTag, "sha") {
		tag := strings.Split(peerTag, "-")[0]

		peerVersion := version.String(tag)
		if !peerVersion.GreaterThan(version.V1_4_6) {
			return nil, nil
		}

		log.Info(fmt.Sprintf("Peer moving to fabric version %s", peerVersion))
	} else {
		if version.GetMajorReleaseVersion(instance.Spec.FabricVersion) == version.V2 {
			return nil, nil
		}
		log.Info(fmt.Sprintf("Peer moving to digest %s", peerTag))
	}

	// Read peer config map and remove keystore value from BCCSP section
	cm, err := initializer.GetCoreFromConfigMap(p.Client, instance)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get '%s' peer's config map", instance.GetName())
	}

	peerConfig := &peerconfig.Core{}
	if err := yaml.Unmarshal(cm.BinaryData["core.yaml"], peerConfig); err != nil {
		return nil, errors.Wrap(err, "invalid peer config")
	}

	// If already nil, don't need to proceed further as config updates are not required
	if peerConfig.Peer.BCCSP.PKCS11.FileKeyStore == nil {
		return nil, nil
	}

	peerConfig.Peer.BCCSP.PKCS11.FileKeyStore = nil

	return peerConfig, nil
}

func (p *Peer) ReconcileFabricPeerMigrationV2_0(instance *current.IBPPeer) error {
	log.Info("Migration to V2 requested, checking if migration is needed")

	migrator := &v2.Migrate{
		DeploymentManager: p.DeploymentManager,
		ConfigMapManager:  &initializer.CoreConfigMap{Config: p.Config.PeerInitConfig, Scheme: p.Scheme, GetLabels: p.GetLabels, Client: p.Client},
		Client:            p.Client,
	}

	if err := fabric.V2Migrate(instance, migrator, instance.Spec.FabricVersion, p.Config.Operator.Peer.Timeouts.DBMigration); err != nil {
		return err
	}

	return nil
}

func (p *Peer) ReconcileFabricPeerMigrationV2_4(instance *current.IBPPeer) error {
	log.Info("Migration to V2.4.x requested, checking if migration is needed")

	migrator := &v2.Migrate{
		DeploymentManager: p.DeploymentManager,
		ConfigMapManager:  &initializer.CoreConfigMap{Config: p.Config.PeerInitConfig, Scheme: p.Scheme, GetLabels: p.GetLabels, Client: p.Client},
		Client:            p.Client,
	}

	if err := fabric.V24Migrate(instance, migrator, instance.Spec.FabricVersion, p.Config.Operator.Peer.Timeouts.DBMigration); err != nil {
		return err
	}

	return nil
}

func (p *Peer) ReconcileFabricPeerMigrationV2_5(instance *current.IBPPeer) error {
	log.Info("Migration to V2.5.x requested, checking if migration is needed")

	migrator := &v25.Migrate{
		DeploymentManager: p.DeploymentManager,
		ConfigMapManager:  &initializer.CoreConfigMap{Config: p.Config.PeerInitConfig, Scheme: p.Scheme, GetLabels: p.GetLabels, Client: p.Client},
		Client:            p.Client,
	}

	if err := fabric.V25Migrate(instance, migrator, instance.Spec.FabricVersion, p.Config.Operator.Peer.Timeouts.DBMigration); err != nil {
		return err
	}

	return nil
}

func (p *Peer) HandleMigrationJobs(listOpt k8sclient.ListOption, instance *current.IBPPeer) (bool, error) {
	status, job, err := p.CheckForRunningJobs(listOpt)
	if err != nil {
		return false, err
	}

	switch status {
	case RUNNING:
		return true, nil
	case COMPLETED:
		jobName := job.GetName()
		log.Info(fmt.Sprintf("Migration job '%s' completed, cleaning up...", jobName))

		migrationJob := &batchv1.Job{
			ObjectMeta: v1.ObjectMeta{
				Name:      jobName,
				Namespace: instance.GetNamespace(),
			},
		}

		if err := p.Client.Delete(context.TODO(), migrationJob); err != nil {
			return false, errors.Wrap(err, "failed to delete migration job after completion")
		}

		// TODO: Need to investigate why job is not adding controller reference to job pod,
		// this manual cleanup should not be required
		podList := &corev1.PodList{}
		if err := p.Client.List(context.TODO(), podList, k8sclient.MatchingLabels{"job-name": jobName}); err != nil {
			return false, errors.Wrap(err, "failed to list db migraton pods")
		}

		if len(podList.Items) == 1 {
			if err := p.Client.Delete(context.TODO(), &podList.Items[0]); err != nil {
				return false, errors.Wrap(err, "failed to delete db migration pod")
			}
		}

		if instance.UsingCouchDB() {
			couchDBPod := &corev1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      fmt.Sprintf("%s-couchdb", instance.GetName()),
					Namespace: instance.GetNamespace(),
				},
			}

			if err := p.Client.Delete(context.TODO(), couchDBPod); err != nil {
				return false, errors.Wrap(err, "failed to delete couchdb pod")
			}
		}

		return false, nil
	default:
		return false, nil
	}
}

type JobStatus string

const (
	COMPLETED JobStatus = "completed"
	RUNNING   JobStatus = "running"
	NOTFOUND  JobStatus = "not-found"
	UNKNOWN   JobStatus = "unknown"
)

func (p *Peer) CheckForRunningJobs(listOpt k8sclient.ListOption) (JobStatus, *jobv1.Job, error) {
	jobList := &batchv1.JobList{}
	if err := p.Client.List(context.TODO(), jobList, listOpt); err != nil {
		return NOTFOUND, nil, nil
	}

	if len(jobList.Items) == 0 {
		return NOTFOUND, nil, nil
	}

	// There should only be one job that is triggered per migration request
	k8sJob := jobList.Items[0]
	job := jobv1.NewWithDefaultsUseExistingName(&k8sJob)

	if len(job.Job.Status.Conditions) > 0 {
		cond := job.Job.Status.Conditions[0]
		if cond.Type == batchv1.JobFailed {
			log.Info(fmt.Sprintf("Job '%s' failed for reason: %s: %s", job.Name, cond.Reason, cond.Message))
		}
	}

	completed, err := job.ContainerFinished(p.Client, "dbmigration")
	if err != nil {
		return UNKNOWN, nil, err
	}

	if completed {
		return COMPLETED, job, nil

	}

	return RUNNING, nil, nil
}

func (p *Peer) UpgradeDBs(instance *current.IBPPeer) error {
	log.Info("Upgrade DBs action requested")
	if err := action.UpgradeDBs(p.DeploymentManager, p.Client, instance, p.Config.Operator.Peer.Timeouts.DBMigration); err != nil {
		return errors.Wrap(err, "failed to reset peer")
	}
	orig := instance.DeepCopy()

	instance.Spec.Action.UpgradeDBs = false
	if err := p.Client.Patch(context.TODO(), instance, k8sclient.MergeFrom(orig)); err != nil {
		return errors.Wrap(err, "failed to reset reenroll action flag")
	}

	return nil
}

func (p *Peer) EnrollForEcert(instance *current.IBPPeer) error {
	log.Info(fmt.Sprintf("Ecert enroll triggered via action parameter for '%s'", instance.GetName()))

	secret := instance.Spec.Secret
	if secret == nil || secret.Enrollment == nil || secret.Enrollment.Component == nil {
		return errors.New("unable to enroll, no ecert enrollment information provided")
	}
	ecertSpec := secret.Enrollment.Component

	storagePath := filepath.Join(p.GetInitStoragePath(instance), "ecert")
	crypto, err := action.Enroll(instance, ecertSpec, storagePath, p.Client, p.Scheme, true, p.Config.Operator.Peer.Timeouts.EnrollJob)
	if err != nil {
		return errors.Wrap(err, "failed to enroll for ecert")
	}

	err = p.Initializer.GenerateSecrets("ecert", instance, crypto)
	if err != nil {
		return errors.Wrap(err, "failed to generate ecert secrets")
	}

	return nil
}

func (p *Peer) EnrollForTLSCert(instance *current.IBPPeer) error {
	log.Info(fmt.Sprintf("TLS cert enroll triggered via action parameter for '%s'", instance.GetName()))

	secret := instance.Spec.Secret
	if secret == nil || secret.Enrollment == nil || secret.Enrollment.TLS == nil {
		return errors.New("unable to enroll, no TLS enrollment information provided")
	}
	tlscertSpec := secret.Enrollment.TLS

	storagePath := filepath.Join(p.GetInitStoragePath(instance), "tls")
	crypto, err := action.Enroll(instance, tlscertSpec, storagePath, p.Client, p.Scheme, false, p.Config.Operator.Peer.Timeouts.EnrollJob)
	if err != nil {
		return errors.Wrap(err, "failed to enroll for TLS cert")
	}

	err = p.Initializer.GenerateSecrets("tls", instance, crypto)
	if err != nil {
		return errors.Wrap(err, "failed to generate ecert secrets")
	}

	return nil
}

func (p *Peer) ReconcileHSMImages(instance *current.IBPPeer) bool {
	hsmConfig, err := commonconfig.ReadHSMConfig(p.Client, instance)
	if err != nil {
		return false
	}

	if hsmConfig.Library.AutoUpdateDisabled {
		return false
	}

	updated := false
	if hsmConfig.Library.Image != "" {
		hsmImage := hsmConfig.Library.Image
		lastIndex := strings.LastIndex(hsmImage, ":")
		image := hsmImage[:lastIndex]
		tag := hsmImage[lastIndex+1:]

		if instance.Spec.Images.HSMImage != image {
			instance.Spec.Images.HSMImage = image
			updated = true
		}

		if instance.Spec.Images.HSMTag != tag {
			instance.Spec.Images.HSMTag = tag
			updated = true
		}
	}

	return updated
}

func (p *Peer) HandleActions(instance *current.IBPPeer, update Update) error {
	orig := instance.DeepCopy()

	if update.EcertReenrollNeeded() {
		if err := p.ReenrollEcert(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetEcertReenroll()
			return err
		}
		instance.ResetEcertReenroll()
	}

	if update.TLSReenrollNeeded() {
		if err := p.ReenrollTLSCert(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetTLSReenroll()
			return err
		}
		instance.ResetTLSReenroll()
	}

	if update.EcertNewKeyReenroll() {
		if err := p.ReenrollEcertNewKey(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetEcertReenroll()
			return err
		}
		instance.ResetEcertReenroll()
	}

	if update.TLScertNewKeyReenroll() {
		if err := p.ReenrollTLSCertNewKey(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetTLSReenroll()
			return err
		}
		instance.ResetTLSReenroll()
	}

	if update.EcertEnroll() {
		if err := p.EnrollForEcert(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetEcertEnroll()
			return err
		}
		instance.ResetEcertEnroll()
	}

	if update.TLSCertEnroll() {
		if err := p.EnrollForTLSCert(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetTLSEnroll()
			return err
		}
		instance.ResetTLSEnroll()
	}

	// Upgrade DBs needs to be one of the last thing that should be performed to allow for other
	// update flags to be processed
	if update.UpgradeDBs() {
		if err := p.UpgradeDBs(instance); err != nil {
			// not adding reset as this action should not be run twice
			// log.Error(err, "Resetting action flag on failure")
			return err
		}
		// Can return without continuing down to restart logic cause resetting a peer will
		// initiate a restart anyways
		instance.ResetUpgradeDBs()

	} else if update.RestartNeeded() {
		if err := p.RestartAction(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetRestart()
			return err
		}
		instance.ResetRestart()
	}

	if err := p.Client.Patch(context.TODO(), instance, k8sclient.MergeFrom(orig)); err != nil {
		return errors.Wrap(err, "failed to reset action flags")
	}

	return nil
}

func (p *Peer) ReenrollEcert(instance *current.IBPPeer) error {
	log.Info("Ecert reenroll triggered via action parameter")
	if err := p.reenrollCert(instance, commoninit.ECERT, false); err != nil {
		return errors.Wrap(err, "ecert reenroll reusing existing private key action failed")
	}
	return nil
}

func (p *Peer) ReenrollEcertNewKey(instance *current.IBPPeer) error {
	log.Info("Ecert with new key reenroll triggered via action parameter")
	if err := p.reenrollCert(instance, commoninit.ECERT, true); err != nil {
		return errors.Wrap(err, "ecert reenroll with new key action failed")
	}
	return nil
}

func (p *Peer) ReenrollTLSCert(instance *current.IBPPeer) error {
	log.Info("TLS reenroll triggered via action parameter")
	if err := p.reenrollCert(instance, commoninit.TLS, false); err != nil {
		return errors.Wrap(err, "tls reenroll reusing existing private key action failed")
	}
	return nil
}

func (p *Peer) ReenrollTLSCertNewKey(instance *current.IBPPeer) error {
	log.Info("TLS with new key reenroll triggered via action parameter")
	if err := p.reenrollCert(instance, commoninit.TLS, true); err != nil {
		return errors.Wrap(err, "tls reenroll with new key action failed")
	}
	return nil
}

func (p *Peer) reenrollCert(instance *current.IBPPeer, certType commoninit.SecretType, newKey bool) error {
	return action.Reenroll(p, p.Client, certType, instance, newKey)
}

func (p *Peer) RestartAction(instance *current.IBPPeer) error {
	log.Info("Restart triggered via action parameter")
	if err := p.Restart.ForRestartAction(instance); err != nil {
		return errors.Wrap(err, "failed to restart peer pods")
	}
	return nil
}

func (p *Peer) HandleRestart(instance *current.IBPPeer, update Update) error {
	// If restart is disabled for components, can return immediately
	if p.Config.Operator.Restart.Disable.Components {
		return nil
	}

	err := p.Restart.TriggerIfNeeded(instance)
	if err != nil {
		return errors.Wrap(err, "failed to restart deployment")
	}

	return nil
}

func (p *Peer) UpdateMSPCertificates(instance *current.IBPPeer) error {
	log.Info("Updating certificates passed in MSP spec")

	updatedPeer, err := p.Initializer.GetUpdatedPeer(instance)
	if err != nil {
		return err
	}

	crypto, err := updatedPeer.GenerateCrypto()
	if err != nil {
		return err
	}

	if crypto != nil {
		err = p.Initializer.UpdateSecretsFromResponse(instance, crypto)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Peer) RenewCert(certType commoninit.SecretType, obj runtime.Object, newKey bool) error {
	instance := obj.(*current.IBPPeer)
	if instance.Spec.Secret == nil {
		return errors.New(fmt.Sprintf("missing secret spec for instance '%s'", instance.GetName()))
	}

	if instance.Spec.Secret.Enrollment != nil {
		log.Info(fmt.Sprintf("Renewing %s certificate for instance '%s'", string(certType), instance.Name))

		hsmEnabled := instance.IsHSMEnabled()
		storagePath := p.GetInitStoragePath(instance)
		spec := instance.Spec.Secret.Enrollment
		bccsp, err := p.GetBCCSPSectionForInstance(instance)
		if err != nil {
			return err
		}

		err = p.CertificateManager.RenewCert(certType, instance, spec, bccsp, storagePath, hsmEnabled, newKey)
		if err != nil {
			return err
		}
	} else {
		return errors.New("cannot auto-renew certificate created by MSP, force renewal required")
	}

	return nil
}

func (p *Peer) CustomLogic(instance *current.IBPPeer, update Update) (*current.CRStatus, *common.Result, error) {
	var status *current.CRStatus
	var err error

	if !p.CanSetCertificateTimer(instance, update) {
		log.Info("Certificate update detected but peer not yet deployed, requeuing request...")
		return status, &common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
		}, nil
	}

	// Check if crypto needs to be backed up before an update overrides exisitng secrets
	if update.CryptoBackupNeeded() {
		log.Info("Performing backup of TLS and ecert crypto")
		err = common.BackupCrypto(p.Client, p.Scheme, instance, p.GetLabels(instance))
		if err != nil {
			return status, nil, errors.Wrap(err, "failed to backup TLS and ecert crypto")
		}
	}

	status, err = p.CheckCertificates(instance)
	if err != nil {
		return status, nil, errors.Wrap(err, "failed to check for expiring certificates")
	}

	if update.CertificateCreated() {
		log.Info(fmt.Sprintf("%s certificate was created", update.GetCreatedCertType()))
		err = p.SetCertificateTimer(instance, update.GetCreatedCertType())
		if err != nil {
			return status, nil, errors.Wrap(err, "failed to set timer for certificate renewal")
		}
	}

	if update.EcertUpdated() {
		log.Info("Ecert was updated")
		err = p.SetCertificateTimer(instance, commoninit.ECERT)
		if err != nil {
			return status, nil, errors.Wrap(err, "failed to set timer for certificate renewal")
		}
	}

	if update.TLSCertUpdated() {
		log.Info("TLS cert was updated")
		err = p.SetCertificateTimer(instance, commoninit.TLS)
		if err != nil {
			return status, nil, errors.Wrap(err, "failed to set timer for certificate renewal")
		}
	}

	return status, nil, err

}

func (p *Peer) CheckCertificates(instance *current.IBPPeer) (*current.CRStatus, error) {
	numSecondsBeforeExpire := instance.Spec.GetNumSecondsWarningPeriod()
	statusType, message, err := p.CertificateManager.CheckCertificatesForExpire(instance, numSecondsBeforeExpire)
	if err != nil {
		return nil, err
	}

	crStatus := &current.CRStatus{
		Type:    statusType,
		Message: message,
	}

	switch statusType {
	case current.Deployed:
		crStatus.Reason = "allPodsDeployed"
	default:
		crStatus.Reason = "certRenewalRequired"
	}

	return crStatus, nil
}

func (p *Peer) SetCertificateTimer(instance *current.IBPPeer, certType commoninit.SecretType) error {
	certName := fmt.Sprintf("%s-%s-signcert", certType, instance.Name)
	numSecondsBeforeExpire := instance.Spec.GetNumSecondsWarningPeriod()
	duration, err := p.CertificateManager.GetDurationToNextRenewal(certType, instance, numSecondsBeforeExpire)
	if err != nil {
		return err
	}

	log.Info((fmt.Sprintf("Setting timer to renew %s %d days before it expires", certName, int(numSecondsBeforeExpire/DaysToSecondsConversion))))

	if p.RenewCertTimers[certName] != nil {
		p.RenewCertTimers[certName].Stop()
		p.RenewCertTimers[certName] = nil
	}
	p.RenewCertTimers[certName] = time.AfterFunc(duration, func() {
		// Check certs for updated status & set status so that reconcile is triggered after cert renewal. Reconcile loop will handle
		// checking certs again to determine whether instance status can return to Deployed
		err := p.UpdateCRStatus(instance)
		if err != nil {
			log.Error(err, "failed to update CR status")
		}

		// get instance
		instanceLatest := &current.IBPPeer{}
		err = p.Client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, instanceLatest)
		if err != nil {
			log.Error(err, "failed to get latest instance")
			return
		}

		err = common.BackupCrypto(p.Client, p.Scheme, instance, p.GetLabels(instance))
		if err != nil {
			log.Error(err, "failed to backup crypto before renewing cert")
			return
		}

		err = p.RenewCert(certType, instanceLatest, false)
		if err != nil {
			log.Info(fmt.Sprintf("Failed to renew %s certificate: %s, status of %s remaining in Warning phase", certType, err, instanceLatest.GetName()))
			return
		}
		log.Info(fmt.Sprintf("%s renewal complete", certName))
	})

	return nil
}

// NOTE: This is called by the timer's subroutine when it goes off, not during a reconcile loop.
// Therefore, it won't be overriden by the "SetStatus" method in ibppeer_controller.go
func (p *Peer) UpdateCRStatus(instance *current.IBPPeer) error {
	status, err := p.CheckCertificates(instance)
	if err != nil {
		return errors.Wrap(err, "failed to check certificates")
	}

	// Get most up-to-date instance at the time of update
	updatedInstance := &current.IBPPeer{}
	err = p.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, updatedInstance)
	if err != nil {
		return errors.Wrap(err, "failed to get new instance")
	}

	// Don't trigger reconcile if status remaining the same
	if updatedInstance.Status.Type == status.Type && updatedInstance.Status.Reason == status.Reason && updatedInstance.Status.Message == status.Message {
		return nil
	}

	updatedInstance.Status.Type = status.Type
	updatedInstance.Status.Reason = status.Reason
	updatedInstance.Status.Message = status.Message
	updatedInstance.Status.Status = current.True
	updatedInstance.Status.LastHeartbeatTime = time.Now().String()

	log.Info(fmt.Sprintf("Updating status of IBPPeer custom resource %s to %s phase", instance.Name, status.Type))
	err = p.Client.UpdateStatus(context.TODO(), updatedInstance)
	if err != nil {
		return errors.Wrapf(err, "failed to update status to %s phase", status.Type)
	}

	return nil
}

// This function checks whether the instance is in Deployed or Warning state when a cert
// update is detected. Only if Deployed or in Warning will a timer be set; otherwise,
// the update will be requeued until the Peer has completed deploying.
func (p *Peer) CanSetCertificateTimer(instance *current.IBPPeer, update Update) bool {
	if update.CertificateCreated() || update.CertificateUpdated() {
		if !(instance.Status.Type == current.Deployed || instance.Status.Type == current.Warning) {
			return false
		}
	}
	return true
}
