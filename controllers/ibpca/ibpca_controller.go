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

package ibpca

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commoncontroller "github.com/IBM-Blockchain/fabric-operator/controllers/common"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/global"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
	baseca "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	k8sca "github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/ca"
	openshiftca "github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/ca"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/staggerrestarts"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/go-test/deep"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	yaml "sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	KIND = "IBPCA"
)

var log = logf.Log.WithName("controller_ibpca")

// Add creates a new IBPCA Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, cfg *config.Config) error {
	r, err := newReconciler(mgr, cfg)
	if err != nil {
		return err
	}
	return add(mgr, r)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, cfg *config.Config) (*ReconcileIBPCA, error) {
	client := k8sclient.New(mgr.GetClient(), &global.ConfigSetter{Config: cfg.Operator.Globals})
	scheme := mgr.GetScheme()

	ibpca := &ReconcileIBPCA{
		client:         client,
		scheme:         scheme,
		Config:         cfg,
		update:         map[string][]Update{},
		mutex:          &sync.Mutex{},
		RestartService: staggerrestarts.New(client, cfg.Operator.Restart.Timeout.Get()),
	}

	switch cfg.Offering {
	case offering.K8S:
		ibpca.Offering = k8sca.New(client, scheme, cfg)
	case offering.OPENSHIFT:
		ibpca.Offering = openshiftca.New(client, scheme, cfg)
	}

	return ibpca, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileIBPCA) error {
	// Create a new controller
	predicateFuncs := predicate.Funcs{
		CreateFunc: r.CreateFunc,
		UpdateFunc: r.UpdateFunc,
	}

	c, err := controller.New("ibpca-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IBPCA
	err = c.Watch(&source.Kind{Type: &current.IBPCA{}}, &handler.EnqueueRequestForObject{}, predicateFuncs)
	if err != nil {
		return err
	}

	// Watch for changes to config maps (Create and Update funcs handle only watching for restart config map)
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, predicateFuncs)
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &current.IBPCA{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to tertiary resource Secrets and requeue the owner IBPPeer
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &current.IBPCA{},
	}, predicateFuncs)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIBPCA{}

//go:generate counterfeiter -o mocks/careconcile.go -fake-name CAReconcile . caReconcile
//counterfeiter:generate . caReconcile
type caReconcile interface {
	Reconcile(*current.IBPCA, baseca.Update) (common.Result, error)
}

// ReconcileIBPCA reconciles a IBPCA object
type ReconcileIBPCA struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client k8sclient.Client
	scheme *runtime.Scheme

	Offering       caReconcile
	Config         *config.Config
	RestartService *staggerrestarts.StaggerRestartsService

	update map[string][]Update
	mutex  *sync.Mutex
}

