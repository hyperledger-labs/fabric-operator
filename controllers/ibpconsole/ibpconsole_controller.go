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

package ibpconsole

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commoncontroller "github.com/IBM-Blockchain/fabric-operator/controllers/common"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/global"
	"github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
	baseconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	k8sconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/console"
	openshiftconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/console"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_ibpconsole")

// Add creates a new IBPPeer Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, config *config.Config) error {
	r, err := newReconciler(mgr, config)
	if err != nil {
		return err
	}
	return add(mgr, r)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, cfg *config.Config) (*ReconcileIBPConsole, error) {
	client := k8sclient.New(mgr.GetClient(), &global.ConfigSetter{Config: cfg.Operator.Globals})
	scheme := mgr.GetScheme()

	ibpconsole := &ReconcileIBPConsole{
		client: client,
		scheme: scheme,
		Config: cfg,
	}

	switch cfg.Offering {
	case offering.K8S:
		ibpconsole.Offering = k8sconsole.New(client, scheme, cfg)
	case offering.OPENSHIFT:
		ibpconsole.Offering = openshiftconsole.New(client, scheme, cfg)
	}

	return ibpconsole, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileIBPConsole) error {
	// Create a new controller
	predicateFuncs := predicate.Funcs{
		CreateFunc: r.CreateFunc,
		UpdateFunc: r.UpdateFunc,
	}

	c, err := controller.New("ibpconsole-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IBPConsole
	err = c.Watch(&source.Kind{Type: &current.IBPConsole{}}, &handler.EnqueueRequestForObject{}, predicateFuncs)
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner IBPPeer
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &current.IBPConsole{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIBPConsole{}

//go:generate counterfeiter -o mocks/consolereconcile.go -fake-name ConsoleReconcile . consoleReconcile

type consoleReconcile interface {
	Reconcile(*current.IBPConsole, baseconsole.Update) (common.Result, error)
}

// ReconcileIBPConsole reconciles a IBPConsole object
type ReconcileIBPConsole struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client k8sclient.Client
	scheme *runtime.Scheme

	Offering consoleReconcile
	Config   *config.Config

	update Update
}

// Reconcile reads that state of the cluster for a IBPConsole object and makes changes based on the state read
// and what is in the IBPConsole.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIBPConsole) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var err error

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info(fmt.Sprintf("Reconciling IBPConsole with update values of [ %+v ]", r.update.GetUpdateStackWithTrues()))

	// Fetch the IBPConsole instance
	instance := &current.IBPConsole{}
	err = r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	result, err := r.Offering.Reconcile(instance, &r.update)
	setStatusErr := r.SetStatus(instance, err)
	if setStatusErr != nil {
		return reconcile.Result{}, operatorerrors.IsBreakingError(setStatusErr, "failed to update status", log)
	}

	if err != nil {
		return reconcile.Result{}, operatorerrors.IsBreakingError(errors.Wrapf(err, "Console instance '%s' encountered error", instance.GetName()), "stopping reconcile loop", log)
	}

	reqLogger.Info(fmt.Sprintf("Finished reconciling IBPConsole '%s' with update values of [ %+v ]", instance.GetName(), r.update.GetUpdateStackWithTrues()))
	return result.Result, nil
}

func (r *ReconcileIBPConsole) SetStatus(instance *current.IBPConsole, reconcileErr error) error {
	err := r.SaveSpecState(instance)
	if err != nil {
		return errors.Wrap(err, "failed to save spec state")
	}

	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.GetName(), Namespace: instance.GetNamespace()}, instance)
	if err != nil {
		return err
	}

	status := instance.Status.CRStatus

	if reconcileErr != nil {
		status.Type = current.Error
		status.Status = current.True
		status.Reason = "errorOccurredDuringReconcile"
		status.Message = reconcileErr.Error()
		status.LastHeartbeatTime = time.Now().String()
		status.ErrorCode = operatorerrors.GetErrorCode(reconcileErr)

		instance.Status = current.IBPConsoleStatus{
			CRStatus: status,
		}

		log.Info(fmt.Sprintf("Updating status of IBPConsole custom resource to %s phase", instance.Status.Type))
		err := r.client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    2,
				Into:     &current.IBPConsole{},
				Strategy: client.MergeFrom,
			},
		})
		if err != nil {
			return err
		}

		return nil
	}

	status.Versions.Reconciled = instance.Spec.Version

	running, err := r.GetPodStatus(instance)
	if err != nil {
		return err
	}

	if running {
		if instance.Status.Type == current.Deployed {
			return nil
		}
		status.Type = current.Deployed
		status.Status = current.True
		status.Reason = "allPodsRunning"
	} else {
		if instance.Status.Type == current.Deploying {
			return nil
		}
		status.Type = current.Deploying
		status.Status = current.True
		status.Reason = "waitingForPods"
	}

	instance.Status = current.IBPConsoleStatus{
		CRStatus: status,
	}
	instance.Status.LastHeartbeatTime = time.Now().String()
	log.Info(fmt.Sprintf("Updating status of IBPConsole custom resource to %s phase", instance.Status.Type))
	err = r.client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
		Resilient: &k8sclient.ResilientPatch{
			Retry:    2,
			Into:     &current.IBPConsole{},
			Strategy: client.MergeFrom,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileIBPConsole) GetPodStatus(instance *current.IBPConsole) (bool, error) {
	labelSelector, err := labels.Parse(fmt.Sprintf("app=%s", instance.Name))
	if err != nil {
		return false, errors.Wrap(err, "failed to parse label selector for app name")
	}

	listOptions := &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     instance.Namespace,
	}

	podList := &corev1.PodList{}
	err = r.client.List(context.TODO(), podList, listOptions)
	if err != nil {
		return false, err
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
	}

	return true, nil
}

