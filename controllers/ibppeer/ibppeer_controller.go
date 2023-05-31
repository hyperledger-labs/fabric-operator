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

package ibppeer

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
	controllerclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering"
	basepeer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	k8speer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/peer"
	openshiftpeer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/peer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/staggerrestarts"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	yaml "sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	KIND = "IBPPeer"
)

var log = logf.Log.WithName("controller_ibppeer")

type CoreConfig interface {
	GetMaxNameLength() *int
}

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
func newReconciler(mgr manager.Manager, cfg *config.Config) (*ReconcileIBPPeer, error) {
	client := controllerclient.New(mgr.GetClient(), &global.ConfigSetter{Config: cfg.Operator.Globals})
	scheme := mgr.GetScheme()

	ibppeer := &ReconcileIBPPeer{
		client:         client,
		scheme:         scheme,
		Config:         cfg,
		update:         map[string][]Update{},
		mutex:          &sync.Mutex{},
		RestartService: staggerrestarts.New(client, cfg.Operator.Restart.Timeout.Get()),
	}

	restClient, err := clientset.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	switch cfg.Offering {
	case offering.K8S:
		ibppeer.Offering = k8speer.New(client, scheme, cfg)
	case offering.OPENSHIFT:
		ibppeer.Offering = openshiftpeer.New(client, scheme, cfg, restClient)
	}

	return ibppeer, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileIBPPeer) error {
	// Create a new controller
	predicateFuncs := predicate.Funcs{
		CreateFunc: r.CreateFunc,
		UpdateFunc: r.UpdateFunc,
		DeleteFunc: r.DeleteFunc,
	}

	c, err := controller.New("ibppeer-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource IBPPeer
	err = c.Watch(&source.Kind{Type: &current.IBPPeer{}}, &handler.EnqueueRequestForObject{}, predicateFuncs)
	if err != nil {
		return err
	}

	// Watch for changes to config maps (Create and Update funcs handle only watching for restart config map)
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}, predicateFuncs)
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner IBPPeer
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &current.IBPPeer{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to tertiary resource Secrets and requeue the owner IBPPeer
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &current.IBPPeer{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to tertiary resource Secrets and requeue the owner IBPPeer
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &current.IBPPeer{},
	}, predicateFuncs)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIBPPeer{}

//go:generate counterfeiter -o mocks/peerreconcile.go -fake-name PeerReconcile . peerReconcile

type peerReconcile interface {
	Reconcile(*current.IBPPeer, basepeer.Update) (common.Result, error)
}

// ReconcileIBPPeer reconciles a IBPPeer object
type ReconcileIBPPeer struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client controllerclient.Client
	scheme *runtime.Scheme

	k8sSecret *corev1.Secret

	Offering       peerReconcile
	Config         *config.Config
	RestartService *staggerrestarts.StaggerRestartsService

	update map[string][]Update
	mutex  *sync.Mutex
}

