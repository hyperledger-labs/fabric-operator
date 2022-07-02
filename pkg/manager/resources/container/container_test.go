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

package container_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/container"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("container", func() {
	var (
		cont *container.Container
	)

	BeforeEach(func() {
		cont = &container.Container{
			Container: &corev1.Container{},
		}
	})

	Context("env vars", func() {
		BeforeEach(func() {
			cont.Env = []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "env1",
					Value: "1.0",
				},
			}

			Expect(cont.Env).To(ContainElement(corev1.EnvVar{Name: "env1", Value: "1.0"}))
		})

		It("updates", func() {
			cont.UpdateEnv("env1", "1.1")
			Expect(len(cont.Env)).To(Equal(1))
			Expect(cont.Env).To(ContainElement(corev1.EnvVar{Name: "env1", Value: "1.1"}))
		})
	})

	Context("set image", func() {
		It("parses sha tags", func() {
			cont.SetImage("ibp-peer", "sha256:12345")
			Expect(cont.Image).To(Equal("ibp-peer@sha256:12345"))
		})
	})
})
