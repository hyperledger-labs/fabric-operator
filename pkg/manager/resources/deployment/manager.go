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

package deployment

import (
	"context"
	"fmt"
	"os"
	"regexp"

	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/go-test/deep"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("deployment_manager")

type Manager struct {
	Client            k8sclient.Client
	Scheme            *runtime.Scheme
	DeploymentFile    string
	IgnoreDifferences []string
	Name              string

	LabelsFunc   func(v1.Object) map[string]string
	OverrideFunc func(v1.Object, *appsv1.Deployment, resources.Action) error
}

func (m *Manager) GetName(instance v1.Object) string {
	return GetName(instance.GetName(), m.Name)
}

func (m *Manager) Reconcile(instance v1.Object, update bool) error {
	name := m.GetName(instance)

	deployment := &appsv1.Deployment{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, deployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Creating deployment '%s'", name))
			deployment, err := m.GetDeploymentBasedOnCRFromFile(instance)
			if err != nil {
				return err
			}

			err = m.Client.Create(context.TODO(), deployment, k8sclient.CreateOption{
				Owner:  instance,
				Scheme: m.Scheme,
			})
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if update {
		log.Info(fmt.Sprintf("Updating deployment '%s'", name))
		err = m.OverrideFunc(instance, deployment, resources.Update)
		if err != nil {
			return operatorerrors.New(operatorerrors.InvalidDeploymentUpdateRequest, err.Error())
		}

		err = m.Client.Patch(context.TODO(), deployment, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    3,
				Into:     &appsv1.Deployment{},
				Strategy: client.MergeFrom,
			},
		})
		if err != nil {
			return err
		}

		// Wait for deployment to get updated before returning

		// TODO: Currently commented this out because with the rolling updates (i.e. for console),
		// it takes longer to wait for the new pod to come up and be running and for the
		// old pod to then terminate. Need to figure out how to resolve this.
		// err := wait.Poll(500*time.Millisecond, 30*time.Second, func() (bool, error) {
		// 	upToDate := m.DeploymentIsUpToDate(instance)
		// 	if upToDate {
		// 		return true, nil
		// 	}
		// 	return false, nil
		// })
		// if err != nil {
		// 	return errors.Wrap(err, "failed to determine if deployment was updated")
		// }
	}

	return nil
}

func (m *Manager) GetDeploymentBasedOnCRFromFile(instance v1.Object) (*appsv1.Deployment, error) {
	deployment, err := util.GetDeploymentFromFile(m.DeploymentFile)
	if err != nil {
		log.Error(err, fmt.Sprintf("Error reading deployment configuration file: %s", m.DeploymentFile))
		return nil, err
	}

	return m.BasedOnCR(instance, deployment)
}

func (m *Manager) CheckForSecretChange(instance v1.Object, secretName string, restartFunc func(string, *appsv1.Deployment) bool) error {
	name := m.GetName(instance)

	deployment := &appsv1.Deployment{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, deployment)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	rv, err := util.GetResourceVerFromSecret(m.Client, secretName, instance.GetNamespace())
	if err == nil && rv != "" {
		// Only if secret change is detected do we update deployment env var with new resource version
		changed := restartFunc(rv, deployment)
		if changed {
			log.Info(fmt.Sprintf("Secret '%s' update detected, triggering deployment restart for peer '%s'", secretName, instance.GetName()))
			err = m.Client.Update(context.TODO(), deployment)
			if err != nil {
				return errors.Wrap(err, "failed to update deployment with secret resource version")
			}
		}
	}

	return nil

}

func (m *Manager) BasedOnCR(instance v1.Object, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	if m.OverrideFunc != nil {
		err := m.OverrideFunc(instance, deployment, resources.Create)
		if err != nil {
			return nil, operatorerrors.New(operatorerrors.InvalidDeploymentCreateRequest, err.Error())
		}
	}

	deployment.Name = m.GetName(instance)
	deployment.Namespace = instance.GetNamespace()
	requiredLabels := m.LabelsFunc(instance)
	labels := deployment.Labels
	if len(labels) == 0 {
		labels = make(map[string]string)
	}
	for requiredKey, requiredElement := range requiredLabels {
		labels[requiredKey] = requiredElement
	}
	deployment.Labels = labels
	deployment.Spec.Template.Labels = labels
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: m.getSelectorLabels(instance),
	}

	return deployment, nil
}

func (m *Manager) CheckState(instance v1.Object) error {
	if instance == nil {
		return nil // Instance has not been reconciled yet
	}

	name := m.GetName(instance)

	// Get the latest version of the instance
	deployment := &appsv1.Deployment{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, deployment)
	if err != nil {
		return nil
	}

	copy := deployment.DeepCopy()
	expectedDeployment, err := m.BasedOnCR(instance, copy)
	if err != nil {
		return err
	}

	deep.MaxDepth = 20
	deep.MaxDiff = 30
	deep.CompareUnexportedFields = true
	deep.LogErrors = true

	if os.Getenv("OPERATOR_DEBUG_DISABLEDEPLOYMENTCHECKS") == "true" {
		return nil
	}

	diff := deep.Equal(deployment.Spec, expectedDeployment.Spec)
	if diff != nil {
		err := m.ignoreDifferences(diff)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("deployment (%s) has been edited manually, and does not match what is expected based on the CR", deployment.GetName()))
		}
	}

	return nil
}

