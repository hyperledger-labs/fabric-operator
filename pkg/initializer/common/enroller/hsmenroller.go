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

package enroller

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric-ca/lib"
	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	jobv1 "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/job"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

//go:generate counterfeiter -o mocks/instance.go -fake-name Instance . Instance
type Instance interface {
	metav1.Object
	EnrollerImage() string
	GetPullSecrets() []corev1.LocalObjectReference
	PVCName() string
	GetResource(current.Component) corev1.ResourceRequirements
}

//go:generate counterfeiter -o mocks/hsmcaclient.go -fake-name HSMCAClient . HSMCAClient
type HSMCAClient interface {
	GetEnrollmentRequest() *current.Enrollment
	GetHomeDir() string
	PingCA(time.Duration) error
	SetHSMLibrary(string)
	GetConfig() *lib.ClientConfig
}

type HSMEnrollJobTimeouts struct {
	JobStart      common.Duration `json:"jobStart" yaml:"jobStart"`
	JobCompletion common.Duration `json:"jobCompletion" yaml:"jobCompletion"`
}

type HSMEnroller struct {
	CAClient HSMCAClient
	Client   k8sclient.Client
	Instance Instance
	Timeouts HSMEnrollJobTimeouts
	Scheme   *runtime.Scheme
	Config   *config.HSMConfig
}

func NewHSMEnroller(cfg *current.Enrollment, instance Instance, caclient HSMCAClient, client k8sclient.Client, scheme *runtime.Scheme, timeouts HSMEnrollJobTimeouts, hsmConfig *config.HSMConfig) *HSMEnroller {
	return &HSMEnroller{
		CAClient: caclient,
		Client:   client,
		Instance: instance,
		Scheme:   scheme,
		Timeouts: timeouts,
		Config:   hsmConfig,
	}
}

func (e *HSMEnroller) GetEnrollmentRequest() *current.Enrollment {
	return e.CAClient.GetEnrollmentRequest()
}

func (e *HSMEnroller) ReadKey() ([]byte, error) {
	return nil, nil
}

func (e *HSMEnroller) PingCA(timeout time.Duration) error {
	return e.CAClient.PingCA(timeout)
}

func (e *HSMEnroller) Enroll() (*config.Response, error) {
	// Deleting CA client config is an unfortunate requirement since the ca client
	// config map was not properly deleted after a successfull reenrollment request.
	// This is problematic when recreating a resource with same name, as it will
	// try to use old settings in the config map, which might no longer apply, thus
	// it must be removed if found before proceeding.
	if err := deleteCAClientConfig(e.Client, e.Instance); err != nil {
		return nil, err
	}

	e.CAClient.SetHSMLibrary(filepath.Join("/hsm/lib", filepath.Base(e.Config.Library.FilePath)))
	if err := createRootTLSSecret(e.Client, e.CAClient, e.Scheme, e.Instance); err != nil {
		return nil, err
	}

	if err := createCAClientConfig(e.Client, e.CAClient, e.Scheme, e.Instance); err != nil {
		return nil, err
	}

	job := e.initHSMJob(e.Instance, e.Timeouts)
	if err := e.Client.Create(context.TODO(), job.Job, k8sclient.CreateOption{
		Owner:  e.Instance,
		Scheme: e.Scheme,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to create HSM ca initialization job")
	}
	log.Info(fmt.Sprintf("Job '%s' created", job.GetName()))

	if err := job.WaitUntilActive(e.Client); err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Job '%s' active", job.GetName()))

	if err := job.WaitUntilFinished(e.Client); err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Job '%s' finished", job.GetName()))

	status, err := job.Status(e.Client)
	if err != nil {
		return nil, err
	}

	switch status {
	case jobv1.FAILED:
		return nil, fmt.Errorf("Job '%s' finished unsuccessfully, not cleaning up pods to allow for error evaluation", job.GetName())
	case jobv1.COMPLETED:
		if err := job.Delete(e.Client); err != nil {
			return nil, err
		}

		if err := deleteRootTLSSecret(e.Client, e.Instance); err != nil {
			return nil, err
		}

		if err := deleteCAClientConfig(e.Client, e.Instance); err != nil {
			return nil, err
		}
	}

	name := fmt.Sprintf("ecert-%s-signcert", e.Instance.GetName())
	err = wait.Poll(2*time.Second, 30*time.Second, func() (bool, error) {
		sec := &corev1.Secret{}
		log.Info(fmt.Sprintf("Waiting for secret '%s' to be created", name))
		err = e.Client.Get(context.TODO(), types.NamespacedName{
			Name:      name,
			Namespace: e.Instance.GetNamespace(),
		}, sec)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret '%s'", name)
	}

	if err := setControllerReferences(e.Client, e.Scheme, e.Instance); err != nil {
		return nil, err
	}

	return &config.Response{}, nil
}

func setControllerReferences(client k8sclient.Client, scheme *runtime.Scheme, instance Instance) error {
	if err := setControllerReferenceFor(fmt.Sprintf("ecert-%s-signcert", instance.GetName()), false, client, scheme, instance); err != nil {
		return err
	}

	if err := setControllerReferenceFor(fmt.Sprintf("ecert-%s-cacerts", instance.GetName()), false, client, scheme, instance); err != nil {
		return err
	}

	if err := setControllerReferenceFor(fmt.Sprintf("ecert-%s-admincerts", instance.GetName()), true, client, scheme, instance); err != nil {
		return err
	}

	if err := setControllerReferenceFor(fmt.Sprintf("ecert-%s-intercerts", instance.GetName()), true, client, scheme, instance); err != nil {
		return err
	}

	return nil
}

func setControllerReferenceFor(name string, skipIfNotFound bool, client k8sclient.Client, scheme *runtime.Scheme, instance Instance) error {
	nn := types.NamespacedName{
		Name:      name,
		Namespace: instance.GetNamespace(),
	}

	sec := &corev1.Secret{}
	if err := client.Get(context.TODO(), nn, sec); err != nil {
		if skipIfNotFound {
			return nil
		}

		return err
	}

	if err := client.Update(context.TODO(), sec, k8sclient.UpdateOption{
		Owner:  instance,
		Scheme: scheme,
	}); err != nil {
		return errors.Wrapf(err, "failed to update secret '%s' with controller reference", instance.GetName())
	}

	return nil
}

func createRootTLSSecret(client k8sclient.Client, caClient HSMCAClient, scheme *runtime.Scheme, instance Instance) error {
	tlsCertBytes, err := caClient.GetEnrollmentRequest().GetCATLSBytes()
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-init-roottls", instance.GetName()),
			Namespace: instance.GetNamespace(),
		},
		Data: map[string][]byte{
			"tlsCert.pem": tlsCertBytes,
		},
	}

	if err := client.Create(context.TODO(), secret, k8sclient.CreateOption{
		Owner:  instance,
		Scheme: scheme,
	}); err != nil {
		return errors.Wrap(err, "failed to create secret")
	}

	return nil
}

