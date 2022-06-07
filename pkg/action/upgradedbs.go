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

package action

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	oconfig "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	controller "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/deployment"
	jobv1 "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/job"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/image"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate counterfeiter -o mocks/deploymentreset.go -fake-name DeploymentReset . DeploymentReset

// DeploymentReset defines the contract to manage deployment reousrce
type DeploymentReset interface {
	Get(v1.Object) (k8sclient.Object, error)
	DeploymentStatus(v1.Object) (appsv1.DeploymentStatus, error)
	GetScheme() *runtime.Scheme
}

//go:generate counterfeiter -o mocks/upgradeinstance.go -fake-name UpgradeInstance . UpgradeInstance

// UpgradeInstance defines the contract to update the insstance database
type UpgradeInstance interface {
	runtime.Object
	v1.Object
	UsingCouchDB() bool
	UsingHSMProxy() bool
	IsHSMEnabled() bool
}

// UpgradeDBs will update the database and peform all necessary clean up and restart logic
func UpgradeDBs(deploymentManager DeploymentReset, client controller.Client, instance UpgradeInstance, timeouts oconfig.DBMigrationTimeouts) error {
	obj, err := deploymentManager.Get(instance)
	if err != nil {
		return errors.Wrap(err, "failed to get deployment")
	}

	dep := deployment.New(obj.(*appsv1.Deployment))
	originalReplicas := dep.Spec.Replicas

	// Need to set replica to 0, otherwise migration job won't be able start to due to
	// volume being attached to another node.
	//
	// Wait for deployment to get marked as unavailable after replica updated to 0
	if err := setReplicaCountAndWait(client, deploymentManager, instance, int32(0), timeouts.ReplicaChange.Get()); err != nil {
		return errors.Wrapf(err, "failed to update deployment for '%s'", instance.GetName())
	}

	if err := waitForPodToDelete(client, instance, timeouts.PodDeletion.Get()); err != nil {
		return err
	}

	var ip string
	if instance.UsingCouchDB() {
		couchDBPod := getCouchDBPod(dep)
		if err := startCouchDBPod(client, couchDBPod); err != nil {
			return err
		}

		ip, err = waitForPodToBeRunning(client, couchDBPod, timeouts.PodStart.Get())
		if err != nil {
			return errors.Wrap(err, "couchdb pod failed to start")
		}
	}

	var hsmConfig *config.HSMConfig
	if !instance.UsingHSMProxy() && instance.IsHSMEnabled() {
		hsmConfig, err = config.ReadHSMConfig(client, instance)
		if err != nil {
			return err
		}
	}

	job := peerDBMigrationJob(dep, instance.(*current.IBPPeer), hsmConfig, ip, timeouts)
	creatOpt := controllerclient.CreateOption{
		Owner:  instance,
		Scheme: deploymentManager.GetScheme(),
	}
	if err := StartJob(client, job.Job, creatOpt); err != nil {
		if instance.UsingCouchDB() {
			log.Info("failed to start db migration job, deleting couchdb pod")
			couchDBPod := &corev1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      fmt.Sprintf("%s-couchdb", instance.GetName()),
					Namespace: instance.GetNamespace(),
				},
			}

			if err := client.Delete(context.TODO(), couchDBPod); err != nil {
				return errors.Wrap(err, "failed to delete couchdb pod")
			}
		}
		return errors.Wrap(err, "failed to start db migration job")
	}
	log.Info(fmt.Sprintf("Job '%s' created", job.GetName()))

	// Wait for job to start and pod to go into running state before reverting
	// back to original replica value
	if err := job.WaitUntilActive(client); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Job '%s' active", job.GetName()))

	if err := job.WaitUntilContainerFinished(client, "dbmigration"); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Job '%s' finished", job.GetName()))

	// Wait for deployment to get marked as available after replica update
	if err := setReplicaCountAndWait(client, deploymentManager, instance, *originalReplicas, timeouts.ReplicaChange.Get()); err != nil {
		return errors.Wrapf(err, "failed to update deployment for '%s'", instance.GetName())
	}

	return nil
}

// StartJob uses the client to create a job on kubernetes client
func StartJob(client controller.Client, job *batchv1.Job, opt controller.CreateOption) error {
	log.Info(fmt.Sprintf("Starting job '%s'", job.GetName()))

	if err := client.Create(context.TODO(), job, opt); err != nil {
		return errors.Wrap(err, "failed to create migration job")
	}

	return nil
}

func startCouchDBPod(client controller.Client, pod *corev1.Pod) error {
	log.Info(fmt.Sprintf("Starting couchdb pod '%s'", pod.GetName()))

	if err := client.Create(context.TODO(), pod); err != nil {
		return errors.Wrap(err, "failed to create couchdb pod")
	}

	return nil
}

