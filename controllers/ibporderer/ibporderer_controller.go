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

package ibporderer

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commoncontroller "github.com/IBM-Blockchain/fabric-operator/controllers/common"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/global"
	orderer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v1"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	k8sorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/orderer"
	openshiftorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/orderer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/staggerrestarts"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	yaml "sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const (
	KIND = "IBPOrderer"
)

var log = logf.Log.WithName("controller_ibporderer")

// Add creates a new IBPOrderer Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, config *config.Config) error {
	r, err := newReconciler(mgr, config)
	if err != nil {
		return err
	}
	return add(mgr, r)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, cfg *config.Config) (*ReconcileIBPOrderer, error) {
	client := k8sclient.New(mgr.GetClient(), &global.ConfigSetter{Config: cfg.Operator.Globals})
	scheme := mgr.GetScheme()

	ibporderer := &ReconcileIBPOrderer{
		client:         client,
		scheme:         scheme,
		Config:         cfg,
		update:         map[string][]Update{},
		mutex:          &sync.Mutex{},
		RestartService: staggerrestarts.New(client, cfg.Operator.Restart.Timeout.Get()),
	}

	switch cfg.Offering {
	case offering.K8S:
		ibporderer.Offering = k8sorderer.New(client, scheme, cfg)
	case offering.OPENSHIFT:
		ibporderer.Offering = openshiftorderer.New(client, scheme, cfg)
	}

	return ibporderer, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileIBPOrderer) error {
	// Create a new controller
	predicateFuncs := predicate.Funcs{
		CreateFunc: r.CreateFunc,
		UpdateFunc: r.UpdateFunc,
		DeleteFunc: r.DeleteFunc,
	}

	c, err := controller.New("ibporderer-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IBPOrderer
	err = c.Watch(&source.Kind{Type: &current.IBPOrderer{}}, &handler.EnqueueRequestForObject{}, predicateFuncs)
	if err != nil {
		return err
	}

	// Watch for changes to config maps (Create and Update funcs handle only watching for restart config map)
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, predicateFuncs)
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner IBPOrderer
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &current.IBPOrderer{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to tertiary resource Secrets and requeue the owner IBPOrderer
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &current.IBPOrderer{},
	}, predicateFuncs)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIBPOrderer{}

//go:generate counterfeiter -o mocks/ordererreconcile.go -fake-name OrdererReconcile . ordererReconcile

type ordererReconcile interface {
	Reconcile(*current.IBPOrderer, baseorderer.Update) (common.Result, error)
}

// ReconcileIBPOrderer reconciles a IBPOrderer object
type ReconcileIBPOrderer struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client k8sclient.Client
	scheme *runtime.Scheme

	Offering       ordererReconcile
	Config         *config.Config
	RestartService *staggerrestarts.StaggerRestartsService

	update map[string][]Update
	mutex  *sync.Mutex
}

// Reconcile reads that state of the cluster for a IBPOrderer object and makes changes based on the state read
// and what is in the IBPOrderer.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIBPOrderer) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var err error

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// If orderer-restart-config configmap is the object being reconciled, reconcile the
	// restart configmap.
	if request.Name == "orderer-restart-config" {
		requeue, err := r.ReconcileRestart(request.Namespace)
		// Error reconciling restart - requeue the request.
		if err != nil {
			return reconcile.Result{}, err
		}
		// Restart reconciled, requeue request if required.
		return reconcile.Result{
			Requeue: requeue,
		}, nil
	}

	reqLogger.Info("Reconciling IBPOrderer")

	// Fetch the IBPOrderer instance
	instance := &current.IBPOrderer{}
	err = r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, operatorerrors.IsBreakingError(err, "failed to reconcile restart", log)
	}

	var maxNameLength *int
	if instance.Spec.ConfigOverride != nil {
		override := &orderer.OrdererOverrides{}
		err := json.Unmarshal(instance.Spec.ConfigOverride.Raw, override)
		if err != nil {
			return reconcile.Result{}, err
		}
		maxNameLength = override.MaxNameLength
	}

	err = util.ValidationChecks(instance.TypeMeta, instance.ObjectMeta, "IBPOrderer", maxNameLength)
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.NodeNumber == nil {
		// If version is nil, then this is a v210 instance and the reconcile
		// loop needs to be triggered to allow for instance migration
		if instance.Status.Version != "" {
			if instance.Status.Type == current.Deployed || instance.Status.Type == current.Warning {
				// This is cluster's update, we don't want to reconcile.
				// It should only be status update
				log.Info(fmt.Sprintf("Update detected on %s cluster spec '%s', not supported", instance.Status.Type, instance.GetName()))
				return reconcile.Result{}, nil
			}
		}
	}

	reqLogger.Info(fmt.Sprintf("Current update stack to process: %+v", GetUpdateStack(r.update)))

	update := r.GetUpdateStatus(instance)
	reqLogger.Info(fmt.Sprintf("Reconciling IBPOrderer '%s' with update values of [ %+v ]", instance.GetName(), update.GetUpdateStackWithTrues()))

	result, err := r.Offering.Reconcile(instance, r.PopUpdate(instance.Name))
	setStatusErr := r.SetStatus(instance, &result, err)
	if setStatusErr != nil {
		return reconcile.Result{}, operatorerrors.IsBreakingError(setStatusErr, "failed to update status", log)
	}

	if err != nil {
		return reconcile.Result{}, operatorerrors.IsBreakingError(errors.Wrapf(err, "Orderer instance '%s' encountered error", instance.GetName()), "stopping reconcile loop", log)
	}

	if result.Requeue {
		r.PushUpdate(instance.Name, *update)
	}

	reqLogger.Info(fmt.Sprintf("Finished reconciling IBPOrderer '%s' with update values of [ %+v ]", instance.GetName(), update.GetUpdateStackWithTrues()))

	// If the stack still has items that require processing, keep reconciling
	// until the stack has been cleared
	_, found := r.update[instance.GetName()]
	if found {
		if len(r.update[instance.GetName()]) > 0 {
			return reconcile.Result{
				Requeue: true,
			}, nil
		}
	}

	return result.Result, nil
}

