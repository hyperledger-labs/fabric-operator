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

package k8sconsole_test

import (
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	managermocks "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/mocks"
	baseconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console"
	baseconsolemocks "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console/mocks"
	k8sconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/console"
	"github.com/IBM-Blockchain/fabric-operator/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("K8s Console", func() {
	var (
		console        *k8sconsole.Console
		instance       *current.IBPConsole
		mockKubeClient *mocks.Client

		deploymentMgr        *managermocks.ResourceManager
		serviceMgr           *managermocks.ResourceManager
		pvcMgr               *managermocks.ResourceManager
		configMapMgr         *managermocks.ResourceManager
		consoleConfigMapMgr  *managermocks.ResourceManager
		deployerConfigMapMgr *managermocks.ResourceManager
		roleMgr              *managermocks.ResourceManager
		roleBindingMgr       *managermocks.ResourceManager
		serviceAccountMgr    *managermocks.ResourceManager
		ingressMgr           *managermocks.ResourceManager
		ingressv1beta1Mgr    *managermocks.ResourceManager
		update               *baseconsolemocks.Update
	)

	BeforeEach(func() {
		mockKubeClient = &mocks.Client{}
		update = &baseconsolemocks.Update{}

		deploymentMgr = &managermocks.ResourceManager{}
		serviceMgr = &managermocks.ResourceManager{}
		pvcMgr = &managermocks.ResourceManager{}
		configMapMgr = &managermocks.ResourceManager{}
		consoleConfigMapMgr = &managermocks.ResourceManager{}
		deployerConfigMapMgr = &managermocks.ResourceManager{}
		roleMgr = &managermocks.ResourceManager{}
		roleBindingMgr = &managermocks.ResourceManager{}
		serviceAccountMgr = &managermocks.ResourceManager{}
		ingressMgr = &managermocks.ResourceManager{}
		ingressv1beta1Mgr = &managermocks.ResourceManager{}

		instance = &current.IBPConsole{
			Spec: current.IBPConsoleSpec{
				License: current.License{
					Accept: true,
				},
				ServiceAccountName: "test",
				AuthScheme:         "couchdb",
				DeployerTimeout:    30000,
				Components:         "athena-components",
				Sessions:           "athena-sessions",
				System:             "athena-system",
				Service:            &current.Service{},
				Email:              "xyz@ibm.com",
				Password:           "cGFzc3dvcmQ=",
				SystemChannel:      "testchainid",
				ImagePullSecrets:   []string{"testsecret"},
				RegistryURL:        "ghcr.io/ibm-blockchain/",
				Kubeconfig:         &[]byte{},
				Images: &current.ConsoleImages{
					ConsoleInitImage:   "fake-init-image",
					ConsoleInitTag:     "1234",
					CouchDBImage:       "fake-couchdb-image",
					CouchDBTag:         "1234",
					ConsoleImage:       "fake-console-image",
					ConsoleTag:         "1234",
					ConfigtxlatorImage: "fake-configtxlator-image",
					ConfigtxlatorTag:   "1234",
					DeployerImage:      "fake-deployer-image",
					DeployerTag:        "1234",
				},
				NetworkInfo: &current.NetworkInfo{
					Domain:      "test.domain",
					ConsolePort: 31010,
					ProxyPort:   31011,
				},
				TLSSecretName: "secret",
				Resources:     &current.ConsoleResources{},
				Storage: &current.ConsoleStorage{
					Console: &current.StorageSpec{
						Size:  "100m",
						Class: "manual",
					},
				},
				PasswordSecretName: "password",
				Versions:           &current.Versions{},
				ConnectionString:   "https://localhost",
			},
		}
		instance.Kind = "IBPConsole"
		instance.Status.Version = version.Operator

		console = &k8sconsole.Console{
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
			IngressManager:        ingressMgr,
			Ingressv1beta1Manager: ingressv1beta1Mgr,
		}
	})

	Context("Reconciles", func() {
		It("returns an error if pvc manager fails to reconcile", func() {
			pvcMgr.ReconcileReturns(errors.New("failed to reconcile pvc"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed PVC reconciliation: failed to reconcile pvc"))
		})

		It("returns an error if service manager fails to reconcile", func() {
			serviceMgr.ReconcileReturns(errors.New("failed to reconcile service"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Service reconciliation: failed to reconcile service"))
		})

		It("returns an error if deployment manager fails to reconcile", func() {
			deploymentMgr.ReconcileReturns(errors.New("failed to reconcile deployment"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Deployment reconciliation: failed to reconcile deployment"))
		})

		It("returns an error if role manager fails to reconcile", func() {
			roleMgr.ReconcileReturns(errors.New("failed to reconcile role"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed RBAC reconciliation: failed to reconcile role"))
		})

		It("returns an error if role binding manager fails to reconcile", func() {
			roleBindingMgr.ReconcileReturns(errors.New("failed to reconcile role binding"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed RBAC reconciliation: failed to reconcile role binding"))
		})

		It("returns an error if service account binding manager fails to reconcile", func() {
			serviceAccountMgr.ReconcileReturns(errors.New("failed to reconcile service account"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed RBAC reconciliation: failed to reconcile service account"))
		})

		It("returns an error if config map manager fails to reconcile", func() {
			configMapMgr.ReconcileReturns(errors.New("failed to reconcile config map"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed ConfigMap reconciliation: failed to reconcile config map"))
		})

		It("returns an error if config map manager fails to reconcile", func() {
			consoleConfigMapMgr.ReconcileReturns(errors.New("failed to reconcile config map"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Console ConfigMap reconciliation: failed to reconcile config map"))
		})

		It("returns an error if config map manager fails to reconcile", func() {
			deployerConfigMapMgr.ReconcileReturns(errors.New("failed to reconcile config map"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Deployer ConfigMap reconciliation: failed to reconcile config map"))
		})

		It("returns an error if ingress manager fails to reconcile", func() {
			ingressMgr.ReconcileReturns(errors.New("failed to reconcile ingress"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Ingress reconciliation: failed to reconcile ingress"))
		})

		It("restarts pods by deleting deployment", func() {
			update.RestartNeededReturns(true)
			_, err := console.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockKubeClient.PatchCallCount()).To(Equal(1))
		})

		It("does not return an error on a successful reconcile", func() {
			_, err := console.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("ValidateSpec", func() {
		It("returns no error if valid spec is passed", func() {
			err := console.ValidateSpec(instance)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error if license is not accepted", func() {
			instance.Spec.License.Accept = false
			err := console.ValidateSpec(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("user must accept license before continuing"))
		})

		It("returns error if serviceaccountname is not passed", func() {
			instance.Spec.ServiceAccountName = ""
			err := console.ValidateSpec(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Service account name not provided"))
		})

		It("returns error if email is not passed", func() {
			instance.Spec.Email = ""
			err := console.ValidateSpec(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("email not provided"))
		})

		It("returns error if password & passwordsecret are not passed", func() {
			instance.Spec.PasswordSecretName = ""
			instance.Spec.Password = ""
			err := console.ValidateSpec(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("password and passwordSecretName both not provided, at least one expected"))
		})

		It("should not return error if password & passwordsecret are not passed when authscheme is ibmid", func() {
			instance.Spec.AuthScheme = "ibmid"
			instance.Spec.PasswordSecretName = ""
			instance.Spec.Password = ""
			err := console.ValidateSpec(instance)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if imagepullsecret is not passed", func() {
			instance.Spec.ImagePullSecrets = nil
			err := console.ValidateSpec(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("imagepullsecrets required"))
		})

		It("returns error if ingress info are not passed", func() {
			instance.Spec.NetworkInfo = nil
			err := console.ValidateSpec(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("network information not provided"))
		})
	})
})