func (r *ReconcileIBPConsole) getIgnoreDiffs() []string {
	return []string{
		`Template\.Spec\.Containers\.slice\[\d\]\.Resources\.Requests\.map\[memory\].s`,
	}
}

func (r *ReconcileIBPConsole) getLabels(instance v1.Object) map[string]string {
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

func (r *ReconcileIBPConsole) getSelectorLabels(instance v1.Object) map[string]string {
	return map[string]string{
		"app": instance.GetName(),
	}
}

func (r *ReconcileIBPConsole) CreateFunc(e event.CreateEvent) bool {
	r.update = Update{}

	console := e.Object.(*current.IBPConsole)
	if console.Status.HasType() {
		cm, err := r.GetSpecState(console)
		if err != nil {
			log.Info(fmt.Sprintf("Failed getting saved console spec '%s', can't perform update checks, triggering reconcile: %s", console.GetName(), err.Error()))
			return true
		}

		specBytes := cm.BinaryData["spec"]
		savedConsole := &current.IBPConsole{}

		err = yaml.Unmarshal(specBytes, &savedConsole.Spec)
		if err != nil {
			log.Info(fmt.Sprintf("Unmarshal failed for saved console spec '%s', can't perform update checks, triggering reconcile: %s", console.GetName(), err.Error()))
			return true
		}

		if !reflect.DeepEqual(console.Spec, savedConsole.Spec) {
			log.Info(fmt.Sprintf("IBPConsole '%s' spec was updated while operator was down, triggering reconcile", console.GetName()))
			r.update.specUpdated = true

			if r.DeployerCMUpdated(console.Spec, savedConsole.Spec) {
				r.update.deployerCMUpdated = true
			}
			if r.ConsoleCMUpdated(console.Spec, savedConsole.Spec) {
				r.update.consoleCMUpdated = true
			}
			if r.EnvCMUpdated(console.Spec, savedConsole.Spec) {
				r.update.envCMUpdated = true
			}

			return true
		}

		// Don't trigger reconcile if spec was not updated during operator restart
		return false
	}

	// If creating resource for the first time, check that a unique name is provided
	err := commoncontroller.ValidateCRName(r.client, console.Name, console.Namespace, commoncontroller.IBPCONSOLE)
	if err != nil {
		log.Error(err, "failed to validate console name")
		operror := operatorerrors.Wrap(err, operatorerrors.InvalidCustomResourceCreateRequest, "failed to validate ibpconsole name")
		err = r.SetStatus(console, operror)
		if err != nil {
			log.Error(err, "failed to set status to error", "console.name", console.Name, "error", "InvalidCustomResourceCreateRequest")
		}

		return false
	}

	return true
}

func (r *ReconcileIBPConsole) UpdateFunc(e event.UpdateEvent) bool {
	r.update = Update{}

	oldConsole := e.ObjectOld.(*current.IBPConsole)
	newConsole := e.ObjectNew.(*current.IBPConsole)

	if util.CheckIfZoneOrRegionUpdated(oldConsole.Spec.Zone, newConsole.Spec.Zone) {
		log.Error(errors.New("Zone update is not allowed"), "invalid spec update")
		return false
	}

	if util.CheckIfZoneOrRegionUpdated(oldConsole.Spec.Region, newConsole.Spec.Region) {
		log.Error(errors.New("Region update is not allowed"), "invalid spec update")
		return false
	}

	if reflect.DeepEqual(oldConsole.Spec, newConsole.Spec) {
		return false
	}

	log.Info(fmt.Sprintf("Spec update detected on IBPConsole custom resource: %s", oldConsole.Name))
	r.update.specUpdated = true

	if newConsole.Spec.Action.Restart {
		r.update.restartNeeded = true
	}

	return true
}

func (r *ReconcileIBPConsole) SaveSpecState(instance *current.IBPConsole) error {
	data, err := yaml.Marshal(instance.Spec)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("%s-spec", instance.GetName()),
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
		},
		BinaryData: map[string][]byte{
			"spec": data,
		},
	}

	err = r.client.CreateOrUpdate(context.TODO(), cm, controllerclient.CreateOrUpdateOption{Owner: instance, Scheme: r.scheme})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileIBPConsole) GetSpecState(instance *current.IBPConsole) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-spec", instance.GetName()),
		Namespace: instance.GetNamespace(),
	}

	err := r.client.Get(context.TODO(), nn, cm)
	if err != nil {
		return nil, err
	}

	return cm, nil
}