func (r *ReconcileIBPOrderer) SetStatus(instance *current.IBPOrderer, result *common.Result, reconcileErr error) error {
	err := r.SaveSpecState(instance)
	if err != nil {
		return errors.Wrap(err, "failed to save spec state")
	}

	// Hierachy of setting status on orderer node instance
	// 1. If error has occurred update status and return
	// 2. If error has not occurred, get list of pods and determine
	// if pods are all running or still waiting to start. If all pods
	// are running mark status as Deployed otherwise mark status as
	// Deploying, but dont update status yet
	// 3. Check to see if a custom status has been passed. If so,
	// set that status but don't update. However, if OverrideUpdateStatus
	// flag is set to true update the status and return
	// 4. If OverrideUpdateStatus was not set in step 3, determine if genesis
	// secret exists for the instance. If genesis secret does not exit update
	// the status to precreate and return

	// Need to get to ensure we are working with the latest state of the instance
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

		instance.Status = current.IBPOrdererStatus{
			CRStatus: status,
		}

		log.Info(fmt.Sprintf("Updating status of IBPOrderer custom resource (%s) to %s phase", instance.GetName(), instance.Status.Type))
		err := r.client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    2,
				Into:     &current.IBPOrderer{},
				Strategy: client.MergeFrom,
			},
		})
		if err != nil {
			return err
		}

		return nil
	}

	status.Versions.Reconciled = instance.Spec.FabricVersion

	// If this is a parent (cluster spec), then ignore setting status. Status should
	// be set by the child nodes only, and child nodes should update the status of parent
	// according to the statuses of the child nodes. This needs to stay after the check for
	// reconcile error otherwise the CR won't get updated with error on parent CR if there
	// are validation errors on the spec.
	if instance.Spec.NodeNumber == nil {
		return nil
	}

	podStatus, err := r.GetPodStatus(instance)
	if err != nil {
		return err
	}

	numberOfPodsRunning := 0
	for _, status := range podStatus {
		if status.Phase == corev1.PodRunning {
			numberOfPodsRunning++
		}
	}

	// if numberOfPodsRunning == len(podStatus) && len(podStatus) > 0 {
	if len(podStatus) > 0 {
		if len(podStatus) == numberOfPodsRunning {
			status.Type = current.Deployed
			status.Status = current.True
			status.Reason = "allPodsRunning"
			status.Message = "allPodsRunning"
		} else {
			status.Type = current.Deploying
			status.Status = current.True
			status.Reason = "waitingForPods"
			status.Message = "waitingForPods"
		}
	}

	// Check if reconcile loop returned an updated status that differs from exisiting status.
	// If so, set status to the reconcile status.
	if result != nil {
		reconcileStatus := result.Status
		if reconcileStatus != nil {
			if instance.Status.Type != reconcileStatus.Type || instance.Status.Reason != reconcileStatus.Reason || instance.Status.Message != reconcileStatus.Message {
				status.Type = reconcileStatus.Type
				status.Status = current.True
				status.Reason = reconcileStatus.Reason
				status.Message = reconcileStatus.Message
				status.LastHeartbeatTime = time.Now().String()

				if result.OverrideUpdateStatus {
					instance.Status = current.IBPOrdererStatus{
						CRStatus: status,
					}

					log.Info(fmt.Sprintf("Updating status returned by reconcile loop of IBPOrderer custom resource (%s) to %s phase", instance.GetName(), instance.Status.Type))
					err := r.client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
						Resilient: &k8sclient.ResilientPatch{
							Retry:    2,
							Into:     &current.IBPOrderer{},
							Strategy: client.MergeFrom,
						},
					})
					if err != nil {
						return err
					}

					return nil
				}
			} else {
				// If the reconcile loop returned an updated status that is the same as the current instance status, then no status update required.
				// NOTE: This will only occur once the instance has hit Deployed state for the first time and would only switch between Deployed and
				// Warning states.
				log.Info(fmt.Sprintf("Reconcile loop returned a status that is the same as %s's current status (%s), not updating status", reconcileStatus.Type, instance.Name))
				return nil
			}
		}

		// There are cases we want to return before checking for genesis secrets, such as updating the spec with default values
		// during prereconcile checks
		if result.OverrideUpdateStatus {
			return nil
		}
	}

	precreated := false
	if instance.Spec.IsUsingChannelLess() {
		log.Info(fmt.Sprintf("IBPOrderer custom resource (%s) is using channel less mode", instance.GetName()))
		precreated = false
	} else {
		err = r.GetGenesisSecret(instance)
		if err != nil {
			log.Info(fmt.Sprintf("IBPOrderer custom resource (%s) pods are waiting for genesis block, setting status to precreate", instance.GetName()))
			precreated = true
		}
	}

	if precreated {
		status.Type = current.Precreated
		status.Status = current.True
		status.Reason = "waiting for genesis block"
		status.Message = "waiting for genesis block"
	}

	// Only update status if status is different from current status
	if status.Type != "" && (instance.Status.Type != status.Type || instance.Status.Reason != status.Reason || instance.Status.Message != status.Message) {
		status.LastHeartbeatTime = time.Now().String()
		log.Info(fmt.Sprintf("Updating status of IBPOrderer custom resource (%s) from %s to %s phase", instance.GetName(), instance.Status.Type, status.Type))

		instance.Status = current.IBPOrdererStatus{
			CRStatus: status,
		}

		err = r.client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    2,
				Into:     &current.IBPOrderer{},
				Strategy: client.MergeFrom,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileIBPOrderer) SaveSpecState(instance *current.IBPOrderer) error {
	data, err := yaml.Marshal(instance.Spec)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-spec", instance.GetName()),
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
		},
		BinaryData: map[string][]byte{
			"spec": data,
		},
	}

	err = r.client.CreateOrUpdate(context.TODO(), cm, k8sclient.CreateOrUpdateOption{Owner: instance, Scheme: r.scheme})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileIBPOrderer) GetSpecState(instance *current.IBPOrderer) (*corev1.ConfigMap, error) {
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

func (r *ReconcileIBPOrderer) GetPodStatus(instance *current.IBPOrderer) (map[string]corev1.PodStatus, error) {
	statuses := map[string]corev1.PodStatus{}

	labelSelector, err := labels.Parse(fmt.Sprintf("app=%s", instance.GetName()))
	if err != nil {
		return statuses, errors.Wrap(err, "failed to parse label selector for app name")
	}

	listOptions := &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     instance.GetNamespace(),
	}

	podList := &corev1.PodList{}
	err = r.client.List(context.TODO(), podList, listOptions)
	if err != nil {
		return statuses, err
	}

	for _, pod := range podList.Items {
		statuses[pod.Name] = pod.Status
	}

	return statuses, nil
}

