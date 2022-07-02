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

package override_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/console/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("Openshift Console Env Config Map Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPConsole
		cm        *corev1.ConfigMap
	)

	BeforeEach(func() {
		var err error
		overrider = &override.Override{}
		instance = &current.IBPConsole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name1",
				Namespace: "ns1",
			},
			Spec: current.IBPConsoleSpec{
				ConnectionString: "connection_string",
				TLSSecretName:    "tls_secret_name",
				System:           "system1",
				NetworkInfo: &current.NetworkInfo{
					Domain:      "test.domain",
					ConsolePort: 31010,
					ProxyPort:   31011,
				},
			},
		}
		cm, err = util.GetConfigMapFromFile("../../../../../definitions/console/configmap.yaml")
		Expect(err).NotTo(HaveOccurred())
	})

	Context("create", func() {
		It("appropriately overrides the respective values for env config map", func() {
			err := overrider.CM(instance, cm, resources.Create, nil)
			Expect(err).NotTo(HaveOccurred())

			By("setting HOST_URL", func() {
				Expect(cm.Data["HOST_URL"]).To(Equal(fmt.Sprintf("https://%s-%s-console.%s:443", instance.GetNamespace(), instance.GetName(), instance.Spec.NetworkInfo.Domain)))
			})

			By("setting HOST_URL", func() {
				Expect(cm.Data["HOST_URL_WS"]).To(Equal(fmt.Sprintf("https://%s-%s-console.%s:443", instance.GetNamespace(), instance.GetName(), instance.Spec.NetworkInfo.Domain)))
			})

			By("setting DB_CONNECTION_STRING", func() {
				Expect(cm.Data["DB_CONNECTION_STRING"]).To(Equal(instance.Spec.ConnectionString))
			})

			By("setting DB_SYSTEM", func() {
				Expect(cm.Data["DB_SYSTEM"]).To(Equal(instance.Spec.System))
			})

			By("setting KEY_FILE_PATH", func() {
				Expect(cm.Data["KEY_FILE_PATH"]).To(Equal("/certs/tls/tls.key"))
			})

			By("setting PEM_FILE_PATH", func() {
				Expect(cm.Data["PEM_FILE_PATH"]).To(Equal("/certs/tls/tls.crt"))
			})
		})
	})
})
