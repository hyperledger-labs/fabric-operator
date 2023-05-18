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

package initializer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	caconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	controller "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// HSMInitJobTimeouts defines timeouts properties
type HSMInitJobTimeouts struct {
	JobStart      common.Duration `json:"jobStart" yaml:"jobStart"`
	JobCompletion common.Duration `json:"jobCompletion" yaml:"jobCompletion"`
}

// HSM implements the ability to initialize HSM CA
type HSM struct {
	Config   *config.HSMConfig
	Timeouts HSMInitJobTimeouts
	Client   controller.Client
	Scheme   *runtime.Scheme
}

// Create creates the crypto and config materical to initialize an HSM based CA
func (h *HSM) Create(instance *current.IBPCA, overrides *v1.ServerConfig, ca IBPCA) (*Response, error) {
	log.Info(fmt.Sprintf("Creating job to initialize ca '%s'", instance.GetName()))

	if err := ca.OverrideServerConfig(overrides); err != nil {
		return nil, err
	}

	if err := createCACryptoSecret(h.Client, h.Scheme, instance, ca); err != nil {
		return nil, err
	}

	if err := createCAConfigMap(h.Client, h.Scheme, instance, h.Config.Library.FilePath, ca); err != nil {
		return nil, err
	}

	dbConfig, err := getDBConfig(instance, ca.GetType())
	if err != nil {
		return nil, errors.Wrapf(err, "failed get DB config for CA '%s'", instance.GetName())
	}

	job := initHSMCAJob(instance, h.Config, dbConfig, ca.GetType())
	setPathsOnJob(h.Config, job)

	if err := h.Client.Create(context.TODO(), job, controller.CreateOption{
		Owner:  instance,
		Scheme: h.Scheme,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to create HSM ca initialization job")
	}

	log.Info(fmt.Sprintf("Job '%s' created", job.GetName()))

	// Wait for job to start and pod to go into running state
	if err := h.waitForJobToBeActive(job); err != nil {
		return nil, err
	}

	status, err := h.waitForJobPodToFinish(job)
	if err != nil {
		return nil, err
	}

	log.Info(fmt.Sprintf("Job '%s' finished", job.GetName()))

	if status.Phase != corev1.PodSucceeded {
		return nil, fmt.Errorf("failed to init '%s' check job '%s' pods for errors", instance.GetName(), job.GetName())
	}

	// For posterity, job is only deleted if successful, not deleting on failure allows logs to be
	// available for review.
	//
	// Don't need to cleanup/delete CACrypto Secret and CAConfig config map created earlier,
	// as the job will update these resources.
	if err := h.deleteJob(job); err != nil {
		return nil, err
	}

	if ca.GetType().Is(caconfig.EnrollmentCA) {
		if err := updateCAConfigMap(h.Client, h.Scheme, instance, ca); err != nil {
			return nil, errors.Wrapf(err, "failed to update CA configmap for CA %s", instance.GetName())
		}
	}

	return nil, nil
}

func createCACryptoSecret(client controller.Client, scheme *runtime.Scheme, instance *current.IBPCA, ca IBPCA) error {
	crypto, err := ca.ParseCrypto()
	if err != nil {
		return err
	}

	var name string
	switch ca.GetType() {
	case caconfig.EnrollmentCA:
		name = fmt.Sprintf("%s-ca-crypto", instance.GetName())
	case caconfig.TLSCA:
		name = fmt.Sprintf("%s-tlsca-crypto", instance.GetName())
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.GetNamespace(),
		},
		Data: crypto,
	}

	if err := client.Create(context.TODO(), secret, controller.CreateOption{
		Owner:  instance,
		Scheme: scheme,
	}); err != nil {
		return errors.Wrap(err, "failed to create initialization crypto secret")
	}

	return nil
}

func createCAConfigMap(client controller.Client, scheme *runtime.Scheme, instance *current.IBPCA, library string, ca IBPCA) error {
	serverConfig := ca.GetServerConfig()
	serverConfig.CAConfig.CSP.PKCS11.Library = filepath.Join("/hsm/lib", filepath.Base(library))

	ca.SetMountPaths()
	configBytes, err := ca.ConfigToBytes()
	if err != nil {
		return err
	}

	var name string
	switch ca.GetType() {
	case caconfig.EnrollmentCA:
		name = fmt.Sprintf("%s-ca-config", instance.GetName())
	case caconfig.TLSCA:
		name = fmt.Sprintf("%s-tlsca-config", instance.GetName())
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "IBPCA",
					APIVersion: "ibp.com/v1beta1",
					Name:       instance.GetName(),
					UID:        instance.GetUID(),
				},
			},
		},
		BinaryData: map[string][]byte{
			"fabric-ca-server-config.yaml": configBytes,
		},
	}

	if err := client.Create(context.TODO(), cm, controller.CreateOption{
		Owner:  instance,
		Scheme: scheme,
	}); err != nil {
		return errors.Wrap(err, "failed to create initialization config map secret")
	}

	return nil
}

