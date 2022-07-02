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

package integration

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type NativeResourcePoller struct {
	Name      string
	Namespace string
	Client    *kubernetes.Clientset
	retry     int
}

func (p *NativeResourcePoller) PVCExists() bool {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	pvcList, err := p.Client.CoreV1().PersistentVolumeClaims(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return false
	}
	for _, pvc := range pvcList.Items {
		if strings.HasPrefix(pvc.Name, p.Name) {
			return true
		}
	}

	return false
}

func (p *NativeResourcePoller) IngressExists() bool {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	ingressList, err := p.Client.NetworkingV1().Ingresses(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return false
	}
	for _, ingress := range ingressList.Items {
		if strings.HasPrefix(ingress.Name, p.Name) {
			return true
		}
	}

	return false
}

func (p *NativeResourcePoller) ServiceExists() bool {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	serviceList, err := p.Client.CoreV1().Services(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return false
	}
	for _, service := range serviceList.Items {
		if strings.HasPrefix(service.Name, p.Name) {
			return true
		}
	}

	return false
}

func (p *NativeResourcePoller) ConfigMapExists() bool {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	cmList, err := p.Client.CoreV1().ConfigMaps(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return false
	}
	for _, cm := range cmList.Items {
		if strings.HasPrefix(cm.Name, p.Name) {
			return true
		}
	}

	return false
}

func (p *NativeResourcePoller) DeploymentExists() bool {
	dep, err := p.Client.AppsV1().Deployments(p.Namespace).Get(context.TODO(), p.Name, metav1.GetOptions{})
	if err == nil && dep != nil {
		return true
	}

	return false
}

func (p *NativeResourcePoller) Deployment() *appsv1.Deployment {
	deps := p.DeploymentList()
	if len(deps.Items) > 0 {
		return &deps.Items[0]
	}
	return nil
}

func (p *NativeResourcePoller) DeploymentList() *appsv1.DeploymentList {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	deps, err := p.Client.AppsV1().Deployments(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return &appsv1.DeploymentList{}
	}
	return deps
}

func (p *NativeResourcePoller) NumberOfDeployments() int {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	deps, err := p.Client.AppsV1().Deployments(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return 0
	}

	return len(deps.Items)
}

func (p *NativeResourcePoller) NumberOfOrdererNodeDeployments() int {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("parent=%s", p.Name),
	}

	deps, err := p.Client.AppsV1().Deployments(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return 0
	}

	return len(deps.Items)
}

func (p *NativeResourcePoller) IsRunning() bool {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("name=%s", p.Name),
	}
	podList, err := p.Client.CoreV1().Pods(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return false
	}
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, p.Name) {
			if pod.Status.Phase == corev1.PodRunning {
				containerStatuses := pod.Status.ContainerStatuses
				for _, status := range containerStatuses {
					if status.State.Running == nil {
						return false
					}
					if !status.Ready {
						return false
					}
				}
				return true
			} else if pod.Status.Phase == corev1.PodPending {
				if p.retry == 0 {
					if len(pod.Status.InitContainerStatuses) == 0 {
						return false
					}
					initContainerStatuses := pod.Status.InitContainerStatuses
					for _, status := range initContainerStatuses {
						if status.State.Waiting != nil {
							if status.State.Waiting.Reason == "CreateContainerConfigError" {
								// Handling this error will make no difference
								_ = p.Client.CoreV1().Pods(p.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
								p.retry = 1
							}
						}
					}
				}
			}
		}
	}

	return false
}

// PodCreated returns true if pod has been created based on app name
func (p *NativeResourcePoller) PodCreated() bool {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	podList, err := p.Client.CoreV1().Pods(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return false
	}
	if len(podList.Items) != 0 {
		return true
	}
	return false
}

