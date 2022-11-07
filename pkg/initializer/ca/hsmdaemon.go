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

	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	caconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	controller "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	jobv1 "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/job"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// HSMDaemon implements the ability to initialize HSM Daemon based CA
type HSMDaemon struct {
	Config   *config.HSMConfig
	Scheme   *runtime.Scheme
	Timeouts HSMInitJobTimeouts
	Client   controller.Client
}

// Create creates the crypto and config materical to initialize an HSM based CA
func (h *HSMDaemon) Create(instance *current.IBPCA, overrides *v1.ServerConfig, ca IBPCA) (*Response, error) {
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

	job := h.initHSMCAJob(instance, dbConfig, ca.GetType())
	if err := h.Client.Create(context.TODO(), job.Job, controller.CreateOption{
		Owner:  instance,
		Scheme: h.Scheme,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to create HSM ca initialization job")
	}
	log.Info(fmt.Sprintf("Job '%s' created", job.GetName()))

	if err := job.WaitUntilActive(h.Client); err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Job '%s' active", job.GetName()))

	if err := job.WaitUntilContainerFinished(h.Client, CertGen); err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Job '%s' finished", job.GetName()))

	status, err := job.ContainerStatus(h.Client, CertGen)
	if err != nil {
		return nil, err
	}

	switch status {
	case jobv1.FAILED:
		return nil, fmt.Errorf("Job '%s' finished unsuccessfully, not cleaning up pods to allow for error evaluation", job.GetName())
	case jobv1.COMPLETED:
		// For posterity, job is only deleted if successful, not deleting on failure allows logs to be
		// examined.
		if err := job.Delete(h.Client); err != nil {
			return nil, err
		}
	}

	if ca.GetType().Is(caconfig.EnrollmentCA) {
		if err := updateCAConfigMap(h.Client, h.Scheme, instance, ca); err != nil {
			return nil, errors.Wrapf(err, "failed to update CA configmap for CA %s", instance.GetName())
		}
	}

	return nil, nil
}

const (
	// HSMClient is the name of container that contain the HSM client library
	HSMClient = "hsm-client"
	// CertGen is the name of container that runs the command to generate the certificate for the CA
	CertGen = "certgen"
)

func (h *HSMDaemon) initHSMCAJob(instance *current.IBPCA, dbConfig *v1.CAConfigDB, caType caconfig.Type) *jobv1.Job {
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

	hsmLibraryPath := h.Config.Library.FilePath
	hsmLibraryName := filepath.Base(hsmLibraryPath)

	t := true
	user := int64(1000)
	root := int64(0)
	backoffLimit := int32(0)
	mountPath := "/shared"
	pvcVolumeName := "fabric-ca"

	batchJob := &batchv1.Job{
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
					ImagePullSecrets:   util.AppendImagePullSecretIfMissing(instance.GetPullSecrets(), h.Config.BuildPullSecret()),
					RestartPolicy:      corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						{
							Name:            HSMClient,
							Image:           h.Config.Library.Image,
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"sh",
								"-c",
								fmt.Sprintf("mkdir -p %s/hsm && dst=\"%s/hsm/%s\" && echo \"Copying %s to ${dst}\" && mkdir -p $(dirname $dst) && cp -r %s $dst", mountPath, mountPath, hsmLibraryName, hsmLibraryPath, hsmLibraryPath),
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:    &user,
								RunAsNonRoot: &t,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared",
									MountPath: mountPath,
								},
							},
							Resources: instance.GetResource("init"),
						},
					},
					Containers: []corev1.Container{
						{
							Name: CertGen,
							Image: image.Format(
								instance.Spec.Images.EnrollerImage,
								instance.Spec.Images.EnrollerTag,
							),
							ImagePullPolicy: corev1.PullAlways,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:                &root,
								Privileged:               &t,
								AllowPrivilegeEscalation: &t,
							},
							Command: []string{
								"sh",
								"-c",
							},
							Args: []string{
								fmt.Sprintf(config.DAEMON_CHECK_CMD+" && /usr/local/bin/enroller ca %s %s %s %s %s %s", instance.GetName(), instance.GetNamespace(), homeDir, cryptoMountPath, secretName, caType),
							},
							Env:       h.Config.GetEnvs(),
							Resources: instance.GetResource(current.ENROLLER),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared",
									MountPath: "/hsm/lib",
									SubPath:   "hsm",
								},
								{
									Name:      "shared",
									MountPath: "/shared",
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
						{
							Name: pvcVolumeName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: instance.PVCName(),
								},
							},
						},
					},
				},
			},
		},
	}
	job := jobv1.New(batchJob, &jobv1.Timeouts{
		WaitUntilActive:   h.Timeouts.JobStart.Get(),
		WaitUntilFinished: h.Timeouts.JobCompletion.Get(),
	})

	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, h.Config.GetVolumes()...)
	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, h.Config.GetVolumeMounts()...)

	if dbConfig != nil {
		// If using postgres with TLS enabled need to mount trusted root TLS certificate for database server
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
	}

	// If daemon settings are configured in HSM config, create a sidecar that is running the daemon image
	if h.Config.Daemon != nil {
		// Certain token information requires to be stored in persistent store, the administrator
		// responsible for configuring HSM sets the HSM config to point to the path where the PVC
		// needs to be mounted.
		var pvcMount *corev1.VolumeMount
		for _, vm := range h.Config.MountPaths {
			if vm.UsePVC {
				pvcMount = &corev1.VolumeMount{
					Name:      pvcVolumeName,
					MountPath: vm.MountPath,
				}
			}
		}

		// Add daemon container to the deployment
		config.AddDaemonContainer(h.Config, job, instance.GetResource(current.HSMDAEMON), pvcMount)

		// If a pvc mount has been configured in HSM config, set the volume mount on the CertGen container
		if pvcMount != nil {
			job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, *pvcMount)
		}
	}

	return job
}
