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

package baseca

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	cav1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	controllerclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common/reconcilechecks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("base_ca")

const (
	DaysToSecondsConversion = int64(24 * 60 * 60)
)

type Override interface {
	Deployment(v1.Object, *appsv1.Deployment, resources.Action) error
	Service(v1.Object, *corev1.Service, resources.Action) error
	PVC(v1.Object, *corev1.PersistentVolumeClaim, resources.Action) error
	Role(v1.Object, *rbacv1.Role, resources.Action) error
	RoleBinding(v1.Object, *rbacv1.RoleBinding, resources.Action) error
	ServiceAccount(v1.Object, *corev1.ServiceAccount, resources.Action) error
	IsPostgres(instance *current.IBPCA) bool
}

//go:generate counterfeiter -o mocks/update.go -fake-name Update . Update

type Update interface {
	SpecUpdated() bool
	CAOverridesUpdated() bool
	TLSCAOverridesUpdated() bool
	ConfigOverridesUpdated() bool
	RestartNeeded() bool
	CACryptoUpdated() bool
	CACryptoCreated() bool
	RenewTLSCert() bool
	FabricVersionUpdated() bool
	ImagesUpdated() bool
	CATagUpdated() bool
}

//go:generate counterfeiter -o mocks/restart_manager.go -fake-name RestartManager . RestartManager

type RestartManager interface {
	ForConfigOverride(instance v1.Object) error
	TriggerIfNeeded(instance restart.Instance) error
	ForTLSReenroll(instance v1.Object) error
	ForRestartAction(instance v1.Object) error
}

type IBPCA interface {
	Initialize(instance *current.IBPCA, update Update) error
	PreReconcileChecks(instance *current.IBPCA, update Update) (bool, error)
	ReconcileManagers(instance *current.IBPCA, update Update) error
	Reconcile(instance *current.IBPCA, update Update) (common.Result, error)
}

//go:generate counterfeiter -o mocks/initialize.go -fake-name InitializeIBPCA . InitializeIBPCA

type InitializeIBPCA interface {
	HandleEnrollmentCAInit(instance *current.IBPCA, update Update) (*initializer.Response, error)
	HandleConfigResources(name string, instance *current.IBPCA, resp *initializer.Response, update Update) error
	HandleTLSCAInit(instance *current.IBPCA, update Update) (*initializer.Response, error)
	SyncDBConfig(*current.IBPCA) (*current.IBPCA, error)
	CreateOrUpdateConfigMap(instance *current.IBPCA, data map[string][]byte, name string) error
	ReadConfigMap(instance *current.IBPCA, name string) (*corev1.ConfigMap, error)
}

//go:generate counterfeiter -o mocks/certificate_manager.go -fake-name CertificateManager . CertificateManager

type CertificateManager interface {
	GetDurationToNextRenewalForCert(string, []byte, v1.Object, int64) (time.Duration, error)
	GetSecret(string, string) (*corev1.Secret, error)
	Expires([]byte, int64) (bool, time.Time, error)
	UpdateSecret(v1.Object, string, map[string][]byte) error
}

var _ IBPCA = &CA{}

type CA struct {
	Client controllerclient.Client
	Scheme *runtime.Scheme
	Config *config.Config

	DeploymentManager     resources.Manager
	ServiceManager        resources.Manager
	PVCManager            resources.Manager
	RoleManager           resources.Manager
	RoleBindingManager    resources.Manager
	ServiceAccountManager resources.Manager

	Override    Override
	Initializer InitializeIBPCA

	CertificateManager CertificateManager
	RenewCertTimers    map[string]*time.Timer

	Restart RestartManager
}