func (r *ReconcileIBPOrderer) GetGenesisSecret(instance *current.IBPOrderer) error {
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-genesis", instance.GetName()),
		Namespace: instance.GetNamespace(),
	}
	err := r.client.Get(context.TODO(), nn, &corev1.Secret{})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileIBPOrderer) CreateFunc(e event.CreateEvent) bool {
	update := Update{}

	switch e.Object.(type) {
	case *current.IBPOrderer:
		orderer := e.Object.(*current.IBPOrderer)
		log.Info(fmt.Sprintf("Create event detected for orderer '%s'", orderer.GetName()))

		if orderer.Status.HasType() {
			log.Info(fmt.Sprintf("Operator restart detected, performing update checks on exisitng orderer '%s'", orderer.GetName()))

			cm, err := r.GetSpecState(orderer)
			if err != nil {
				log.Info(fmt.Sprintf("Failed getting saved orderer spec '%s', can't perform update checks, triggering reconcile: %s", orderer.GetName(), err.Error()))
				return true
			}

			specBytes := cm.BinaryData["spec"]
			savedOrderer := &current.IBPOrderer{}

			err = yaml.Unmarshal(specBytes, &savedOrderer.Spec)
			if err != nil {
				log.Info(fmt.Sprintf("Unmarshal failed for saved orderer spec '%s', can't perform update checks, triggering reconcile: %s", orderer.GetName(), err.Error()))
				return true
			}

			if !reflect.DeepEqual(orderer.Spec, savedOrderer.Spec) {
				log.Info(fmt.Sprintf("IBPOrderer '%s' spec was updated while operator was down", orderer.GetName()))
				update.specUpdated = true
			}

			if !reflect.DeepEqual(orderer.Spec.ConfigOverride, savedOrderer.Spec.ConfigOverride) {
				log.Info(fmt.Sprintf("IBPOrderer '%s' overrides were updated while operator was down", orderer.GetName()))
				update.overridesUpdated = true
			}

			update.imagesUpdated = imagesUpdated(savedOrderer, orderer)
			update.fabricVersionUpdated = fabricVersionUpdated(savedOrderer, orderer)
			if fabricVersionUpdatedTo149plusOr221plus(savedOrderer, orderer) {
				log.Info(fmt.Sprintf("Fabric version update detected from '%s' to '%s' setting tls cert created flag '%s'", savedOrderer.Spec.FabricVersion, orderer.Spec.FabricVersion, orderer.GetName()))
				update.tlsCertCreated = true
			}

			log.Info(fmt.Sprintf("Create event triggering reconcile for updating orderer '%s'", orderer.GetName()))
			r.PushUpdate(orderer.GetName(), update)
		}

		// If creating resource for the first time, check that a unique name is provided
		err := commoncontroller.ValidateCRName(r.client, orderer.Name, orderer.Namespace, commoncontroller.IBPORDERER)
		if err != nil {
			log.Error(err, "failed to validate orderer name")
			operror := operatorerrors.Wrap(err, operatorerrors.InvalidCustomResourceCreateRequest, "failed to validate custom resource name")
			err = r.SetStatus(orderer, nil, operror)
			if err != nil {
				log.Error(err, "failed to set status to error", "orderer.name", orderer.Name, "error", "InvalidCustomResourceCreateRequest")
			}
			return false
		}

		log.Info(fmt.Sprintf("Create event triggering reconcile for creating orderer '%s'", orderer.GetName()))

	case *corev1.Secret:
		secret := e.Object.(*corev1.Secret)

		if secret.OwnerReferences == nil || len(secret.OwnerReferences) == 0 {
			isOrdererSecret, err := r.AddOwnerReferenceToSecret(secret)
			if err != nil || !isOrdererSecret {
				return false
			}
		}

		if secret.OwnerReferences[0].Kind == KIND {
			log.Info(fmt.Sprintf("Create event detected for secret '%s'", secret.GetName()))
			instanceName := secret.OwnerReferences[0].Name
			if util.IsSecretTLSCert(secret.Name) {
				update.tlsCertCreated = true
			} else if util.IsSecretEcert(secret.Name) {
				update.ecertCreated = true
			} else {
				return false
			}

			log.Info(fmt.Sprintf("Orderer crypto create triggering reconcile on IBPOrderer custom resource %s: update [ %+v ]", instanceName, update.GetUpdateStackWithTrues()))
			r.PushUpdate(instanceName, update)
		}

	case *appsv1.Deployment:
		dep := e.Object.(*appsv1.Deployment)
		log.Info(fmt.Sprintf("Create event detected by IBPOrderer controller for deployment '%s', triggering reconcile", dep.GetName()))

	case *corev1.ConfigMap:
		cm := e.Object.(*corev1.ConfigMap)
		if cm.Name == "orderer-restart-config" {
			log.Info(fmt.Sprintf("Create event detected by IBPOrderer contoller for config map '%s', triggering restart reconcile", cm.GetName()))
		} else {
			return false
		}

	}

	return true
}