func (m *Manager) RestoreState(instance v1.Object) error {
	if instance == nil {
		return nil // Instance has not been reconciled yet
	}

	name := m.GetName(instance)
	deployment := &appsv1.Deployment{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, deployment)
	if err != nil {
		return nil
	}

	deployment, err = m.BasedOnCR(instance, deployment)
	if err != nil {
		return err
	}

	err = m.Client.Patch(context.TODO(), deployment, nil, k8sclient.PatchOption{
		Resilient: &k8sclient.ResilientPatch{
			Retry:    2,
			Into:     &appsv1.Deployment{},
			Strategy: client.MergeFrom,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) Get(instance v1.Object) (client.Object, error) {
	if instance == nil {
		return nil, nil // Instance has not been reconciled yet
	}

	name := m.GetName(instance)
	deployment := &appsv1.Deployment{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: instance.GetNamespace()}, deployment)
	if err != nil {
		return nil, err
	}

	return deployment, nil
}

func (m *Manager) Exists(instance v1.Object) bool {
	dep, err := m.Get(instance)
	if err != nil || dep == nil {
		return false
	}

	return true
}

func (m *Manager) Delete(instance v1.Object) error {
	dep, err := m.Get(instance)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	if dep == nil {
		return nil
	}

	err = m.Client.Delete(context.TODO(), dep)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (m *Manager) getSelectorLabels(instance v1.Object) map[string]string {
	return map[string]string{
		"app": instance.GetName(),
	}
}

func (m *Manager) ignoreDifferences(diff []string) error {
	diffs := []string{}
	for _, d := range diff {
		found := false
		for _, i := range m.differenceToIgnore() {
			regex := regexp.MustCompile(i)
			found = regex.MatchString(d)
			if found {
				break
			}
		}
		if !found {
			diffs = append(diffs, d)
			return fmt.Errorf("unexpected mismatch: %s", d)
		}
	}
	return nil
}

func (m *Manager) differenceToIgnore() []string {
	d := []string{
		"TypeMeta", "ObjectMeta",
		"VolumeSource.Secret.DefaultMode",
		"VolumeSource.ConfigMap.DefaultMode",
		"TerminationMessagePath",
		"TerminationMessagePolicy",
		"SecurityContext.ProcMount",
		"Template.Spec.TerminationGracePeriodSeconds",
		"Template.Spec.DNSPolicy",
		"Template.Spec.DeprecatedServiceAccount",
		"Template.Spec.SchedulerName",
		"RevisionHistoryLimit",
		"RestartPolicy",
		"ProgressDeadlineSeconds",
		"LivenessProbe.SuccessThreshold",
		"LivenessProbe.FailureThreshold",
		"LivenessProbe.InitialDelaySeconds",
		"LivenessProbe.PeriodSeconds",
		"LivenessProbe.TimeoutSeconds",
		"ReadinessProbe.SuccessThreshold",
		"ReadinessProbe.FailureThreshold",
		"ReadinessProbe.InitialDelaySeconds",
		"ReadinessProbe.PeriodSeconds",
		"ReadinessProbe.TimeoutSeconds",
		"StartupProbe.SuccessThreshold",
		"StartupProbe.FailureThreshold",
		"StartupProbe.InitialDelaySeconds",
		"StartupProbe.PeriodSeconds",
		"StartupProbe.TimeoutSeconds",
		"ValueFrom.FieldRef.APIVersion",
		"Template.Spec.Affinity",
		"Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms",
		"Strategy.RollingUpdate",
	}
	d = append(d, m.IgnoreDifferences...)
	return d
}

func (m *Manager) DeploymentIsUpToDate(instance v1.Object) bool {
	deployment := &appsv1.Deployment{}
	err := m.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: m.GetName(instance), Namespace: instance.GetNamespace()},
		deployment,
	)
	if err != nil {
		return false
	}

	if deployment.Status.Replicas > 0 {
		if deployment.Status.Replicas != deployment.Status.UpdatedReplicas {
			return false
		}
	}

	return true
}

func (m *Manager) DeploymentStatus(instance v1.Object) (appsv1.DeploymentStatus, error) {
	deployment := &appsv1.Deployment{}
	err := m.Client.Get(
		context.TODO(),
		types.NamespacedName{Name: m.GetName(instance), Namespace: instance.GetNamespace()},
		deployment,
	)
	if err != nil {
		return appsv1.DeploymentStatus{}, err
	}

	return deployment.Status, nil
}

func (m *Manager) SetCustomName(name string) {
	// NO-OP
}

func (m *Manager) GetScheme() *runtime.Scheme {
	return m.Scheme
}

func GetName(instanceName string, suffix ...string) string {
	if len(suffix) != 0 {
		if suffix[0] != "" {
			return fmt.Sprintf("%s-%s", instanceName, suffix[0])
		}
	}
	return instanceName
}