func deleteRootTLSSecret(client k8sclient.Client, instance Instance) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-init-roottls", instance.GetName()),
			Namespace: instance.GetNamespace(),
		},
	}

	if err := client.Delete(context.TODO(), secret); err != nil {
		return errors.Wrap(err, "failed to delete secret")
	}

	return nil
}

func createCAClientConfig(client k8sclient.Client, caClient HSMCAClient, scheme *runtime.Scheme, instance Instance) error {
	configBytes, err := yaml.Marshal(caClient.GetConfig())
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-init-config", instance.GetName()),
			Namespace: instance.GetNamespace(),
		},
		BinaryData: map[string][]byte{
			"fabric-ca-client-config.yaml": configBytes,
		},
	}

	if err := client.Create(context.TODO(), cm, k8sclient.CreateOption{
		Owner:  instance,
		Scheme: scheme,
	}); err != nil {
		return errors.Wrap(err, "failed to create config map")
	}

	return nil
}

func deleteCAClientConfig(k8sClient k8sclient.Client, instance Instance) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-init-config", instance.GetName()),
			Namespace: instance.GetNamespace(),
		},
	}

	if err := k8sClient.Delete(context.TODO(), cm); client.IgnoreNotFound(err) != nil {
		return errors.Wrap(err, "failed to delete config map")
	}

	return nil
}

func (e *HSMEnroller) initHSMJob(instance Instance, timeouts HSMEnrollJobTimeouts) *jobv1.Job {
	hsmConfig := e.Config
	req := e.CAClient.GetEnrollmentRequest()

	hsmLibraryPath := hsmConfig.Library.FilePath
	hsmLibraryName := filepath.Base(hsmLibraryPath)

	jobName := fmt.Sprintf("%s-enroll", instance.GetName())

	f := false
	user := int64(0)
	backoffLimit := int32(0)
	mountPath := "/shared"

	k8sJob := &batchv1.Job{
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
							Name:            "init",
							Image:           instance.EnrollerImage(),
							ImagePullPolicy: corev1.PullAlways,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:    &user,
								RunAsNonRoot: &f,
							},
							Env: hsmConfig.GetEnvs(),
							Command: []string{
								"sh",
								"-c",
								fmt.Sprintf("/usr/local/bin/enroller node enroll %s %s %s %s %s %s %s %s %s", e.CAClient.GetHomeDir(), "/tmp/fabric-ca-client-config.yaml", req.CAHost, req.CAPort, req.CAName, instance.GetName(), instance.GetNamespace(), req.EnrollID, req.EnrollSecret),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tlscertfile",
									MountPath: fmt.Sprintf("%s/tlsCert.pem", e.CAClient.GetHomeDir()),
									SubPath:   "tlsCert.pem",
								},
								{
									Name:      "clientconfig",
									MountPath: fmt.Sprintf("/tmp/%s", "fabric-ca-client-config.yaml"),
									SubPath:   "fabric-ca-client-config.yaml",
								},
								{
									Name:      "shared",
									MountPath: "/hsm/lib",
									SubPath:   "hsm",
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
							Name: "tlscertfile",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: fmt.Sprintf("%s-init-roottls", instance.GetName()),
								},
							},
						},
						{
							Name: "clientconfig",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-init-config", instance.GetName()),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	job := jobv1.New(k8sJob, &jobv1.Timeouts{
		WaitUntilActive:   timeouts.JobStart.Get(),
		WaitUntilFinished: timeouts.JobCompletion.Get(),
	})

	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, hsmConfig.GetVolumes()...)
	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, hsmConfig.GetVolumeMounts()...)

	return job
}
