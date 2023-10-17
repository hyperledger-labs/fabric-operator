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

package baseconsole_test

import (
	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	cmocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	consolev1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/console/v1"
	managermocks "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/mocks"
	baseconsole "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/console/mocks"
	"github.com/IBM-Blockchain/fabric-operator/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("Base Console", func() {
	var (
		console        *baseconsole.Console
		instance       *current.IBPConsole
		mockKubeClient *cmocks.Client

		deploymentMgr        *managermocks.ResourceManager
		serviceMgr           *managermocks.ResourceManager
		deployerServiceMgr   *managermocks.ResourceManager
		pvcMgr               *managermocks.ResourceManager
		configMapMgr         *managermocks.ResourceManager
		consoleConfigMapMgr  *managermocks.ResourceManager
		deployerConfigMapMgr *managermocks.ResourceManager
		roleMgr              *managermocks.ResourceManager
		roleBindingMgr       *managermocks.ResourceManager
		serviceAccountMgr    *managermocks.ResourceManager
		update               *mocks.Update
		restartMgr           *mocks.RestartManager
	)

	BeforeEach(func() {
		logf.SetLogger(zap.New())
		mockKubeClient = &cmocks.Client{}
		update = &mocks.Update{}
		restartMgr = &mocks.RestartManager{}

		deploymentMgr = &managermocks.ResourceManager{}
		serviceMgr = &managermocks.ResourceManager{}
		deployerServiceMgr = &managermocks.ResourceManager{}
		pvcMgr = &managermocks.ResourceManager{}
		configMapMgr = &managermocks.ResourceManager{}
		consoleConfigMapMgr = &managermocks.ResourceManager{}
		deployerConfigMapMgr = &managermocks.ResourceManager{}
		roleMgr = &managermocks.ResourceManager{}
		roleBindingMgr = &managermocks.ResourceManager{}
		serviceAccountMgr = &managermocks.ResourceManager{}

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
				RegistryURL:        "ghcr.io/ibm-blockchain/ibp-temp/",
				Kubeconfig:         &[]byte{},
				ConnectionString:   "https://localhost",
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
				Versions:           &current.Versions{},
				PasswordSecretName: "password",
			},
		}
		instance.Kind = "IBPConsole"
		instance.Status.Version = version.Operator

		console = &baseconsole.Console{
			Client: mockKubeClient,
			Scheme: &runtime.Scheme{},
			Config: &config.Config{},

			DeploymentManager:        deploymentMgr,
			ServiceManager:           serviceMgr,
			DeployerServiceManager:   deployerServiceMgr,
			PVCManager:               pvcMgr,
			ConfigMapManager:         configMapMgr,
			ConsoleConfigMapManager:  consoleConfigMapMgr,
			DeployerConfigMapManager: deployerConfigMapMgr,
			RoleManager:              roleMgr,
			RoleBindingManager:       roleBindingMgr,
			ServiceAccountManager:    serviceAccountMgr,
			Restart:                  restartMgr,
		}
	})

	Context("Reconciles", func() {
		It("returns nil and request will be requeued if instance version is updated", func() {
			instance.Status.Version = ""
			_, err := console.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockKubeClient.PatchStatusCallCount()).To(Equal(1))
		})
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

		It("returns no error if dev mode is disabled & deployer service manager fails to reconcile", func() {
			deployerServiceMgr.ReconcileReturns(errors.New("failed to reconcile service"))
			_, err := console.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error if deployer service manager fails to reconcile", func() {
			instance.Spec.FeatureFlags = &consolev1.FeatureFlags{
				DevMode: true,
			}
			deployerServiceMgr.ReconcileReturns(errors.New("failed to reconcile service"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Deployer Service reconciliation: failed to reconcile service"))
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

		It("restarts pods by deleting deployment", func() {
			update.RestartNeededReturns(true)
			_, err := console.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockKubeClient.PatchCallCount()).To(Equal(1))
		})

		It("returns error if trigger restart fails", func() {
			restartMgr.TriggerIfNeededReturns(errors.New("failed to trigger restart"))
			_, err := console.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to restart deployment: failed to trigger restart"))
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
			instance.Spec.ImagePullSecrets = []string{}
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

	Context("CreateCouchdbCredentials", func() {
		It("does not update connectionstring if it is not blank", func() {
			connectionString := "https://fake.url"
			instance.Spec.ConnectionString = connectionString
			updated := console.CreateCouchdbCredentials(instance)
			Expect(updated).To(BeFalse())
			Expect(instance.Spec.ConnectionString).To(Equal(connectionString))
		})

		It("does not update connectionstring if it is not blank & is https", func() {
			connectionString := "https://localhost:5984"
			instance.Spec.ConnectionString = connectionString
			updated := console.CreateCouchdbCredentials(instance)
			Expect(updated).To(BeFalse())
			Expect(instance.Spec.ConnectionString).To(Equal(connectionString))
		})

		It("does update connectionstring if it is missing creds", func() {
			connectionString := "http://localhost:5984"
			instance.Spec.ConnectionString = connectionString
			updated := console.CreateCouchdbCredentials(instance)
			Expect(updated).To(BeTrue())
			Expect(instance.Spec.ConnectionString).NotTo(Equal(connectionString))
		})

		It("does not update connectionstring if it is has creds already", func() {
			connectionString := "http://user:pass@localhost:5984"
			instance.Spec.ConnectionString = connectionString
			updated := console.CreateCouchdbCredentials(instance)
			Expect(updated).To(BeFalse())
			Expect(instance.Spec.ConnectionString).To(Equal(connectionString))
		})
	})
})
