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

package baseconsole

import (
	"context"
	"fmt"
	"os"
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sruntime "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("base_console")

type Override interface {
	Deployment(v1.Object, *appsv1.Deployment, resources.Action) error
	Service(v1.Object, *corev1.Service, resources.Action) error
	DeployerService(v1.Object, *corev1.Service, resources.Action) error
	ServiceAccount(v1.Object, *corev1.ServiceAccount, resources.Action) error
	PVC(v1.Object, *corev1.PersistentVolumeClaim, resources.Action) error
	CM(v1.Object, *corev1.ConfigMap, resources.Action, map[string]interface{}) error
	ConsoleCM(v1.Object, *corev1.ConfigMap, resources.Action, map[string]interface{}) error
	DeployerCM(v1.Object, *corev1.ConfigMap, resources.Action, map[string]interface{}) error
}

//go:generate counterfeiter -o mocks/update.go -fake-name Update . Update

type Update interface {
	SpecUpdated() bool
	DeployerCMUpdated() bool
	ConsoleCMUpdated() bool
	EnvCMUpdated() bool
	RestartNeeded() bool
}

//go:generate counterfeiter -o mocks/restart_manager.go -fake-name RestartManager . RestartManager

type RestartManager interface {
	ForConfigMapUpdate(instance v1.Object) error
	TriggerIfNeeded(instance restart.Instance) error
	ForRestartAction(instance v1.Object) error
}

type IBPConsole interface {
	PreReconcileChecks(instance *current.IBPConsole) (bool, error)
	CheckStates(instance *current.IBPConsole, update bool) error
	ReconcileManagers(instance *current.IBPConsole, update bool) error
	Reconcile(instance *current.IBPConsole, update Update) (common.Result, error)
}

var _ IBPConsole = &Console{}

type Console struct {
	Client k8sclient.Client
	Scheme *runtime.Scheme
	Config *config.Config

	DeploymentManager        resources.Manager
	ServiceManager           resources.Manager
	DeployerServiceManager   resources.Manager
	PVCManager               resources.Manager
	ConfigMapManager         resources.Manager
	ConsoleConfigMapManager  resources.Manager
	DeployerConfigMapManager resources.Manager
	RoleManager              resources.Manager
	RoleBindingManager       resources.Manager
	ServiceAccountManager    resources.Manager

	Override Override

	Restart RestartManager
}

func New(client k8sclient.Client, scheme *runtime.Scheme, config *config.Config, o Override) *Console {
	console := &Console{
		Client:   client,
		Scheme:   scheme,
		Config:   config,
		Override: o,
		Restart:  restart.New(client, config.Operator.Restart.WaitTime.Get(), config.Operator.Restart.Timeout.Get()),
	}

	console.CreateManagers()
	return console
}

func (c *Console) CreateManagers() {
	options := map[string]interface{}{}

	options["userid"] = util.GenerateRandomString(10)
	options["password"] = util.GenerateRandomString(10)

	consoleConfig := c.Config.ConsoleInitConfig

	override := c.Override
	resourceManager := resourcemanager.New(c.Client, c.Scheme)
	c.DeploymentManager = resourceManager.CreateDeploymentManager("", override.Deployment, c.GetLabels, consoleConfig.DeploymentFile)
	c.ServiceManager = resourceManager.CreateServiceManager("", override.Service, c.GetLabels, consoleConfig.ServiceFile)
	c.DeployerServiceManager = resourceManager.CreateServiceManager("", override.Service, c.GetLabels, consoleConfig.DeployerServiceFile)
	c.PVCManager = resourceManager.CreatePVCManager("", override.PVC, c.GetLabels, consoleConfig.PVCFile)
	c.ConfigMapManager = resourceManager.CreateConfigMapManager("", override.CM, c.GetLabels, consoleConfig.CMFile, nil)
	c.ConsoleConfigMapManager = resourceManager.CreateConfigMapManager("console", override.ConsoleCM, c.GetLabels, consoleConfig.ConsoleCMFile, options)
	c.DeployerConfigMapManager = resourceManager.CreateConfigMapManager("deployer", override.DeployerCM, c.GetLabels, consoleConfig.DeployerCMFile, options)
	c.RoleManager = resourceManager.CreateRoleManager("", nil, c.GetLabels, consoleConfig.RoleFile)
	c.RoleBindingManager = resourceManager.CreateRoleBindingManager("", nil, c.GetLabels, consoleConfig.RoleBindingFile)
	c.ServiceAccountManager = resourceManager.CreateServiceAccountManager("", nil, c.GetLabels, consoleConfig.ServiceAccountFile)
}

func (c *Console) PreReconcileChecks(instance *current.IBPConsole) (bool, error) {
	var maxNameLength *int
	if instance.Spec.ConfigOverride != nil {
		maxNameLength = instance.Spec.ConfigOverride.MaxNameLength
	}
	err := util.ValidationChecks(instance.TypeMeta, instance.ObjectMeta, "IBPConsole", maxNameLength)
	if err != nil {
		return false, err
	}

	// check if all required values are passed
	err = c.ValidateSpec(instance)
	if err != nil {
		return false, err
	}

	zoneUpdated, err := c.SelectZone(instance)
	if err != nil {
		return false, err
	}

	regionUpdated, err := c.SelectRegion(instance)
	if err != nil {
		return false, err
	}

	passwordUpdated, err := c.CreatePasswordSecretIfRequired(instance)
	if err != nil {
		return false, err
	}

	kubeconfigUpdated, err := c.CreateKubernetesSecretIfRequired(instance)
	if err != nil {
		return false, err
	}

	connectionStringUpdated := c.CreateCouchdbCredentials(instance)

	update := passwordUpdated || zoneUpdated || regionUpdated || kubeconfigUpdated || connectionStringUpdated

	if update {
		log.Info(fmt.Sprintf("passwordUpdated %t, zoneUpdated %t, regionUpdated %t, kubeconfigUpdated %t, connectionstringUpdated %t",
			passwordUpdated, zoneUpdated, regionUpdated, kubeconfigUpdated, connectionStringUpdated))
	}

	return update, err
}

func (c *Console) Reconcile(instance *current.IBPConsole, update Update) (common.Result, error) {
	var err error

	versionSet, err := c.SetVersion(instance)
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

	instanceUpdated, err := c.PreReconcileChecks(instance)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed pre reconcile checks")
	}

	if instanceUpdated {
		log.Info("Updating instance after pre reconcile checks")
		err := c.Client.Patch(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPConsole{},
				Strategy: client.MergeFrom,
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

	log.Info("Reconciling managers ...")
	err = c.ReconcileManagers(instance, update.SpecUpdated())
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to reconcile managers")
	}

	err = c.CheckStates(instance, update.SpecUpdated())
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check and restore state")
	}

	err = c.CheckForConfigMapUpdates(instance, update)
	if err != nil {
		return common.Result{}, errors.Wrap(err, "failed to check for config map updates")
	}

	err = c.HandleActions(instance, update)
	if err != nil {
		return common.Result{}, err
	}

	if err := c.HandleRestart(instance, update); err != nil {
		return common.Result{}, err
	}

	return common.Result{}, nil
}