// Reconcile reads that state of the cluster for a IBPPeer object and makes changes based on the state read
// and what is in the IBPPeer.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIBPPeer) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var err error

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// If peer-restart-config configmap is the object being reconciled, reconcile the
	// restart configmap.
	if request.Name == "peer-restart-config" {
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

	reqLogger.Info("Reconciling IBPPeer")

	// Fetch the IBPPeer instance
	instance := &current.IBPPeer{}
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

	var maxNameLength *int

	co, err := instance.GetConfigOverride()
	if err != nil {
		return reconcile.Result{}, err
	}

	configOverride := co.(CoreConfig)
	maxNameLength = configOverride.GetMaxNameLength()

	err = util.ValidationChecks(instance.TypeMeta, instance.ObjectMeta, "IBPPeer", maxNameLength)
	if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info(fmt.Sprintf("Current update stack to process: %+v", GetUpdateStack(r.update)))

	update := r.GetUpdateStatus(instance)
	reqLogger.Info(fmt.Sprintf("Reconciling IBPPeer '%s' with update values of [ %+v ]", instance.GetName(), update.GetUpdateStackWithTrues()))

	result, err := r.Offering.Reconcile(instance, r.PopUpdate(instance.GetName()))
	setStatusErr := r.SetStatus(instance, result.Status, err)
	if setStatusErr != nil {
		return reconcile.Result{}, operatorerrors.IsBreakingError(setStatusErr, "failed to update status", log)
	}

	if err != nil {
		return reconcile.Result{}, operatorerrors.IsBreakingError(errors.Wrapf(err, "Peer instance '%s' encountered error", instance.GetName()), "stopping reconcile loop", log)
	}

	if result.Requeue {
		r.PushUpdate(instance.GetName(), *update)
	}

	reqLogger.Info(fmt.Sprintf("Finished reconciling IBPPeer '%s' with update values of [ %+v ]", instance.GetName(), update.GetUpdateStackWithTrues()))

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

func (r *ReconcileIBPPeer) SetStatus(instance *current.IBPPeer, reconcileStatus *current.CRStatus, reconcileErr error) error {
	err := r.SaveSpecState(instance)
	if err != nil {
		return errors.Wrap(err, "failed to save spec state")
	}

	// This is get is required but should not be, the reason we need to get the latest instance is because
	// there is code between the reconcile start and SetStatus that ends up updating the instance. Since
	// instance gets updated, but we are still working with original (outdated) version of instance, trying
	// to update it fails with "object as been modified".
	//
	// TODO: Instance should only be updated at the start of reconcile (e.g. PreReconcileChecks), and if is updated
	// the request should be requeued and not processed. The only only time the intance should be updated is in
	// SetStatus
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

		instance.Status = current.IBPPeerStatus{
			CRStatus: status,
		}

		log.Info(fmt.Sprintf("Updating status of IBPPeer custom resource to %s phase", instance.Status.Type))
		err = r.client.PatchStatus(context.TODO(), instance, nil, controllerclient.PatchOption{
			Resilient: &controllerclient.ResilientPatch{
				Retry:    2,
				Into:     &current.IBPPeer{},
				Strategy: k8sclient.MergeFrom,
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

			instance.Status = current.IBPPeerStatus{
				CRStatus: status,
			}

			log.Info(fmt.Sprintf("Updating status of IBPPeer custom resource to %s phase", instance.Status.Type))
			err := r.client.PatchStatus(context.TODO(), instance, nil, controllerclient.PatchOption{
				Resilient: &controllerclient.ResilientPatch{
					Retry:    2,
					Into:     &current.IBPPeer{},
					Strategy: k8sclient.MergeFrom,
				},
			})
			if err != nil {
				return err
			}

			return nil
		}
	}

	running, err := r.GetPodStatus(instance)
	if err != nil {
		return err
	}

	if running {
		if instance.Status.Type == current.Deployed || instance.Status.Type == current.Warning {
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

	instance.Status = current.IBPPeerStatus{
		CRStatus: status,
	}
	instance.Status.LastHeartbeatTime = time.Now().String()
	log.Info(fmt.Sprintf("Updating status of IBPPeer custom resource to %s phase", instance.Status.Type))
	err = r.client.PatchStatus(context.TODO(), instance, nil, controllerclient.PatchOption{
		Resilient: &controllerclient.ResilientPatch{
			Retry:    2,
			Into:     &current.IBPPeer{},
			Strategy: k8sclient.MergeFrom,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileIBPPeer) SaveSpecState(instance *current.IBPPeer) error {
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

func (r *ReconcileIBPPeer) GetSpecState(instance *current.IBPPeer) (*corev1.ConfigMap, error) {
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

func (r *ReconcileIBPPeer) GetPodStatus(instance *current.IBPPeer) (bool, error) {
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

func (r *ReconcileIBPPeer) getIgnoreDiffs() []string {
	return []string{
		`Template\.Spec\.Containers\.slice\[\d\]\.Resources\.Requests\.map\[memory\].s`,
		`Template\.Spec\.InitContainers\.slice\[\d\]\.Resources\.Requests\.map\[memory\].s`,
		`Ports\.slice\[\d\]\.Protocol`,
	}
}

func (r *ReconcileIBPPeer) getSelectorLabels(instance *current.IBPPeer) map[string]string {
	label := os.Getenv("OPERATOR_LABEL_PREFIX")
	if label == "" {
		label = "fabric"
	}

	return map[string]string{
		"app":                          instance.Name,
		"creator":                      label,
		"orgname":                      instance.Spec.MSPID,
		"release":                      "operator",
		"helm.sh/chart":                "ibm-" + label,
		"app.kubernetes.io/name":       label,
		"app.kubernetes.io/instance":   label + "peer",
		"app.kubernetes.io/managed-by": label + "-operator",
	}
}

func (r *ReconcileIBPPeer) CreateFunc(e event.CreateEvent) bool {
	update := Update{}

	switch e.Object.(type) {
	case *current.IBPPeer:
		peer := e.Object.(*current.IBPPeer)
		log.Info(fmt.Sprintf("Create event detected for peer '%s'", peer.GetName()))

		if peer.Status.HasType() {
			cm, err := r.GetSpecState(peer)
			if err != nil {
				log.Info(fmt.Sprintf("Failed getting saved peer spec '%s', can't perform update checks, triggering reconcile: %s", peer.GetName(), err.Error()))
				return true
			}

			specBytes := cm.BinaryData["spec"]
			savedPeer := &current.IBPPeer{}

			err = yaml.Unmarshal(specBytes, &savedPeer.Spec)
			if err != nil {
				log.Info(fmt.Sprintf("Unmarshal failed for saved peer spec '%s', can't perform update checks, triggering reconcile: %s", peer.GetName(), err.Error()))
				return true
			}

			if !reflect.DeepEqual(peer.Spec, savedPeer.Spec) {
				log.Info(fmt.Sprintf("IBPPeer '%s' spec was updated while operator was down", peer.GetName()))
				update.specUpdated = true
			}

			if !reflect.DeepEqual(peer.Spec.ConfigOverride, savedPeer.Spec.ConfigOverride) {
				log.Info(fmt.Sprintf("IBPPeer '%s' overrides were updated while operator was down", peer.GetName()))
				update.overridesUpdated = true
			}

			update.imagesUpdated = imagesUpdated(savedPeer, peer)
			update.fabricVersionUpdated = fabricVersionUpdated(savedPeer, peer)

			log.Info(fmt.Sprintf("Create event triggering reconcile for updating peer '%s'", peer.GetName()))
			r.PushUpdate(peer.GetName(), update)
			return true
		}

		// If creating resource for the first time, check that a unique name is provided
		err := commoncontroller.ValidateCRName(r.client, peer.Name, peer.Namespace, commoncontroller.IBPPEER)
		if err != nil {
			log.Error(err, "failed to validate peer name")
			operror := operatorerrors.Wrap(err, operatorerrors.InvalidCustomResourceCreateRequest, "failed to validate custom resource name")
			err = r.SetStatus(peer, nil, operror)
			if err != nil {
				log.Error(err, "failed to set status to error", "peer.name", peer.Name, "error", "InvalidCustomResourceCreateRequest")
			}
			return false
		}

		log.Info(fmt.Sprintf("Create event triggering reconcile for creating peer '%s'", peer.GetName()))

	case *corev1.Secret:
		secret := e.Object.(*corev1.Secret)

		if secret.OwnerReferences == nil || len(secret.OwnerReferences) == 0 {
			isPeerSecret, err := r.AddOwnerReferenceToSecret(secret)
			if err != nil || !isPeerSecret {
				return false
			}
		}

		if secret.OwnerReferences[0].Kind == KIND {
			log.Info(fmt.Sprintf("Create event detected for secret '%s'", secret.GetName()))
			instanceName := secret.OwnerReferences[0].Name

			if util.IsSecretTLSCert(secret.Name) {
				update.tlsCertCreated = true
				log.Info(fmt.Sprintf("TLS cert create detected on IBPPeer custom resource %s", instanceName))
			} else if util.IsSecretEcert(secret.Name) {
				update.ecertCreated = true
				log.Info(fmt.Sprintf("Ecert create detected on IBPPeer custom resource %s", instanceName))
			} else {
				return false
			}

			log.Info(fmt.Sprintf("Peer crypto create triggering reconcile on IBPPeer custom resource %s: update [ %+v ]", instanceName, update.GetUpdateStackWithTrues()))
			r.PushUpdate(instanceName, update)
		}

	case *appsv1.Deployment:
		dep := e.Object.(*appsv1.Deployment)
		log.Info(fmt.Sprintf("Create event detected by IBPPeer controller for deployment '%s', triggering reconcile", dep.GetName()))
	case *corev1.ConfigMap:
		cm := e.Object.(*corev1.ConfigMap)
		if cm.Name == "peer-restart-config" {
			log.Info(fmt.Sprintf("Create event detected by IBPPeer contoller for config map '%s', triggering restart reconcile", cm.GetName()))
		} else {
			return false
		}

	}

	return true
}

func (r *ReconcileIBPPeer) UpdateFunc(e event.UpdateEvent) bool {
	update := Update{}

	switch e.ObjectOld.(type) {
	case *current.IBPPeer:
		oldPeer := e.ObjectOld.(*current.IBPPeer)
		newPeer := e.ObjectNew.(*current.IBPPeer)
		log.Info(fmt.Sprintf("Update event detected for peer '%s'", oldPeer.GetName()))

		if util.CheckIfZoneOrRegionUpdated(oldPeer.Spec.Zone, newPeer.Spec.Zone) {
			log.Error(errors.New("Zone update is not allowed"), "invalid spec update")
			return false
		}

		if util.CheckIfZoneOrRegionUpdated(oldPeer.Spec.Region, newPeer.Spec.Region) {
			log.Error(errors.New("Region update is not allowed"), "invalid spec update")
			return false
		}

		if reflect.DeepEqual(oldPeer.Spec, newPeer.Spec) {
			return false
		}
		log.Info(fmt.Sprintf("%s spec updated", oldPeer.GetName()))
		update.specUpdated = true

		// Check for changes to peer tag to determine if any migration logic needs to be executed
		// from old peer version to new peer version
		if oldPeer.Spec.Images != nil && newPeer.Spec.Images != nil {
			if oldPeer.Spec.Images.PeerTag != newPeer.Spec.Images.PeerTag {
				log.Info(fmt.Sprintf("Peer tag update from %s to %s", oldPeer.Spec.Images.PeerTag, newPeer.Spec.Images.PeerTag))
				update.peerTagUpdated = true
			}
		}

		if !reflect.DeepEqual(oldPeer.Spec.ConfigOverride, newPeer.Spec.ConfigOverride) {
			log.Info(fmt.Sprintf("%s config override updated", oldPeer.GetName()))
			update.overridesUpdated = true
		}

		update.mspUpdated = commoncontroller.MSPInfoUpdateDetected(oldPeer.Spec.Secret, newPeer.Spec.Secret)

		if newPeer.Spec.Action.Restart == true {
			update.restartNeeded = true
		}

		if oldPeer.Spec.Action.Reenroll.Ecert != newPeer.Spec.Action.Reenroll.Ecert {
			update.ecertReenrollNeeded = newPeer.Spec.Action.Reenroll.Ecert
		}

		if oldPeer.Spec.Action.Reenroll.TLSCert != newPeer.Spec.Action.Reenroll.TLSCert {
			update.tlsReenrollNeeded = newPeer.Spec.Action.Reenroll.TLSCert
		}

		if oldPeer.Spec.Action.Reenroll.EcertNewKey != newPeer.Spec.Action.Reenroll.EcertNewKey {
			update.ecertNewKeyReenroll = newPeer.Spec.Action.Reenroll.EcertNewKey
		}

		if oldPeer.Spec.Action.Reenroll.TLSCertNewKey != newPeer.Spec.Action.Reenroll.TLSCertNewKey {
			update.tlscertNewKeyReenroll = newPeer.Spec.Action.Reenroll.TLSCertNewKey
		}

		oldVer := version.String(oldPeer.Spec.FabricVersion)
		newVer := version.String(newPeer.Spec.FabricVersion)

		// check if this V1 -> V2.2.x / V2.4.x peer migration
		if (oldPeer.Spec.FabricVersion == "" ||
			version.GetMajorReleaseVersion(oldPeer.Spec.FabricVersion) == version.V1) &&
			version.GetMajorReleaseVersion(newPeer.Spec.FabricVersion) == version.V2 {
			update.migrateToV2 = true
			if newVer.EqualWithoutTag(version.V2_5_1) || newVer.GreaterThan(version.V2_5_1) {
				update.migrateToV24 = true
				update.migrateToV25 = true
			} else if newVer.EqualWithoutTag(version.V2_4_1) || newVer.GreaterThan(version.V2_4_1) {
				update.migrateToV24 = true
			}
		}

		// check if this V2.2.x -> V2.4.x/V2.5.x peer migration
		if (version.GetMajorReleaseVersion(oldPeer.Spec.FabricVersion) == version.V2) &&
			oldVer.LessThan(version.V2_4_1) {
			update.migrateToV24 = true
			if newVer.EqualWithoutTag(version.V2_5_1) || newVer.GreaterThan(version.V2_5_1) {
				update.migrateToV25 = true
			}
		}

		// check if this V2.4.x -> V2.5.x peer migration
		if (version.GetMajorReleaseVersion(oldPeer.Spec.FabricVersion) == version.V2) &&
			oldVer.LessThan(version.V2_5_1) {
			if newVer.EqualWithoutTag(version.V2_5_1) || newVer.GreaterThan(version.V2_5_1) {
				update.migrateToV25 = true
			}
		}

		if newPeer.Spec.Action.UpgradeDBs == true {
			update.upgradedbs = true
		}

		if newPeer.Spec.Action.Enroll.Ecert == true {
			update.ecertEnroll = true
		}

		if newPeer.Spec.Action.Enroll.TLSCert == true {
			update.tlscertEnroll = true
		}

		if oldPeer.Spec.NodeOUDisabled() != newPeer.Spec.NodeOUDisabled() {
			update.nodeOUUpdated = true
		}

		// if use updates NumSecondsWarningPeriod field once we have already run the reconcile
		// we need to retrigger the timer logic
		if oldPeer.Spec.NumSecondsWarningPeriod != newPeer.Spec.NumSecondsWarningPeriod {
			update.ecertUpdated = true
			update.tlsCertUpdated = true
			log.Info(fmt.Sprintf("%s NumSecondsWarningPeriod updated", oldPeer.Name))
		}

		update.imagesUpdated = imagesUpdated(oldPeer, newPeer)
		update.fabricVersionUpdated = fabricVersionUpdated(oldPeer, newPeer)

		log.Info(fmt.Sprintf("Spec update triggering reconcile on IBPPeer custom resource %s, update [ %+v ]", oldPeer.Name, update.GetUpdateStackWithTrues()))
		r.PushUpdate(oldPeer.Name, update)
		return true

	case *corev1.Secret:
		oldSecret := e.ObjectOld.(*corev1.Secret)
		newSecret := e.ObjectNew.(*corev1.Secret)

		if oldSecret.OwnerReferences == nil || len(oldSecret.OwnerReferences) == 0 {
			isPeerSecret, err := r.AddOwnerReferenceToSecret(oldSecret)
			if err != nil || !isPeerSecret {
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
				log.Info(fmt.Sprintf("TLS cert update detected on IBPPeer custom resource %s", instanceName))
			} else if util.IsSecretEcert(oldSecret.Name) {
				update.ecertUpdated = true
				log.Info(fmt.Sprintf("ecert update detected on IBPPeer custom resource %s", instanceName))
			} else {
				return false
			}

			log.Info(fmt.Sprintf("Peer crypto update triggering reconcile on IBPPeer custom resource %s: update [ %+v ]", instanceName, update.GetUpdateStackWithTrues()))
			r.PushUpdate(instanceName, update)
			return true
		}

	case *appsv1.Deployment:
		oldDeployment := e.ObjectOld.(*appsv1.Deployment)
		log.Info(fmt.Sprintf("Spec update detected by IBPPeer controller on deployment '%s'", oldDeployment.GetName()))

	case *corev1.ConfigMap:
		cm := e.ObjectOld.(*corev1.ConfigMap)
		if cm.Name == "peer-restart-config" {
			log.Info("Update event detected for peer-restart-config, triggering restart reconcile")
			return true
		}

	}

	return false
}

// DeleteFunc will perform any necessary clean up, such as removing artificates that were
// left dangling after the deletion of the peer resource
func (r *ReconcileIBPPeer) DeleteFunc(e event.DeleteEvent) bool {
	switch e.Object.(type) {
	case *current.IBPPeer:
		peer := e.Object.(*current.IBPPeer)
		log.Info(fmt.Sprintf("Peer (%s) deleted", peer.GetName()))

		// Deleting this config map manually, in 2.5.1 release of operator this config map was created
		// without proper controller references set and was not cleaned up on peer resource deletion.
		log.Info(fmt.Sprintf("Deleting %s-init-config config map, if found", peer.GetName()))
		if err := r.client.Delete(context.TODO(), &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      fmt.Sprintf("%s-init-config", peer.GetName()),
				Namespace: peer.GetNamespace(),
			},
		}); client.IgnoreNotFound(err) != nil {
			log.Info(fmt.Sprintf("failed to delete config map: %s", err))
		}

	case *appsv1.Deployment:
		dep := e.Object.(*appsv1.Deployment)
		log.Info(fmt.Sprintf("Delete detected by IBPPeer controller on deployment '%s'", dep.GetName()))
	case *corev1.Secret:
		secret := e.Object.(*corev1.Secret)
		log.Info(fmt.Sprintf("Delete detected by IBPPeer controller on secret '%s'", secret.GetName()))
	case *corev1.ConfigMap:
		cm := e.Object.(*corev1.ConfigMap)
		log.Info(fmt.Sprintf("Delete detected by IBPPeer controller on configmap '%s'", cm.GetName()))

	}

	return true
}

func (r *ReconcileIBPPeer) GetUpdateStatusAtElement(instance *current.IBPPeer, index int) *Update {
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

func (r *ReconcileIBPPeer) GetUpdateStatus(instance *current.IBPPeer) *Update {
	return r.GetUpdateStatusAtElement(instance, 0)
}

func (r *ReconcileIBPPeer) PushUpdate(instanceName string, update Update) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.update[instanceName] = r.AppendUpdateIfMissing(r.update[instanceName], update)
}

func (r *ReconcileIBPPeer) PopUpdate(instanceName string) *Update {
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

func (r *ReconcileIBPPeer) AppendUpdateIfMissing(updates []Update, update Update) []Update {
	for _, u := range updates {
		if u == update {
			return updates
		}
	}
	return append(updates, update)
}

func (r *ReconcileIBPPeer) AddOwnerReferenceToSecret(secret *corev1.Secret) (bool, error) {
	// Peer secrets we are looking to add owner references to are named:
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

	peerList := &current.IBPPeerList{}
	err := r.client.List(context.TODO(), peerList, listOptions)
	if err != nil {
		return false, errors.Wrap(err, "failed to get list of peers")
	}

	for _, o := range peerList.Items {
		peer := o
		if peer.Name == instanceName {
			// Instance 'i' found in list of orderers
			err := r.client.Update(context.TODO(), secret, controllerclient.UpdateOption{
				Owner:  &peer,
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

func (r *ReconcileIBPPeer) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&current.IBPPeer{}).
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

func (r *ReconcileIBPPeer) ReconcileRestart(namespace string) (bool, error) {
	requeue, err := r.RestartService.Reconcile("peer", namespace)
	if err != nil {
		log.Error(err, "failed to reconcile restart queues in peer-restart-config")
		return false, err
	}

	return requeue, nil
}
