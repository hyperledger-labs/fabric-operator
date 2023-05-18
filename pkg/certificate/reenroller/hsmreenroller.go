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

package reenroller

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric-ca/lib"
	"github.com/hyperledger/fabric-ca/lib/tls"
	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
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

type Instance interface {
	metav1.Object
	EnrollerImage() string
	GetPullSecrets() []corev1.LocalObjectReference
	GetResource(current.Component) corev1.ResourceRequirements
	PVCName() string
}

type HSMReenroller struct {
	CAClient *lib.Client
	Identity Identity

	HomeDir   string
	Config    *current.Enrollment
	BCCSP     bool
	Timeout   time.Duration
	HSMConfig *config.HSMConfig
	Instance  Instance
	Client    k8sclient.Client
	Scheme    *runtime.Scheme
	NewKey    bool
}

func NewHSMReenroller(cfg *current.Enrollment, homeDir string, bccsp *commonapi.BCCSP, timeoutstring string, hsmConfig *config.HSMConfig, instance Instance, client k8sclient.Client, scheme *runtime.Scheme, newKey bool) (*HSMReenroller, error) {
	if cfg == nil {
		return nil, errors.New("unable to reenroll, Enrollment config must be passed")
	}

	err := EnrollmentConfigValidation(cfg)
	if err != nil {
		return nil, err
	}

	caclient := &lib.Client{
		HomeDir: homeDir,
		Config: &lib.ClientConfig{
			TLS: tls.ClientTLSConfig{
				Enabled:   true,
				CertFiles: []string{"tlsCert.pem"},
			},
			URL: fmt.Sprintf("https://%s:%s", cfg.CAHost, cfg.CAPort),
		},
	}

	bccsp.PKCS11.Library = filepath.Join("/hsm/lib", filepath.Base(hsmConfig.Library.FilePath))

	caclient = GetClient(caclient, bccsp)

	timeout, err := time.ParseDuration(timeoutstring)
	if err != nil || timeoutstring == "" {
		timeout = time.Duration(60 * time.Second)
	}

	r := &HSMReenroller{
		CAClient:  caclient,
		HomeDir:   homeDir,
		Config:    cfg,
		Timeout:   timeout,
		HSMConfig: hsmConfig,
		Instance:  instance,
		Client:    client,
		Scheme:    scheme,
		NewKey:    newKey,
	}

	if bccsp != nil {
		r.BCCSP = true
	}

	return r, nil
}