func New(client controllerclient.Client, scheme *runtime.Scheme, config *config.Config, o Override) *CA {
	ca := &CA{
		Client:   client,
		Scheme:   scheme,
		Config:   config,
		Override: o,
	}
	ca.CreateManagers()
	ca.Initializer = NewInitializer(config.CAInitConfig, scheme, client, ca.GetLabels, config.Operator.CA.Timeouts.HSMInitJob)
	ca.Restart = restart.New(client, config.Operator.Restart.WaitTime.Get(), config.Operator.Restart.Timeout.Get())
	ca.CertificateManager = &certificate.CertificateManager{
		Client: client,
		Scheme: scheme,
	}
	ca.RenewCertTimers = make(map[string]*time.Timer)

	return ca
}

func (ca *CA) CreateManagers() {
	override := ca.Override
	resourceManager := resourcemanager.New(ca.Client, ca.Scheme)
	ca.DeploymentManager = resourceManager.CreateDeploymentManager("", override.Deployment, ca.GetLabels, ca.Config.CAInitConfig.DeploymentFile)
	ca.ServiceManager = resourceManager.CreateServiceManager("", override.Service, ca.GetLabels, ca.Config.CAInitConfig.ServiceFile)
	ca.PVCManager = resourceManager.CreatePVCManager("", override.PVC, ca.GetLabels, ca.Config.CAInitConfig.PVCFile)
	ca.RoleManager = resourceManager.CreateRoleManager("", override.Role, ca.GetLabels, ca.Config.CAInitConfig.RoleFile)
	ca.RoleBindingManager = resourceManager.CreateRoleBindingManager("", override.RoleBinding, ca.GetLabels, ca.Config.CAInitConfig.RoleBindingFile)
	ca.ServiceAccountManager = resourceManager.CreateServiceAccountManager("", override.ServiceAccount, ca.GetLabels, ca.Config.CAInitConfig.ServiceAccountFile)
}