func (r *ReconcileIBPOrderer) UpdateFunc(e event.UpdateEvent) bool {
	update := Update{}

	switch e.ObjectOld.(type) {
	case *current.IBPOrderer:
		oldOrderer := e.ObjectOld.(*current.IBPOrderer)
		newOrderer := e.ObjectNew.(*current.IBPOrderer)
		log.Info(fmt.Sprintf("Update event detected for orderer '%s'", oldOrderer.GetName()))

		if oldOrderer.Spec.NodeNumber == nil {
			if oldOrderer.Status.Type != newOrderer.Status.Type {
				log.Info(fmt.Sprintf("Parent orderer %s status updated from %s to %s", oldOrderer.Name, oldOrderer.Status.Type, newOrderer.Status.Type))
			}

			if oldOrderer.Status.Type == current.Deployed || oldOrderer.Status.Type == current.Error || oldOrderer.Status.Type == current.Warning {
				// Parent orderer has been fully deployed by this point
				log.Info(fmt.Sprintf("Ignoring the IBPOrderer cluster (parent) update after %s", oldOrderer.Status.Type))
				return false
			}

		}

		if util.CheckIfZoneOrRegionUpdated(oldOrderer.Spec.Zone, newOrderer.Spec.Zone) {
			log.Error(errors.New("Zone update is not allowed"), "invalid spec update")
			return false
		}

		if util.CheckIfZoneOrRegionUpdated(oldOrderer.Spec.Region, newOrderer.Spec.Region) {
			log.Error(errors.New("Region update is not allowed"), "invalid spec update")
			return false
		}

		// Need to trigger update when status has changed is to allow us update the
		// status of the parent, since the status of the parent depends on the status
		// of its children .Only flag status update when there is a meaninful change
		// and not everytime the heartbeat is updated
		if oldOrderer.Status != newOrderer.Status {
			if oldOrderer.Status.Type != newOrderer.Status.Type ||
				oldOrderer.Status.Reason != newOrderer.Status.Reason ||
				oldOrderer.Status.Message != newOrderer.Status.Message {

				log.Info(fmt.Sprintf("%s status changed to '%+v' from '%+v'", oldOrderer.GetName(), newOrderer.Status, oldOrderer.Status))
				update.statusUpdated = true
			}
		}

		if !reflect.DeepEqual(oldOrderer.Spec.ConfigOverride, newOrderer.Spec.ConfigOverride) {
			log.Info(fmt.Sprintf("%s config override updated", oldOrderer.GetName()))
			update.overridesUpdated = true
		}

		if !reflect.DeepEqual(oldOrderer.Spec, newOrderer.Spec) {
			log.Info(fmt.Sprintf("%s spec updated", oldOrderer.GetName()))
			update.specUpdated = true
		}

		// Check for changes to orderer tag to determine if any migration logic needs to be executed
		// from old orderer version to new orderer version
		if oldOrderer.Spec.Images != nil && newOrderer.Spec.Images != nil {
			if oldOrderer.Spec.Images.OrdererTag != newOrderer.Spec.Images.OrdererTag {
				log.Info(fmt.Sprintf("%s orderer tag updated from %s to %s", oldOrderer.GetName(), oldOrderer.Spec.Images.OrdererTag, newOrderer.Spec.Images.OrdererTag))
				update.ordererTagUpdated = true
			}
		}

		if fabricVersionUpdatedTo149plusOr221plus(oldOrderer, newOrderer) {
			log.Info(fmt.Sprintf("Fabric version update detected from '%s' to '%s' setting tls cert created flag '%s'", oldOrderer.Spec.FabricVersion, newOrderer.Spec.FabricVersion, newOrderer.GetName()))
			update.tlsCertCreated = true
		}

		update.mspUpdated = commoncontroller.MSPInfoUpdateDetected(oldOrderer.Spec.Secret, newOrderer.Spec.Secret)

		if newOrderer.Spec.Action.Restart {
			update.restartNeeded = true
		}

		if oldOrderer.Spec.Action.Reenroll.Ecert != newOrderer.Spec.Action.Reenroll.Ecert {
			update.ecertReenrollNeeded = newOrderer.Spec.Action.Reenroll.Ecert
		}

		if oldOrderer.Spec.Action.Reenroll.TLSCert != newOrderer.Spec.Action.Reenroll.TLSCert {
			update.tlscertReenrollNeeded = newOrderer.Spec.Action.Reenroll.TLSCert
		}

		if oldOrderer.Spec.Action.Reenroll.EcertNewKey != newOrderer.Spec.Action.Reenroll.EcertNewKey {
			update.ecertNewKeyReenroll = newOrderer.Spec.Action.Reenroll.EcertNewKey
		}

		if oldOrderer.Spec.Action.Reenroll.TLSCertNewKey != newOrderer.Spec.Action.Reenroll.TLSCertNewKey {
			update.tlscertNewKeyReenroll = newOrderer.Spec.Action.Reenroll.TLSCertNewKey
		}

		if newOrderer.Spec.Action.Enroll.Ecert {
			update.ecertEnroll = true
		}

		if newOrderer.Spec.Action.Enroll.TLSCert {
			update.tlscertEnroll = true
		}

		update.deploymentUpdated = deploymentUpdated(oldOrderer, newOrderer)
		oldVer := version.String(oldOrderer.Spec.FabricVersion)
		newVer := version.String(newOrderer.Spec.FabricVersion)

		// check if this V1 -> V2.2.x/V2.4.x/v2.5.x orderer migration
		if (oldOrderer.Spec.FabricVersion == "" ||
			version.GetMajorReleaseVersion(oldOrderer.Spec.FabricVersion) == version.V1) &&
			version.GetMajorReleaseVersion(newOrderer.Spec.FabricVersion) == version.V2 {
			update.migrateToV2 = true
			if newVer.EqualWithoutTag(version.V2_5_1) || newVer.GreaterThan(version.V2_5_1) {
				update.migrateToV25 = true
				// Re-enrolling tls cert to include admin hostname in SAN (for orderers >=2.5.1)
				update.tlscertReenrollNeeded = true
			} else if newVer.EqualWithoutTag(version.V2_4_1) || newVer.GreaterThan(version.V2_4_1) {
				update.migrateToV24 = true
				// Re-enrolling tls cert to include admin hostname in SAN (for orderers >=2.4.1)
				update.tlscertReenrollNeeded = true
			}
		}

		// check if this V2.2.x -> V2.4.x/2.5.x orderer migration
		if (version.GetMajorReleaseVersion(oldOrderer.Spec.FabricVersion) == version.V2) &&
			oldVer.LessThan(version.V2_4_1) {
			if newVer.EqualWithoutTag(version.V2_5_1) || newVer.GreaterThan(version.V2_5_1) {
				update.migrateToV25 = true
				// Re-enrolling tls cert to include admin hostname in SAN (for orderers >=2.4.1)
				update.tlscertReenrollNeeded = true
			} else if newVer.EqualWithoutTag(version.V2_4_1) || newVer.GreaterThan(version.V2_4_1) {
				update.migrateToV24 = true
				// Re-enrolling tls cert to include admin hostname in SAN (for orderers >=2.4.1)
				update.tlscertReenrollNeeded = true
			}
		}

		// check if this V2.4.x -> V2.5.x orderer migration
		if (version.GetMajorReleaseVersion(oldOrderer.Spec.FabricVersion) == version.V2) &&
			oldVer.LessThan(version.V2_5_1) &&
			(newVer.EqualWithoutTag(version.V2_5_1) || newVer.GreaterThan(version.V2_5_1)) {
			update.migrateToV25 = true
			//Orderers >=2.4.1 alredy has the tls-cert renewed, we do not do this in this upgrade
			//update.tlscertReenrollNeeded = true
		}

		if oldOrderer.Spec.NodeOUDisabled() != newOrderer.Spec.NodeOUDisabled() {
			update.nodeOUUpdated = true
		}

		// if use updates NumSecondsWarningPeriod field once we have already run the reconcile
		// we need to retrigger the timer logic
		if oldOrderer.Spec.NumSecondsWarningPeriod != newOrderer.Spec.NumSecondsWarningPeriod {
			update.ecertUpdated = true
			update.tlsCertUpdated = true
			log.Info(fmt.Sprintf("%s NumSecondsWarningPeriod updated", oldOrderer.GetName()))
		}

		if update.Detected() {
			log.Info(fmt.Sprintf("Spec update triggering reconcile on IBPOrderer custom resource %s: update [ %+v ]", oldOrderer.GetName(), update.GetUpdateStackWithTrues()))
			r.PushUpdate(oldOrderer.GetName(), update)
			return true
		}

	case *appsv1.Deployment:
		oldDeployment := e.ObjectOld.(*appsv1.Deployment)
		log.Info(fmt.Sprintf("Spec update detected by IBPOrderer controller on deployment '%s'", oldDeployment.GetName()))

	case *corev1.Secret:
		oldSecret := e.ObjectOld.(*corev1.Secret)
		newSecret := e.ObjectNew.(*corev1.Secret)

		if oldSecret.OwnerReferences == nil || len(oldSecret.OwnerReferences) == 0 {
			isOrdererSecret, err := r.AddOwnerReferenceToSecret(oldSecret)
			if err != nil || !isOrdererSecret {
				return false
			}
		}

		if oldSecret.OwnerReferences[0].Kind == KIND {
			if reflect.DeepEqual(oldSecret.Data, newSecret.Data) {
				return false
			}

			log.Info(fmt.Sprintf("Update event detected on secret '%s'", oldSecret.GetName()))
			instanceName := oldSecret.OwnerReferences[0].Name
			if util.IsSecretTLSCert(oldSecret.Name) {
				update.tlsCertUpdated = true
				log.Info(fmt.Sprintf("TLS cert updated for %s", instanceName))
			}
			if util.IsSecretEcert(oldSecret.Name) {
				update.ecertUpdated = true
				log.Info(fmt.Sprintf("ecert updated for %s", instanceName))
			}

			if update.CertificateUpdated() {
				log.Info(fmt.Sprintf("Orderer crypto update triggering reconcile on IBPOrderer custom resource %s: update [ %+v ]", instanceName, update.GetUpdateStackWithTrues()))
				r.PushUpdate(instanceName, update)
				return true
			}
		}

	case *corev1.ConfigMap:
		cm := e.ObjectOld.(*corev1.ConfigMap)
		if cm.Name == "orderer-restart-config" {
			log.Info("Update event detected for orderer-restart-config, triggering restart reconcile")
			return true
		}
	}

	return false
}