func updateCAConfigMap(client controller.Client, scheme *runtime.Scheme, instance *current.IBPCA, ca IBPCA) error {
	serverConfig := ca.GetServerConfig()
	serverConfig.CAfiles = []string{"/data/tlsca/fabric-ca-server-config.yaml"}

	configBytes, err := ca.ConfigToBytes()
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s-ca-config", instance.GetName())

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "IBPCA",
					APIVersion: "ibp.com/v1beta1",
					Name:       instance.GetName(),
					UID:        instance.GetUID(),
				},
			},
		},
		BinaryData: map[string][]byte{
			"fabric-ca-server-config.yaml": configBytes,
		},
	}

	if err := client.Update(context.TODO(), cm, controller.UpdateOption{
		Owner:  instance,
		Scheme: scheme,
	}); err != nil {
		return errors.Wrapf(err, "failed to update config map '%s'", name)
	}

	return nil
}

func (h *HSM) waitForJobToBeActive(job *batchv1.Job) error {
	err := wait.Poll(2*time.Second, h.Timeouts.JobStart.Duration, func() (bool, error) {
		log.Info(fmt.Sprintf("Waiting for job '%s' to start", job.GetName()))

		j := &batchv1.Job{}
		err := h.Client.Get(context.TODO(), types.NamespacedName{
			Name:      job.GetName(),
			Namespace: job.GetNamespace(),
		}, j)
		if err != nil {
			return false, nil
		}

		if j.Status.Active >= int32(1) {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "job failed to start")
	}
	return nil
}

func (h *HSM) waitForJobPodToFinish(job *batchv1.Job) (*corev1.PodStatus, error) {
	var err error
	var status *corev1.PodStatus

	err = wait.Poll(2*time.Second, h.Timeouts.JobCompletion.Duration, func() (bool, error) {
		log.Info(fmt.Sprintf("Waiting for job pod '%s' to finish", job.GetName()))

		status, err = h.podStatus(job)
		if err != nil {
			log.Info(fmt.Sprintf("job pod err: %s", err))
			return false, nil
		}

		if status.Phase == corev1.PodFailed || status.Phase == corev1.PodSucceeded {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "pod for job '%s' failed to finish", job.GetName())
	}

	return status, nil
}

func (h *HSM) podStatus(job *batchv1.Job) (*corev1.PodStatus, error) {
	labelSelector, err := labels.Parse(fmt.Sprintf("job-name=%s", job.GetName()))
	if err != nil {
		return nil, err
	}

	opts := &k8sclient.ListOptions{
		LabelSelector: labelSelector,
	}

	pods := &corev1.PodList{}
	if err := h.Client.List(context.TODO(), pods, opts); err != nil {
		return nil, err
	}

	if len(pods.Items) != 1 {
		return nil, errors.New("incorrect number of job pods found")
	}

	for _, pod := range pods.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil || containerStatus.State.Running != nil {
				return &pod.Status, nil
			}
		}

		return &pod.Status, nil
	}

	return nil, errors.New("unable to get pod status")
}

func (h *HSM) deleteJob(job *batchv1.Job) error {
	if err := h.Client.Delete(context.TODO(), job); err != nil {
		return err
	}

	// TODO: Need to investigate why job is not adding controller reference to job pod,
	// this manual cleanup should not be required
	podList := &corev1.PodList{}
	if err := h.Client.List(context.TODO(), podList, k8sclient.MatchingLabels{"job-name": job.Name}); err != nil {
		return errors.Wrap(err, "failed to list job pods")
	}

	for _, pod := range podList.Items {
		podListItem := pod
		if err := h.Client.Delete(context.TODO(), &podListItem); err != nil {
			return errors.Wrapf(err, "failed to delete pod '%s'", podListItem.Name)
		}
	}

	return nil
}

func setPathsOnJob(hsmConfig *config.HSMConfig, job *batchv1.Job) {
	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, hsmConfig.GetVolumes()...)
	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, hsmConfig.GetVolumeMounts()...)
}

