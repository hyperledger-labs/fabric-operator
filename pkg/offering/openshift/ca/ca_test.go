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

package openshiftca_test

import (
	"encoding/json"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"context"

	corev1 "k8s.io/api/core/v1"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	managermocks "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/mocks"
	baseca "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca"
	basecamocks "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca/mocks"
	openshiftca "github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/ca"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/ca/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Openshift CA", func() {
	const (
		defaultConfigs = "../../../../defaultconfig/ca"
		testdataDir    = "../../../../testdata"

		testCert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo"
	)

	var (
		ca             *openshiftca.CA
		instance       *current.IBPCA
		mockKubeClient *mocks.Client

		deploymentMgr          *managermocks.ResourceManager
		serviceMgr             *managermocks.ResourceManager
		pvcMgr                 *managermocks.ResourceManager
		roleMgr                *managermocks.ResourceManager
		roleBindingMgr         *managermocks.ResourceManager
		serviceAccountMgr      *managermocks.ResourceManager
		caRouteManager         *managermocks.ResourceManager
		operationsRouteManager *managermocks.ResourceManager

		initMock *basecamocks.InitializeIBPCA
		update   *basecamocks.Update
		certMgr  *basecamocks.CertificateManager
	)

	Context("Reconciles", func() {
		BeforeEach(func() {
			mockKubeClient = &mocks.Client{}
			update = &basecamocks.Update{}

			replicas := int32(1)
			instance = &current.IBPCA{
				TypeMeta: metav1.TypeMeta{
					Kind: "IBPCA",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ca1",
					Namespace: "test",
				},
				Spec: current.IBPCASpec{
					Domain:        "domain",
					Images:        &current.CAImages{},
					Replicas:      &replicas,
					FabricVersion: "1.4.9-0",
				},
				Status: current.IBPCAStatus{
					CRStatus: current.CRStatus{
						Version: version.Operator,
					},
				},
			}

			mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *corev1.Secret:
					o := obj.(*corev1.Secret)
					switch types.Name {
					case instance.Name + "-ca-crypto":
						o.Name = instance.Name + "-ca-crypto"
						o.Namespace = instance.Namespace
						o.Data = map[string][]byte{"tls-cert.pem": []byte(testCert)}
					case instance.Name + "-tlsca-crypto":
						o.Name = instance.Name + "-tlsca-crypto"
						o.Namespace = instance.Namespace
						o.Data = map[string][]byte{"cert.pem": []byte(testCert)}
					}
				}
				return nil
			}
			deploymentMgr = &managermocks.ResourceManager{}
			serviceMgr = &managermocks.ResourceManager{}
			pvcMgr = &managermocks.ResourceManager{}
			roleMgr = &managermocks.ResourceManager{}
			roleBindingMgr = &managermocks.ResourceManager{}
			serviceAccountMgr = &managermocks.ResourceManager{}
			caRouteManager = &managermocks.ResourceManager{}
			operationsRouteManager = &managermocks.ResourceManager{}
			initMock = &basecamocks.InitializeIBPCA{}
			restartMgr := &basecamocks.RestartManager{}
			certMgr = &basecamocks.CertificateManager{}

			cfg := &config.Config{
				CAInitConfig: &initializer.Config{
					CADefaultConfigPath:     filepath.Join(defaultConfigs, "/ca.yaml"),
					CAOverrideConfigPath:    filepath.Join(testdataDir, "init/override.yaml"),
					TLSCADefaultConfigPath:  filepath.Join(defaultConfigs, "tlsca.yaml"),
					TLSCAOverrideConfigPath: filepath.Join(testdataDir, "init/override.yaml"),
					SharedPath:              "shared",
				},
				Operator: config.Operator{
					Versions: &deployer.Versions{
						CA: map[string]deployer.VersionCA{
							"1.4.9-0": {},
						},
					},
				},
			}

			certMgr.GetSecretReturns(&corev1.Secret{}, nil)
			deploymentMgr.ExistsReturns(true)
			ca = &openshiftca.CA{
				CA: &baseca.CA{
					Client:                mockKubeClient,
					Scheme:                &runtime.Scheme{},
					DeploymentManager:     deploymentMgr,
					ServiceManager:        serviceMgr,
					PVCManager:            pvcMgr,
					RoleManager:           roleMgr,
					RoleBindingManager:    roleBindingMgr,
					ServiceAccountManager: serviceAccountMgr,
					Override:              &override.Override{},
					Config:                cfg,
					Initializer:           initMock,
					Restart:               restartMgr,
					CertificateManager:    certMgr,
				},
				CARouteManager:         caRouteManager,
				OperationsRouteManager: operationsRouteManager,
				Override:               &override.Override{},
			}
		})

		It("returns a breaking error if initialization fails", func() {
			initMock.HandleEnrollmentCAInitReturns(nil, errors.New("failed to init"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Code: 20 - failed to initialize ca: failed to init"))
			Expect(operatorerrors.IsBreakingError(err, "msg", nil)).NotTo(HaveOccurred())
		})

		It("returns an error if pvc manager fails to reconcile", func() {
			pvcMgr.ReconcileReturns(errors.New("failed to reconcile pvc"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed PVC reconciliation: failed to reconcile pvc"))
		})

		It("returns an error if service manager fails to reconcile", func() {
			serviceMgr.ReconcileReturns(errors.New("failed to reconcile service"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Service reconciliation: failed to reconcile service"))
		})

		It("returns an error if role manager fails to reconcile", func() {
			roleMgr.ReconcileReturns(errors.New("failed to reconcile role"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to reconcile role"))
		})

		It("returns an error if role binding manager fails to reconcile", func() {
			roleBindingMgr.ReconcileReturns(errors.New("failed to reconcile role binding"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to reconcile role binding"))
		})

		It("returns an error if service account manager fails to reconcile", func() {
			serviceAccountMgr.ReconcileReturns(errors.New("failed to reconcile service account"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to reconcile service account"))
		})

		It("returns an error if deployment manager fails to reconcile", func() {
			deploymentMgr.ReconcileReturns(errors.New("failed to reconcile deployment"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Deployment reconciliation: failed to reconcile deployment"))
		})

		It("returns an error if ca route manager fails to reconcile", func() {
			caRouteManager.ReconcileReturns(errors.New("failed to reconcile ca route"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed CA Route reconciliation: failed to reconcile ca route"))
		})

		It("returns an error if operations route manager fails to reconcile", func() {
			operationsRouteManager.ReconcileReturns(errors.New("failed to reconcile operations route"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Operations Route reconciliation: failed to reconcile operations route"))
		})

		It("returns an error if restart fails", func() {
			update.RestartNeededReturns(true)
			mockKubeClient.PatchReturns(errors.New("patch failed"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).Should(MatchError(ContainSubstring("patch failed")))
		})

		It("reconciles IBPCA", func() {
			_, err := ca.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("AddTLSCryptoIfMissing", func() {
		It("adds tls crypto", func() {
			mockKubeClient.GetReturns(errors.New("fake error"))
			err := ca.AddTLSCryptoIfMissing(instance, &current.CAEndpoints{})
			Expect(err).NotTo(HaveOccurred())

			caOverrides := &v1.ServerConfig{}
			err = json.Unmarshal(instance.Spec.ConfigOverride.CA.Raw, caOverrides)
			Expect(err).NotTo(HaveOccurred())

			Expect(caOverrides.TLS.CertFile).NotTo(Equal(""))
			Expect(caOverrides.TLS.KeyFile).NotTo(Equal(""))
		})
	})
})