func (p *NativeResourcePoller) PodIsRunning() bool {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	podList, err := p.Client.CoreV1().Pods(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return false
	}
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, p.Name) {
			switch pod.Status.Phase {
			case corev1.PodRunning:
				containerStatuses := pod.Status.ContainerStatuses
				for _, status := range containerStatuses {
					if status.State.Running == nil {
						fmt.Fprintf(GinkgoWriter, "For pod '%s', container '%s' is not yet running\n", pod.Name, status.Name)
						return false
					}
					if !status.Ready {
						fmt.Fprintf(GinkgoWriter, "For pod '%s', container '%s' is not yet ready\n", pod.Name, status.Name)
						return false
					}
				}
				fmt.Fprintf(GinkgoWriter, "'%s' and it's containers are ready and running\n", pod.Name)
				return true
			case corev1.PodPending:
				p.CheckForStuckPod(pod)
			}
		}
	}

	return false
}

func (p *NativeResourcePoller) CheckForStuckPod(pod corev1.Pod) bool {
	fmt.Fprintf(GinkgoWriter, "'%s' in pending state, waiting for pod to start running...\n", pod.Name)
	if p.retry > 0 {
		return false // Out of retries, return
	}

	if len(pod.Status.InitContainerStatuses) == 0 {
		return false // No containers found, unable to get status to determine if pod is running
	}

	initContainerStatuses := pod.Status.InitContainerStatuses
	for _, status := range initContainerStatuses {
		if status.State.Waiting != nil {
			fmt.Fprintf(GinkgoWriter, "'%s' is waiting, with reason '%s'\n", pod.Name, status.State.Waiting.Reason)

			// Intermittent issues are see on pods with shared volume mounts that are deleted and created in
			// quick succession, in suchs situation the pods sometimes ends up with an error stating that it
			// can't mount to subPath. This can be resolved by deleting the pod and let it try again to
			// acquire the volume mount. The code below mimics this solution by deleting the pod, which is
			// brought back by the deployment and pod comes up fine. This is more of a hack to resolve this
			// issue in test, the root cause might live in portworx or in operator.
			if status.State.Waiting.Reason == "CreateContainerConfigError" {
				fmt.Fprintf(GinkgoWriter, "Deleting pod '%s'\n", pod.Name)
				err := p.Client.CoreV1().Pods(p.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
				if err != nil {
					fmt.Fprintf(GinkgoWriter, "Deleting pod '%s' failed: %s\n", pod.Name, err)
				}
				p.retry = 1
			}
		}
	}

	return true
}

func (p *NativeResourcePoller) GetPods() []corev1.Pod {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	podList, err := p.Client.CoreV1().Pods(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return nil
	}
	return podList.Items
}

func (p *NativeResourcePoller) GetRunningPods() []corev1.Pod {
	opts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", p.Name),
	}
	podList, err := p.Client.CoreV1().Pods(p.Namespace).List(context.TODO(), opts)
	if err != nil {
		return nil
	}
	pods := []corev1.Pod{}
	for _, pod := range podList.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			containerStatuses := pod.Status.ContainerStatuses

			readyContainers := 0
			numOfContainers := len(containerStatuses)

			for _, status := range containerStatuses {
				if status.Ready && status.State.Running != nil {
					readyContainers++
				}
			}
			if readyContainers == numOfContainers {
				pods = append(pods, pod)
			}

		case corev1.PodPending:
			p.CheckForStuckPod(pod)
		}
	}

	return pods
}

func (p *NativeResourcePoller) TestAffinityZone(dep *appsv1.Deployment) bool {
	zoneExp := "topology.kubernetes.io/zone"

	affinity := dep.Spec.Template.Spec.Affinity.NodeAffinity
	if affinity != nil {
		nodes := affinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
		for _, node := range nodes {
			for _, expr := range node.MatchExpressions {
				depExp := expr.Key
				if zoneExp == depExp {
					return true
				}
			}
		}
	} else {
		return false
	}

	return false
}

func (p *NativeResourcePoller) TestAffinityRegion(dep *appsv1.Deployment) bool {
	regionExp := "topology.kubernetes.io/region"

	affinity := dep.Spec.Template.Spec.Affinity.NodeAffinity
	if affinity != nil {
		nodes := affinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
		for _, node := range nodes {
			for _, expr := range node.MatchExpressions {
				depExp := expr.Key
				if regionExp == depExp {
					return true
				}
			}
		}
	} else {
		return false
	}

	return false
}