func (c *Console) SetVersion(instance *current.IBPConsole) (bool, error) {
	if instance.Status.Version == "" || !version.String(instance.Status.Version).Equal(version.Operator) {
		log.Info("Version of Operator: ", "version", version.Operator)
		log.Info("Version of CR: ", "version", instance.Status.Version)
		log.Info(fmt.Sprintf("Setting '%s' to version '%s'", instance.Name, version.Operator))

		instance.Status.Version = version.Operator
		err := c.Client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPConsole{},
				Strategy: client.MergeFrom,
			},
		})
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (c *Console) ReconcileManagers(instance *current.IBPConsole, update bool) error {
	var err error

	if strings.Contains(instance.Spec.ConnectionString, "localhost") || instance.Spec.ConnectionString == "" {
		err = c.PVCManager.Reconcile(instance, update)
		if err != nil {
			return errors.Wrap(err, "failed PVC reconciliation")
		}
	}

	err = c.ServiceManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Service reconciliation")
	}

	if instance.Spec.FeatureFlags != nil && instance.Spec.FeatureFlags.DevMode {
		c.DeployerServiceManager.SetCustomName(instance.GetName() + "-deployer-" + instance.Namespace)
		err = c.DeployerServiceManager.Reconcile(instance, update)
		if err != nil {
			return errors.Wrap(err, "failed Deployer Service reconciliation")
		}
	}

	err = c.ReconcileRBAC(instance)
	if err != nil {
		return errors.Wrap(err, "failed RBAC reconciliation")
	}

	err = c.ConfigMapManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed ConfigMap reconciliation")
	}

	err = c.DeployerConfigMapManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Deployer ConfigMap reconciliation")
	}

	err = c.ConsoleConfigMapManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Console ConfigMap reconciliation")
	}

	err = c.DeploymentManager.Reconcile(instance, update)
	if err != nil {
		return errors.Wrap(err, "failed Deployment reconciliation")
	}

	return nil
}

