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

package configmap_test

import (
	"context"
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart/configmap"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Configmap", func() {

	var (
		mockClient *controllermocks.Client
		manager    *configmap.Manager
	)

	BeforeEach(func() {
		mockClient = &controllermocks.Client{}
		manager = configmap.NewManager(mockClient)
	})

	Context("get restart config from", func() {
		It("returns error if fails to get config map", func() {
			mockClient.GetReturns(errors.New("fake error"))
			err := manager.GetRestartConfigFrom("test-config", "namespace", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get test-config config map"))
		})

		It("returns error if fails to unmarshal data", func() {
			into := &TestConfig{}
			mockClient.GetStub = func(ctx context.Context, ns types.NamespacedName, obj client.Object) error {
				o := obj.(*corev1.ConfigMap)
				o.Name = "test-config"
				o.Namespace = ns.Namespace
				o.BinaryData = map[string][]byte{
					"restart-config.yaml": []byte("test"),
				}
				return nil
			}

			err := manager.GetRestartConfigFrom("test-config", "namespace", into)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unmarshal test-config config map"))
		})

		It("unmarshals config map data into struct", func() {
			into := &TestConfig{}
			mockClient.GetStub = func(ctx context.Context, ns types.NamespacedName, obj client.Object) error {
				cfg := &TestConfig{
					Field: "test",
				}
				bytes, _ := json.Marshal(cfg)

				o := obj.(*corev1.ConfigMap)
				o.Name = "test-config"
				o.Namespace = ns.Namespace
				o.BinaryData = map[string][]byte{
					"restart-config.yaml": bytes,
				}
				return nil
			}

			err := manager.GetRestartConfigFrom("test-config", "namespace", into)
			Expect(err).NotTo(HaveOccurred())
			Expect(into.Field).To(Equal("test"))
		})
	})

	It("update config", func() {
		mockClient.CreateOrUpdateReturns(errors.New("fake error"))
		err := manager.UpdateConfig("test-config", "ns", &TestConfig{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to create or update test-config config map"))
	})
})

type TestConfig struct {
	Field string
}