func (r *ReconcileIBPOrderer) DeleteFunc(e event.DeleteEvent) bool {
	switch e.Object.(type) {
	case *current.IBPOrderer:
		oldOrderer := e.Object.(*current.IBPOrderer)

		if oldOrderer.Spec.NodeNumber != nil {
			log.Info(fmt.Sprintf("Orderer node %d (%s) deleted", *oldOrderer.Spec.NodeNumber, oldOrderer.GetName()))

			// Deleting this config map manually, in 2.5.1 release of operator this config map was created
			// without proper controller references set and was not cleaned up on orderer resource deletion.
			log.Info(fmt.Sprintf("Deleting %s-init-config config map, if found", oldOrderer.GetName()))
			if err := r.client.Delete(context.TODO(), &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-init-config", oldOrderer.GetName()),
					Namespace: oldOrderer.GetNamespace(),
				},
			}); client.IgnoreNotFound(err) != nil {
				log.Info(fmt.Sprintf("failed to delete config map: %s", err))
			}

			parentName := oldOrderer.ObjectMeta.Labels["parent"]
			labelSelector, err := labels.Parse(fmt.Sprintf("parent=%s", parentName))
			if err != nil {
				log.Info(fmt.Sprintf("failed to parse selector for parent name: %s", err.Error()))
				return false
			}

			listOptions := &client.ListOptions{
				LabelSelector: labelSelector,
				Namespace:     oldOrderer.GetNamespace(),
			}

			ordererList := &current.IBPOrdererList{}
			err = r.client.List(context.TODO(), ordererList, listOptions)
			if err != nil {
				log.Info(fmt.Sprintf("Ignoring Deletion of Orderer node %d (%s) due to error in getting list of other nodes: %s", *oldOrderer.Spec.NodeNumber, oldOrderer.GetName(), err.Error()))
				return false
			}

			log.Info(fmt.Sprintf("There are %d child nodes for the orderer parent %s.", len(ordererList.Items), parentName))

			if len(ordererList.Items) == 0 {
				log.Info(fmt.Sprintf("Deleting Parent (%s) of Orderer node %d (%s) as all nodes are deleted.", parentName, *oldOrderer.Spec.NodeNumber, oldOrderer.GetName()))
				parent := &current.IBPOrderer{}
				parent.SetName(parentName)
				parent.SetNamespace(oldOrderer.GetNamespace())

				err := r.client.Delete(context.TODO(), parent)
				if err != nil {
					log.Error(err, fmt.Sprintf("Error deleting parent (%s) of Orderer node %d (%s).", parentName, *oldOrderer.Spec.NodeNumber, oldOrderer.GetName()))
				}
				return false
			}

			log.Info(fmt.Sprintf("Ignoring Deletion of Orderer node %d (%s) as there are %d nodes of parent still around", *oldOrderer.Spec.NodeNumber, oldOrderer.GetName(), len(ordererList.Items)))
			return false
		}

		log.Info(fmt.Sprintf("Orderer parent %s deleted", oldOrderer.GetName()))
		parentName := oldOrderer.GetName()
		labelSelector, err := labels.Parse(fmt.Sprintf("parent=%s", parentName))
		if err != nil {
			log.Info(fmt.Sprintf("failed to parse selector for parent name: %s", err.Error()))
		}

		listOptions := &client.ListOptions{
			LabelSelector: labelSelector,
			Namespace:     oldOrderer.GetNamespace(),
		}

		ordererList := &current.IBPOrdererList{}
		err = r.client.List(context.TODO(), ordererList, listOptions)
		if err != nil {
			log.Info(fmt.Sprintf("Ignoring Deletion of Orderer parent %s due to error in getting list of child nodes: %s", oldOrderer.GetName(), err.Error()))
			return false
		}

		log.Info(fmt.Sprintf("There are %d child nodes for the orderer parent %s.", len(ordererList.Items), parentName))

		for _, item := range ordererList.Items {
			log.Info(fmt.Sprintf("Deleting child node %s", item.GetName()))

			child := &current.IBPOrderer{}
			child.SetName(item.GetName())
			child.SetNamespace(item.GetNamespace())

			err := r.client.Delete(context.TODO(), child)
			if err != nil {
				log.Error(err, fmt.Sprintf("Error child node (%s) of Orderer (%s).", child.GetName(), parentName))
			}
		}

		return false

	case *appsv1.Deployment:
		dep := e.Object.(*appsv1.Deployment)
		log.Info(fmt.Sprintf("Delete detected by IBPOrderer controller on deployment '%s'", dep.GetName()))
	case *corev1.Secret:
		secret := e.Object.(*corev1.Secret)
		log.Info(fmt.Sprintf("Delete detected by IBPOrderer controller on secret '%s'", secret.GetName()))
	case *corev1.ConfigMap:
		cm := e.Object.(*corev1.ConfigMap)
		log.Info(fmt.Sprintf("Delete detected by IBPOrderer controller on configmap '%s'", cm.GetName()))
	}

	return true
}

