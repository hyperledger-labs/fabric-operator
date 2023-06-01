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

package baseorderer

import (
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
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer"
	ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v1"
	v2ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v2"
	v24ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v24"
	v25ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v25"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/validator"
	controllerclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common/reconcilechecks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

const (
	NODE                    = "node"
	DaysToSecondsConversion = int64(24 * 60 * 60)
)

type Override interface {
	Deployment(v1.Object, *appsv1.Deployment, resources.Action) error
	Service(v1.Object, *corev1.Service, resources.Action) error
	PVC(v1.Object, *corev1.PersistentVolumeClaim, resources.Action) error
	EnvCM(v1.Object, *corev1.ConfigMap, resources.Action, map[string]interface{}) error
	OrdererNode(v1.Object, *current.IBPOrderer, resources.Action) error
}

//go:generate counterfeiter -o mocks/deployment_manager.go -fake-name DeploymentManager . DeploymentManager

type DeploymentManager interface {
	resources.Manager
	CheckForSecretChange(v1.Object, string, func(string, *appsv1.Deployment) bool) error
}

//go:generate counterfeiter -o mocks/initializeibporderer.go -fake-name InitializeIBPOrderer . InitializeIBPOrderer

type InitializeIBPOrderer interface {
	GenerateSecrets(commoninit.SecretType, *current.IBPOrderer, *commonconfig.Response) error
	Create(initializer.OrdererConfig, initializer.IBPOrderer, string) (*initializer.Response, error)
	Update(initializer.OrdererConfig, initializer.IBPOrderer) (*initializer.Response, error)
	CreateOrUpdateConfigMap(*current.IBPOrderer, initializer.OrdererConfig) error
	GetConfigFromConfigMap(instance *current.IBPOrderer) (*corev1.ConfigMap, error)
	MissingCrypto(*current.IBPOrderer) bool
	Delete(*current.IBPOrderer) error
	CheckIfAdminCertsUpdated(*current.IBPOrderer) (bool, error)
	UpdateAdminSecret(*current.IBPOrderer) error
	GetInitOrderer(instance *current.IBPOrderer, storagePath string) (*initializer.Orderer, error)
	GetUpdatedOrderer(instance *current.IBPOrderer) (*initializer.Orderer, error)
	UpdateSecrets(prefix commoninit.SecretType, instance *current.IBPOrderer, crypto *commonconfig.Response) error
	GenerateSecretsFromResponse(instance *current.IBPOrderer, cryptoResponse *commonconfig.CryptoResponse) error
	UpdateSecretsFromResponse(instance *current.IBPOrderer, cryptoResponse *commonconfig.CryptoResponse) error
	GetCrypto(instance *current.IBPOrderer) (*commonconfig.CryptoResponse, error)
	GetCoreConfigFromFile(instance *current.IBPOrderer, file string) (initializer.OrdererConfig, error)
	GetCoreConfigFromBytes(instance *current.IBPOrderer, bytes []byte) (initializer.OrdererConfig, error)
}

//go:generate counterfeiter -o mocks/update.go -fake-name Update . Update

type Update interface {
	SpecUpdated() bool
	ConfigOverridesUpdated() bool
	TLSCertUpdated() bool
	EcertUpdated() bool
	OrdererTagUpdated() bool
	CertificateUpdated() bool
	RestartNeeded() bool
	EcertReenrollNeeded() bool
	TLScertReenrollNeeded() bool
	EcertNewKeyReenroll() bool
	TLScertNewKeyReenroll() bool
	DeploymentUpdated() bool
	MSPUpdated() bool
	EcertEnroll() bool
	TLScertEnroll() bool
	CertificateCreated() bool
	GetCreatedCertType() commoninit.SecretType
	CryptoBackupNeeded() bool
	MigrateToV2() bool
	MigrateToV24() bool
	MigrateToV25() bool
	NodeOUUpdated() bool
	ImagesUpdated() bool
	FabricVersionUpdated() bool
}

type IBPOrderer interface {
	Initialize(instance *current.IBPOrderer, update Update) error
	PreReconcileChecks(instance *current.IBPOrderer, update Update) (bool, error)
	ReconcileManagers(instance *current.IBPOrderer, update Update, genesisBlock []byte) error
	Reconcile(instance *current.IBPOrderer, update Update) (common.Result, error)
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
	TriggerIfNeeded(instance restart.Instance) error
	ForRestartAction(instance v1.Object) error
}

type OrdererConfig interface {
	MergeWith(interface{}, bool) error
	ToBytes() ([]byte, error)
	UsingPKCS11() bool
	SetPKCS11Defaults(bool)
	GetBCCSPSection() *commonapi.BCCSP
	SetDefaultKeyStore()
	SetBCCSPLibrary(string)
}

type Manager struct {
	Client controllerclient.Client
	Scheme *runtime.Scheme
	Config *config.Config
}

func (m *Manager) GetNode(nodeNumber int, renewCertTimers map[string]*time.Timer, restartManager RestartManager) *Node {
	return NewNode(m.Client, m.Scheme, m.Config, fmt.Sprintf("%s%d", NODE, nodeNumber), renewCertTimers, restartManager)
}

var _ IBPOrderer = &Node{}

type Node struct {
	Client controllerclient.Client
	Scheme *runtime.Scheme
	Config *config.Config

	DeploymentManager     DeploymentManager
	ServiceManager        resources.Manager
	PVCManager            resources.Manager
	EnvConfigMapManager   resources.Manager
	RoleManager           resources.Manager
	RoleBindingManager    resources.Manager
	ServiceAccountManager resources.Manager

	Override    Override
	Initializer InitializeIBPOrderer
	Name        string

	CertificateManager CertificateManager
	RenewCertTimers    map[string]*time.Timer

	Restart RestartManager
}

func NewNode(client controllerclient.Client, scheme *runtime.Scheme, config *config.Config, name string, renewCertTimers map[string]*time.Timer, restartManager RestartManager) *Node {
	n := &Node{
		Client: client,
		Scheme: scheme,
		Config: config,
		Override: &override.Override{
			Name:   name,
			Client: client,
			Config: config,
		},
		Name:            name,
		RenewCertTimers: renewCertTimers,
		Restart:         restartManager,
	}
	n.CreateManagers()

	validator := &validator.Validator{
		Client: client,
	}

	n.Initializer = initializer.New(client, scheme, config.OrdererInitConfig, name, validator)
	n.CertificateManager = certificate.New(client, scheme)

	return n
}

func NewNodeWithOverrides(client controllerclient.Client, scheme *runtime.Scheme, config *config.Config, name string, o Override, renewCertTimers map[string]*time.Timer, restartManager RestartManager) *Node {
	n := &Node{
		Client:          client,
		Scheme:          scheme,
		Config:          config,
		Override:        o,
		Name:            name,
		RenewCertTimers: renewCertTimers,
		Restart:         restartManager,
	}
	n.CreateManagers()

	validator := &validator.Validator{
		Client: client,
	}

	n.Initializer = initializer.New(client, scheme, config.OrdererInitConfig, name, validator)
	n.CertificateManager = certificate.New(client, scheme)

	return n
}

func (n *Node) CreateManagers() {
	override := n.Override
	resourceManager := resourcemanager.New(n.Client, n.Scheme)
	n.DeploymentManager = resourceManager.CreateDeploymentManager("", override.Deployment, n.GetLabels, n.Config.OrdererInitConfig.DeploymentFile)
	n.ServiceManager = resourceManager.CreateServiceManager("", override.Service, n.GetLabels, n.Config.OrdererInitConfig.ServiceFile)
	n.PVCManager = resourceManager.CreatePVCManager("", override.PVC, n.GetLabels, n.Config.OrdererInitConfig.PVCFile)
	n.EnvConfigMapManager = resourceManager.CreateConfigMapManager("env", override.EnvCM, n.GetLabels, n.Config.OrdererInitConfig.CMFile, nil)
	n.RoleManager = resourceManager.CreateRoleManager("", nil, n.GetLabels, n.Config.OrdererInitConfig.RoleFile)
	n.RoleBindingManager = resourceManager.CreateRoleBindingManager("", nil, n.GetLabels, n.Config.OrdererInitConfig.RoleBindingFile)
	n.ServiceAccountManager = resourceManager.CreateServiceAccountManager("", nil, n.GetLabels, n.Config.OrdererInitConfig.ServiceAccountFile)
}

func (n *Node) Reconcile(instance *current.IBPOrderer, update Update) (common.Result, error) {
	log.Info(fmt.Sprintf("Reconciling node instance '%s' ... update: %+v", instance.Name, update))
	var err error
	var status *current.CRStatus

	versionSet, err := n.SetVersion(instance)
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

	instanceUpdated, err := n.PreReconcileChecks(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed pre reconcile checks")
	}
	externalEndpointUpdated := n.UpdateExternalEndpoint(instance)

	if instanceUpdated || externalEndpointUpdated {
		log.Info(fmt.Sprintf("Updating instance after pre reconcile checks: %t, updating external endpoint: %t",
			instanceUpdated, externalEndpointUpdated))

		err = n.Client.Patch(context.TODO(), instance, nil, controllerclient.PatchOption{
			Resilient: &controllerclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPOrderer{},
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
			Status: &current.CRStatus{
				Type:    current.Initializing,
				Reason:  "Setting default values for either zone, region, and/or external endpoint",
				Message: "Operator has updated spec with defaults as part of initialization",
			},
		}, nil
	}

	err = n.Initialize(instance, update)
	if err != nil {
		return common.Result{}, operatorerrors.Wrap(err, operatorerrors.OrdererInitilizationFailed, "failed to initialize orderer node")
	}

	err = n.ReconcileManagers(instance, update, nil)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to reconcile managers")
	}

	err = n.UpdateConnectionProfile(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to create connection profile")
	}

	err = n.CheckStates(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check and restore state")
	}

	// custom product logic can be implemented here
	// No-Op atm
	status, result, err := n.CustomLogic(instance, update)

	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to run custom offering logic")
	}

	if result != nil {
		return *result, nil
	}

	if update.MSPUpdated() {
		err = n.UpdateMSPCertificates(instance)
		if err != nil {
			if err != nil {
				return common.Result{}, errors.Wrap(err, "failed to update certificates passed in MSP spec")
			}
		}
	}

	if update.EcertUpdated() {
		log.Info("Ecert was updated")
		// Request deployment restart for tls cert update
		err = n.Restart.ForCertUpdate(commoninit.ECERT, instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	if update.TLSCertUpdated() {
		log.Info("TLS cert was updated")
		// Request deployment restart for ecert update
		err = n.Restart.ForCertUpdate(commoninit.TLS, instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	if err := n.HandleActions(instance, update); err != nil {
		return common.Result{}, errors.Wrap(err, "failed to handle actions")
	}

	if err := n.HandleRestart(instance, update); err != nil {
		return common.Result{}, err
	}

	return common.Result{
		Status: status,
	}, nil
}

// PreReconcileChecks validate CR request before starting reconcile flow
func (n *Node) PreReconcileChecks(instance *current.IBPOrderer, update Update) (bool, error) {
	var err error

	imagesUpdated, err := reconcilechecks.FabricVersionHelper(instance, n.Config.Operator.Versions, update)
	if err != nil {
		return false, errors.Wrap(err, "failed during version and image checks")
	}

	if instance.Spec.HSMSet() {
		err = util.ValidateHSMProxyURL(instance.Spec.HSM.PKCS11Endpoint)
		if err != nil {
			return false, errors.Wrapf(err, "invalid HSM endpoint for orderer instance '%s'", instance.GetName())
		}
	}

	if !instance.Spec.DomainSet() {
		return false, fmt.Errorf("domain not set for orderer instance '%s'", instance.GetName())
	}

	if instance.Spec.Action.Enroll.Ecert && instance.Spec.Action.Reenroll.Ecert {
		return false, errors.New("both enroll and renenroll action requested for ecert, must only select one")
	}

	if instance.Spec.Action.Enroll.TLSCert && instance.Spec.Action.Reenroll.TLSCert {
		return false, errors.New("both enroll and renenroll action requested for TLS cert, must only select one")
	}

	if instance.Spec.Action.Enroll.Ecert && instance.Spec.Action.Reenroll.EcertNewKey {
		return false, errors.New("both enroll and renenroll with new key action requested for ecert, must only select one")
	}

	if instance.Spec.Action.Enroll.TLSCert && instance.Spec.Action.Reenroll.TLSCertNewKey {
		return false, errors.New("both enroll and renenroll with new key action requested for TLS cert, must only select one")
	}

	if instance.Spec.Action.Reenroll.Ecert && instance.Spec.Action.Reenroll.EcertNewKey {
		return false, errors.New("both reenroll and renenroll with new key action requested for ecert, must only select one")
	}

	if instance.Spec.Action.Reenroll.TLSCert && instance.Spec.Action.Reenroll.TLSCertNewKey {
		return false, errors.New("both reenroll and renenroll with new key action requested for TLS cert, must only select one")
	}

	zoneUpdated, err := n.SelectZone(instance)
	if err != nil {
		return false, err
	}

	regionUpdated, err := n.SelectRegion(instance)
	if err != nil {
		return false, err
	}

	hsmImageUpdated := n.ReconcileHSMImages(instance)

	var replicasUpdated bool
	if instance.Spec.Replicas == nil {
		replicas := int32(1)
		instance.Spec.Replicas = &replicas
		replicasUpdated = true
	}

	updated := zoneUpdated || regionUpdated || hsmImageUpdated || replicasUpdated || imagesUpdated

	if updated {
		log.Info(fmt.Sprintf("zoneUpdated %t, regionUpdated %t, hsmImageUpdated %t, replicasUpdated %t, imagesUpdated %t",
			zoneUpdated, regionUpdated, hsmImageUpdated, replicasUpdated, imagesUpdated))
	}

	return updated, nil
}

func (n *Node) Initialize(instance *current.IBPOrderer, update Update) error {
	var err error

	log.Info(fmt.Sprintf("Checking if initialization needed for node: %s", instance.GetName()))

	// TODO: Add checks to determine if initialization is neeeded. Split this method into
	// two, one should handle initialization during the create event of a CR and the other
	// should update events

	// Service account is required by HSM init job
	err = n.ReconcileRBAC(instance)
	if err != nil {
		return errors.Wrap(err, "failed RBAC reconciliation")
	}

	if instance.IsHSMEnabled() {
		// If HSM config not found, HSM proxy is being used
		if instance.UsingHSMProxy() {
			err = os.Setenv("PKCS11_PROXY_SOCKET", instance.Spec.HSM.PKCS11Endpoint)
			if err != nil {
				return err
			}
		} else {

			hsmConfig, err := commonconfig.ReadHSMConfig(n.Client, instance)
			if err != nil {
				return errors.New("using non-proxy HSM, but no HSM config defined as config map 'ibp-hsm-config'")
			}

			if hsmConfig.Daemon != nil {
				log.Info("Using daemon based HSM, creating pvc...")
				n.PVCManager.SetCustomName(instance.Spec.CustomNames.PVC.Orderer)
				err = n.PVCManager.Reconcile(instance, update.SpecUpdated())
				if err != nil {
					return errors.Wrap(err, "failed PVC reconciliation")
				}
			}
		}
	}

	initOrderer, err := n.Initializer.GetInitOrderer(instance, n.GetInitStoragePath(instance))
	if err != nil {
		return err
	}
	initOrderer.UsingHSMProxy = instance.UsingHSMProxy()

	ordererConfig := n.Config.OrdererInitConfig.OrdererFile
	if version.GetMajorReleaseVersion(instance.Spec.FabricVersion) == version.V2 {
		currentVer := version.String(instance.Spec.FabricVersion)
		if currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_5_1) {
			ordererConfig = n.Config.OrdererInitConfig.OrdererV25File
		} else if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.GreaterThan(version.V2_4_1) {
			ordererConfig = n.Config.OrdererInitConfig.OrdererV24File
		} else if currentVer.LessThan(version.V2_4_1) {
			ordererConfig = n.Config.OrdererInitConfig.OrdererV2File
		}
	}

	initOrderer.Config, err = n.Initializer.GetCoreConfigFromFile(instance, ordererConfig)
	if err != nil {
		return err
	}

	updated := update.ConfigOverridesUpdated() || update.NodeOUUpdated()
	if update.ConfigOverridesUpdated() {
		err = n.InitializeUpdateConfigOverride(instance, initOrderer)
		if err != nil {
			return err
		}
		// Request deployment restart for config override update
		if err := n.Restart.ForConfigOverride(instance); err != nil {
			return err
		}
	}
	if update.NodeOUUpdated() {
		err = n.InitializeUpdateNodeOU(instance)
		if err != nil {
			return err
		}
		// Request deloyment restart for node OU update
		if err = n.Restart.ForNodeOU(instance); err != nil {
			return err
		}
	}
	if !updated {
		err = n.InitializeCreate(instance, initOrderer)
		if err != nil {
			return err
		}
	}

	updateNeeded, err := n.Initializer.CheckIfAdminCertsUpdated(instance)
	if err != nil {
		return err
	}

	if updateNeeded {
		err = n.Initializer.UpdateAdminSecret(instance)
		if err != nil {
			return err
		}
		// Request deployment restart for admin cert updates
		if err = n.Restart.ForAdminCertUpdate(instance); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) InitializeCreate(instance *current.IBPOrderer, initOrderer *initializer.Orderer) error {
	// TODO: Should also check for secrets not just config map
	if n.ConfigExists(instance) {
		log.Info(fmt.Sprintf("Config '%s-config' exists, not reinitializing node", instance.GetName()))
		return nil
	}

	log.Info(fmt.Sprintf("Running initialization for create event on node '%s', since config '%s-config' does not exists", instance.GetName(), instance.GetName()))
	configOverride, err := instance.GetConfigOverride()
	if err != nil {
		return err
	}
	resp, err := n.Initializer.Create(configOverride.(OrdererConfig), initOrderer, n.GetInitStoragePath(instance))
	if err != nil {
		return err
	}

	if resp != nil {
		if resp.Crypto != nil {
			if !instance.Spec.NodeOUDisabled() {
				if err := resp.Crypto.VerifyCertOU("orderer"); err != nil {
					return err
				}
			}

			err = n.Initializer.GenerateSecretsFromResponse(instance, resp.Crypto)
			if err != nil {
				return err
			}
		}

		if resp.Config != nil {
			log.Info(fmt.Sprintf("Create config map for '%s'...", instance.GetName()))
			if instance.IsHSMEnabled() && !instance.UsingHSMProxy() {
				hsmConfig, err := commonconfig.ReadHSMConfig(n.Client, instance)
				if err != nil {
					return err
				}
				resp.Config.SetBCCSPLibrary(filepath.Join("/hsm/lib", filepath.Base(hsmConfig.Library.FilePath)))
			}

			err = n.Initializer.CreateOrUpdateConfigMap(instance, resp.Config)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *Node) ConfigExists(instance *current.IBPOrderer) bool {
	name := fmt.Sprintf("%s-config", instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: instance.Namespace,
	}

	cm := &corev1.ConfigMap{}
	err := n.Client.Get(context.TODO(), namespacedName, cm)
	if err != nil {
		return false
	}

	return true
}

func (n *Node) InitializeUpdateConfigOverride(instance *current.IBPOrderer, initOrderer *initializer.Orderer) error {
	log.Info(fmt.Sprintf("Running initialization update config override for node: %s", instance.GetName()))

	if n.Initializer.MissingCrypto(instance) {
		log.Info("Missing crypto for node")
		// If crypto is missing, we should run the create logic
		err := n.InitializeCreate(instance, initOrderer)
		if err != nil {
			return err
		}

		return nil
	}

	cm, err := n.Initializer.GetConfigFromConfigMap(instance)
	if err != nil {
		return err
	}

	initOrderer.Config, err = n.Initializer.GetCoreConfigFromBytes(instance, cm.BinaryData["orderer.yaml"])
	if err != nil {
		return err
	}

	configOverride, err := instance.GetConfigOverride()
	if err != nil {
		return err
	}

	resp, err := n.Initializer.Update(configOverride.(OrdererConfig), initOrderer)
	if err != nil {
		return err
	}

	if resp != nil && resp.Config != nil {
		log.Info(fmt.Sprintf("Update config map for '%s'...", instance.GetName()))
		err = n.Initializer.CreateOrUpdateConfigMap(instance, resp.Config)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) InitializeUpdateNodeOU(instance *current.IBPOrderer) error {
	log.Info(fmt.Sprintf("Running initialize update node OU enabled: %t for orderer '%s", !instance.Spec.NodeOUDisabled(), instance.GetName()))

	crypto, err := n.Initializer.GetCrypto(instance)
	if err != nil {
		return err
	}

	if !instance.Spec.NodeOUDisabled() {
		if err := crypto.VerifyCertOU("orderer"); err != nil {
			return err

		}
	} else {
		// If nodeOUDisabled, admin certs are required
		if crypto.Enrollment.AdminCerts == nil {
			return errors.New("node OU disabled, admin certs are required but missing")
		}
	}

	// Update config.yaml in config map
	err = n.Initializer.CreateOrUpdateConfigMap(instance, nil)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) ReconcileManagers(instance *current.IBPOrderer, updated Update, genesisBlock []byte) error {
	var err error

	update := updated.SpecUpdated()

	n.PVCManager.SetCustomName(instance.Spec.CustomNames.PVC.Orderer)
	err = n.PVCManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrapf(err, "failed PVC reconciliation")
	}

	err = n.ServiceManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Service reconciliation")
	}

	err = n.ReconcileRBAC(instance)
	if err != nil {
		return errors.Wrap(err, "failed RBAC reconciliation")
	}

	err = n.EnvConfigMapManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Env ConfigMap reconciliation")
	}

	if instance.Spec.IsUsingChannelLess() {
		log.Info("Node is in channel less mode - ending reconcile")
	} else if !instance.Spec.IsPrecreateOrderer() {
		log.Info("Node is not precreate - reconciling genesis secret")
		err = n.ReconcileGenesisSecret(instance)
		if err != nil {
			return errors.Wrap(err, "failed Genesis Secret reconciliation")
		}
	}

	err = n.DeploymentManager.Reconcile(instance, updated.DeploymentUpdated())
	if err != nil {
		return errors.Wrap(err, "failed Deployment reconciliation")
	}

	return nil
}

func (n *Node) CheckStates(instance *current.IBPOrderer) error {
	// Don't need to check state if the state is being updated via CR. State needs
	// to be checked if operator detects changes to a resources that was not triggered
	// via CR.
	if n.DeploymentManager.Exists(instance) {
		err := n.DeploymentManager.CheckState(instance)
		if err != nil {
			log.Error(err, "unexpected state")
			err = n.DeploymentManager.RestoreState(instance)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *Node) SetVersion(instance *current.IBPOrderer) (bool, error) {
	if instance.Status.Version == "" || !version.String(instance.Status.Version).Equal(version.Operator) {
		log.Info("Version of Operator: ", "version", version.Operator)
		log.Info(fmt.Sprintf("Version of CR '%s': %s", instance.GetName(), instance.Status.Version))
		log.Info(fmt.Sprintf("Setting '%s' to version '%s'", instance.Name, version.Operator))

		instance.Status.Version = version.Operator
		err := n.Client.PatchStatus(context.TODO(), instance, nil, controllerclient.PatchOption{
			Resilient: &controllerclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPOrderer{},
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

func (n *Node) GetLabels(instance v1.Object) map[string]string {
	parts := strings.Split(instance.GetName(), "node")
	label := os.Getenv("OPERATOR_LABEL_PREFIX")
	if label == "" {
		label = "fabric"
	}

	if len(parts) > 1 {
		ordererclustername := strings.Join(parts[:len(parts)-1], "node")
		orderingnode := "node" + parts[len(parts)-1]
		return map[string]string{
			"app":                          instance.GetName(),
			"creator":                      label,
			"orderingservice":              ordererclustername,
			"orderingnode":                 orderingnode,
			"parent":                       ordererclustername,
			"app.kubernetes.io/name":       label,
			"app.kubernetes.io/instance":   label + "orderer",
			"app.kubernetes.io/managed-by": label + "-operator",
		}
	}

	return map[string]string{
		"app":                          instance.GetName(),
		"creator":                      label,
		"orderingservice":              fmt.Sprintf("%s", instance.GetName()),
		"app.kubernetes.io/name":       label,
		"app.kubernetes.io/instance":   label + "orderer",
		"app.kubernetes.io/managed-by": label + "-operator",
	}
}

func (n *Node) Delete(instance *current.IBPOrderer) error {
	log.Info(fmt.Sprintf("Deleting node '%s'", n.Name))
	err := n.ServiceManager.Delete(instance)
	if err != nil {
		return errors.Wrapf(err, "failed to delete service '%s'", n.ServiceManager.GetName(instance))
	}

	err = n.PVCManager.Delete(instance)
	if err != nil {
		return errors.Wrapf(err, "failed to delete pvc '%s'", n.ServiceManager.GetName(instance))
	}

	err = n.EnvConfigMapManager.Delete(instance)
	if err != nil {
		return errors.Wrapf(err, "failed to delete config map '%s'", n.ServiceManager.GetName(instance))
	}

	err = n.Initializer.Delete(instance)
	if err != nil {
		return errors.Wrapf(err, "failed to delete secrets")
	}

	// Important: This must always be the last resource to be deleted
	err = n.DeploymentManager.Delete(instance)
	if err != nil {
		return errors.Wrapf(err, "failed to delete deployment '%s'", n.DeploymentManager.GetName(instance))
	}

	return nil
}

func (n *Node) ReconcileGenesisSecret(instance *current.IBPOrderer) error {
	namespacedName := types.NamespacedName{
		Name:      instance.Name + "-genesis",
		Namespace: instance.Namespace,
	}

	secret := &corev1.Secret{}
	err := n.Client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return n.CreateGenesisSecret(instance)
		}
		// Error reading the object - requeue the request.
		return err
	}
	return nil
}

func (n *Node) CreateGenesisSecret(instance *current.IBPOrderer) error {
	data := map[string][]byte{}

	genesisBlock, err := util.Base64ToBytes(instance.Spec.GenesisBlock)
	if err != nil {
		return errors.Wrap(err, "failed to decode genesis block")
	}

	data["orderer.block"] = genesisBlock
	s := &corev1.Secret{
		Data: data,
	}
	s.Name = instance.Name + "-genesis"
	s.Namespace = instance.Namespace
	s.Labels = n.GetLabels(instance)

	err = n.Client.CreateOrUpdate(context.TODO(), s, controllerclient.CreateOrUpdateOption{
		Owner:  instance,
		Scheme: n.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create genesis secret")
	}

	return nil
}

func (n *Node) ReconcileRBAC(instance *current.IBPOrderer) error {
	var err error

	err = n.RoleManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	err = n.RoleBindingManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	err = n.ServiceAccountManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) SelectZone(instance *current.IBPOrderer) (bool, error) {
	if instance.Spec.Zone == "select" {
		log.Info("Selecting zone...")
		zone := util.GetZone(n.Client)
		log.Info(fmt.Sprintf("Zone set to: '%s'", zone))
		instance.Spec.Zone = zone
		return true, nil
	}
	if instance.Spec.Zone != "" {
		err := util.ValidateZone(n.Client, instance.Spec.Zone)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (n *Node) SelectRegion(instance *current.IBPOrderer) (bool, error) {
	if instance.Spec.Region == "select" {
		log.Info("Selecting region...")
		region := util.GetRegion(n.Client)
		log.Info(fmt.Sprintf("Region set to: '%s'", region))
		instance.Spec.Region = region
		return true, nil
	}
	if instance.Spec.Region != "" {
		err := util.ValidateRegion(n.Client, instance.Spec.Region)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (n *Node) UpdateExternalEndpoint(instance *current.IBPOrderer) bool {
	if instance.Spec.ExternalAddress == "" {
		instance.Spec.ExternalAddress = instance.Namespace + "-" + instance.Name + "-orderer" + "." + instance.Spec.Domain + ":443"
		return true
	}
	return false
}

func (n *Node) UpdateConnectionProfile(instance *current.IBPOrderer) error {
	var err error

	endpoints := n.GetEndpoints(instance)

	tlscert, err := common.GetTLSSignCertEncoded(n.Client, instance)
	if err != nil {
		return err
	}

	tlscacerts, err := common.GetTLSCACertEncoded(n.Client, instance)
	if err != nil {
		return err
	}

	tlsintercerts, err := common.GetTLSIntercertEncoded(n.Client, instance)
	if err != nil {
		return err
	}

	ecert, err := common.GetEcertSignCertEncoded(n.Client, instance)
	if err != nil {
		return err
	}

	cacert, err := common.GetEcertCACertEncoded(n.Client, instance)
	if err != nil {
		return err
	}

	admincerts, err := common.GetEcertAdmincertEncoded(n.Client, instance)
	if err != nil {
		return err
	}

	if len(tlsintercerts) > 0 {
		tlscacerts = tlsintercerts
	}

	err = n.UpdateConnectionProfileConfigmap(instance, *endpoints, tlscert, tlscacerts, ecert, cacert, admincerts)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) UpdateConnectionProfileConfigmap(instance *current.IBPOrderer, endpoints current.OrdererEndpoints, tlscert string, tlscacerts []string, ecert string, cacert []string, admincerts []string) error {

	// TODO add ecert.intermediatecerts and ecert.admincerts
	// TODO add tls.cacerts
	// TODO get the whole PeerConnectionProfile object from caller??
	name := instance.Name + "-connection-profile"
	connectionProfile := &current.OrdererConnectionProfile{
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
		return errors.Wrap(err, "failed to marshal connection profile")
	}
	cm := &corev1.ConfigMap{
		BinaryData: map[string][]byte{"profile.json": bytes},
	}
	cm.Name = name
	cm.Namespace = instance.Namespace
	cm.Labels = n.GetLabels(instance)

	nn := types.NamespacedName{
		Name:      name,
		Namespace: instance.GetNamespace(),
	}

	err = n.Client.Get(context.TODO(), nn, &corev1.ConfigMap{})
	if err == nil {
		log.Info(fmt.Sprintf("Update connection profile configmap '%s' for %s", nn.Name, instance.Name))
		err = n.Client.Update(context.TODO(), cm, controllerclient.UpdateOption{Owner: instance, Scheme: n.Scheme})
		if err != nil {
			return errors.Wrap(err, "failed to update connection profile configmap")
		}
	} else {
		log.Info(fmt.Sprintf("Create connection profile configmap '%s' for %s", nn.Name, instance.Name))
		err = n.Client.Create(context.TODO(), cm, controllerclient.CreateOption{Owner: instance, Scheme: n.Scheme})
		if err != nil {
			return errors.Wrap(err, "failed to create connection profile configmap")
		}
	}

	return nil
}

func (n *Node) GetEndpoints(instance *current.IBPOrderer) *current.OrdererEndpoints {
	endpoints := &current.OrdererEndpoints{
		API:        "grpcs://" + instance.Namespace + "-" + instance.Name + "-orderer." + instance.Spec.Domain + ":443",
		Operations: "https://" + instance.Namespace + "-" + instance.Name + "-operations." + instance.Spec.Domain + ":443",
		Grpcweb:    "https://" + instance.Namespace + "-" + instance.Name + "-grpcweb." + instance.Spec.Domain + ":443",
	}
	currentVer := version.String(instance.Spec.FabricVersion)
	if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_4_1) {
		endpoints.Admin = "https://" + instance.Namespace + "-" + instance.Name + "-admin." + instance.Spec.Domain + ":443"
	}
	return endpoints
}

func (n *Node) UpdateParentStatus(instance *current.IBPOrderer) error {
	parentName := instance.Labels["parent"]

	nn := types.NamespacedName{
		Name:      parentName,
		Namespace: instance.GetNamespace(),
	}

	log.Info(fmt.Sprintf("Node '%s' is setting parent '%s' status", instance.GetName(), parentName))

	parentInstance := &current.IBPOrderer{}
	err := n.Client.Get(context.TODO(), nn, parentInstance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Parent '%s' for node '%s' not found, skipping setting parent status", parentName, instance.GetName()))
			return nil
		}
		return err
	}

	// If parent is deployed and child was not updated to warning state, no longer update the parent
	if parentInstance.Status.Type == current.Deployed && instance.Status.Type != current.Warning {
		log.Info(fmt.Sprintf("Parent '%s' is in 'Deployed' state, can't update status", parentName))
		return nil
	}

	labelSelector, err := labels.Parse(fmt.Sprintf("parent=%s", parentName))
	if err != nil {
		return errors.Wrap(err, "failed to parse selector for parent name")
	}

	listOptions := &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     instance.GetNamespace(),
	}

	ordererList := &current.IBPOrdererList{}
	err = n.Client.List(context.TODO(), ordererList, listOptions)
	if err != nil {
		return err
	}

	clustersize := parentInstance.Spec.ClusterSize

	var returnStatus current.IBPCRStatusType
	reason := "No reason"

	log.Info(fmt.Sprintf("Found %d nodes, original cluster size %d", len(ordererList.Items), clustersize))

	updateStatus := false
	errorstateNodes := []string{}
	deployingstateNodes := []string{}
	precreatedNodes := []string{}
	deployedNodes := []string{}
	warningNodes := []string{}

	for _, node := range ordererList.Items {
		if node.Status.Type == current.Error {
			log.Info(fmt.Sprintf("Node %s is in Error state", node.GetName()))
			errorstateNodes = append(errorstateNodes, node.GetName())
		} else if node.Status.Type == current.Deploying {
			log.Info(fmt.Sprintf("Node %s is in Deploying state", node.GetName()))
			deployingstateNodes = append(deployingstateNodes, node.GetName())
		} else if node.Status.Type == current.Precreated {
			log.Info(fmt.Sprintf("Node %s is in Precreating state", node.GetName()))
			precreatedNodes = append(precreatedNodes, node.GetName())
		} else if node.Status.Type == current.Warning {
			log.Info(fmt.Sprintf("Node %s is in Warning state", node.GetName()))
			warningNodes = append(warningNodes, node.GetName())
		} else if node.Status.Type == current.Deployed {
			log.Info(fmt.Sprintf("Node %s is in Deployed state", node.GetName()))
			deployedNodes = append(deployedNodes, node.GetName())
		}
	}

	if len(deployingstateNodes) != 0 {
		log.Info("Nodes are in deploying state currently, not updating parent status")
		updateStatus = false
	} else if len(errorstateNodes) != 0 {
		updateStatus = true
		reason = "The orderer nodes " + strings.Join(errorstateNodes[:], ",") + " are in Error state"
		returnStatus = current.Error
	} else if len(precreatedNodes) != 0 {
		updateStatus = true
		reason = "The orderer nodes " + strings.Join(precreatedNodes[:], ",") + " are in Precreated state"
		returnStatus = current.Precreated
	} else if len(warningNodes) != 0 {
		updateStatus = true
		reason = "The orderer nodes " + strings.Join(warningNodes[:], ",") + " are in Warning state"
		returnStatus = current.Warning
	} else if len(deployedNodes) != 0 {
		updateStatus = true
		returnStatus = current.Deployed
		reason = "All nodes are deployed"
	}

	if updateStatus {
		parentInstance.Status.Type = returnStatus
		parentInstance.Status.Status = current.True
		parentInstance.Status.Reason = reason
		parentInstance.Status.LastHeartbeatTime = time.Now().String()

		log.Info(fmt.Sprintf("Setting parent status to: %+v", parentInstance.Status))
		err = n.Client.UpdateStatus(context.TODO(), parentInstance)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) GetInitStoragePath(instance *current.IBPOrderer) string {
	if n.Config != nil && n.Config.OrdererInitConfig != nil && n.Config.OrdererInitConfig.StoragePath != "" {
		return filepath.Join(n.Config.OrdererInitConfig.StoragePath, instance.GetName())
	}

	return filepath.Join("/", "ordererinit", instance.GetName())
}

func (n *Node) GetBCCSPSectionForInstance(instance *current.IBPOrderer) (*commonapi.BCCSP, error) {
	var bccsp *commonapi.BCCSP
	if instance.IsHSMEnabled() {
		co, err := instance.GetConfigOverride()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get configoverride")
		}

		configOverride := co.(OrdererConfig)
		configOverride.SetPKCS11Defaults(instance.UsingHSMProxy())
		bccsp = configOverride.GetBCCSPSection()
	}

	return bccsp, nil
}

func (n *Node) ReconcileFabricOrdererMigration(instance *current.IBPOrderer) error {
	ordererConfig, err := n.FabricOrdererMigration(instance)
	if err != nil {
		return errors.Wrap(err, "failed to migrate orderer between fabric versions")
	}

	if ordererConfig != nil {
		log.Info("Orderer config updated during fabric orderer migration, updating config map...")
		if err := n.Initializer.CreateOrUpdateConfigMap(instance, ordererConfig); err != nil {
			return errors.Wrapf(err, "failed to create/update '%s' orderer's config map", instance.GetName())
		}
	}

	return nil
}

// Moving to fabric version above 1.4.6 require that the `msp/keystore` value be removed
// from BCCSP section if configured to use PKCS11 (HSM). NOTE: This does not support
// migration across major release, will not cover migration orderer from 1.4.x to 2.x
func (n *Node) FabricOrdererMigration(instance *current.IBPOrderer) (*ordererconfig.Orderer, error) {
	if !instance.IsHSMEnabled() {
		return nil, nil
	}

	ordererTag := instance.Spec.Images.OrdererTag
	if !strings.Contains(ordererTag, "sha") {
		tag := strings.Split(ordererTag, "-")[0]

		ordererVersion := version.String(tag)
		if !ordererVersion.GreaterThan(version.V1_4_6) {
			return nil, nil
		}

		log.Info(fmt.Sprintf("Orderer moving to fabric version %s", ordererVersion))
	} else {
		if instance.Spec.FabricVersion == version.V2 {
			return nil, nil
		}
		log.Info(fmt.Sprintf("Orderer moving to digest %s", ordererTag))
	}

	// Read orderer config map and remove keystore value from BCCSP section
	cm, err := n.Initializer.GetConfigFromConfigMap(instance)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get '%s' orderer's config map", instance.GetName())
	}

	ordererConfig := &ordererconfig.Orderer{}
	if err := yaml.Unmarshal(cm.BinaryData["orderer.yaml"], ordererConfig); err != nil {
		return nil, errors.Wrap(err, "invalid orderer config")
	}

	// If already nil, don't need to proceed further as config updates are not required
	if ordererConfig.General.BCCSP.PKCS11.FileKeyStore == nil {
		return nil, nil
	}

	ordererConfig.General.BCCSP.PKCS11.FileKeyStore = nil

	return ordererConfig, nil
}

func (n *Node) UpdateMSPCertificates(instance *current.IBPOrderer) error {
	log.Info("Updating certificates passed in MSP spec")
	updatedOrderer, err := n.Initializer.GetUpdatedOrderer(instance)
	if err != nil {
		return err
	}

	crypto, err := updatedOrderer.GenerateCrypto()
	if err != nil {
		return err
	}

	if crypto != nil {
		err = n.Initializer.UpdateSecrets("ecert", instance, crypto.Enrollment)
		if err != nil {
			return errors.Wrap(err, "failed to update ecert secrets")
		}

		err = n.Initializer.UpdateSecrets("tls", instance, crypto.TLS)
		if err != nil {
			return errors.Wrap(err, "failed to update tls secrets")
		}

		err = n.Initializer.UpdateSecrets("clientauth", instance, crypto.ClientAuth)
		if err != nil {
			return errors.Wrap(err, "failed to update client auth secrets")
		}
	}

	return nil
}

func (n *Node) RenewCert(certType commoninit.SecretType, obj runtime.Object, newKey bool) error {
	instance := obj.(*current.IBPOrderer)
	if instance.Spec.Secret == nil {
		return errors.New(fmt.Sprintf("missing secret spec for instance '%s'", instance.GetName()))
	}

	if instance.Spec.Secret.Enrollment != nil {
		log.Info(fmt.Sprintf("Renewing %s certificate for instance '%s'", string(certType), instance.Name))

		hsmEnabled := instance.IsHSMEnabled()
		spec := instance.Spec.Secret.Enrollment
		storagePath := n.GetInitStoragePath(instance)
		bccsp, err := n.GetBCCSPSectionForInstance(instance)
		if err != nil {
			return err
		}

		err = n.CertificateManager.RenewCert(certType, instance, spec, bccsp, storagePath, hsmEnabled, newKey)
		if err != nil {
			return err
		}
	} else {
		return errors.New("cannot auto-renew certificate created by MSP, force renewal required")
	}

	return nil
}

func (n *Node) EnrollForEcert(instance *current.IBPOrderer) error {
	log.Info(fmt.Sprintf("Ecert enroll triggered via action parameter for '%s'", instance.GetName()))

	secret := instance.Spec.Secret
	if secret == nil || secret.Enrollment == nil || secret.Enrollment.Component == nil {
		return errors.New("unable to enroll, no ecert enrollment information provided")
	}
	ecertSpec := secret.Enrollment.Component

	storagePath := filepath.Join(n.GetInitStoragePath(instance), "ecert")
	crypto, err := action.Enroll(instance, ecertSpec, storagePath, n.Client, n.Scheme, true, n.Config.Operator.Orderer.Timeouts.EnrollJob)
	if err != nil {
		return errors.Wrap(err, "failed to enroll for ecert")
	}

	err = n.Initializer.GenerateSecrets("ecert", instance, crypto)
	if err != nil {
		return errors.Wrap(err, "failed to generate ecert secrets")
	}

	return nil
}

func (n *Node) EnrollForTLSCert(instance *current.IBPOrderer) error {
	log.Info(fmt.Sprintf("TLS cert enroll triggered via action parameter for '%s'", instance.GetName()))

	secret := instance.Spec.Secret
	if secret == nil || secret.Enrollment == nil || secret.Enrollment.TLS == nil {
		return errors.New("unable to enroll, no TLS enrollment information provided")
	}
	tlscertSpec := secret.Enrollment.TLS

	storagePath := filepath.Join(n.GetInitStoragePath(instance), "tls")
	crypto, err := action.Enroll(instance, tlscertSpec, storagePath, n.Client, n.Scheme, false, n.Config.Operator.Orderer.Timeouts.EnrollJob)
	if err != nil {
		return errors.Wrap(err, "failed to enroll for TLS cert")
	}

	err = n.Initializer.GenerateSecrets("tls", instance, crypto)
	if err != nil {
		return errors.Wrap(err, "failed to generate ecert secrets")
	}

	return nil
}

func (n *Node) FabricOrdererMigrationV2_0(instance *current.IBPOrderer) error {
	log.Info(fmt.Sprintf("Orderer instance '%s' migrating to v2", instance.GetName()))

	initOrderer, err := n.Initializer.GetInitOrderer(instance, n.GetInitStoragePath(instance))
	if err != nil {
		return err
	}
	initOrderer.UsingHSMProxy = instance.UsingHSMProxy()

	ordererConfig := n.Config.OrdererInitConfig.OrdererFile
	if version.GetMajorReleaseVersion(instance.Spec.FabricVersion) == version.V2 {
		currentVer := version.String(instance.Spec.FabricVersion)
		if currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_5_1) {
			ordererConfig = n.Config.OrdererInitConfig.OrdererV25File
		} else if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.GreaterThan(version.V2_4_1) {
			ordererConfig = n.Config.OrdererInitConfig.OrdererV24File
		} else {
			ordererConfig = n.Config.OrdererInitConfig.OrdererV2File
		}
	}

	switch version.GetMajorReleaseVersion(instance.Spec.FabricVersion) {
	case version.V2:
		currentVer := version.String(instance.Spec.FabricVersion)
		if currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_5_1) {
			log.Info("v2.5.x Fabric Orderer requested")
			v25config, err := v25ordererconfig.ReadOrdererFile(ordererConfig)
			if err != nil {
				return errors.Wrap(err, "failed to read v2.5.x default config file")
			}
			initOrderer.Config = v25config
		} else if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.GreaterThan(version.V2_4_1) {
			log.Info("v2.4.x Fabric Orderer requested")
			v24config, err := v24ordererconfig.ReadOrdererFile(ordererConfig)
			if err != nil {
				return errors.Wrap(err, "failed to read v2.4.x default config file")
			}
			initOrderer.Config = v24config
		} else if currentVer.LessThan(version.V2_4_1) {
			log.Info("v2.2.x Fabric Orderer requested")
			v2config, err := v2ordererconfig.ReadOrdererFile(ordererConfig)
			if err != nil {
				return errors.Wrap(err, "failed to read v2.2.x default config file")
			}
			initOrderer.Config = v2config
		}
	case version.V1:
		fallthrough
	default:
		// Choosing to default to v1.4 to not break backwards comptability, if coming
		// from a previous version of operator the 'FabricVersion' field would not be set and would
		// result in an error. // TODO: Determine if we want to throw error or handle setting
		// FabricVersion as part of migration logic.
		oconfig, err := ordererconfig.ReadOrdererFile(ordererConfig)
		if err != nil {
			return errors.Wrap(err, "failed to read v1.4 default config file")
		}
		initOrderer.Config = oconfig
	}

	configOverride, err := instance.GetConfigOverride()
	if err != nil {
		return err
	}

	err = initOrderer.OverrideConfig(configOverride.(OrdererConfig))
	if err != nil {
		return err
	}

	if instance.IsHSMEnabled() && !instance.UsingHSMProxy() {
		log.Info(fmt.Sprintf("During orderer '%s' migration, detected using HSM sidecar, setting library path", instance.GetName()))
		hsmConfig, err := commonconfig.ReadHSMConfig(n.Client, instance)
		if err != nil {
			return err
		}
		initOrderer.Config.SetBCCSPLibrary(filepath.Join("/hsm/lib", filepath.Base(hsmConfig.Library.FilePath)))
	}

	err = n.Initializer.CreateOrUpdateConfigMap(instance, initOrderer.GetConfig())
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) FabricOrdererMigrationV2_4(instance *current.IBPOrderer) error {
	log.Info(fmt.Sprintf("Orderer instance '%s' migrating to v2.4.x", instance.GetName()))

	initOrderer, err := n.Initializer.GetInitOrderer(instance, n.GetInitStoragePath(instance))
	if err != nil {
		return err
	}

	ordererConfig, err := v24ordererconfig.ReadOrdererFile(n.Config.OrdererInitConfig.OrdererV24File)
	if err != nil {
		return errors.Wrap(err, "failed to read v2.4.x default config file")
	}

	// removed the field from the struct
	// ordererConfig.FileLedger.Prefix = ""

	name := fmt.Sprintf("%s-env", instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: instance.Namespace,
	}

	cm := &corev1.ConfigMap{}
	err = n.Client.Get(context.TODO(), namespacedName, cm)
	if err != nil {
		return errors.Wrap(err, "failed to get env configmap")
	}

	// Add configs for 2.4.x
	trueVal := true
	ordererConfig.Admin.TLs.Enabled = &trueVal
	ordererConfig.Admin.TLs.ClientAuthRequired = &trueVal

	intermediateExists := util.IntermediateSecretExists(n.Client, instance.Namespace, fmt.Sprintf("ecert-%s-intercerts", instance.Name)) &&
		util.IntermediateSecretExists(n.Client, instance.Namespace, fmt.Sprintf("tls-%s-intercerts", instance.Name))
	intercertPath := "/certs/msp/tlsintermediatecerts/intercert-0.pem"
	currentVer := version.String(instance.Spec.FabricVersion)
	if currentVer.EqualWithoutTag(version.V2_4_1) || currentVer.GreaterThan(version.V2_4_1) {
		// Enable Channel participation for 2.4.x orderers
		cm.Data["ORDERER_CHANNELPARTICIPATION_ENABLED"] = "true"

		cm.Data["ORDERER_ADMIN_TLS_ENABLED"] = "true"
		cm.Data["ORDERER_ADMIN_TLS_CERTIFICATE"] = "/certs/tls/signcerts/cert.pem"
		cm.Data["ORDERER_ADMIN_TLS_PRIVATEKEY"] = "/certs/tls/keystore/key.pem"
		cm.Data["ORDERER_ADMIN_TLS_CLIENTAUTHREQUIRED"] = "true"
		// override the default value 127.0.0.1:9443
		cm.Data["ORDERER_ADMIN_LISTENADDRESS"] = "0.0.0.0:9443"
		if intermediateExists {
			// override intermediate cert paths for root and clientroot cas
			cm.Data["ORDERER_ADMIN_TLS_ROOTCAS"] = intercertPath
			cm.Data["ORDERER_ADMIN_TLS_CLIENTROOTCAS"] = intercertPath
		} else {
			cm.Data["ORDERER_ADMIN_TLS_ROOTCAS"] = "/certs/msp/tlscacerts/cacert-0.pem"
			cm.Data["ORDERER_ADMIN_TLS_CLIENTROOTCAS"] = "/certs/msp/tlscacerts/cacert-0.pem"
		}
	}

	err = n.Client.Update(context.TODO(), cm, controllerclient.UpdateOption{Owner: instance, Scheme: n.Scheme})
	if err != nil {
		return errors.Wrap(err, "failed to update env configmap")
	}

	initOrderer.Config = ordererConfig
	configOverride, err := instance.GetConfigOverride()
	if err != nil {
		return err
	}

	err = initOrderer.OverrideConfig(configOverride.(OrdererConfig))
	if err != nil {
		return err
	}

	if instance.IsHSMEnabled() && !instance.UsingHSMProxy() {
		log.Info(fmt.Sprintf("During orderer '%s' migration, detected using HSM sidecar, setting library path", instance.GetName()))
		hsmConfig, err := commonconfig.ReadHSMConfig(n.Client, instance)
		if err != nil {
			return err
		}
		initOrderer.Config.SetBCCSPLibrary(filepath.Join("/hsm/lib", filepath.Base(hsmConfig.Library.FilePath)))
	}

	err = n.Initializer.CreateOrUpdateConfigMap(instance, initOrderer.GetConfig())
	if err != nil {
		return err
	}
	return nil
}

func (n *Node) FabricOrdererMigrationV2_5(instance *current.IBPOrderer) error {
	log.Info(fmt.Sprintf("Orderer instance '%s' migrating to v2.5.x", instance.GetName()))

	initOrderer, err := n.Initializer.GetInitOrderer(instance, n.GetInitStoragePath(instance))
	if err != nil {
		return err
	}

	ordererConfig, err := v25ordererconfig.ReadOrdererFile(n.Config.OrdererInitConfig.OrdererV25File)
	if err != nil {
		return errors.Wrap(err, "failed to read v2.5.x default config file")
	}

	// removed the field from the struct
	// ordererConfig.FileLedger.Prefix = ""

	name := fmt.Sprintf("%s-env", instance.GetName())
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: instance.Namespace,
	}

	cm := &corev1.ConfigMap{}
	err = n.Client.Get(context.TODO(), namespacedName, cm)
	if err != nil {
		return errors.Wrap(err, "failed to get env configmap")
	}

	// Add configs for 2.5.x
	trueVal := true
	ordererConfig.Admin.TLs.Enabled = &trueVal
	ordererConfig.Admin.TLs.ClientAuthRequired = &trueVal

	intermediateExists := util.IntermediateSecretExists(n.Client, instance.Namespace, fmt.Sprintf("ecert-%s-intercerts", instance.Name)) &&
		util.IntermediateSecretExists(n.Client, instance.Namespace, fmt.Sprintf("tls-%s-intercerts", instance.Name))
	intercertPath := "/certs/msp/tlsintermediatecerts/intercert-0.pem"
	currentVer := version.String(instance.Spec.FabricVersion)
	if currentVer.EqualWithoutTag(version.V2_5_1) || currentVer.GreaterThan(version.V2_5_1) {
		// Enable Channel participation for 2.5.x orderers
		cm.Data["ORDERER_CHANNELPARTICIPATION_ENABLED"] = "true"

		cm.Data["ORDERER_GENERAL_CLUSTER_SENDBUFFERSIZE"] = "100"

		cm.Data["ORDERER_ADMIN_TLS_ENABLED"] = "true"
		cm.Data["ORDERER_ADMIN_TLS_CERTIFICATE"] = "/certs/tls/signcerts/cert.pem"
		cm.Data["ORDERER_ADMIN_TLS_PRIVATEKEY"] = "/certs/tls/keystore/key.pem"
		cm.Data["ORDERER_ADMIN_TLS_CLIENTAUTHREQUIRED"] = "true"
		// override the default value 127.0.0.1:9443
		cm.Data["ORDERER_ADMIN_LISTENADDRESS"] = "0.0.0.0:9443"
		if intermediateExists {
			// override intermediate cert paths for root and clientroot cas
			cm.Data["ORDERER_ADMIN_TLS_ROOTCAS"] = intercertPath
			cm.Data["ORDERER_ADMIN_TLS_CLIENTROOTCAS"] = intercertPath
		} else {
			cm.Data["ORDERER_ADMIN_TLS_ROOTCAS"] = "/certs/msp/tlscacerts/cacert-0.pem"
			cm.Data["ORDERER_ADMIN_TLS_CLIENTROOTCAS"] = "/certs/msp/tlscacerts/cacert-0.pem"
		}
	}

	err = n.Client.Update(context.TODO(), cm, controllerclient.UpdateOption{Owner: instance, Scheme: n.Scheme})
	if err != nil {
		return errors.Wrap(err, "failed to update env configmap")
	}

	initOrderer.Config = ordererConfig
	configOverride, err := instance.GetConfigOverride()
	if err != nil {
		return err
	}

	err = initOrderer.OverrideConfig(configOverride.(OrdererConfig))
	if err != nil {
		return err
	}

	if instance.IsHSMEnabled() && !instance.UsingHSMProxy() {
		log.Info(fmt.Sprintf("During orderer '%s' migration, detected using HSM sidecar, setting library path", instance.GetName()))
		hsmConfig, err := commonconfig.ReadHSMConfig(n.Client, instance)
		if err != nil {
			return err
		}
		initOrderer.Config.SetBCCSPLibrary(filepath.Join("/hsm/lib", filepath.Base(hsmConfig.Library.FilePath)))
	}

	err = n.Initializer.CreateOrUpdateConfigMap(instance, initOrderer.GetConfig())
	if err != nil {
		return err
	}
	return nil
}

func (n *Node) ReconcileHSMImages(instance *current.IBPOrderer) bool {
	hsmConfig, err := commonconfig.ReadHSMConfig(n.Client, instance)
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

func (n *Node) HandleActions(instance *current.IBPOrderer, update Update) error {
	orig := instance.DeepCopy()

	if update.EcertReenrollNeeded() {
		if err := n.ReenrollEcert(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetEcertReenroll()
			return err
		}
		instance.ResetEcertReenroll()
	}

	if update.TLScertReenrollNeeded() {
		if err := n.ReenrollTLSCert(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetTLSReenroll()
			return err
		}
		instance.ResetTLSReenroll()
	}

	if update.EcertNewKeyReenroll() {
		if err := n.ReenrollEcertNewKey(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetEcertReenroll()
			return err
		}
		instance.ResetEcertReenroll()
	}

	if update.TLScertNewKeyReenroll() {
		if err := n.ReenrollTLSCertNewKey(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetTLSReenroll()
			return err
		}
		instance.ResetTLSReenroll()
	}

	if update.EcertEnroll() {
		if err := n.EnrollForEcert(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetEcertEnroll()
			return err
		}
		instance.ResetEcertEnroll()
	}

	if update.TLScertEnroll() {
		if err := n.EnrollForTLSCert(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetTLSEnroll()
			return err
		}
		instance.ResetTLSEnroll()
	}

	// This should be the last action checked
	if update.RestartNeeded() {
		if err := n.RestartAction(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetRestart()
			return err
		}
		instance.ResetRestart()
	}

	if err := n.Client.Patch(context.TODO(), instance, k8sclient.MergeFrom(orig)); err != nil {
		return errors.Wrap(err, "failed to reset action flags")
	}

	return nil
}

func (n *Node) ReenrollEcert(instance *current.IBPOrderer) error {
	log.Info("Ecert reenroll triggered via action parameter")
	if err := n.reenrollCert(instance, commoninit.ECERT, false); err != nil {
		return errors.Wrap(err, "ecert reenroll reusing existing private key action failed")
	}
	return nil
}

func (n *Node) ReenrollEcertNewKey(instance *current.IBPOrderer) error {
	log.Info("Ecert with new key reenroll triggered via action parameter")
	if err := n.reenrollCert(instance, commoninit.ECERT, true); err != nil {
		return errors.Wrap(err, "ecert reenroll with new key action failed")
	}
	return nil
}

func (n *Node) ReenrollTLSCert(instance *current.IBPOrderer) error {
	log.Info("TLS reenroll triggered via action parameter")
	if err := n.reenrollCert(instance, commoninit.TLS, false); err != nil {
		return errors.Wrap(err, "tls reenroll reusing existing private key action failed")
	}
	return nil
}

func (n *Node) ReenrollTLSCertNewKey(instance *current.IBPOrderer) error {
	log.Info("TLS with new key reenroll triggered via action parameter")
	if err := n.reenrollCert(instance, commoninit.TLS, true); err != nil {
		return errors.Wrap(err, "tls reenroll with new key action failed")
	}
	return nil
}

func (n *Node) reenrollCert(instance *current.IBPOrderer, certType commoninit.SecretType, newKey bool) error {
	return action.Reenroll(n, n.Client, certType, instance, newKey)
}

func (n *Node) RestartAction(instance *current.IBPOrderer) error {
	log.Info("Restart triggered via action parameter")
	if err := n.Restart.ForRestartAction(instance); err != nil {
		return errors.Wrap(err, "failed to restart orderer node pods")
	}
	return nil
}

func (n *Node) HandleRestart(instance *current.IBPOrderer, update Update) error {
	// If restart is disabled for components, can return immediately
	if n.Config.Operator.Restart.Disable.Components {
		return nil
	}

	err := n.Restart.TriggerIfNeeded(instance)
	if err != nil {
		return errors.Wrap(err, "failed to restart deployment")
	}

	return nil
}

func (n *Node) CustomLogic(instance *current.IBPOrderer, update Update) (*current.CRStatus, *common.Result, error) {
	var status *current.CRStatus
	var err error
	if !n.CanSetCertificateTimer(instance, update) {
		log.Info("Certificate update detected but all nodes not yet deployed, requeuing request...")
		return status, &common.Result{
			Result: reconcile.Result{
				Requeue: true,
			},
		}, nil
	}

	// Check if crypto needs to be backed up before an update overrides exisitng secrets
	if update.CryptoBackupNeeded() {
		log.Info("Performing backup of TLS and ecert crypto")
		err = common.BackupCrypto(n.Client, n.Scheme, instance, n.GetLabels(instance))
		if err != nil {
			return status, nil, errors.Wrap(err, "failed to backup TLS and ecert crypto")
		}
	}

	status, err = n.CheckCertificates(instance)
	if err != nil {
		return status, nil, errors.Wrap(err, "failed to check for expiring certificates")
	}

	if update.CertificateCreated() {
		log.Info(fmt.Sprintf("%s certificate was created, setting timer for certificate renewal", update.GetCreatedCertType()))
		err = n.SetCertificateTimer(instance, update.GetCreatedCertType())
		if err != nil {
			return status, nil, errors.Wrap(err, "failed to set timer for certificate renewal")
		}
	}

	if update.EcertUpdated() {
		log.Info("Ecert was updated, setting timer for certificate renewal")
		err = n.SetCertificateTimer(instance, commoninit.ECERT)
		if err != nil {
			return status, nil, errors.Wrap(err, "failed to set timer for certificate renewal")
		}
	}

	if update.TLSCertUpdated() {
		log.Info("TLS cert was updated, setting timer for certificate renewal")
		err = n.SetCertificateTimer(instance, commoninit.TLS)
		if err != nil {
			return status, nil, errors.Wrap(err, "failed to set timer for certificate renewal")
		}
	}
	return status, nil, err

}

func (n *Node) CheckCertificates(instance *current.IBPOrderer) (*current.CRStatus, error) {
	numSecondsBeforeExpire := instance.Spec.GetNumSecondsWarningPeriod()
	statusType, message, err := n.CertificateManager.CheckCertificatesForExpire(instance, numSecondsBeforeExpire)
	if err != nil {
		return nil, err
	}

	crStatus := &current.CRStatus{
		Type:    statusType,
		Message: message,
	}

	switch statusType {
	case current.Deployed:
		crStatus.Reason = "allPodsRunning"
		if message == "" {
			crStatus.Message = "allPodsRunning"
		}
	default:
		crStatus.Reason = "certRenewalRequired"
	}

	return crStatus, nil
}

func (n *Node) SetCertificateTimer(instance *current.IBPOrderer, certType commoninit.SecretType) error {
	certName := fmt.Sprintf("%s-%s-signcert", certType, instance.Name)
	numSecondsBeforeExpire := instance.Spec.GetNumSecondsWarningPeriod()
	duration, err := n.CertificateManager.GetDurationToNextRenewal(certType, instance, numSecondsBeforeExpire)
	if err != nil {
		return err
	}

	log.Info((fmt.Sprintf("Creating timer to renew %s %d days before it expires", certName, int(numSecondsBeforeExpire/DaysToSecondsConversion))))

	if n.RenewCertTimers[certName] != nil {
		n.RenewCertTimers[certName].Stop()
		n.RenewCertTimers[certName] = nil
	}
	n.RenewCertTimers[certName] = time.AfterFunc(duration, func() {
		// Check certs for updated status & set status so that reconcile is triggered after cert renewal. Reconcile loop will handle
		// checking certs again to determine whether instance status can return to Deployed
		err := n.UpdateCRStatus(instance)
		if err != nil {
			log.Error(err, "failed to update CR status")
		}

		// get instance
		instanceLatest := &current.IBPOrderer{}
		err = n.Client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, instanceLatest)
		if err != nil {
			log.Error(err, "failed to get latest instance")
			return
		}

		// Orderer TLS certs can be auto-renewed for 1.4.9+ or 2.2.1+ orderers
		if certType == commoninit.TLS {
			// if renewal is disabled
			if n.Config.Operator.Orderer.Renewals.DisableTLScert {
				log.Info(fmt.Sprintf("%s cannot be auto-renewed because orderer tls renewal is disabled", certName))
				return
			}
			switch version.GetMajorReleaseVersion(instanceLatest.Spec.FabricVersion) {
			case version.V2:
				if version.String(instanceLatest.Spec.FabricVersion).LessThan("2.2.1") {
					log.Info(fmt.Sprintf("%s cannot be auto-renewed because v2 orderer is less than 2.2.1, force renewal required", certName))
					return
				}
			case version.V1:
				if version.String(instanceLatest.Spec.FabricVersion).LessThan("1.4.9") {
					log.Info(fmt.Sprintf("%s cannot be auto-renewed because v1.4 orderer less than 1.4.9, force renewal required", certName))
					return
				}
			default:
				log.Info(fmt.Sprintf("%s cannot be auto-renewed, force renewal required", certName))
				return
			}
		}

		err = common.BackupCrypto(n.Client, n.Scheme, instance, n.GetLabels(instance))
		if err != nil {
			log.Error(err, "failed to backup crypto before renewing cert")
			return
		}

		err = n.RenewCert(certType, instanceLatest, false)
		if err != nil {
			log.Info(fmt.Sprintf("Failed to renew %s certificate: %s, status of %s remaining in Warning phase", certType, err, instanceLatest.GetName()))
			return
		}
		log.Info(fmt.Sprintf("%s renewal complete", certName))
	})

	return nil
}

// NOTE: This is called by the timer's subroutine when it goes off, not during a reconcile loop.
// Therefore, it won't be overriden by the "SetStatus" method in ibporderer_controller.go
func (n *Node) UpdateCRStatus(instance *current.IBPOrderer) error {
	status, err := n.CheckCertificates(instance)
	if err != nil {
		return errors.Wrap(err, "failed to check certificates")
	}

	// Get most up-to-date instance at the time of update
	updatedInstance := &current.IBPOrderer{}
	err = n.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, updatedInstance)
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

	log.Info(fmt.Sprintf("Updating status of IBPOrderer node %s to %s phase", instance.Name, status.Type))
	err = n.Client.UpdateStatus(context.TODO(), updatedInstance)
	if err != nil {
		return errors.Wrapf(err, "failed to update status to %s phase", status.Type)
	}

	return nil
}

// This function checks whether the parent orderer node (if parent exists) or node itself  is in
// Deployed or Warning state. We don't want to set a timer to renew certifictes before all nodes
// are Deployed as a certificate renewal updates the parent status to Warning while renewing.
func (n *Node) CanSetCertificateTimer(instance *current.IBPOrderer, update Update) bool {
	if update.CertificateCreated() || update.CertificateUpdated() {
		parentName := instance.Labels["parent"]
		if parentName == "" {
			// If parent not found, check individual node
			if !(instance.Status.Type == current.Deployed || instance.Status.Type == current.Warning) {
				log.Info(fmt.Sprintf("%s has no parent, node not yet deployed", instance.Name))
				return false
			} else {
				log.Info(fmt.Sprintf("%s has no parent, node is deployed", instance.Name))
				return true
			}
		}

		nn := types.NamespacedName{
			Name:      parentName,
			Namespace: instance.GetNamespace(),
		}

		parentInstance := &current.IBPOrderer{}
		err := n.Client.Get(context.TODO(), nn, parentInstance)
		if err != nil {
			log.Error(err, fmt.Sprintf("%s parent not found", instance.Name))
			return false
		}

		// If parent not yet deployed, but cert update detected, then prevent timer from being set until parent
		// (and subequently all child nodes) are deployed
		if !(parentInstance.Status.Type == current.Deployed || parentInstance.Status.Type == current.Warning) {
			log.Info(fmt.Sprintf("%s has parent, parent not yet deployed", instance.Name))
			return false
		}
	}

	log.Info(fmt.Sprintf("%s has parent, parent is deployed", instance.Name))
	return true
}
