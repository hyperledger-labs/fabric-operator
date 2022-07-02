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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	consolev1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/console/v1"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/console/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/console/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

var _ = Describe("Openshift Console Config Map Overrides", func() {
	var (
		overrider *override.Override
		instance  *current.IBPConsole
		cm        *corev1.ConfigMap
	)

	BeforeEach(func() {
		var err error

		cm, err = util.GetConfigMapFromFile("../../../../../definitions/console/console-configmap.yaml")
		Expect(err).NotTo(HaveOccurred())

		overrider = &override.Override{}
		instance = &current.IBPConsole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "consolecm",
				Namespace: "consolecmns",
			},
			Spec: current.IBPConsoleSpec{
				Email:            "test@ibm.com",
				AuthScheme:       "scheme1",
				ConfigtxlatorURL: "configtx.ibm.com",
				DeployerURL:      "deployer.ibm.com",
				DeployerTimeout:  5,
				Components:       "component1",
				Sessions:         "session1",
				System:           "system1",
				SystemChannel:    "channel1",
				FeatureFlags: &consolev1.FeatureFlags{
					CreateChannelEnabled: true,
				},
				ClusterData: &consolev1.IBPConsoleClusterData{
					Zones: []string{"zone1"},
					Type:  "type1",
				},
				NetworkInfo: &current.NetworkInfo{
					Domain: "ibm.com",
				},
			},
		}
	})

	Context("create", func() {
		It("returns an error if domain not provided", func() {
			instance.Spec.NetworkInfo.Domain = ""
			err := overrider.ConsoleCM(instance, cm, resources.Create, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("domain not provided"))
		})

		It("overrides values based on spec", func() {
			err := overrider.ConsoleCM(instance, cm, resources.Create, nil)
			Expect(err).NotTo(HaveOccurred())

			config := &v1.ConsoleSettingsConfig{}
			err = yaml.Unmarshal([]byte(cm.Data["settings.yaml"]), config)
			Expect(err).NotTo(HaveOccurred())

			CommonConsoleCMOverrides(instance, config)
		})
	})

	Context("update", func() {
		It("overrides values based on spec", func() {
			err := overrider.ConsoleCM(instance, cm, resources.Update, nil)
			Expect(err).NotTo(HaveOccurred())

			config := &v1.ConsoleSettingsConfig{}
			err = yaml.Unmarshal([]byte(cm.Data["settings.yaml"]), config)
			Expect(err).NotTo(HaveOccurred())

			CommonConsoleCMOverrides(instance, config)
		})
	})
})

func CommonConsoleCMOverrides(instance *current.IBPConsole, config *v1.ConsoleSettingsConfig) {
	By("setting email", func() {
		Expect(config.Email).To(Equal(instance.Spec.Email))
	})

	By("setting auth scheme", func() {
		Expect(config.AuthScheme).To(Equal(instance.Spec.AuthScheme))
	})

	By("setting configtxlator URL", func() {
		Expect(config.Configtxlator).To(Equal(instance.Spec.ConfigtxlatorURL))
	})

	By("setting Deployer URL", func() {
		Expect(config.DeployerURL).To(Equal(instance.Spec.DeployerURL))
	})

	By("setting Deployer timeout", func() {
		Expect(config.DeployerTimeout).To(Equal(instance.Spec.DeployerTimeout))
	})

	By("setting components", func() {
		Expect(config.DBCustomNames.Components).To(Equal(instance.Spec.Components))
	})

	By("setting sessions", func() {
		Expect(config.DBCustomNames.Sessions).To(Equal(instance.Spec.Sessions))
	})

	By("setting system", func() {
		Expect(config.DBCustomNames.System).To(Equal(instance.Spec.System))
	})

	By("setting system channel", func() {
		Expect(config.SystemChannelID).To(Equal(instance.Spec.SystemChannel))
	})

	By("setting Proxy TLS Reqs", func() {
		Expect(config.ProxyTLSReqs).To(Equal("always"))
	})

	By("settings feature flags", func() {
		Expect(config.Featureflags).To(Equal(instance.Spec.FeatureFlags))
	})

	By("settings cluster data", func() {
		Expect(config.ClusterData).To(Equal(instance.Spec.ClusterData))
	})
}