func (r *ReconcileIBPOrderer) GetUpdateStatusAtElement(instance *current.IBPOrderer, index int) *Update {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	update := Update{}
	_, ok := r.update[instance.GetName()]
	if !ok {
		return &update
	}

	if len(r.update[instance.GetName()]) >= 1 {
		update = r.update[instance.GetName()][index]
	}

	return &update
}

func (r *ReconcileIBPOrderer) GetUpdateStatus(instance *current.IBPOrderer) *Update {
	return r.GetUpdateStatusAtElement(instance, 0)
}

func (r *ReconcileIBPOrderer) PushUpdate(instanceName string, update Update) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.update[instanceName] = r.AppendUpdateIfMissing(r.update[instanceName], update)
}

func (r *ReconcileIBPOrderer) PopUpdate(instanceName string) *Update {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	update := Update{}
	if len(r.update[instanceName]) >= 1 {
		update = r.update[instanceName][0]
		if len(r.update[instanceName]) == 1 {
			r.update[instanceName] = []Update{}
		} else {
			r.update[instanceName] = r.update[instanceName][1:]
		}
	}

	return &update
}

func (r *ReconcileIBPOrderer) AppendUpdateIfMissing(updates []Update, update Update) []Update {
	for _, u := range updates {
		if u == update {
			return updates
		}
	}
	return append(updates, update)
}