func (r *HSMReenroller) IsCAReachable() bool {
	log.Info("Check if CA is reachable before triggering enroll job")

	timeout := r.Timeout
	url := fmt.Sprintf("https://%s:%s/cainfo", r.Config.CAHost, r.Config.CAPort)

	// Convert TLS certificate from base64 to file
	tlsCertBytes, err := util.Base64ToBytes(r.Config.CATLS.CACert)
	if err != nil {
		log.Error(err, "Cannot convert TLS Certificate from base64")
		return false
	}

	err = wait.Poll(500*time.Millisecond, timeout, func() (bool, error) {
		err = util.HealthCheck(url, tlsCertBytes, timeout)
		if err == nil {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		log.Error(err, "Health check failed")
		return false
	}

	return true
}

func (r *HSMReenroller) Reenroll() (*config.Response, error) {
	if !r.IsCAReachable() {
		return nil, errors.New("unable to enroll, CA is not reachable")
	}

	// Deleting CA client config is an unfortunate requirement since the ca client
	// config map was not properly deleted after a successfull reenrollment request.
	// This is problematic when recreating a resource with same name, as it will
	// try to use old settings in the config map, which might no longer apply, thus
	// it must be removed if found before proceeding.
	if err := deleteCAClientConfig(r.Client, r.Instance); err != nil {
		return nil, err
	}

	if err := createRootTLSSecret(r.Client, r.Instance, r.Scheme, r.Config.CATLS.CACert); err != nil {
		return nil, err
	}

	if err := createCAClientConfig(r.Client, r.Instance, r.Scheme, r.CAClient.Config); err != nil {
		return nil, err
	}

	job := r.initHSMJob(r.Instance, r.HSMConfig, r.Timeout)
	if err := r.Client.Create(context.TODO(), job.Job, k8sclient.CreateOption{
		Owner:  r.Instance,
		Scheme: r.Scheme,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to create HSM ca initialization job")
	}
	log.Info(fmt.Sprintf("Job '%s' created", job.GetName()))

	if err := job.WaitUntilActive(r.Client); err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Job '%s' active", job.GetName()))

	if err := job.WaitUntilFinished(r.Client); err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Job '%s' finished", job.GetName()))

	status, err := job.Status(r.Client)
	if err != nil {
		return nil, err
	}

	switch status {
	case jobv1.FAILED:
		return nil, fmt.Errorf("Job '%s' finished unsuccessfully, not cleaning up pods to allow for error evaluation", job.GetName())
	case jobv1.COMPLETED:
		if err := job.Delete(r.Client); err != nil {
			return nil, err
		}

		if err := deleteRootTLSSecret(r.Client, r.Instance); err != nil {
			return nil, err
		}

		if err := deleteCAClientConfig(r.Client, r.Instance); err != nil {
			return nil, err
		}
	}

	if err := r.setControllerReferences(); err != nil {
		return nil, err
	}

	return &config.Response{}, nil
}

func (r *HSMReenroller) setControllerReferences() error {
	if err := setControllerReferenceFor(r.Client, r.Instance, r.Scheme, fmt.Sprintf("ecert-%s-signcert", r.Instance.GetName()), false); err != nil {
		return err
	}

	if err := setControllerReferenceFor(r.Client, r.Instance, r.Scheme, fmt.Sprintf("ecert-%s-cacerts", r.Instance.GetName()), false); err != nil {
		return err
	}

	if err := setControllerReferenceFor(r.Client, r.Instance, r.Scheme, fmt.Sprintf("ecert-%s-intercerts", r.Instance.GetName()), true); err != nil {
		return err
	}

	return nil
}

func setControllerReferenceFor(client k8sclient.Client, instance Instance, scheme *runtime.Scheme, name string, skipIfNotFound bool) error {
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

func createRootTLSSecret(client k8sclient.Client, instance Instance, scheme *runtime.Scheme, cert string) error {
	tlsCertBytes, err := util.Base64ToBytes(cert)
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

func createCAClientConfig(client k8sclient.Client, instance Instance, scheme *runtime.Scheme, config *lib.ClientConfig) error {
	configBytes, err := yaml.Marshal(config)
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
		return errors.Wrap(err, "failed to delete confk8smap")
	}

	return nil
}

func (r *HSMReenroller) initHSMJob(instance Instance, hsmConfig *config.HSMConfig, timeout time.Duration) *jobv1.Job {
	hsmLibraryPath := hsmConfig.Library.FilePath
	hsmLibraryName := filepath.Base(hsmLibraryPath)

	jobName := fmt.Sprintf("%s-reenroll", instance.GetName())

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
							Command: []string{
								"sh",
								"-c",
								fmt.Sprintf("/usr/local/bin/enroller node reenroll %s %s %s %s %s %s %s %s %s %t", r.HomeDir, "/tmp/fabric-ca-client-config.yaml", r.Config.CAHost, r.Config.CAPort, r.Config.CAName, instance.GetName(), instance.GetNamespace(), r.Config.EnrollID, fmt.Sprintf("%s/cert.pem", r.HomeDir), r.NewKey),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tlscertfile",
									MountPath: fmt.Sprintf("%s/tlsCert.pem", r.HomeDir),
									SubPath:   "tlsCert.pem",
								},
								{
									Name:      "certfile",
									MountPath: fmt.Sprintf("%s/cert.pem", r.HomeDir),
									SubPath:   "cert.pem",
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
							Name: "certfile",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: fmt.Sprintf("ecert-%s-signcert", instance.GetName()),
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
		WaitUntilActive:   timeout,
		WaitUntilFinished: timeout,
	})

	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, hsmConfig.GetVolumes()...)
	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, hsmConfig.GetVolumeMounts()...)

	return job
}