func getDBConfig(instance *current.IBPCA, caType caconfig.Type) (*v1.CAConfigDB, error) {
	var rawMessage *[]byte
	switch caType {
	case caconfig.EnrollmentCA:
		if instance.Spec.ConfigOverride != nil && instance.Spec.ConfigOverride.CA != nil {
			rawMessage = &instance.Spec.ConfigOverride.CA.Raw
		}
	case caconfig.TLSCA:
		if instance.Spec.ConfigOverride != nil && instance.Spec.ConfigOverride.TLSCA != nil {
			rawMessage = &instance.Spec.ConfigOverride.TLSCA.Raw
		}
	}

	if rawMessage == nil {
		return &v1.CAConfigDB{}, nil
	}

	caOverrides := &v1.ServerConfig{}
	err := yaml.Unmarshal(*rawMessage, caOverrides)
	if err != nil {
		return nil, err
	}

	return caOverrides.CAConfig.DB, nil
}

func initHSMCAJob(instance *current.IBPCA, hsmConfig *config.HSMConfig, dbConfig *v1.CAConfigDB, caType caconfig.Type) *batchv1.Job {
	var typ string

	switch caType {
	case caconfig.EnrollmentCA:
		typ = "ca"
	case caconfig.TLSCA:
		typ = "tlsca"
	}

	cryptoMountPath := fmt.Sprintf("/crypto/%s", typ)
	homeDir := fmt.Sprintf("/tmp/data/%s/%s", instance.GetName(), typ)
	secretName := fmt.Sprintf("%s-%s-crypto", instance.GetName(), typ)
	jobName := fmt.Sprintf("%s-%s-init", instance.GetName(), typ)

	hsmLibraryPath := hsmConfig.Library.FilePath
	hsmLibraryName := filepath.Base(hsmLibraryPath)

	f := false
	user := int64(0)
	backoffLimit := int32(0)
	mountPath := "/shared"
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: instance.GetNamespace(),
			Labels: map[string]string{
				"name":  jobName,
				"owner": instance.GetName(),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: instance.GetName(),
					ImagePullSecrets:   util.AppendImagePullSecretIfMissing(instance.GetPullSecrets(), hsmConfig.BuildPullSecret()),
					RestartPolicy:      corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						{
							Name:            "hsm-client",
							Image:           hsmConfig.Library.Image,
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
								{
									Name:      "shared",
									MountPath: mountPath,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:              resource.MustParse("0.1"),
									corev1.ResourceMemory:           resource.MustParse("100Mi"),
									corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:              resource.MustParse("1"),
									corev1.ResourceMemory:           resource.MustParse("500Mi"),
									corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "init",
							Image: image.Format(
								instance.Spec.Images.EnrollerImage,
								instance.Spec.Images.EnrollerTag,
							),
							ImagePullPolicy: corev1.PullAlways,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:    &user,
								RunAsNonRoot: &f,
							},
							Command: []string{
								"sh",
								"-c",
								fmt.Sprintf("/usr/local/bin/enroller ca %s %s %s %s %s %s", instance.GetName(), instance.GetNamespace(), homeDir, cryptoMountPath, secretName, caType),
							},
							Env: hsmConfig.GetEnvs(),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared",
									MountPath: "/hsm/lib",
									SubPath:   "hsm",
								},
								{
									Name:      "caconfig",
									MountPath: fmt.Sprintf("/tmp/data/%s/%s/fabric-ca-server-config.yaml", instance.GetName(), typ),
									SubPath:   "fabric-ca-server-config.yaml",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "shared",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: corev1.StorageMediumMemory,
								},
							},
						},
						{
							Name: "caconfig",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-%s-config", instance.GetName(), typ),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if dbConfig == nil {
		return job
	}

	// If using postgres with TLS enabled need to mount trusted root TLS certificate for databae server
	if strings.ToLower(dbConfig.Type) == "postgres" {
		if dbConfig.TLS.IsEnabled() {
			job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts,
				corev1.VolumeMount{
					Name:      "cacrypto",
					MountPath: fmt.Sprintf("/crypto/%s/db-certfile0.pem", typ),
					SubPath:   "db-certfile0.pem",
				})

			job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes,
				corev1.Volume{
					Name: "cacrypto",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: fmt.Sprintf("%s-%s-crypto", instance.GetName(), typ),
							Items: []corev1.KeyToPath{
								{
									Key:  "db-certfile0.pem",
									Path: "db-certfile0.pem",
								},
							},
						},
					},
				},
			)
		}
	}

	return job
}