func getCouchDBPod(dep *deployment.Deployment) *corev1.Pod {
	couchdb := dep.MustGetContainer("couchdb")

	localSpecCopy := dep.Spec.Template.Spec.DeepCopy()
	volumes := localSpecCopy.Volumes
	// Remove ledgerdb volume from couchddb pod
	for i, volume := range volumes {
		if volume.Name == "fabric-peer-0" {
			// Remove the ledgerdb data from couchdb container
			volumes[i] = volumes[len(volumes)-1]
			volumes = volumes[:len(volumes)-1]
			break
		}
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-couchdb", dep.GetName()),
			Namespace: dep.GetNamespace(),
			Labels: map[string]string{
				"app": dep.Name,
			},
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: dep.Spec.Template.Spec.ImagePullSecrets,
			RestartPolicy:    corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				*couchdb.Container,
			},
			Volumes: volumes,
		},
	}
}

func waitForPodToDelete(client controller.Client, instance metav1.Object, timeout time.Duration) error {
	err := wait.Poll(2*time.Second, timeout, func() (bool, error) {
		log.Info(fmt.Sprintf("Waiting for pod for deployment '%s' to be deleted", instance.GetName()))

		labelSelector, err := labels.Parse(fmt.Sprintf("app=%s", instance.GetName()))
		if err != nil {
			return false, nil
		}

		opts := &k8sclient.ListOptions{
			LabelSelector: labelSelector,
		}

		pods := &corev1.PodList{}
		if err := client.List(context.TODO(), pods, opts); err != nil {
			return false, nil
		}

		if len(pods.Items) == 0 {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to delete pod associated with '%s'", instance.GetName())
	}
	return nil
}

func waitForPodToBeRunning(client controller.Client, pod *corev1.Pod, timeout time.Duration) (string, error) {
	var podIP string
	p := &corev1.Pod{}

	err := wait.Poll(2*time.Second, timeout, func() (bool, error) {
		log.Info(fmt.Sprintf("Waiting for couchdb pod '%s' to be running", pod.GetName()))

		label := fmt.Sprintf("app=%s", pod.Labels["app"])
		labelSelector, err := labels.Parse(label)
		if err != nil {
			return false, err
		}

		opts := &k8sclient.ListOptions{
			LabelSelector: labelSelector,
		}

		pods := &corev1.PodList{}
		if err := client.List(context.TODO(), pods, opts); err != nil {
			return false, err
		}

		if len(pods.Items) != 1 {
			return false, nil
		}

		p = &pods.Items[0]
		if len(p.Status.ContainerStatuses) > 0 && p.Status.ContainerStatuses[0].State.Running != nil {
			if p.Status.ContainerStatuses[0].Ready {
				return true, nil
			}
		}

		return false, nil
	})
	if err != nil {
		return podIP, errors.Wrapf(err, "pod '%s' not running", pod.GetName())
	}

	if p != nil {
		podIP = p.Status.PodIP
	}

	return podIP, nil
}

func setReplicaCountAndWait(client controller.Client, deploymentManager DeploymentReset, instance metav1.Object, count int32, timeout time.Duration) error {
	obj, err := deploymentManager.Get(instance)
	if err != nil {
		return errors.Wrap(err, "failed to get deployment")
	}
	dep := deployment.New(obj.DeepCopyObject().(*appsv1.Deployment))

	if err := setReplicaCountOnDeployment(client, obj, dep, count); err != nil {
		return err
	}

	err = wait.Poll(2*time.Second, timeout, func() (bool, error) {
		log.Info(fmt.Sprintf("Waiting for deployment '%s' replicas to go to %d", dep.GetName(), count))
		status, err := deploymentManager.DeploymentStatus(instance)
		if err == nil {
			if status.Replicas == count {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to determine if deployment is available")
	}

	return nil
}

func setReplicaCountOnDeployment(client controller.Client, obj k8sclient.Object, dep *deployment.Deployment, count int32) error {
	dep.Deployment.Spec.Replicas = &count
	if err := client.Patch(context.TODO(), dep.Deployment, k8sclient.MergeFrom(obj)); err != nil {
		return errors.Wrapf(err, "failed to update replica to %d", count)
	}
	return nil
}

// Copy of container that is passed but updated with new command
func peerDBMigrationJob(dep *deployment.Deployment, instance *current.IBPPeer, hsmConfig *config.HSMConfig, couchdbIP string, timeouts oconfig.DBMigrationTimeouts) *jobv1.Job {
	cont := dep.MustGetContainer("peer")
	envs := []string{
		"LICENSE",
		"FABRIC_CFG_PATH",
		"CORE_PEER_MSPCONFIGPATH",
		"CORE_PEER_FILESYSTEMPATH",
		"CORE_PEER_TLS_ENABLED",
		"CORE_PEER_TLS_CERT_FILE",
		"CORE_PEER_TLS_KEY_FILE",
		"CORE_PEER_TLS_ROOTCERT_FILE",
		"CORE_PEER_LOCALMSPID",
		"CORE_LEDGER_STATE_COUCHDBCONFIG_USERNAME",
		"CORE_LEDGER_STATE_COUCHDBCONFIG_PASSWORD",
		"CORE_LEDGER_STATE_STATEDATABASE",
	}

	backoffLimit := int32(0)
	envVars := cont.GetEnvs(envs)
	envVars = append(envVars,
		corev1.EnvVar{
			Name:  "FABRIC_LOGGING_SPEC",
			Value: "debug",
		},
	)

	if couchdbIP != "" {
		envVars = append(envVars,
			corev1.EnvVar{
				Name:  "CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS",
				Value: fmt.Sprintf("%s:5984", couchdbIP),
			},
		)
	}

	command := `echo "Migrating peer's database" && peer node upgrade-dbs && mkdir -p /data/status && ts=$(date +%Y%m%d-%H%M%S) && touch /data/status/migrated_to_v2-$ts`

	if instance.UsingHSMProxy() {
		envVars = append(envVars,
			corev1.EnvVar{
				Name:  "PKCS11_PROXY_SOCKET",
				Value: instance.Spec.HSM.PKCS11Endpoint,
			},
		)
	}

	localSpecCopy := dep.Spec.Template.Spec.DeepCopy()
	volumes := localSpecCopy.Volumes

	if instance.UsingCouchDB() {
		// Remove statedb volume from migration pod
		for i, volume := range volumes {
			if volume.Name == "db-data" {
				// Remove the statedb data from couchdb container
				volumes[i] = volumes[len(volumes)-1]
				volumes = volumes[:len(volumes)-1]
				break
			}
		}
	}

	k8sJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-dbmigration", instance.GetName()),
			Namespace: dep.GetNamespace(),
			Labels: map[string]string{
				"job-name": fmt.Sprintf("%s-dbmigration", instance.GetName()),
				"owner":    instance.GetName(),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ImagePullSecrets: dep.Spec.Template.Spec.ImagePullSecrets,
					RestartPolicy:    corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "dbmigration",
							Image:           image.Format(instance.Spec.Images.PeerImage, instance.Spec.Images.PeerTag),
							ImagePullPolicy: cont.ImagePullPolicy,
							Command: []string{
								"sh",
								"-c",
								command,
							},
							Env:             envVars,
							Resources:       cont.Resources,
							SecurityContext: cont.SecurityContext,
							VolumeMounts:    cont.VolumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	job := jobv1.New(k8sJob, &jobv1.Timeouts{
		WaitUntilActive:   timeouts.JobStart.Get(),
		WaitUntilFinished: timeouts.JobCompletion.Get(),
	})

	if hsmConfig != nil {
		migrationCont := job.MustGetContainer("dbmigration")
		migrationCont.Env = append(migrationCont.Env, hsmConfig.Envs...)

		volume := corev1.Volume{
			Name: "shared",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		}
		job.Spec.Template.Spec.Volumes = util.AppendVolumeIfMissing(job.Spec.Template.Spec.Volumes, volume)

		initCont := HSMInitContainer(instance, hsmConfig)
		job.Spec.Template.Spec.InitContainers = append(job.Spec.Template.Spec.InitContainers, *initCont.Container)

		if hsmConfig.Daemon != nil {
			// Unable to launch daemon if not running priviledged moe
			t := true
			migrationCont.SecurityContext.Privileged = &t
			migrationCont.SecurityContext.AllowPrivilegeEscalation = &t

			// This is the shared volume where the file 'pkcsslotd-luanched' is touched to let
			// other containers know that the daemon has successfully launched.
			migrationCont.AppendVolumeMountIfMissing("shared", "/shared")

			// Update command in deployment to ensure that deamon is running before starting the ca
			migrationCont.Command = []string{
				"sh",
				"-c",
				fmt.Sprintf("%s && %s", config.DAEMON_CHECK_CMD, command),
			}

			var pvcMount *corev1.VolumeMount
			for _, vm := range hsmConfig.MountPaths {
				if vm.UsePVC {
					pvcMount = &corev1.VolumeMount{
						Name:      "fabric-peer-0",
						MountPath: vm.MountPath,
					}
				}
			}

			// Add daemon container to the job
			config.AddDaemonContainer(hsmConfig, job, instance.GetResource(current.HSMDAEMON), pvcMount)

			// If a pvc mount has been configured in HSM config, set the volume mount on the CertGen container
			if pvcMount != nil {
				migrationCont.AppendVolumeMountIfMissing(pvcMount.Name, pvcMount.MountPath)
			}
		}
	}

	return job
}

// HSMInitContainer creates a container that copies the HSM library to shared volume
func HSMInitContainer(instance *current.IBPPeer, hsmConfig *config.HSMConfig) *container.Container {
	hsmLibraryPath := hsmConfig.Library.FilePath
	hsmLibraryName := filepath.Base(hsmLibraryPath)

	f := false
	user := int64(0)
	mountPath := "/shared"
	initCont := &container.Container{
		Container: &corev1.Container{
			Name:            "hsm-client",
			Image:           image.Format(instance.Spec.Images.HSMImage, instance.Spec.Images.HSMTag),
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
					corev1.ResourceCPU:    resource.MustParse("0.1"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
		},
	}

	return initCont
}