func (ca *CA) Reconcile(instance *current.IBPCA, update Update) (common.Result, error) {
	var err error

	versionSet, err := ca.SetVersion(instance)
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

	instanceUpdated, err := ca.PreReconcileChecks(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed pre reconcile checks")
	}

	if instanceUpdated {
		log.Info("Updating instance after pre reconcile checks")
		err := ca.Client.Patch(context.TODO(), instance, nil, controllerclient.PatchOption{
			Resilient: &controllerclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPCA{},
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

	err = ca.AddTLSCryptoIfMissing(instance, ca.GetEndpointsDNS(instance))
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to generate tls crypto")
	}

	err = ca.Initialize(instance, update)
	if err != nil {
		return common.Result{}, operatorerrors.Wrap(err, operatorerrors.CAInitilizationFailed, "failed to initialize ca")
	}

	err = ca.ReconcileManagers(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to reconcile managers")
	}

	err = ca.UpdateConnectionProfile(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to create connection profile")
	}

	err = ca.CheckStates(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check and restore state")
	}

	if update.CACryptoUpdated() {
		log.Info("TLS crypto updated, triggering restart")
		err = ca.Restart.ForTLSReenroll(instance)
		if err != nil {
			return common.Result{}, errors.Wrap(err, "failed to update restart config")
		}
	}

	err = ca.HandleActions(instance, update)
	if err != nil {
		return common.Result{}, err
	}

	err = ca.HandleRestart(instance, update)
	if err != nil {
		return common.Result{}, err
	}

	return common.Result{}, nil
}

// PreReconcileChecks validate CR request before starting reconcile flow
func (ca *CA) PreReconcileChecks(instance *current.IBPCA, update Update) (bool, error) {
	var err error

	imagesUpdated, err := reconcilechecks.FabricVersionHelper(instance, ca.Config.Operator.Versions, update)
	if err != nil {
		return false, errors.Wrap(err, "failed to during version and image checks")
	}

	var maxNameLength *int
	if instance.Spec.ConfigOverride != nil {
		maxNameLength = instance.Spec.ConfigOverride.MaxNameLength
	}
	err = util.ValidationChecks(instance.TypeMeta, instance.ObjectMeta, "IBPCA", maxNameLength)
	if err != nil {
		return false, err
	}

	if instance.Spec.HSMSet() {
		err = util.ValidateHSMProxyURL(instance.Spec.HSM.PKCS11Endpoint)
		if err != nil {
			return false, errors.Wrapf(err, "invalid HSM endpoint for ca instance '%s'", instance.GetName())
		}
	}

	if !instance.Spec.DomainSet() {
		return false, fmt.Errorf("domain not set for ca instance '%s'", instance.GetName())
	}

	zoneUpdated, err := ca.SelectZone(instance)
	if err != nil {
		return false, err
	}

	regionUpdated, err := ca.SelectRegion(instance)
	if err != nil {
		return false, err
	}

	hsmImageUpdated := ca.ReconcileHSMImages(instance)

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

func (ca *CA) SetVersion(instance *current.IBPCA) (bool, error) {
	if instance.Status.Version == "" || !version.String(instance.Status.Version).Equal(version.Operator) {
		log.Info("Version of Operator: ", "version", version.Operator)
		log.Info("Version of CR: ", "version", instance.Status.Version)
		log.Info(fmt.Sprintf("Setting '%s' to version '%s'", instance.Name, version.Operator))

		instance.Status.Version = version.Operator
		err := ca.Client.PatchStatus(context.TODO(), instance, nil, controllerclient.PatchOption{
			Resilient: &controllerclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPCA{},
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

func (ca *CA) Initialize(instance *current.IBPCA, update Update) error {
	var err error

	// TODO: Add checks to determine if initialization is neeeded. Split this method into
	// two, one should handle initialization during the create event of a CR and the other
	// should update events

	// Service account is required by job
	err = ca.ReconcileRBAC(instance)
	if err != nil {
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
			hsmConfig, err := commonconfig.ReadHSMConfig(ca.Client, instance)
			if err != nil {
				return errors.New("using non-proxy HSM, but no HSM config defined as config map 'ibp-hsm-config'")
			}

			if hsmConfig.Daemon != nil {
				log.Info("Using daemon based HSM, creating pvc...")
				ca.PVCManager.SetCustomName(instance.Spec.CustomNames.PVC.CA)
				err = ca.PVCManager.Reconcile(instance, update.SpecUpdated())
				if err != nil {
					return errors.Wrap(err, "failed PVC reconciliation")
				}
			}
		}
	}

	instance, err = ca.Initializer.SyncDBConfig(instance)
	if err != nil {
		return err
	}

	eresp, err := ca.Initializer.HandleEnrollmentCAInit(instance, update)
	if err != nil {
		return err
	}

	if eresp != nil {
		err = ca.Initializer.HandleConfigResources(fmt.Sprintf("%s-ca", instance.GetName()), instance, eresp, update)
		if err != nil {
			return err
		}
	}

	tresp, err := ca.Initializer.HandleTLSCAInit(instance, update)
	if err != nil {
		return err
	}

	if tresp != nil {
		err = ca.Initializer.HandleConfigResources(fmt.Sprintf("%s-tlsca", instance.GetName()), instance, tresp, update)
		if err != nil {
			return err
		}
	}

	// If deployment exists, and configoverride update detected need to restart pod(s) to pick up
	// the latest configuration from configmap and secret
	if ca.DeploymentManager.Exists(instance) && update.ConfigOverridesUpdated() {
		// Request deployment restart for config override update
		if err := ca.Restart.ForConfigOverride(instance); err != nil {
			return err
		}
	}

	return nil
}

func (ca *CA) SelectZone(instance *current.IBPCA) (bool, error) {
	if instance.Spec.Zone == "select" {
		zone := util.GetZone(ca.Client)
		instance.Spec.Zone = zone
		return true, nil
	}
	if instance.Spec.Zone != "" {
		err := util.ValidateZone(ca.Client, instance.Spec.Zone)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (ca *CA) SelectRegion(instance *current.IBPCA) (bool, error) {
	if instance.Spec.Region == "select" {
		region := util.GetRegion(ca.Client)
		instance.Spec.Region = region
		return true, nil
	}
	if instance.Spec.Region != "" {
		err := util.ValidateRegion(ca.Client, instance.Spec.Region)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (ca *CA) ReconcileManagers(instance *current.IBPCA, updated Update) error {
	var err error

	update := updated.SpecUpdated()

	if !ca.Override.IsPostgres(instance) {
		log.Info("Using sqlite database, creating pvc...")
		ca.PVCManager.SetCustomName(instance.Spec.CustomNames.PVC.CA)
		err = ca.PVCManager.Reconcile(instance, update)
		if err != nil {
			return errors.Wrap(err, "failed PVC reconciliation")
		}
	}

	err = ca.ServiceManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Service reconciliation")
	}

	err = ca.ReconcileRBAC(instance)
	if err != nil {
		return errors.Wrap(err, "failed RBAC reconciliation")
	}

	err = ca.DeploymentManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Deployment reconciliation")
	}

	// TODO: Can this be removed?
	err = ca.createSecret(instance, "-ca")
	if err != nil {
		return errors.Wrap(err, "failed CA Secret reconciliation")
	}

	// TODO: Can this be removed?
	err = ca.createSecret(instance, "-tlsca")
	if err != nil {
		return errors.Wrap(err, "failed TLS Secret reconciliation")
	}

	return nil
}

func (ca *CA) ReconcileRBAC(instance *current.IBPCA) error {
	var err error

	err = ca.RoleManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	err = ca.RoleBindingManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	err = ca.ServiceAccountManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	return nil
}

func (ca *CA) UpdateConnectionProfile(instance *current.IBPCA) error {
	var err error

	endpoints := ca.GetEndpoints(instance)

	cacrypto, err := common.GetCACryptoEncoded(ca.Client, instance)
	if err != nil {
		return err
	}
	tlscacrypto, err := common.GetTLSCACryptoEncoded(ca.Client, instance)
	if err != nil {
		return err
	}

	err = ca.UpdateConnectionProfileConfigmap(instance, *endpoints, cacrypto.TLSCert, cacrypto.Cert, tlscacrypto.Cert)
	if err != nil {
		return err
	}

	return nil
}

func (ca *CA) UpdateConnectionProfileConfigmap(instance *current.IBPCA, endpoints current.CAEndpoints, tlscert, cacert, tlscacert string) error {
	var err error

	name := instance.Name + "-connection-profile"
	nn := types.NamespacedName{
		Name:      name,
		Namespace: instance.GetNamespace(),
	}

	log.Info(fmt.Sprintf("Create connection profle configmap called for %s", instance.Name))
	connectionProfile := &current.CAConnectionProfile{
		Endpoints: endpoints,
		TLS: &current.ConnectionProfileTLS{
			Cert: tlscert,
		},
		CA: &current.MSP{
			SignCerts: cacert,
		},
		TLSCA: &current.MSP{
			SignCerts: tlscacert,
		},
	}

	bytes, err := json.Marshal(connectionProfile)
	if err != nil {
		return errors.Wrap(err, "failed to marshal connectionprofile")
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: instance.GetNamespace(),
			Labels:    ca.GetLabels(instance),
		},
		BinaryData: map[string][]byte{"profile.json": bytes},
	}

	err = ca.Client.Get(context.TODO(), nn, &corev1.ConfigMap{})
	if err == nil {
		err = ca.Client.Update(context.TODO(), cm, controllerclient.UpdateOption{Owner: instance, Scheme: ca.Scheme})
		if err != nil {
			return errors.Wrap(err, "failed to update connection profile configmap")
		}
	} else {
		err = ca.Client.Create(context.TODO(), cm, controllerclient.CreateOption{Owner: instance, Scheme: ca.Scheme})
		if err != nil {
			return errors.Wrap(err, "failed to create connection profile configmap")
		}
	}

	return nil
}

func (ca *CA) GetEndpoints(instance *current.IBPCA) *current.CAEndpoints {
	endpoints := &current.CAEndpoints{
		API:        "https://" + instance.Namespace + "-" + instance.Name + "-ca." + instance.Spec.Domain + ":443",
		Operations: "https://" + instance.Namespace + "-" + instance.Name + "-operations." + instance.Spec.Domain + ":443",
	}
	return endpoints
}

func (ca *CA) CheckStates(instance *current.IBPCA) error {
	// Check state if deployment exists, make sure that deployment matches what is expected
	// base on IBPCA spec
	if ca.DeploymentManager.Exists(instance) {
		err := ca.DeploymentManager.CheckState(instance)
		if err != nil {
			log.Error(err, "unexpected state")
			err = ca.DeploymentManager.RestoreState(instance)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (ca *CA) GetLabels(instance v1.Object) map[string]string {
	return instance.GetLabels()
}

// TODO: Can this be removed?
func (ca *CA) createSecret(instance *current.IBPCA, suffix string) error {
	secretCA := &corev1.Secret{}
	secretCA.Name = instance.Name + suffix
	secretCA.Namespace = instance.Namespace
	secretCA.Labels = ca.GetLabels(instance)

	secretCA.Data = map[string][]byte{}
	secretCA.Data["_shared_creation"] = []byte("-----BEGIN")

	err := ca.Client.Create(context.TODO(), secretCA, controllerclient.CreateOption{
		Owner:  instance,
		Scheme: ca.Scheme,
	})
	if err != nil {
		return err
	}

	return nil
}

func (ca *CA) CreateCACryptoSecret(instance *current.IBPCA, caCrypto map[string][]byte) error {
	// Create CA secret with crypto
	secret := &corev1.Secret{
		Data: caCrypto,
		Type: corev1.SecretTypeOpaque,
	}
	secret.Name = instance.Name + "-ca-crypto"
	secret.Namespace = instance.Namespace
	secret.Labels = ca.GetLabels(instance)

	err := ca.Client.Create(context.TODO(), secret, controllerclient.CreateOption{
		Owner:  instance,
		Scheme: ca.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create CA crypto secret")
	}

	return nil
}

func (ca *CA) CreateTLSCACryptoSecret(instance *current.IBPCA, tlscaCrypto map[string][]byte) error {
	// Create TLSCA secret with crypto
	secret := &corev1.Secret{
		Data: tlscaCrypto,
		Type: corev1.SecretTypeOpaque,
	}
	secret.Name = instance.Name + "-tlsca-crypto"
	secret.Namespace = instance.Namespace
	secret.Labels = ca.GetLabels(instance)

	err := ca.Client.Create(context.TODO(), secret, controllerclient.CreateOption{
		Owner:  instance,
		Scheme: ca.Scheme,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create TLS CA crypto secret")
	}

	return nil
}

func (ca *CA) AddTLSCryptoIfMissing(instance *current.IBPCA, endpoints *current.CAEndpoints) error {
	var err error
	caOverrides := &cav1.ServerConfig{}

	genTLSCrypto := func() error {
		tlskey, tlscert, err := ca.GenTLSCrypto(instance, endpoints)
		if err != nil {
			return err
		}

		base64cert := base64.StdEncoding.EncodeToString(tlscert)
		base64key := base64.StdEncoding.EncodeToString(tlskey)

		caOverrides.TLS = cav1.ServerTLSConfig{
			Enabled:  pointer.True(),
			CertFile: base64cert,
			KeyFile:  base64key,
		}

		caBytes, err := json.Marshal(caOverrides)
		if err != nil {
			return err
		}

		instance.Spec.ConfigOverride.CA = &runtime.RawExtension{Raw: caBytes}
		return nil
	}

	// check for cert
	err = ca.CheckForTLSSecret(instance)
	if err != nil {
		log.Info(fmt.Sprintf("No TLS crypto configurated for CA '%s', generating TLS crypto...", instance.GetName()))
		// that means secret is not found on cluster
		if instance.Spec.ConfigOverride == nil {
			instance.Spec.ConfigOverride = &current.ConfigOverride{}
			err := genTLSCrypto()
			if err != nil {
				return err
			}

			return nil
		}

		if instance.Spec.ConfigOverride.CA == nil {
			err := genTLSCrypto()
			if err != nil {
				return err
			}

			return nil
		}

		if instance.Spec.ConfigOverride != nil && instance.Spec.ConfigOverride.CA != nil {
			err = json.Unmarshal(instance.Spec.ConfigOverride.CA.Raw, caOverrides)
			if err != nil {
				return err
			}

			if caOverrides.TLS.CertFile == "" {
				err := genTLSCrypto()
				if err != nil {
					return err
				}
			}

			return nil
		}
	}

	return nil
}

func (ca *CA) GenTLSCrypto(instance *current.IBPCA, endpoints *current.CAEndpoints) ([]byte, []byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate key")
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := crand.Int(crand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate serial number")
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 87600) // valid for 10 years

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Issuer: pkix.Name{
			Country:            []string{"US"},
			Province:           []string{"North Carolina"},
			Locality:           []string{"Durham"},
			Organization:       []string{"IBM"},
			OrganizationalUnit: []string{"Blockchain"},
			CommonName:         endpoints.API,
		},
		Subject: pkix.Name{
			Country:            []string{"US"},
			Province:           []string{"North Carolina"},
			Locality:           []string{"Durham"},
			Organization:       []string{"IBM"},
			OrganizationalUnit: []string{"Blockchain"},
			CommonName:         endpoints.API,
		},

		NotBefore: notBefore,
		NotAfter:  notAfter,
	}

	ip := net.ParseIP(endpoints.API)
	if ip == nil {
		template.DNSNames = append(template.DNSNames, endpoints.API)
		template.DNSNames = append(template.DNSNames, strings.Replace(endpoints.API, "-ca.", ".", -1))
	} else {
		template.IPAddresses = append(template.IPAddresses, ip)
	}
	ip = net.ParseIP(endpoints.Operations)
	if ip == nil {
		template.DNSNames = append(template.DNSNames, endpoints.Operations)
	} else {
		template.IPAddresses = append(template.IPAddresses, ip)
	}

	derBytes, err := x509.CreateCertificate(crand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create certificate")
	}

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal key")
	}

	certPEM := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}
	keyPEM := &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}

	certBytes := pem.EncodeToMemory(certPEM)
	keyBytes = pem.EncodeToMemory(keyPEM)

	return keyBytes, certBytes, nil
}

func (ca *CA) CheckForTLSSecret(instance *current.IBPCA) error {
	secret := &corev1.Secret{}
	err := ca.Client.Get(context.TODO(), types.NamespacedName{
		Name:      fmt.Sprintf("%s-tlsca-crypto", instance.Name),
		Namespace: instance.Namespace}, secret)
	return err
}

func (ca *CA) CheckCertificates(instance *current.IBPCA) (*current.CRStatus, error) {
	secret, err := ca.CertificateManager.GetSecret(
		fmt.Sprintf("%s-ca-crypto", instance.GetName()),
		instance.GetNamespace(),
	)

	numSecondsBeforeExpire := instance.GetNumSecondsWarningPeriod()
	expiring, expireDate, err := ca.CertificateManager.Expires(secret.Data["tls-cert.pem"], numSecondsBeforeExpire)
	if err != nil {
		return nil, err
	}

	var message string
	statusType := current.Deployed

	if expiring {
		statusType = current.Warning
		// Check if tls cert's expiration date has already passed
		if expireDate.Before(time.Now()) {
			statusType = current.Error
			message += fmt.Sprintf("TLS cert for '%s' has expired", instance.GetName())
		} else {
			message += fmt.Sprintf("TLS cert for '%s' expires on %s", instance.GetName(), expireDate.String())
		}
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

func (ca *CA) RenewCert(instance *current.IBPCA, endpoints *current.CAEndpoints) error {
	log.Info(fmt.Sprintf("Renewing TLS certificate for CA '%s'", instance.GetName()))

	tlskey, tlscert, err := ca.GenTLSCrypto(instance, endpoints)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s-ca-crypto", instance.GetName())
	secret, err := ca.CertificateManager.GetSecret(
		name,
		instance.GetNamespace(),
	)

	secret.Data["tls-cert.pem"] = tlscert
	secret.Data["tls-key.pem"] = tlskey
	secret.Data["operations-cert.pem"] = tlscert
	secret.Data["operations-key.pem"] = tlskey

	if err := ca.CertificateManager.UpdateSecret(instance, name, secret.Data); err != nil {
		return err
	}

	return nil
}

func (ca *CA) GetEndpointsDNS(instance *current.IBPCA) *current.CAEndpoints {
	return &current.CAEndpoints{
		API:        fmt.Sprintf("%s-%s-ca.%s", instance.Namespace, instance.Name, instance.Spec.Domain),
		Operations: fmt.Sprintf("%s-%s-operations.%s", instance.Namespace, instance.Name, instance.Spec.Domain),
	}
}

func (ca *CA) ReconcileHSMImages(instance *current.IBPCA) bool {
	hsmConfig, err := commonconfig.ReadHSMConfig(ca.Client, instance)
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

func (ca *CA) HandleActions(instance *current.IBPCA, update Update) error {
	orig := instance.DeepCopy()

	if update.RenewTLSCert() {
		if err := common.BackupCACrypto(ca.Client, ca.Scheme, instance, ca.GetLabels(instance)); err != nil {
			return errors.Wrap(err, "failed to backup crypto before renewing cert")
		}

		if err := ca.RenewCert(instance, ca.GetEndpointsDNS(instance)); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetTLSRenew()
			return err
		}
		instance.ResetTLSRenew()
	}

	if update.RestartNeeded() {
		if err := ca.RestartAction(instance); err != nil {
			log.Error(err, "Resetting action flag on failure")
			instance.ResetRestart()
			return err
		}
		instance.ResetRestart()
	}

	if err := ca.Client.Patch(context.TODO(), instance, k8sclient.MergeFrom(orig)); err != nil {
		return errors.Wrap(err, "failed to reset action flags")
	}

	return nil
}

func (ca *CA) RestartAction(instance *current.IBPCA) error {
	log.Info("Restart triggered via action parameter")
	if err := ca.Restart.ForRestartAction(instance); err != nil {
		return errors.Wrap(err, "failed to restart ca pods")
	}
	return nil
}

func (ca *CA) HandleRestart(instance *current.IBPCA, update Update) error {
	// If restart is disabled for components, can return immediately
	if ca.Config.Operator.Restart.Disable.Components {
		return nil
	}

	err := ca.Restart.TriggerIfNeeded(instance)
	if err != nil {
		return errors.Wrap(err, "failed to restart deployment")
	}

	return nil
}

func (ca *CA) ReconcileFabricCAMigration(instance *current.IBPCA) error {
	cmname := fmt.Sprintf("%s-ca-config", instance.GetName())
	cm, err := ca.Initializer.ReadConfigMap(instance, cmname)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Migrating config map '%s'", cmname))

	serverConfig := &cav1.ServerConfig{}
	err = yaml.Unmarshal(cm.BinaryData["fabric-ca-server-config.yaml"], serverConfig)
	if err != nil {
		return err
	}

	if serverConfig.CA.ReenrollIgnoreCertExpiry == pointer.True() {
		// if it is already updated no need to update configmap
		return nil
	} else {
		serverConfig.CA.ReenrollIgnoreCertExpiry = pointer.True()
	}

	caConfigBytes, err := yaml.Marshal(serverConfig)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Updating config map '%s'", cmname))

	cm.BinaryData["fabric-ca-server-config.yaml"] = caConfigBytes

	err = ca.Initializer.CreateOrUpdateConfigMap(instance, cm.BinaryData, cmname)
	if err != nil {
		return err
	}
	return nil
}
