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

package openshiftpeer_test

import (
	"context"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	cmocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	peerinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	managermocks "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/mocks"
	basepeer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/mocks"
	peermocks "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/mocks"
	openshiftpeer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/openshift/peer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Openshift Peer", func() {
	var (
		peer           *openshiftpeer.Peer
		instance       *current.IBPPeer
		mockKubeClient *cmocks.Client
		cfg            *config.Config

		deploymentMgr          *peermocks.DeploymentManager
		peerRouteManager       *managermocks.ResourceManager
		operationsRouteManager *managermocks.ResourceManager
		grpcRouteManager       *managermocks.ResourceManager
		update                 *mocks.Update
	)

	Context("Reconciles", func() {
		BeforeEach(func() {
			mockKubeClient = &cmocks.Client{}
			update = &mocks.Update{}

			replicas := int32(1)
			instance = &current.IBPPeer{
				TypeMeta: metav1.TypeMeta{
					Kind: "IBPPeer",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "peer1",
					Namespace: "random",
				},
				Spec: current.IBPPeerSpec{
					PeerExternalEndpoint: "address",
					Domain:               "domain",
					DindArgs:             []string{"fake", "args"},
					StateDb:              "couchdb",
					Replicas:             &replicas,
					Images:               &current.PeerImages{},
					FabricVersion:        "1.4.9",
				},
				Status: current.IBPPeerStatus{
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
					case "tls-" + instance.Name + "-signcert":
						o.Name = "tls-" + instance.Name + "-signcert"
						o.Namespace = instance.Namespace
						o.Data = map[string][]byte{"cert.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
					case "tls-" + instance.Name + "-cacerts":
						o.Name = "tls-" + instance.Name + "-cacerts"
						o.Namespace = instance.Namespace
						o.Data = map[string][]byte{"cert.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
					case "ecert-" + instance.Name + "-signcert":
						o.Name = "ecert-" + instance.Name + "-signcert"
						o.Namespace = instance.Namespace
						o.Data = map[string][]byte{"cert.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
					case "ecert-" + instance.Name + "-cacerts":
						o.Name = "ecert-" + instance.Name + "-cacerts"
						o.Namespace = instance.Namespace
						o.Data = map[string][]byte{"cacert-0.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
					}
				}
				return nil
			}

			deploymentMgr = &peermocks.DeploymentManager{}
			serviceMgr := &managermocks.ResourceManager{}
			pvcMgr := &managermocks.ResourceManager{}
			couchPvcMgr := &managermocks.ResourceManager{}
			configMapMgr := &managermocks.ResourceManager{}
			roleMgr := &managermocks.ResourceManager{}
			roleBindingMgr := &managermocks.ResourceManager{}
			serviceAccountMgr := &managermocks.ResourceManager{}
			certificateMgr := &peermocks.CertificateManager{}
			restartMgr := &peermocks.RestartManager{}

			peerRouteManager = &managermocks.ResourceManager{}
			operationsRouteManager = &managermocks.ResourceManager{}
			grpcRouteManager = &managermocks.ResourceManager{}

			scheme := &runtime.Scheme{}
			cfg = &config.Config{
				PeerInitConfig: &peerinit.Config{
					OUFile:       "../../../../defaultconfig/peer/ouconfig.yaml",
					CorePeerFile: "../../../../defaultconfig/peer/core.yaml",
				},
			}
			initializer := &peermocks.InitializeIBPPeer{}
			initializer.GetInitPeerReturns(&peerinit.Peer{}, nil)
			peer = &openshiftpeer.Peer{
				Peer: &basepeer.Peer{
					Config:                  cfg,
					Client:                  mockKubeClient,
					Scheme:                  scheme,
					DeploymentManager:       deploymentMgr,
					ServiceManager:          serviceMgr,
					PVCManager:              pvcMgr,
					StateDBPVCManager:       couchPvcMgr,
					FluentDConfigMapManager: configMapMgr,
					RoleManager:             roleMgr,
					RoleBindingManager:      roleBindingMgr,
					ServiceAccountManager:   serviceAccountMgr,
					Initializer:             initializer,
					CertificateManager:      certificateMgr,
					Restart:                 restartMgr,
				},
				RouteManager:           peerRouteManager,
				OperationsRouteManager: operationsRouteManager,
				GRPCRouteManager:       grpcRouteManager,
			}
		})

		It("returns an error if peer route manager fails to reconcile", func() {
			peerRouteManager.ReconcileReturns(errors.New("failed to reconcile peer route"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Peer Route reconciliation: failed to reconcile peer route"))
		})

		It("returns an error if operations route manager fails to reconcile", func() {
			operationsRouteManager.ReconcileReturns(errors.New("failed to reconcile operations route"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Operations Route reconciliation: failed to reconcile operations route"))
		})

		It("returns an error if grpc web route manager fails to reconcile", func() {
			grpcRouteManager.ReconcileReturns(errors.New("failed to reconcile grpc web route"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Peer GRPC Route reconciliation: failed to reconcile grpc web route"))
		})

		// Disabling this test because the function uses rest client which cannot be mocked
		// It("adds dind args in CR if not passed", func() {
		// 	mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
		// 		switch obj.(type) {
		// 		case *openshiftv1.ClusterVersion:
		// 			cv := &openshiftv1.ClusterVersion{
		// 				Spec: openshiftv1.ClusterVersionSpec{
		// 					Channel: "stable-4.2",
		// 				},
		// 			}

		// 			obj = cv.DeepCopy()
		// 		}

		// 		return nil

		// 	}
		// 	_, err := peer.SelectDinDArgs(instance)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	Expect(len(instance.Spec.DindArgs)).NotTo(Equal(0))
		// })

		It("returns a breaking error if initialization fails", func() {
			cfg.PeerInitConfig.CorePeerFile = "../../../../defaultconfig/peer/badfile.yaml"
			peer.Initializer = peerinit.New(cfg.PeerInitConfig, nil, nil, nil, nil, enroller.HSMEnrollJobTimeouts{})
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Code: 22 - failed to initialize peer: open"))
			Expect(operatorerrors.IsBreakingError(err, "msg", nil)).NotTo(HaveOccurred())
		})

		It("reconciles openshift peer", func() {
			_, err := peer.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
