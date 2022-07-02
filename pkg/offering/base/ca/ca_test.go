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

package baseca_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	cmocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	managermocks "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/mocks"
	baseca "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca/mocks"
	basecamocks "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca/mocks"
	override "github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/ca/override"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/version"
)

var _ = Describe("Base CA", func() {
	const (
		defaultConfigs = "../../../../defaultconfig/ca"
		testdataDir    = "../../../../testdata"

		keyBase64  = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBdFJBUDlMemUyZEc1cm1rbmcvdVVtREFZU0VwUElqRFdUUDhqUjMxcUJ5Yjc3YWUrCnk3UTRvRnZod1lDVUhsUWVTWjFKeTdUUHpEcitoUk5hdDJYNGdGYUpGYmVFbC9DSHJ3Rk1mNzNzQStWV1pHdnkKdXhtbjB2bEdYMW5zSEo5aUdIUS9qR2FvV1FJYzlVbnpHWi8yWStlZkpxOWd3cDBNemFzWWZkdXordXVBNlp4VAp5TTdDOWFlWmxYL2ZMYmVkSXVXTzVzaXhPSlZQeUVpcWpkd0RiY1AxYy9mRCtSMm1DbmM3VGovSnVLK1poTGxPCnhGcVlFRmtROHBmSi9LY1pabVF1QURZVFh6RGp6OENxcTRTRU5ySzI0b2hQQkN2SGgyanplWjhGdGR4MmpSSFQKaXdCZWZEYWlSWVBSOUM4enk4K1Z2Wmt6S0hQV3N5aENiNUMrN1FJREFRQUJBb0lCQUZROGhzL2IxdW9Mc3BFOApCdEJXaVVsTWh0K0xBc25yWXFncnd5UU5hdmlzNEdRdXVJdFk2MGRmdCtZb2hjQ2ViZ0RkbG1tWlUxdTJ6cGJtCjdEdUt5MVFaN21rV0dpLytEWUlUM3AxSHBMZ2pTRkFzRUorUFRnN1BQamc2UTZrRlZjUCt3Vm4yb0xmWVRkU28KZE5zbEdxSmNMaVQzVHRMNzhlcjFnTTE5RzN6T3J1ZndrSGJSYU1BRmtvZ1ExUlZLSWpnVGUvbmpIMHFHNW9JagoxNEJLeFFKTUZFTG1pQk50NUx5OVMxWWdxTDRjbmNtUDN5L1QyNEdodVhNckx0eTVOeVhnS0dFZ1pUTDMzZzZvCnYreDFFMFRURWRjMVQvWVBGWkdBSXhHdWRKNWZZZ2JtWU9LZ09mUHZFOE9TbEV6OW56aHNnckVZYjdQVThpZDUKTHFycVJRRUNnWUVBNjIyT3RIUmMxaVY1ZXQxdHQydTVTTTlTS2h2b0lPT3d2Q3NnTEI5dDJzNEhRUlRYN0RXcAo0VDNpUC9leEl5OXI3bTIxNFo5MEgzZlpVNElSUkdHSUxKUVMrYzRQNVA4cHJFTDcyd1dIWlpQTTM3QlZTQ1U3CkxOTXl4TkRjeVdjSUJIVFh4NUY2eXhLNVFXWTg5MVB0eDlDamJFSEcrNVJVdDA4UVlMWDlUQTBDZ1lFQXhPSmYKcXFjeThMOVZyYUFVZG9lbGdIU0NGSkJRR3hMRFNSQlJSTkRIOUJhaWlZOCtwZzd2TExTRXFMRFpsbkZPbFkrQQpiRENEQ0RtdHhwRXViY0x6b3FnOXhlQTZ0eXZZWkNWalY5dXVzNVh1Wmk1VDBBUHhCdm56OHNNa3dRY3RQWkRQCk8zQTN4WllkZzJBRmFrV1BmT1FFbjVaK3F4TU13SG9VZ1ZwQkptRUNnWUJ2Q2FjcTJVOEgrWGpJU0ROOU5TT1kKZ1ovaEdIUnRQcmFXcVVodFJ3MkxDMjFFZHM0NExEOUphdVNSQXdQYThuelhZWXROTk9XU0NmYkllaW9tdEZHRApwUHNtTXRnd1MyQ2VUS0Y0OWF5Y2JnOU0yVi8vdlAraDdxS2RUVjAwNkpGUmVNSms3K3FZYU9aVFFDTTFDN0swCmNXVUNwQ3R6Y014Y0FNQmF2THNRNlFLQmdHbXJMYmxEdjUxaXM3TmFKV0Z3Y0MwL1dzbDZvdVBFOERiNG9RV1UKSUowcXdOV2ZvZm95TGNBS3F1QjIrbkU2SXZrMmFiQ25ZTXc3V0w4b0VJa3NodUtYOVgrTVZ6Y1VPekdVdDNyaQpGeU9mcHJJRXowcm5zcWNSNUJJNUZqTGJqVFpyMEMyUWp2NW5FVFAvaHlpQWFRQ1l5THAyWlVtZ0Vjb0VPNWtwClBhcEJBb0dBZVV0WjE0SVp2cVorQnAxR1VqSG9PR0pQVnlJdzhSRUFETjRhZXRJTUlQRWFVaDdjZUtWdVN6VXMKci9WczA1Zjg0cFBVaStuUTUzaGo2ZFhhYTd1UE1aMFBnNFY4cS9UdzJMZ3BWWndVd0ltZUQrcXNsbldha3VWMQpMSnp3SkhOa3pOWE1OMmJWREFZTndSamNRSmhtbzF0V2xHYlpRQjNoSkEwR2thWGZPa2c9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg=="
		certBase64 = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURBekNDQWV1Z0F3SUJBZ0lKQU9xQ1VmaFNjcWtlTUEwR0NTcUdTSWIzRFFFQkJRVUFNQmd4RmpBVUJnTlYKQkFNTURYQnZjM1JuY21WekxuUmxjM1F3SGhjTk1Ua3dOekl6TVRrd09UVTRXaGNOTWprd056SXdNVGt3T1RVNApXakFZTVJZd0ZBWURWUVFEREExd2IzTjBaM0psY3k1MFpYTjBNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DCkFROEFNSUlCQ2dLQ0FRRUF0UkFQOUx6ZTJkRzVybWtuZy91VW1EQVlTRXBQSWpEV1RQOGpSMzFxQnliNzdhZSsKeTdRNG9Gdmh3WUNVSGxRZVNaMUp5N1RQekRyK2hSTmF0Mlg0Z0ZhSkZiZUVsL0NIcndGTWY3M3NBK1ZXWkd2eQp1eG1uMHZsR1gxbnNISjlpR0hRL2pHYW9XUUljOVVuekdaLzJZK2VmSnE5Z3dwME16YXNZZmR1eit1dUE2WnhUCnlNN0M5YWVabFgvZkxiZWRJdVdPNXNpeE9KVlB5RWlxamR3RGJjUDFjL2ZEK1IybUNuYzdUai9KdUsrWmhMbE8KeEZxWUVGa1E4cGZKL0tjWlptUXVBRFlUWHpEano4Q3FxNFNFTnJLMjRvaFBCQ3ZIaDJqemVaOEZ0ZHgyalJIVAppd0JlZkRhaVJZUFI5Qzh6eTgrVnZaa3pLSFBXc3loQ2I1Qys3UUlEQVFBQm8xQXdUakFkQmdOVkhRNEVGZ1FVCi9mZ01BcExIMXBvcFFoS25KTmgrVk04QUtQZ3dId1lEVlIwakJCZ3dGb0FVL2ZnTUFwTEgxcG9wUWhLbkpOaCsKVk04QUtQZ3dEQVlEVlIwVEJBVXdBd0VCL3pBTkJna3Foa2lHOXcwQkFRVUZBQU9DQVFFQURjOUc4M05LaWw3ZQpoVFlvR1piejhFV1o4c0puVnY4azMwRDlydUY1OXFvT0ppZGorQUhNbzNHOWtud1lvbGFGbmJwb093cElOZ3g1CnYvL21aU3VldlFMZUZKRlN1UjBheVQ1WFYxcjljNUZGQ2JSaEp0cE4rOEdTT29tRUFSYTNBVGVFSG5WeVpaYkMKWkFQQUxMVXlVeUVrSDR3Q0RZUGtYa3dWQVVlR2FGVmNqZWR0eGJ3Z2k0dG0rSFZoTEt5Y0NoZ25YUVhxQ2srTwo2RHJIc0Z0STVTNWQvQlBPbE1Yc28vNUFielBGelpVVVg4OEhkVUhWSWlqM0luMXdUbWhtREtwdzZ6dmcvNjIxCjRhcGhDOWJ2bXAxeUVOUklzb0xiMGlMWVAzRSswU0ZkZC9IRnRhVXV3eUx6cnl4R2xrdG1BVUJWNVdYZEQxMkIKTU1mQnhvNFVYUT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
	)

	AfterEach(func() {
		err := os.RemoveAll("shared")
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		ca             *baseca.CA
		instance       *current.IBPCA
		mockKubeClient *cmocks.Client

		deploymentMgr     *managermocks.ResourceManager
		serviceMgr        *managermocks.ResourceManager
		pvcMgr            *managermocks.ResourceManager
		roleMgr           *managermocks.ResourceManager
		roleBindingMgr    *managermocks.ResourceManager
		serviceAccountMgr *managermocks.ResourceManager

		initMock   *basecamocks.InitializeIBPCA
		update     *mocks.Update
		restartMgr *basecamocks.RestartManager
		certMgr    *basecamocks.CertificateManager
	)

	BeforeEach(func() {
		mockKubeClient = &cmocks.Client{}
		update = &mocks.Update{}

		replicas := int32(1)
		instance = &current.IBPCA{
			Status: current.IBPCAStatus{
				CRStatus: current.CRStatus{
					Version: version.Operator,
				},
			},
			Spec: current.IBPCASpec{
				Domain: "domain",
				HSM: &current.HSM{
					PKCS11Endpoint: "tcp://0.0.0.0:2345",
				},
				Images: &current.CAImages{
					CAImage:     "caimage",
					CATag:       "2.0.0",
					CAInitImage: "cainitimage",
					CAInitTag:   "2.0.0",
				},
				Replicas:      &replicas,
				FabricVersion: "1.4.9-0",
			},
		}
		instance.Kind = "IBPCA"
		instance.Name = "ca1"
		instance.Namespace = "test"

		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *corev1.Secret:
				o := obj.(*corev1.Secret)
				switch types.Name {
				case instance.Name + "-ca-crypto":
					o.Name = instance.Name + "-ca-crypto"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{
						"tls-cert.pem":        []byte(certBase64),
						"cert.pem":            []byte(certBase64),
						"operations-cert.pem": []byte(certBase64),
					}
				case instance.Name + "-tlsca-crypto":
					o.Name = instance.Name + "-tlsca-crypto"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{
						"cert.pem": []byte(certBase64),
					}
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
		initMock = &basecamocks.InitializeIBPCA{}
		restartMgr = &basecamocks.RestartManager{}
		certMgr = &basecamocks.CertificateManager{}

		initMock.SyncDBConfigReturns(instance, nil)

		config := &config.Config{
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
						"1.4.9-0": {
							Default: true,
							Image: deployer.CAImages{
								CAImage:     "caimage",
								CATag:       "1.4.9",
								CAInitImage: "cainitimage",
								CAInitTag:   "1.4.9",
							},
						},
					},
				},
			},
		}

		deploymentMgr.ExistsReturns(true)
		ca = &baseca.CA{
			DeploymentManager:     deploymentMgr,
			ServiceManager:        serviceMgr,
			PVCManager:            pvcMgr,
			RoleManager:           roleMgr,
			RoleBindingManager:    roleBindingMgr,
			ServiceAccountManager: serviceAccountMgr,
			Client:                mockKubeClient,
			Scheme:                &runtime.Scheme{},
			Override:              &override.Override{},
			Config:                config,
			Initializer:           initMock,
			Restart:               restartMgr,
			CertificateManager:    certMgr,
		}
	})

	Context("Reconciles", func() {
		It("requeues request and returns nil if instance version is updated", func() {
			instance.Status.CRStatus.Version = ""
			_, err := ca.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockKubeClient.PatchStatusCallCount()).To(Equal(1))
		})

		It("returns a breaking error if initialization fails", func() {
			initMock.HandleEnrollmentCAInitReturns(nil, errors.New("failed to init"))
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Code: 20 - failed to initialize ca: failed to init"))
			Expect(operatorerrors.IsBreakingError(err, "msg", nil)).NotTo(HaveOccurred())
		})

		It("returns an error for invalid HSM endpoint", func() {
			instance.Spec.HSM.PKCS11Endpoint = "tcp://:2345"
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("failed pre reconcile checks: invalid HSM endpoint for ca instance '%s': missing IP address", instance.Name)))
		})

		It("returns an error domain is not set", func() {
			instance.Spec.Domain = ""
			_, err := ca.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("failed pre reconcile checks: domain not set for ca instance '%s'", instance.Name)))
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

	Context("initialize", func() {
		It("returns an error if enrollment ca init fails", func() {
			msg := "failed to init enrollment ca"
			initMock.HandleEnrollmentCAInitReturns(nil, errors.New(msg))
			err := ca.Initialize(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if unable to create create config resources for enrollment ca", func() {
			msg := "failed to create config resources for enrollment ca"
			initMock.HandleEnrollmentCAInitReturns(&initializer.Response{}, nil)
			initMock.HandleConfigResourcesReturns(errors.New(msg))
			err := ca.Initialize(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if tls ca init fails", func() {
			msg := "failed to init tls ca"
			initMock.HandleTLSCAInitReturns(nil, errors.New(msg))
			err := ca.Initialize(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if unable to create create config resources for tls ca", func() {
			msg := "failed to create config resources for tls ca"
			initMock.HandleEnrollmentCAInitReturns(&initializer.Response{Config: &v1.ServerConfig{}}, nil)
			initMock.HandleTLSCAInitReturns(&initializer.Response{Config: &v1.ServerConfig{}}, nil)
			initMock.HandleConfigResourcesReturnsOnCall(1, errors.New(msg))
			err := ca.Initialize(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("triggers deployment restart if deployment exists and overrides update detected", func() {
			deploymentMgr.ExistsReturns(true)
			update.ConfigOverridesUpdatedReturns(true)

			err := ca.Initialize(instance, update)
			Expect(err).NotTo(HaveOccurred())
			Expect(restartMgr.ForConfigOverrideCallCount()).To(Equal(1))
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

	Context("image overrides", func() {
		var images *current.CAImages

		Context("using registry url", func() {
			BeforeEach(func() {
				images = &current.CAImages{
					CAImage:     "caimage",
					CATag:       "2.0.0",
					CAInitImage: "cainitimage",
					CAInitTag:   "2.0.0",
				}
			})

			It("overrides images based with registry url and does not append more value on each call", func() {
				images.Override(images, "ghcr.io/ibm-blockchain/", "amd64")
				Expect(images.CAImage).To(Equal("ghcr.io/ibm-blockchain/caimage"))
				Expect(images.CATag).To(Equal("2.0.0"))
				Expect(images.CAInitImage).To(Equal("ghcr.io/ibm-blockchain/cainitimage"))
				Expect(images.CAInitTag).To(Equal("2.0.0"))
			})

			It("overrides images based with registry url and does not append more value on each call", func() {
				images.Override(images, "ghcr.io/ibm-blockchain/images/", "s390x")
				Expect(images.CAImage).To(Equal("ghcr.io/ibm-blockchain/images/caimage"))
				Expect(images.CATag).To(Equal("2.0.0"))
				Expect(images.CAInitImage).To(Equal("ghcr.io/ibm-blockchain/images/cainitimage"))
				Expect(images.CAInitTag).To(Equal("2.0.0"))

			})
		})

		Context("using fully qualified path", func() {
			BeforeEach(func() {
				images = &current.CAImages{
					CAImage:     "ghcr.io/ibm-blockchain/caimage",
					CATag:       "2.0.0",
					CAInitImage: "ghcr.io/ibm-blockchain/cainitimage",
					CAInitTag:   "2.0.0",
				}
			})

			It("keeps images and adds arch to tag", func() {
				images.Override(images, "", "s390")
				Expect(images.CAImage).To(Equal("ghcr.io/ibm-blockchain/caimage"))
				Expect(images.CATag).To(Equal("2.0.0"))
				Expect(images.CAInitImage).To(Equal("ghcr.io/ibm-blockchain/cainitimage"))
				Expect(images.CAInitTag).To(Equal("2.0.0"))
			})
		})
	})

	Context("pre reconcile checks", func() {
		Context("version and images", func() {
			Context("create CR", func() {
				It("returns an error if fabric version is not set in spec", func() {
					instance.Spec.FabricVersion = ""
					_, err := ca.PreReconcileChecks(instance, update)
					Expect(err).To(MatchError(ContainSubstring("fabric version is not set")))
				})

				Context("images section blank", func() {
					BeforeEach(func() {
						instance.Spec.Images = nil
					})

					It("normalizes fabric version and requests a requeue", func() {
						instance.Spec.FabricVersion = "1.4.9"
						requeue, err := ca.PreReconcileChecks(instance, update)
						Expect(err).NotTo(HaveOccurred())
						Expect(requeue).To(Equal(true))
						Expect(instance.Spec.FabricVersion).To(Equal("1.4.9-0"))
					})

					It("returns an error if fabric version not supported", func() {
						instance.Spec.FabricVersion = "0.0.1"
						_, err := ca.PreReconcileChecks(instance, update)
						Expect(err).To(MatchError(ContainSubstring("fabric version '0.0.1' is not supported")))
					})

					When("version is passed without hyphen", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = "1.4.9"
						})

						It("finds default version for release and updates images section", func() {
							requeue, err := ca.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(true))
							Expect(*instance.Spec.Images).To(Equal(current.CAImages{
								CAImage:     "caimage",
								CATag:       "1.4.9",
								CAInitImage: "cainitimage",
								CAInitTag:   "1.4.9",
							}))
						})
					})

					When("version is passed with hyphen", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = "1.4.9-0"
						})

						It("looks images and updates images section", func() {
							requeue, err := ca.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(true))
							Expect(*instance.Spec.Images).To(Equal(current.CAImages{
								CAImage:     "caimage",
								CATag:       "1.4.9",
								CAInitImage: "cainitimage",
								CAInitTag:   "1.4.9",
							}))
						})
					})
				})

				Context("images section passed", func() {
					BeforeEach(func() {
						instance.Spec.Images = &current.CAImages{
							CAImage:     "ghcr.io/ibm-blockchain/caimage",
							CATag:       "2.0.0",
							CAInitImage: "ghcr.io/ibm-blockchain/cainitimage",
							CAInitTag:   "2.0.0",
						}
					})

					When("version is not passed", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = ""
						})

						It("returns an error", func() {
							_, err := ca.PreReconcileChecks(instance, update)
							Expect(err).To(MatchError(ContainSubstring("fabric version is not set")))
						})
					})

					When("version is passed", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = "2.0.0-8"
						})

						It("persists current spec configuration", func() {
							requeue, err := ca.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("2.0.0-8"))
							Expect(*instance.Spec.Images).To(Equal(current.CAImages{
								CAImage:     "ghcr.io/ibm-blockchain/caimage",
								CATag:       "2.0.0",
								CAInitImage: "ghcr.io/ibm-blockchain/cainitimage",
								CAInitTag:   "2.0.0",
							}))
						})
					})
				})
			})

			Context("update CR", func() {
				BeforeEach(func() {
					instance.Spec.FabricVersion = "2.0.1-0"
					instance.Spec.Images = &current.CAImages{
						CAImage:     "ghcr.io/ibm-blockchain/caimage",
						CATag:       "2.0.1",
						CAInitImage: "ghcr.io/ibm-blockchain/cainitimage",
						CAInitTag:   "2.0.1",
					}
				})

				When("images updated", func() {
					BeforeEach(func() {
						update.ImagesUpdatedReturns(true)
						instance.Spec.Images = &current.CAImages{
							CAImage:     "ghcr.io/ibm-blockchain/caimage",
							CATag:       "2.0.8",
							CAInitImage: "ghcr.io/ibm-blockchain/cainitimage",
							CAInitTag:   "2.0.8",
						}
					})

					Context("and version updated", func() {
						BeforeEach(func() {
							update.FabricVersionUpdatedReturns(true)
							instance.Spec.FabricVersion = "2.0.1-8"
						})

						It("persists current spec configuration", func() {
							requeue, err := ca.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("2.0.1-8"))
							Expect(*instance.Spec.Images).To(Equal(current.CAImages{
								CAImage:     "ghcr.io/ibm-blockchain/caimage",
								CATag:       "2.0.8",
								CAInitImage: "ghcr.io/ibm-blockchain/cainitimage",
								CAInitTag:   "2.0.8",
							}))
						})
					})

					Context("and version not updated", func() {
						It("persists current spec configuration", func() {
							requeue, err := ca.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("2.0.1-0"))
							Expect(*instance.Spec.Images).To(Equal(current.CAImages{
								CAImage:     "ghcr.io/ibm-blockchain/caimage",
								CATag:       "2.0.8",
								CAInitImage: "ghcr.io/ibm-blockchain/cainitimage",
								CAInitTag:   "2.0.8",
							}))
						})
					})
				})

				When("images not updated", func() {
					Context("and version updated during operator migration", func() {
						BeforeEach(func() {
							update.FabricVersionUpdatedReturns(true)
							instance.Spec.FabricVersion = "unsupported"
						})

						It("persists current spec configuration", func() {
							requeue, err := ca.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("unsupported"))
							Expect(*instance.Spec.Images).To(Equal(current.CAImages{
								CAImage:     "ghcr.io/ibm-blockchain/caimage",
								CATag:       "2.0.1",
								CAInitImage: "ghcr.io/ibm-blockchain/cainitimage",
								CAInitTag:   "2.0.1",
							}))
						})
					})

					Context("and version updated (not during operator migration)", func() {
						BeforeEach(func() {
							update.FabricVersionUpdatedReturns(true)
						})

						When("using non-hyphenated version", func() {
							BeforeEach(func() {
								instance.Spec.FabricVersion = "1.4.9"
							})

							It("looks images and updates images section", func() {
								requeue, err := ca.PreReconcileChecks(instance, update)
								Expect(err).NotTo(HaveOccurred())
								Expect(requeue).To(Equal(true))
								Expect(instance.Spec.FabricVersion).To(Equal("1.4.9-0"))
								Expect(*instance.Spec.Images).To(Equal(current.CAImages{
									CAImage:     "caimage",
									CATag:       "1.4.9",
									CAInitImage: "cainitimage",
									CAInitTag:   "1.4.9",
								}))
							})
						})

						When("using hyphenated version", func() {
							BeforeEach(func() {
								instance.Spec.FabricVersion = "1.4.9-0"
							})

							It("looks images and updates images section", func() {
								requeue, err := ca.PreReconcileChecks(instance, update)
								Expect(err).NotTo(HaveOccurred())
								Expect(requeue).To(Equal(true))
								Expect(instance.Spec.FabricVersion).To(Equal("1.4.9-0"))
								Expect(*instance.Spec.Images).To(Equal(current.CAImages{
									CAImage:     "caimage",
									CATag:       "1.4.9",
									CAInitImage: "cainitimage",
									CAInitTag:   "1.4.9",
								}))
							})
						})
					})
				})
			})
		})

		Context("hsm image updates", func() {
			var (
				hsmConfig = &commonconfig.HSMConfig{
					Library: commonconfig.Library{
						Image: "ghcr.io/ibm-blockchain/hsmimage:1.0.0",
					},
				}
			)

			BeforeEach(func() {
				mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
					switch obj.(type) {
					case *corev1.ConfigMap:
						o := obj.(*corev1.ConfigMap)

						bytes, err := yaml.Marshal(hsmConfig)
						Expect(err).NotTo(HaveOccurred())

						o.Data = map[string]string{
							"ibp-hsm-config.yaml": string(bytes),
						}
					}
					return nil
				}
			})

			It("updates hsm image and tag if passed through operator config", func() {
				updated, err := ca.PreReconcileChecks(instance, update)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated).To(Equal(true))
				Expect(instance.Spec.Images.HSMImage).To(Equal("ghcr.io/ibm-blockchain/hsmimage"))
				Expect(instance.Spec.Images.HSMTag).To(Equal("1.0.0"))
			})

			It("doesn't update hsm image and tag if hsm update is disabled", func() {
				hsmConfig.Library.AutoUpdateDisabled = true

				updated, err := ca.PreReconcileChecks(instance, update)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated).To(Equal(false))
				Expect(instance.Spec.Images.HSMImage).To(Equal(""))
				Expect(instance.Spec.Images.HSMTag).To(Equal(""))
			})
		})
	})

	Context("update connection profile", func() {
		It("returns error if fails to get cert", func() {
			mockKubeClient.GetReturns(errors.New("get error"))
			err := ca.UpdateConnectionProfile(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get error"))
		})

		It("updates connection profile cm", func() {
			err := ca.UpdateConnectionProfile(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockKubeClient.GetCallCount()).To(Equal(3))

			_, obj, _ := mockKubeClient.UpdateArgsForCall(0)
			configmap := obj.(*corev1.ConfigMap)
			connectionprofile := &current.CAConnectionProfile{}
			err = json.Unmarshal(configmap.BinaryData["profile.json"], connectionprofile)
			Expect(err).NotTo(HaveOccurred())

			certEncoded := base64.StdEncoding.EncodeToString([]byte(certBase64))
			Expect(connectionprofile.TLS.Cert).To(Equal(certEncoded))
			Expect(connectionprofile.CA.SignCerts).To(Equal(certEncoded))
			Expect(connectionprofile.TLSCA.SignCerts).To(Equal(certEncoded))
		})
	})
})