func (c *Console) ReconcileRBAC(instance *current.IBPConsole) error {
	var err error

	err = c.RoleManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	err = c.RoleBindingManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	err = c.ServiceAccountManager.Reconcile(instance, false)
	if err != nil {
		return err
	}

	return nil
}

func (c *Console) CheckStates(instance *current.IBPConsole, update bool) error {
	// Don't need to check state if the state is being updated via CR. State needs
	// to be checked if operator detects changes to a resources that was not triggered
	// via CR.
	if c.DeploymentManager.Exists(instance) {
		err := c.DeploymentManager.CheckState(instance)
		if err != nil {
			log.Info(fmt.Sprintf("unexpected state found for deployment, restoring state: %s", err.Error()))
			err = c.DeploymentManager.RestoreState(instance)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Console) SelectZone(instance *current.IBPConsole) (bool, error) {
	if instance.Spec.Zone == "select" {
		zone := util.GetZone(c.Client)
		instance.Spec.Zone = zone
		return true, nil
	}
	if instance.Spec.Zone != "" {
		err := util.ValidateZone(c.Client, instance.Spec.Zone)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (c *Console) SelectRegion(instance *current.IBPConsole) (bool, error) {
	if instance.Spec.Region == "select" {
		region := util.GetRegion(c.Client)
		instance.Spec.Region = region
		return true, nil
	}
	if instance.Spec.Region != "" {
		err := util.ValidateRegion(c.Client, instance.Spec.Region)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

func (c *Console) CreatePasswordSecretIfRequired(instance *current.IBPConsole) (bool, error) {
	namespace := instance.Namespace
	passwordSecretName := instance.Spec.PasswordSecretName
	password := instance.Spec.Password

	authscheme := instance.Spec.AuthScheme

	// if password is blank and passwordSecret is set
	if password == "" && passwordSecretName != "" {
		userSecret := &corev1.Secret{}
		err := c.Client.Get(context.TODO(), types.NamespacedName{Name: passwordSecretName, Namespace: namespace}, userSecret)
		if err != nil {
			return false, errors.Wrap(err, "failed to get provided console password secret, password is blank & secret is set")
		}
		return false, nil
	}

	if passwordSecretName == "" && authscheme == "ibmid" {
		password = "unused"
	}

	if password == "" && passwordSecretName == "" {
		return false, errors.New("both password and password secret are NOT set")
	}

	if passwordSecretName == "" && password != "" {
		passwordSecretName = instance.Name + "-console-pw"
		err := c.CreateUserSecret(instance, passwordSecretName, password)
		if err != nil {
			return false, errors.Wrap(err, "failed to create user secret")
		} else {
			instance.Spec.Password = ""
			instance.Spec.PasswordSecretName = passwordSecretName
			return true, nil
		}
	}

	return false, nil

}

func (c *Console) CreateKubernetesSecretIfRequired(instance *current.IBPConsole) (bool, error) {
	namespace := instance.Namespace
	kubeconfigsecretname := instance.Spec.KubeconfigSecretName
	kubeconfig := instance.Spec.Kubeconfig

	// if password is blank and passwordSecret is set
	if kubeconfig == nil && kubeconfigsecretname != "" {
		kubeconfigSecret := &corev1.Secret{}
		err := c.Client.Get(context.TODO(), types.NamespacedName{Name: kubeconfigsecretname, Namespace: namespace}, kubeconfigSecret)
		if err != nil {
			return false, errors.Wrap(err, "failed to get kubeconifg secret")
		}
		return false, nil
	}

	if kubeconfigsecretname == "" && kubeconfig != nil && string(*kubeconfig) != "" {
		kubeconfigsecretname = instance.Name + "-kubeconfig"
		err := c.CreateKubeconfigSecret(instance, kubeconfigsecretname, kubeconfig)
		if err != nil {
			return false, errors.Wrap(err, "failed to create kubeconfig secret")
		} else {
			empty := make([]byte, 0)
			instance.Spec.Kubeconfig = &empty
			instance.Spec.KubeconfigSecretName = kubeconfigsecretname
			return true, nil
		}
	}

	if kubeconfig != nil && string(*kubeconfig) != "" && kubeconfigsecretname != "" {
		return false, errors.New("both kubeconfig and kubeconfig secret name are set")
	}

	return false, nil
}

func (c *Console) CreateKubeconfigSecret(instance *current.IBPConsole, kubeocnfigSecretName string, kubeconfig *[]byte) error {
	kubeconfigSecret := &corev1.Secret{}
	kubeconfigSecret.Name = kubeocnfigSecretName
	kubeconfigSecret.Namespace = instance.Namespace
	kubeconfigSecret.Labels = c.GetLabels(instance)

	kubeconfigSecret.Data = map[string][]byte{}
	kubeconfigSecret.Data["kubeconfig.yaml"] = *kubeconfig

	err := c.Client.Create(context.TODO(), kubeconfigSecret, k8sclient.CreateOption{Owner: instance, Scheme: c.Scheme})
	if err != nil {
		return err
	}

	return nil
}

func (c *Console) CreateUserSecret(instance *current.IBPConsole, passwordSecretName, password string) error {
	userSecret := &corev1.Secret{}
	userSecret.Name = passwordSecretName
	userSecret.Namespace = instance.Namespace
	userSecret.Labels = c.GetLabels(instance)

	userSecret.Data = map[string][]byte{}
	userSecret.Data["password"] = []byte(password)

	err := c.Client.Create(context.TODO(), userSecret, k8sclient.CreateOption{Owner: instance, Scheme: c.Scheme})
	if err != nil {
		return err
	}

	return nil
}

func (c *Console) ValidateSpec(instance *current.IBPConsole) error {
	if instance.Spec.NetworkInfo == nil {
		return errors.New("network information not provided")
	}

	if instance.Spec.NetworkInfo.Domain == "" {
		return errors.New("domain not provided in network information")
	}

	if !instance.Spec.License.Accept {
		return errors.New("user must accept license before continuing")
	}

	if instance.Spec.ServiceAccountName == "" {
		return errors.New("Service account name not provided")
	}

	if instance.Spec.Email == "" {
		return errors.New("email not provided")
	}

	if instance.Spec.AuthScheme != "ibmid" && instance.Spec.Password == "" && instance.Spec.PasswordSecretName == "" {
		return errors.New("password and passwordSecretName both not provided, at least one expected")
	}

	if instance.Spec.ImagePullSecrets == nil || len(instance.Spec.ImagePullSecrets) == 0 {
		return errors.New("imagepullsecrets required")
	}

	if instance.Spec.RegistryURL != "" && !strings.HasSuffix(instance.Spec.RegistryURL, "/") {
		instance.Spec.RegistryURL = instance.Spec.RegistryURL + "/"
	}

	return nil
}

func (c *Console) GetLabels(instance v1.Object) map[string]string {
	label := os.Getenv("OPERATOR_LABEL_PREFIX")
	if label == "" {
		label = "fabric"
	}

	return map[string]string{
		"app":                          instance.GetName(),
		"creator":                      label,
		"release":                      "operator",
		"helm.sh/chart":                "ibm-" + label,
		"app.kubernetes.io/name":       label,
		"app.kubernetes.io/instance":   label + "console",
		"app.kubernetes.io/managed-by": label + "-operator",
	}
}

func (c *Console) HandleActions(instance *current.IBPConsole, update Update) error {
	orig := instance.DeepCopy()

	if update.RestartNeeded() {
		if err := c.Restart.ForRestartAction(instance); err != nil {
			return errors.Wrap(err, "failed to restart console pods")
		}
		instance.ResetRestart()
	}

	if err := c.Client.Patch(context.TODO(), instance, client.MergeFrom(orig)); err != nil {
		return errors.Wrap(err, "failed to reset action flag")
	}

	return nil
}

func (c *Console) CreateCouchdbCredentials(instance *current.IBPConsole) bool {
	if instance.Spec.ConnectionString != "" && instance.Spec.ConnectionString != "http://localhost:5984" {
		return false
	}

	couchdbUser := util.GenerateRandomString(32)
	couchdbPassword := util.GenerateRandomString(32)
	connectionString := fmt.Sprintf("http://%s:%s@localhost:5984", couchdbUser, couchdbPassword)
	instance.Spec.ConnectionString = connectionString
	// TODO save deployer docs for SW?
	// instance.Spec.Deployer.ConnectionString = connectionString

	return true
}

func (c *Console) CheckForConfigMapUpdates(instance *current.IBPConsole, update Update) error {
	if update.DeployerCMUpdated() || update.ConsoleCMUpdated() || update.EnvCMUpdated() {
		err := c.Restart.ForConfigMapUpdate(instance)
		if err != nil {
			return errors.Wrap(err, "failed to update restart config")
		}
	}

	return nil
}

func (c *Console) HandleRestart(instance *current.IBPConsole, update Update) error {
	// If restart is disabled for components, can return immediately
	if c.Config.Operator.Restart.Disable.Components {
		return nil
	}

	err := c.Restart.TriggerIfNeeded(instance)
	if err != nil {
		return errors.Wrap(err, "failed to restart deployment")
	}

	return nil
}

func (c *Console) NetworkPolicyReconcile(instance *current.IBPConsole) error {
	if c.Config.Operator.Console.ApplyNetworkPolicy == "" || c.Config.Operator.Console.ApplyNetworkPolicy == "false" {
		return nil
	}

	log.Info("IBPOPERATOR_CONSOLE_APPLYNETWORKPOLICY set applying network policy")
	err := c.CreateNetworkPolicyIfNotExists(instance, c.Config.ConsoleInitConfig.NetworkPolicyIngressFile, instance.GetName()+"-ingress")
	if err != nil {
		log.Error(err, "Cannot install ingress network policy")
	}

	err = c.CreateNetworkPolicyIfNotExists(instance, c.Config.ConsoleInitConfig.NetworkPolicyDenyAllFile, instance.GetName()+"-denyall")
	if err != nil {
		log.Error(err, "Cannot install denyall network policy")
	}

	return nil
}

func (c *Console) CreateNetworkPolicyIfNotExists(instance *current.IBPConsole, filename string, policyname string) error {
	policy, err := util.GetNetworkPolicyFromFile(filename)
	if err != nil {
		return err
	}

	policy.Namespace = instance.Namespace
	policy.Name = policyname
	policy.Spec.PodSelector.MatchLabels = c.GetLabelsForNetworkPolicy(instance)

	newPolicy := policy.DeepCopy()
	err = c.Client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.GetName()}, newPolicy)
	if err != nil {
		if k8sruntime.IgnoreNotFound(err) == nil {
			log.Info("network policy not found, applying now")
			err1 := c.Client.Create(context.TODO(), policy, k8sclient.CreateOption{Owner: instance, Scheme: c.Scheme})
			if err1 != nil {
				log.Error(err1, "Error applying network policy")
			}
		} else {
			log.Error(err, "Error getting network policy")
			return nil
		}
	} else {
		log.Info("network policy found, not applying")
		return nil
	}
	return nil
}

func (c *Console) GetLabelsForNetworkPolicy(instance v1.Object) map[string]string {
	label := os.Getenv("OPERATOR_LABEL_PREFIX")
	if label == "" {
		label = "fabric"
	}

	return map[string]string{
		"app.kubernetes.io/name": label,
	}
}