// Reconcile reads that state of the cluster for a IBPCA object and makes changes based on the state read
// and what is in the IBPCA.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=persistentvolumeclaims;persistentvolumes,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups="",resources=pods;pods/log;persistentvolumeclaims;persistentvolumes;services;endpoints;events;configmaps;secrets;nodes;serviceaccounts,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups="batch",resources=jobs,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups="authorization.openshift.io";"rbac.authorization.k8s.io",resources=roles;rolebinding,verbs=get;list;watch;create;update;patch;delete;deletecollection;bind;escalate
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;create
// +kubebuilder:rbac:groups=apps,resourceNames=ibp-operator,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=ibp.com,resources=ibpcas.ibp.com;ibppeers.ibp.com;ibporderers.ibp.com;ibpcas;ibppeers;ibporderers;ibpconsoles;ibpcas/finalizers;ibppeer/finalizers;ibporderers/finalizers;ibpconsole/finalizers;ibpcas/status;ibppeers/status;ibporderers/status;ibpconsoles/status,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=extensions;networking.k8s.io;config.openshift.io,resources=ingresses;networkpolicies,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete;deletecollection
func (r *ReconcileIBPCA) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var err error

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// If ca-restart-config configmap is the object being reconciled, reconcile the
	// restart configmap.
	if request.Name == "ca-restart-config" {
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

	reqLogger.Info("Reconciling IBPCA")

	// Fetch the IBPCA instance
	instance := &current.IBPCA{}
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

	reqLogger.Info(fmt.Sprintf("Current update stack to process: %+v", GetUpdateStack(r.update)))

	update := r.GetUpdateStatus(instance)
	reqLogger.Info(fmt.Sprintf("Reconciling IBPCA '%s' with update values of [ %+v ]", instance.GetName(), update.GetUpdateStackWithTrues()))

	result, err := r.Offering.Reconcile(instance, r.PopUpdate(instance.GetName()))
	setStatusErr := r.SetStatus(instance, result.Status, err)
	if setStatusErr != nil {
		return reconcile.Result{}, operatorerrors.IsBreakingError(setStatusErr, "failed to update status", log)
	}

	if err != nil {
		return reconcile.Result{}, operatorerrors.IsBreakingError(errors.Wrapf(err, "CA instance '%s' encountered error", instance.GetName()), "stopping reconcile loop", log)
	}

	if result.Requeue {
		r.PushUpdate(instance.GetName(), *update)
	}

	reqLogger.Info(fmt.Sprintf("Finished reconciling IBPCA '%s' with update values of [ %+v ]", instance.GetName(), update.GetUpdateStackWithTrues()))

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

func (r *ReconcileIBPCA) SetStatus(instance *current.IBPCA, reconcileStatus *current.CRStatus, reconcileErr error) error {
	log.Info(fmt.Sprintf("Setting status for '%s'", instance.GetName()))

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

		instance.Status = current.IBPCAStatus{
			CRStatus: status,
		}

		log.Info(fmt.Sprintf("Updating status of IBPCA custom resource to %s phase", instance.Status.Type))
		err = r.client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    2,
				Into:     &current.IBPCA{},
				Strategy: client.MergeFrom,
			},
		})
		if err != nil {
			return err
		}

		return nil
	}

	status.Versions.Reconciled = instance.Spec.FabricVersion

	// Check if reconcile loop returned an updated status that differs from exisiting status.
	// If so, set status to the reconcile status.
	if reconcileStatus != nil {
		if instance.Status.Type != reconcileStatus.Type || instance.Status.Reason != reconcileStatus.Reason || instance.Status.Message != reconcileStatus.Message {
			status.Type = reconcileStatus.Type
			status.Status = current.True
			status.Reason = reconcileStatus.Reason
			status.Message = reconcileStatus.Message
			status.LastHeartbeatTime = time.Now().String()

			instance.Status = current.IBPCAStatus{
				CRStatus: status,
			}

			log.Info(fmt.Sprintf("Updating status of IBPPeer custom resource to %s phase", instance.Status.Type))
			err := r.client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
				Resilient: &k8sclient.ResilientPatch{
					Retry:    2,
					Into:     &current.IBPCA{},
					Strategy: client.MergeFrom,
				},
			})
			if err != nil {
				return err
			}

			return nil
		}
	}

	running, err := r.PodsRunning(instance)
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
		status.Message = "All pods running"
	} else {
		if instance.Status.Type == current.Deploying {
			return nil
		}
		status.Type = current.Deploying
		status.Status = current.True
		status.Reason = "waitingForPods"
		status.Message = "Waiting for pods"
	}

	instance.Status = current.IBPCAStatus{
		CRStatus: status,
	}
	instance.Status.LastHeartbeatTime = time.Now().String()
	log.Info(fmt.Sprintf("Updating status of IBPCA custom resource to %s phase", instance.Status.Type))
	err = r.client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
		Resilient: &k8sclient.ResilientPatch{
			Retry:    2,
			Into:     &current.IBPCA{},
			Strategy: client.MergeFrom,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileIBPCA) SaveSpecState(instance *current.IBPCA) error {
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

	err = r.client.CreateOrUpdate(context.TODO(), cm, k8sclient.CreateOrUpdateOption{
		Owner:  instance,
		Scheme: r.scheme,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileIBPCA) GetSpecState(instance *current.IBPCA) (*corev1.ConfigMap, error) {
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

func (r *ReconcileIBPCA) PodsRunning(instance *current.IBPCA) (bool, error) {
	labelSelector, err := labels.Parse(fmt.Sprintf("app=%s", instance.GetName()))
	if err != nil {
		return false, errors.Wrap(err, "failed to parse label selector for app name")
	}

	listOptions := &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     instance.GetNamespace(),
	}

	podList := &corev1.PodList{}
	err = r.client.List(context.TODO(), podList, listOptions)
	if err != nil {
		return false, err
	}

	if len(podList.Items) == 0 {
		return false, nil
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
	}

	return true, nil
}

func (r *ReconcileIBPCA) getIgnoreDiffs() []string {
	return []string{
		`Template\.Spec\.Containers\.slice\[\d\]\.Resources\.Requests\.map\[memory\].s`,
		`Template\.Spec\.InitContainers\.slice\[\d\]\.Resources`,
		`Ports\.slice\[\d\]\.Protocol`,
	}
}

func (r *ReconcileIBPCA) getLabels(instance v1.Object) map[string]string {
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
		"app.kubernetes.io/instance":   label + "ca",
		"app.kubernetes.io/managed-by": label + "-operator",
	}
}

func (r *ReconcileIBPCA) getSelectorLabels(instance v1.Object) map[string]string {
	return map[string]string{
		"app": instance.GetName(),
	}
}

// TODO: Move to predicate.go
func (r *ReconcileIBPCA) CreateFunc(e event.CreateEvent) bool {
	update := Update{}

	switch e.Object.(type) {
	case *current.IBPCA:
		ca := e.Object.(*current.IBPCA)
		log.Info(fmt.Sprintf("Create event detected for ca '%s'", ca.GetName()))

		// Operator restart detected, want to trigger update logic for CA resource if changes detected
		if ca.Status.HasType() {
			log.Info(fmt.Sprintf("Operator restart detected, running update flow on existing ca '%s'", ca.GetName()))

			// Get the spec state of the resource before the operator went down, this
			// will be used to compare to see if the spec of resources has changed
			cm, err := r.GetSpecState(ca)
			if err != nil {
				log.Info(fmt.Sprintf("Failed getting saved ca spec '%s', triggering create: %s", ca.GetName(), err.Error()))
				return true
			}

			specBytes := cm.BinaryData["spec"]
			existingCA := &current.IBPCA{}
			err = yaml.Unmarshal(specBytes, &existingCA.Spec)
			if err != nil {
				log.Info(fmt.Sprintf("Unmarshal failed for saved ca spec '%s', triggering create: %s", ca.GetName(), err.Error()))
				return true
			}

			diff := deep.Equal(ca.Spec, existingCA.Spec)
			if diff != nil {
				log.Info(fmt.Sprintf("IBPCA '%s' spec was updated while operator was down", ca.GetName()))
				log.Info(fmt.Sprintf("Difference detected: %s", diff))
				update.specUpdated = true
			}

			// If existing CA spec did not have config overrides defined but new spec does,
			// trigger update logic for both CA and TLSCA overrides
			if ca.Spec.ConfigOverride == nil && existingCA.Spec.ConfigOverride != nil {
				log.Info(fmt.Sprintf("IBPCA '%s' CA and TLSCA overrides were updated while operator was down", ca.GetName()))
				update.caOverridesUpdated = true
				update.tlscaOverridesUpdated = true
			}

			// If existing CA spec had config overrides defined, need to further check to see if CA or
			// TLSCA specs have been updated and trigger update for the one on which updates are detected.
			if ca.Spec.ConfigOverride != nil && existingCA.Spec.ConfigOverride != nil {
				if ca.Spec.ConfigOverride.CA != nil && existingCA.Spec.ConfigOverride.CA != nil {
					if !reflect.DeepEqual(ca.Spec.ConfigOverride.CA, existingCA.Spec.ConfigOverride.CA) {
						log.Info(fmt.Sprintf("IBPCA '%s' CA overrides were updated while operator was down", ca.GetName()))
						update.caOverridesUpdated = true
					}
				}

				if ca.Spec.ConfigOverride.TLSCA != nil && existingCA.Spec.ConfigOverride.TLSCA != nil {
					if !reflect.DeepEqual(ca.Spec.ConfigOverride.TLSCA, existingCA.Spec.ConfigOverride.TLSCA) {
						log.Info(fmt.Sprintf("IBPCA '%s' TLSCA overrides were updated while operator was down", ca.GetName()))
						update.tlscaOverridesUpdated = true
					}
				}
			}

			update.imagesUpdated = imagesUpdated(existingCA, ca)
			update.fabricVersionUpdated = fabricVersionUpdated(existingCA, ca)

			log.Info(fmt.Sprintf("Create event triggering reconcile for updating ca '%s'", ca.GetName()))
			r.PushUpdate(ca.GetName(), update)
			return true
		}

		// TODO: This seems more appropriate for the PreReconcileCheck method rather than the predicate function. Not
		// sure if there was reason for putting it here, but if not we should consider moving it
		//
		// If creating resource for the first time, check that a unique name is provided
		err := commoncontroller.ValidateCRName(r.client, ca.Name, ca.Namespace, commoncontroller.IBPCA)
		if err != nil {
			log.Error(err, "failed to validate ca name")
			operror := operatorerrors.Wrap(err, operatorerrors.InvalidCustomResourceCreateRequest, "failed to validate custom resource name")

			err = r.SetStatus(ca, nil, operror)
			if err != nil {
				log.Error(err, "failed to set status to error", "ca.name", ca.Name, "error", "InvalidCustomResourceCreateRequest")
			}
			return false
		}

		log.Info(fmt.Sprintf("Create event triggering reconcile for creating ca '%s'", ca.GetName()))

	case *corev1.Secret:
		secret := e.Object.(*corev1.Secret)

		if secret.OwnerReferences == nil || len(secret.OwnerReferences) == 0 {
			isCASecret, err := r.AddOwnerReferenceToSecret(secret)
			if err != nil || !isCASecret {
				return false
			}
		}

		if secret.OwnerReferences[0].Kind == KIND {
			instanceName := secret.OwnerReferences[0].Name
			log.Info(fmt.Sprintf("Create event detected for secret '%s'", secret.GetName()))

			if strings.HasSuffix(secret.Name, "-ca-crypto") {
				update.caCryptoCreated = true
				log.Info(fmt.Sprintf("CA crypto created, triggering reconcile for IBPCA custom resource %s: update [ %+v ]", instanceName, update.GetUpdateStackWithTrues()))
			} else {
				return false
			}

			r.PushUpdate(instanceName, update)
		}

	case *appsv1.Deployment:
		dep := e.Object.(*appsv1.Deployment)
		log.Info(fmt.Sprintf("Create event detected by IBPCA controller for deployment '%s', triggering reconcile", dep.GetName()))

	case *corev1.ConfigMap:
		cm := e.Object.(*corev1.ConfigMap)
		if cm.Name == "ca-restart-config" {
			log.Info(fmt.Sprintf("Create event detected by IBPCA contoller for config map '%s', triggering restart reconcile", cm.GetName()))
		} else {
			return false
		}
	}

	return true
}

// TODO: Move to predicate.go
func (r *ReconcileIBPCA) UpdateFunc(e event.UpdateEvent) bool {
	update := Update{}

	switch e.ObjectOld.(type) {
	case *current.IBPCA:
		oldCA := e.ObjectOld.(*current.IBPCA)
		newCA := e.ObjectNew.(*current.IBPCA)
		log.Info(fmt.Sprintf("Update event detected for ca '%s'", oldCA.GetName()))

		if util.CheckIfZoneOrRegionUpdated(oldCA.Spec.Zone, newCA.Spec.Zone) {
			log.Error(errors.New("Zone update is not allowed"), "invalid spec update")
			return false
		}

		if util.CheckIfZoneOrRegionUpdated(oldCA.Spec.Region, newCA.Spec.Region) {
			log.Error(errors.New("Region update is not allowed"), "invalid spec update")
			return false
		}

		if reflect.DeepEqual(oldCA.Spec, newCA.Spec) {
			return false
		}

		update.specUpdated = true

		// Check for changes to ca tag to determine if any migration logic needs to be executed
		if oldCA.Spec.Images != nil && newCA.Spec.Images != nil {
			if oldCA.Spec.Images.CATag != newCA.Spec.Images.CATag {
				log.Info(fmt.Sprintf("CA tag update from %s to %s", oldCA.Spec.Images.CATag, newCA.Spec.Images.CATag))
				update.caTagUpdated = true
			}
		}

		if oldCA.Spec.ConfigOverride == nil {
			if newCA.Spec.ConfigOverride != nil {
				update.caOverridesUpdated = true
				update.tlscaOverridesUpdated = true
			}
		} else {
			if !reflect.DeepEqual(oldCA.Spec.ConfigOverride.CA, newCA.Spec.ConfigOverride.CA) {
				update.caOverridesUpdated = true
			}

			if !reflect.DeepEqual(oldCA.Spec.ConfigOverride.TLSCA, newCA.Spec.ConfigOverride.TLSCA) {
				update.tlscaOverridesUpdated = true
			}
		}

		if newCA.Spec.Action.Restart {
			update.restartNeeded = true
		}

		if newCA.Spec.Action.Renew.TLSCert {
			update.renewTLSCert = true
		}

		update.imagesUpdated = imagesUpdated(oldCA, newCA)
		update.fabricVersionUpdated = fabricVersionUpdated(oldCA, newCA)

		log.Info(fmt.Sprintf("Spec update triggering reconcile on IBPCA custom resource %s: update [ %+v ]", oldCA.Name, update.GetUpdateStackWithTrues()))
		r.PushUpdate(oldCA.GetName(), update)
		return true

	case *corev1.Secret:
		oldSecret := e.ObjectOld.(*corev1.Secret)
		newSecret := e.ObjectNew.(*corev1.Secret)

		if oldSecret.OwnerReferences == nil || len(oldSecret.OwnerReferences) == 0 {
			isCASecret, err := r.AddOwnerReferenceToSecret(oldSecret)
			if err != nil || !isCASecret {
				return false
			}
		}

		if oldSecret.OwnerReferences[0].Kind == KIND {
			if reflect.DeepEqual(oldSecret.Data, newSecret.Data) {
				return false
			}

			instanceName := oldSecret.OwnerReferences[0].Name
			log.Info(fmt.Sprintf("Update event detected for secret '%s'", oldSecret.GetName()))

			if util.IsSecretTLSCert(oldSecret.Name) {
				update.caCryptoUpdated = true
			} else {
				return false
			}

			log.Info(fmt.Sprintf("CA crypto update triggering reconcile on IBPCA custom resource %s: update [ %+v ]", instanceName, update.GetUpdateStackWithTrues()))
			r.PushUpdate(instanceName, update)
			return true
		}

	case *appsv1.Deployment:
		dep := e.ObjectOld.(*appsv1.Deployment)
		log.Info(fmt.Sprintf("Spec update detected by IBPCA controller for deployment '%s'", dep.GetName()))

	case *corev1.ConfigMap:
		cm := e.ObjectOld.(*corev1.ConfigMap)
		if cm.Name == "ca-restart-config" {
			log.Info("Update event detected for ca-restart-config, triggering restart reconcile")
			return true
		}

	}

	return false
}

func (r *ReconcileIBPCA) GetUpdateStatusAtElement(instance *current.IBPCA, index int) *Update {
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

func (r *ReconcileIBPCA) GetUpdateStatus(instance *current.IBPCA) *Update {
	return r.GetUpdateStatusAtElement(instance, 0)
}

func (r *ReconcileIBPCA) PushUpdate(instance string, update Update) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.update[instance] = r.AppendUpdateIfMissing(r.update[instance], update)
}

func (r *ReconcileIBPCA) PopUpdate(instance string) *Update {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	update := Update{}
	if len(r.update[instance]) >= 1 {
		update = r.update[instance][0]
		if len(r.update[instance]) == 1 {
			r.update[instance] = []Update{}
		} else {
			r.update[instance] = r.update[instance][1:]
		}
	}

	return &update
}

func (r *ReconcileIBPCA) AppendUpdateIfMissing(updates []Update, update Update) []Update {
	for _, u := range updates {
		if u == update {
			return updates
		}
	}
	return append(updates, update)
}

func (r *ReconcileIBPCA) AddOwnerReferenceToSecret(secret *corev1.Secret) (bool, error) {
	// CA secrets we are looking to add owner references to are named:
	// <instance name>-ca
	// <instance name>-ca-crypto
	// <instance name>-tlsca
	// <instance name>-tlsca-crypto

	items := strings.Split(secret.Name, "-")
	var instanceName string

	if strings.Contains(secret.Name, "-ca-crypto") || strings.Contains(secret.Name, "-tlsca-crypto") {
		// If -ca-crypto or -tlsca-crypto, construct instance name from all but last 2 items
		instanceName = strings.Join(items[:len(items)-2], "-")
	} else if strings.Contains(secret.Name, "-ca") || strings.Contains(secret.Name, "-tlsca") {
		// If -ca-crypto or -tlsca-crypto, construct instance name from all but last item
		instanceName = strings.Join(items[:len(items)-1], "-")
	} else {
		return false, nil
	}

	listOptions := &client.ListOptions{
		Namespace: secret.Namespace,
	}

	caList := &current.IBPCAList{}
	err := r.client.List(context.TODO(), caList, listOptions)
	if err != nil {
		return false, errors.Wrap(err, "failed to get list of CAs")
	}

	for _, o := range caList.Items {
		ca := o
		if ca.Name == instanceName {
			// Instance 'i' found in list of orderers
			err = r.client.Update(context.TODO(), secret, k8sclient.UpdateOption{
				Owner:  &ca,
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

func (r *ReconcileIBPCA) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&current.IBPCA{}).
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

func (r *ReconcileIBPCA) ReconcileRestart(namespace string) (bool, error) {
	requeue, err := r.RestartService.Reconcile("ca", namespace)
	if err != nil {
		log.Error(err, "failed to reconcile restart queues in ca-restart-config")
		return false, err
	}

	return requeue, nil
}
