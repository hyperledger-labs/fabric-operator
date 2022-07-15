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

package k8speer_test

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
	k8speer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/k8s/peer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("K8s Peer", func() {
	var (
		peer           *k8speer.Peer
		instance       *current.IBPPeer
		mockKubeClient *cmocks.Client
		cfg            *config.Config

		deploymentMgr     *mocks.DeploymentManager
		serviceMgr        *managermocks.ResourceManager
		pvcMgr            *managermocks.ResourceManager
		couchPvcMgr       *managermocks.ResourceManager
		configMapMgr      *managermocks.ResourceManager
		roleMgr           *managermocks.ResourceManager
		roleBindingMgr    *managermocks.ResourceManager
		serviceAccountMgr *managermocks.ResourceManager
		ingressMgr        *managermocks.ResourceManager
		update            *mocks.Update
		certificateMgr    *mocks.CertificateManager
	)

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
			case *current.IBPPeer:
				o := obj.(*current.IBPPeer)
				o.Kind = "IBPPeer"
				instance = o
			case *corev1.Service:
				o := obj.(*corev1.Service)
				o.Spec.Type = corev1.ServiceTypeNodePort
				o.Spec.Ports = append(o.Spec.Ports, corev1.ServicePort{
					Name: "peer-api",
					TargetPort: intstr.IntOrString{
						IntVal: 7051,
					},
					NodePort: int32(7051),
				})
			case *corev1.Secret:
				o := obj.(*corev1.Secret)
				switch types.Name {
				case "ecert-" + instance.Name + "-cacerts":
					o.Name = "tls-" + instance.Name + "-signcert"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"cacert-0.pem": []byte("")}
				default:
					o.Name = "tls-" + instance.Name + "-signcert"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"cert.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
				}
			}
			return nil
		}
		instance.Status.Version = version.Operator

		deploymentMgr = &mocks.DeploymentManager{}
		serviceMgr = &managermocks.ResourceManager{}
		pvcMgr = &managermocks.ResourceManager{}
		couchPvcMgr = &managermocks.ResourceManager{}
		configMapMgr = &managermocks.ResourceManager{}
		roleMgr = &managermocks.ResourceManager{}
		roleBindingMgr = &managermocks.ResourceManager{}
		serviceAccountMgr = &managermocks.ResourceManager{}
		ingressMgr = &managermocks.ResourceManager{}
		certificateMgr = &mocks.CertificateManager{}
		restartMgr := &mocks.RestartManager{}

		scheme := &runtime.Scheme{}
		cfg = &config.Config{
			PeerInitConfig: &peerinit.Config{
				OUFile:       "../../../../defaultconfig/peer/ouconfig.yaml",
				CorePeerFile: "../../../../defaultconfig/peer/core.yaml",
			},
		}
		initializer := &mocks.InitializeIBPPeer{}
		initializer.GetInitPeerReturns(&peerinit.Peer{}, nil)
		peer = &k8speer.Peer{
			Peer: &basepeer.Peer{
				Client: mockKubeClient,
				Scheme: scheme,
				Config: cfg,

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
			IngressManager: ingressMgr,
		}
	})

	Context("Reconciles", func() {
		It("returns an error if pvc manager fails to reconcile", func() {
			pvcMgr.ReconcileReturns(errors.New("failed to reconcile pvc"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed PVC reconciliation: failed to reconcile pvc"))
		})

		It("returns an error if couch pvc manager fails to reconcile", func() {
			couchPvcMgr.ReconcileReturns(errors.New("failed to reconcile couch pvc"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed CouchDB PVC reconciliation: failed to reconcile couch pvc"))
		})

		It("returns an error if service manager fails to reconcile", func() {
			serviceMgr.ReconcileReturns(errors.New("failed to reconcile service"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Service reconciliation: failed to reconcile service"))
		})

		It("returns an error if deployment manager fails to reconcile", func() {
			deploymentMgr.ReconcileReturns(errors.New("failed to reconcile deployment"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Deployment reconciliation: failed to reconcile deployment"))
		})

		It("returns an error if role manager fails to reconcile", func() {
			roleMgr.ReconcileReturns(errors.New("failed to reconcile role"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to reconcile role"))
		})

		It("returns an error if role binding manager fails to reconcile", func() {
			roleBindingMgr.ReconcileReturns(errors.New("failed to reconcile role binding"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to reconcile role binding"))
		})

		It("returns an error if service account binding manager fails to reconcile", func() {
			serviceAccountMgr.ReconcileReturns(errors.New("failed to reconcile service account"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to reconcile service account"))
		})

		It("returns an error if config map manager fails to reconcile", func() {
			configMapMgr.ReconcileReturns(errors.New("failed to reconcile config map"))
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed FluentD ConfigMap reconciliation: failed to reconcile config map"))
		})

		It("returns a breaking error if initialization fails", func() {
			cfg.PeerInitConfig.CorePeerFile = "../../../../../defaultconfig/peer/badfile.yaml"
			peer.Initializer = peerinit.New(cfg.PeerInitConfig, nil, nil, nil, nil, enroller.HSMEnrollJobTimeouts{})
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Code: 22 - failed to initialize peer: open"))
			Expect(operatorerrors.IsBreakingError(err, "msg", nil)).NotTo(HaveOccurred())
		})

		It("does not return an error on a successful reconcile", func() {
			_, err := peer.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("ExternalEndpoint", func() {
		It("Updates the external endpoint, When external endpoint is not defined", func() {
			instance.Namespace = "namespace"
			instance.Name = "name"
			instance.Spec.PeerExternalEndpoint = ""
			instance.Spec.Domain = "1.2.3.4"

			updated := peer.UpdateExternalEndpoint(instance)
			Expect(updated).To(Equal(true))
			Expect(instance.Spec.PeerExternalEndpoint).To(Equal("namespace-name-peer.1.2.3.4:443"))
		})
	})
})
