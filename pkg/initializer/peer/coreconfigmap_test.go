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

package initializer_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v2"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/mocks"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("core config map", func() {
	var (
		coreCM   *initializer.CoreConfigMap
		instance *current.IBPPeer
		client   *mocks.Client
	)

	BeforeEach(func() {
		client = &mocks.Client{}
		coreCM = &initializer.CoreConfigMap{
			Config: &initializer.Config{
				CorePeerFile:    "../../../defaultconfig/peer/core.yaml",
				CorePeerV2File:  "../../../defaultconfig/peer/v2/core.yaml",
				CorePeerV25File: "../../../defaultconfig/peer/v25/core.yaml",
				OUFile:          "../../../defaultconfig/peer/ouconfig.yaml",
				InterOUFile:     "../../../defaultconfig/peer/ouconfig-inter.yaml",
			},
			Client:    client,
			GetLabels: func(o metav1.Object) map[string]string { return map[string]string{} },
		}

		instance = &current.IBPPeer{}

		client.GetStub = func(ctx context.Context, types types.NamespacedName, obj k8sclient.Object) error {
			switch obj.(type) {
			case *corev1.ConfigMap:
				if types.Name == fmt.Sprintf("%s-config", instance.Name) {
					cm := obj.(*corev1.ConfigMap)
					cm.BinaryData = map[string][]byte{}
				}
			}
			return nil
		}
	})

	Context("get core config", func() {
		It("returns config map containing peer's core config", func() {
			cm, err := coreCM.GetCoreConfig(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(cm).NotTo(BeNil())
		})
	})

	Context("create or update config map", func() {
		BeforeEach(func() {
			client.GetStub = func(ctx context.Context, types types.NamespacedName, obj k8sclient.Object) error {
				switch obj.(type) {
				case *corev1.ConfigMap:
					if types.Name == fmt.Sprintf("%s-config", instance.Name) {
						cm := obj.(*corev1.ConfigMap)
						cm.BinaryData = map[string][]byte{}
					}
				}
				return nil
			}

		})

		It("adds default configs", func() {
			err := coreCM.CreateOrUpdate(instance, &v2.Core{})
			Expect(err).NotTo(HaveOccurred())

			By("adding node OU config section", func() {
				_, obj, _ := client.CreateOrUpdateArgsForCall(0)

				cm := obj.(*corev1.ConfigMap)
				Expect(cm.BinaryData["config.yaml"]).To(ContainSubstring("Enable: true"))
			})
		})
	})

	Context("add node ou to config map", func() {
		When("nodeoudisabled is set to false", func() {
			BeforeEach(func() {
				f := false
				instance.Spec.DisableNodeOU = &f
			})

			It("adds nodeou configs as enabled", func() {
				err := coreCM.AddNodeOU(instance)
				Expect(err).NotTo(HaveOccurred())

				_, obj, _ := client.CreateOrUpdateArgsForCall(0)

				cm := obj.(*corev1.ConfigMap)
				Expect(cm.BinaryData["config.yaml"]).To(ContainSubstring("Enable: true"))
			})
		})

		When("nodeoudisabled is set to true", func() {
			BeforeEach(func() {
				t := true
				instance.Spec.DisableNodeOU = &t
			})

			It("adds nodeou configs as disabled", func() {
				err := coreCM.AddNodeOU(instance)
				Expect(err).NotTo(HaveOccurred())

				_, obj, _ := client.CreateOrUpdateArgsForCall(0)

				cm := obj.(*corev1.ConfigMap)
				Expect(cm.BinaryData["config.yaml"]).To(ContainSubstring("Enable: false"))
			})
		})
	})
})
