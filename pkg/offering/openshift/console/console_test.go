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

package openshiftconsole_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	managermocks "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/mocks"
	baseconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console"
	baseconsolemocks "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console/mocks"
	openshiftconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/console"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Openshift Console", func() {

	var (
		console        *openshiftconsole.Console
		instance       *current.IBPConsole
		mockKubeClient *mocks.Client

		consoleRouteManager *managermocks.ResourceManager
		proxyRouteManager   *managermocks.ResourceManager
		deploymentMgr       *managermocks.ResourceManager
		update              *baseconsolemocks.Update
	)

	Context("Reconciles", func() {
		BeforeEach(func() {
			mockKubeClient = &mocks.Client{}
			update = &baseconsolemocks.Update{}
			instance = &current.IBPConsole{
				Spec: current.IBPConsoleSpec{
					License: current.License{
						Accept: true,
					},
					Email:              "xyz@ibm.com",
					PasswordSecretName: "secret",
					ImagePullSecrets:   []string{"testsecret"},
					RegistryURL:        "ghcr.io/ibm-blockchain/",
					ServiceAccountName: "test",
					NetworkInfo: &current.NetworkInfo{
						Domain: "test-domain",
					},
					Versions:         &current.Versions{},
					ConnectionString: "http://fake.url",
				},
			}
			instance.Kind = "IBPConsole"
			instance.Name = "route1"
			instance.Namespace = "testNS"
			instance.Status.Version = version.Operator

			deploymentMgr = &managermocks.ResourceManager{}
			serviceMgr := &managermocks.ResourceManager{}
			pvcMgr := &managermocks.ResourceManager{}
			configMapMgr := &managermocks.ResourceManager{}
			consoleConfigMapMgr := &managermocks.ResourceManager{}
			deployerConfigMapMgr := &managermocks.ResourceManager{}
			roleMgr := &managermocks.ResourceManager{}
			roleBindingMgr := &managermocks.ResourceManager{}
			serviceAccountMgr := &managermocks.ResourceManager{}

			consoleRouteManager = &managermocks.ResourceManager{}
			proxyRouteManager = &managermocks.ResourceManager{}

			deploymentMgr.ExistsReturns(true)
			console = &openshiftconsole.Console{
				Console: &baseconsole.Console{
					Client: mockKubeClient,
					Scheme: &runtime.Scheme{},
					Config: &config.Config{},

					DeploymentManager:        deploymentMgr,
					ServiceManager:           serviceMgr,
					PVCManager:               pvcMgr,
					ConfigMapManager:         configMapMgr,
					ConsoleConfigMapManager:  consoleConfigMapMgr,
					DeployerConfigMapManager: deployerConfigMapMgr,
					RoleManager:              roleMgr,
					RoleBindingManager:       roleBindingMgr,
					ServiceAccountManager:    serviceAccountMgr,
					Restart:                  &baseconsolemocks.RestartManager{},
				},
				RouteManager:      consoleRouteManager,
				ProxyRouteManager: proxyRouteManager,
			}
		})

		It("returns an error if console route manager fails to reconcile", func() {
			consoleRouteManager.ReconcileReturns(errors.New("failed to reconcile ca route"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Console Route reconciliation: failed to reconcile ca route"))
		})

		It("returns an error if proxy route manager fails to reconcile", func() {
			proxyRouteManager.ReconcileReturns(errors.New("failed to reconcile operations route"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Proxy Route reconciliation: failed to reconcile operations route"))
		})

		It("restarts pods by deleting deployment", func() {
			update.RestartNeededReturns(true)
			_, err := console.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockKubeClient.PatchCallCount()).To(Equal(1))
		})

		It("reconciles IBPConsole", func() {
			_, err := console.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