func deploymentUpdated(oldOrderer, newOrderer *current.IBPOrderer) bool {
	if !reflect.DeepEqual(oldOrderer.Spec.Images, newOrderer.Spec.Images) {
		log.Info(fmt.Sprintf("Images updated for '%s', deployment will be updated", newOrderer.Name))
		return true
	}

	if !reflect.DeepEqual(oldOrderer.Spec.Replicas, newOrderer.Spec.Replicas) {
		log.Info(fmt.Sprintf("Replica size updated for '%s', deployment will be updated", newOrderer.Name))
		return true
	}

	if !reflect.DeepEqual(oldOrderer.Spec.Resources, newOrderer.Spec.Resources) {
		log.Info(fmt.Sprintf("Resources updated for '%s', deployment will be updated", newOrderer.Name))
		return true
	}

	if !reflect.DeepEqual(oldOrderer.Spec.Storage, newOrderer.Spec.Storage) {
		log.Info(fmt.Sprintf("Storage updated for '%s', deployment will be updated", newOrderer.Name))
		return true
	}

	if len(oldOrderer.Spec.ImagePullSecrets) != len(newOrderer.Spec.ImagePullSecrets) {
		log.Info(fmt.Sprintf("ImagePullSecret updated for '%s', deployment will be updated", newOrderer.Name))
		return true
	}
	for i, v := range newOrderer.Spec.ImagePullSecrets {
		if v != oldOrderer.Spec.ImagePullSecrets[i] {
			log.Info(fmt.Sprintf("ImagePullSecret updated for '%s', deployment will be updated", newOrderer.Name))
			return true
		}
	}

	return false
}

