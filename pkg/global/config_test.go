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

package global_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/global"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Global config", func() {
	var (
		f    = false
		root = int64(0)

		configSetter *global.ConfigSetter
	)

	BeforeEach(func() {
		configSetter = &global.ConfigSetter{
			Config: config.Globals{
				SecurityContext: &container.SecurityContext{
					RunAsNonRoot:             &f,
					Privileged:               &f,
					RunAsUser:                &root,
					AllowPrivilegeEscalation: &f,
				},
			},
		}
	})

	Context("security context on containers", func() {
		Context("job", func() {
			var job *batchv1.Job

			BeforeEach(func() {
				job = &batchv1.Job{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								InitContainers: []corev1.Container{
									{
										Name: "initcontainer1",
									},
									{
										Name: "initcontainer2",
									},
								},
								Containers: []corev1.Container{
									{
										Name: "container1",
									},
									{
										Name: "container2",
									},
								},
							},
						},
					},
				}
			})

			It("updates security context", func() {
				configSetter.UpdateSecurityContextForAllContainers(job)

				for _, cont := range job.Spec.Template.Spec.InitContainers {
					Expect(*cont.SecurityContext).To(MatchFields(IgnoreExtras, Fields{
						"RunAsNonRoot":             Equal(&f),
						"Privileged":               Equal(&f),
						"RunAsUser":                Equal(&root),
						"AllowPrivilegeEscalation": Equal(&f),
					}))
				}

				for _, cont := range job.Spec.Template.Spec.Containers {
					Expect(*cont.SecurityContext).To(MatchFields(IgnoreExtras, Fields{
						"RunAsNonRoot":             Equal(&f),
						"Privileged":               Equal(&f),
						"RunAsUser":                Equal(&root),
						"AllowPrivilegeEscalation": Equal(&f),
					}))
				}
			})
		})

		Context("deployment", func() {
			var dep *appsv1.Deployment

			BeforeEach(func() {
				dep = &appsv1.Deployment{
					Spec: appsv1.DeploymentSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								InitContainers: []corev1.Container{
									{
										Name: "initcontainer1",
									},
									{
										Name: "initcontainer2",
									},
								},
								Containers: []corev1.Container{
									{
										Name: "container1",
									},
									{
										Name: "container2",
									},
								},
							},
						},
					},
				}
			})

			It("updates security context", func() {
				configSetter.UpdateSecurityContextForAllContainers(dep)

				for _, cont := range dep.Spec.Template.Spec.InitContainers {
					Expect(*cont.SecurityContext).To(MatchFields(IgnoreExtras, Fields{
						"RunAsNonRoot":             Equal(&f),
						"Privileged":               Equal(&f),
						"RunAsUser":                Equal(&root),
						"AllowPrivilegeEscalation": Equal(&f),
					}))
				}

				for _, cont := range dep.Spec.Template.Spec.Containers {
					Expect(*cont.SecurityContext).To(MatchFields(IgnoreExtras, Fields{
						"RunAsNonRoot":             Equal(&f),
						"Privileged":               Equal(&f),
						"RunAsUser":                Equal(&root),
						"AllowPrivilegeEscalation": Equal(&f),
					}))
				}
			})
		})
	})
})