func (r *ReconcileIBPConsole) DeployerCMUpdated(old, new current.IBPConsoleSpec) bool {
	if !reflect.DeepEqual(old.ImagePullSecrets, new.ImagePullSecrets) {
		return true
	}
	if !reflect.DeepEqual(old.Deployer, new.Deployer) {
		return true
	}
	if old.NetworkInfo.Domain != new.NetworkInfo.Domain {
		return true
	}
	if old.RegistryURL != new.RegistryURL {
		return true
	}
	if !reflect.DeepEqual(old.Arch, new.Arch) {
		return true
	}
	if !reflect.DeepEqual(old.Versions, new.Versions) {
		return true
	}
	// Uncomment if MustGather changes are ported into release 2.5.2
	// if old.Images.MustgatherImage != new.Images.MustgatherImage {
	// 	return true
	// }
	// if old.Images.MustgatherTag != new.Images.MustgatherTag {
	// 	return true
	// }
	if !reflect.DeepEqual(old.Storage, new.Storage) {
		return true
	}
	if !reflect.DeepEqual(old.CRN, new.CRN) {
		return true
	}

	oldOverrides, err := old.GetOverridesDeployer()
	if err != nil {
		return false
	}
	newOverrides, err := new.GetOverridesDeployer()
	if err != nil {
		return false
	}
	if !reflect.DeepEqual(oldOverrides, newOverrides) {
		return true
	}

	return false
}

func (r *ReconcileIBPConsole) ConsoleCMUpdated(old, new current.IBPConsoleSpec) bool {
	if !reflect.DeepEqual(old.IBMID, new.IBMID) {
		return true
	}
	if old.IAMApiKey != new.IAMApiKey {
		return true
	}
	if old.SegmentWriteKey != new.SegmentWriteKey {
		return true
	}
	if old.Email != new.Email {
		return true
	}
	if old.AuthScheme != new.AuthScheme {
		return true
	}
	if old.ConfigtxlatorURL != new.ConfigtxlatorURL {
		return true
	}
	if old.DeployerURL != new.DeployerURL {
		return true
	}
	if old.DeployerTimeout != new.DeployerTimeout {
		return true
	}
	if old.Components != new.Components {
		return true
	}
	if old.Sessions != new.Sessions {
		return true
	}
	if old.System != new.System {
		return true
	}
	if old.SystemChannel != new.SystemChannel {
		return true
	}
	if !reflect.DeepEqual(old.Proxying, new.Proxying) {
		return true
	}
	if !reflect.DeepEqual(old.FeatureFlags, new.FeatureFlags) {
		return true
	}
	if !reflect.DeepEqual(old.ClusterData, new.ClusterData) {
		return true
	}
	if !reflect.DeepEqual(old.CRN, new.CRN) {
		return true
	}

	oldOverrides, err := old.GetOverridesConsole()
	if err != nil {
		return false
	}
	newOverrides, err := new.GetOverridesConsole()
	if err != nil {
		return false
	}
	if !reflect.DeepEqual(oldOverrides, newOverrides) {
		return true
	}

	return false
}

func (r *ReconcileIBPConsole) EnvCMUpdated(old, new current.IBPConsoleSpec) bool {
	if old.ConnectionString != new.ConnectionString {
		return true
	}
	if old.System != new.System {
		return true
	}
	if old.TLSSecretName != new.TLSSecretName {
		return true
	}

	return false
}

func (r *ReconcileIBPConsole) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&current.IBPConsole{}).
		Complete(r)
}

type Update struct {
	specUpdated       bool
	restartNeeded     bool
	deployerCMUpdated bool
	consoleCMUpdated  bool
	envCMUpdated      bool
}

func (u *Update) SpecUpdated() bool {
	return u.specUpdated
}

func (u *Update) RestartNeeded() bool {
	return u.restartNeeded
}

func (u *Update) DeployerCMUpdated() bool {
	return u.deployerCMUpdated
}

func (u *Update) ConsoleCMUpdated() bool {
	return u.consoleCMUpdated
}

func (u *Update) EnvCMUpdated() bool {
	return u.envCMUpdated
}

func (u *Update) GetUpdateStackWithTrues() string {
	stack := ""

	if u.specUpdated {
		stack += "specUpdated "
	}
	if u.restartNeeded {
		stack += "restartNeeded "
	}
	if u.deployerCMUpdated {
		stack += "deployerCMUpdated "
	}
	if u.consoleCMUpdated {
		stack += "consoleCMUpdated "
	}
	if u.envCMUpdated {
		stack += "envCMUpdated "
	}

	if len(stack) == 0 {
		stack = "emptystack "
	}

	return stack
}