func (r *ReconcileIBPOrderer) AddOwnerReferenceToSecret(secret *corev1.Secret) (bool, error) {
	// Orderer secrets we are looking to add owner references to are named:
	// <prefix>-<instance name>-<type>
	// <instance name>-init-rootcert

	// The following secrets are created by operator, and will have owner references:
	// <instance name>-genesis
	// <instance name>-crypto-backup
	// <instance name>-secret

	items := strings.Split(secret.Name, "-")
	if len(items) < 3 {
		// Secret names we are looking for will be split into at least 3 strings:
		// [prefix, instance name, type] OR [instance name, "init", "rootcert"]
		return false, nil
	}

	// Account for the case where the instance's name is hyphenated
	var instanceName string
	if strings.Contains(secret.Name, "-init-rootcert") {
		instanceName = strings.Join(items[:len(items)-2], "-") // instance name contains all but last 2 items
	} else {
		instanceName = strings.Join(items[1:len(items)-1], "-") // instance name contains all but first and last item
	}

	listOptions := &client.ListOptions{
		Namespace: secret.Namespace,
	}

	ordererList := &current.IBPOrdererList{}
	err := r.client.List(context.TODO(), ordererList, listOptions)
	if err != nil {
		return false, errors.Wrap(err, "failed to get list of orderers")
	}

	for _, o := range ordererList.Items {
		orderer := o
		if orderer.Name == instanceName {
			// Instance 'i' found in list of orderers
			err := r.client.Update(context.TODO(), secret, k8sclient.UpdateOption{
				Owner:  &orderer,
				Scheme: r.scheme,
			})
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

func (r *ReconcileIBPOrderer) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&current.IBPOrderer{}).
		Complete(r)
}

func GetUpdateStack(allUpdates map[string][]Update) string {
	stack := ""

	for orderer, updates := range allUpdates {
		currentStack := ""
		for index, update := range updates {
			currentStack += fmt.Sprintf("{ %s}", update.GetUpdateStackWithTrues())
			if index != len(updates)-1 {
				currentStack += " , "
			}
		}
		stack += fmt.Sprintf("%s: [ %s ] ", orderer, currentStack)
	}

	return stack
}

func (r *ReconcileIBPOrderer) ReconcileRestart(namespace string) (bool, error) {
	requeue, err := r.RestartService.Reconcile("orderer", namespace)
	if err != nil {
		log.Error(err, "failed to reconcile restart queues in orderer-restart-config")
		return false, err
	}

	return requeue, nil
}
