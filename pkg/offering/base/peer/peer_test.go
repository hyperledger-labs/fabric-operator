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

package basepeer_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	cmocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/mspparser"
	peerinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	pconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
	managermocks "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/mocks"
	basepeer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/mocks"
	peermocks "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Base Peer", func() {
	var (
		peer           *basepeer.Peer
		instance       *current.IBPPeer
		mockKubeClient *cmocks.Client
		cfg            *config.Config

		deploymentMgr     *peermocks.DeploymentManager
		serviceMgr        *managermocks.ResourceManager
		pvcMgr            *managermocks.ResourceManager
		couchPvcMgr       *managermocks.ResourceManager
		configMapMgr      *managermocks.ResourceManager
		roleMgr           *managermocks.ResourceManager
		roleBindingMgr    *managermocks.ResourceManager
		serviceAccountMgr *managermocks.ResourceManager

		certificateMgr *peermocks.CertificateManager
		initializer    *peermocks.InitializeIBPPeer
		update         *mocks.Update
	)

	BeforeEach(func() {
		mockKubeClient = &cmocks.Client{}
		update = &mocks.Update{}

		replicas := int32(1)
		instance = &current.IBPPeer{
			Spec: current.IBPPeerSpec{
				PeerExternalEndpoint: "address",
				Domain:               "domain",
				HSM: &current.HSM{
					PKCS11Endpoint: "tcp://0.0.0.0:2347",
				},
				StateDb: "couchdb",
				Images: &current.PeerImages{
					PeerTag: "1.4.7-20200611",
				},
				Replicas:      &replicas,
				FabricVersion: "1.4.9",
			},
		}
		instance.Kind = "IBPPeer"
		instance.Name = "peer1"
		instance.Namespace = "random"

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
				case "tls-" + instance.Name + "-signcert":
					o.Name = "tls-" + instance.Name + "-signcert"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"cert.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
				case "tls-" + instance.Name + "-keystore":
					o.Name = "tls-" + instance.Name + "-keystore"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"key.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
				case "tls-" + instance.Name + "-cacerts":
					o.Name = "tls-" + instance.Name + "-cacerts"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"key.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
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
		instance.Status.Version = version.Operator

		deploymentMgr = &peermocks.DeploymentManager{}
		serviceMgr = &managermocks.ResourceManager{}
		pvcMgr = &managermocks.ResourceManager{}
		couchPvcMgr = &managermocks.ResourceManager{}
		configMapMgr = &managermocks.ResourceManager{}
		roleMgr = &managermocks.ResourceManager{}
		roleBindingMgr = &managermocks.ResourceManager{}
		serviceAccountMgr = &managermocks.ResourceManager{}

		scheme := &runtime.Scheme{}
		cfg = &config.Config{
			PeerInitConfig: &peerinit.Config{
				OUFile:       "../../../../defaultconfig/peer/ouconfig.yaml",
				CorePeerFile: "../../../../defaultconfig/peer/core.yaml",
			},
			Operator: config.Operator{
				Versions: &deployer.Versions{
					Peer: map[string]deployer.VersionPeer{
						"1.4.9-0": {
							Default: true,
							Image: deployer.PeerImages{
								PeerImage:     "peerimage",
								PeerTag:       "1.4.9",
								PeerInitImage: "peerinitimage",
								PeerInitTag:   "1.4.9",
							},
						},
					},
				},
			},
		}
		initializer = &peermocks.InitializeIBPPeer{}
		initializer.GetInitPeerReturns(&peerinit.Peer{}, nil)

		certificateMgr = &peermocks.CertificateManager{}
		restartMgr := &peermocks.RestartManager{}
		peer = &basepeer.Peer{
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

			CertificateManager: certificateMgr,
			RenewCertTimers:    make(map[string]*time.Timer),

			Restart: restartMgr,
		}
	})

	Context("pre reconcile checks", func() {
		Context("version and images", func() {
			Context("create CR", func() {
				It("returns an error if fabric version is not set in spec", func() {
					instance.Spec.FabricVersion = ""
					_, err := peer.PreReconcileChecks(instance, update)
					Expect(err).To(MatchError(ContainSubstring("fabric version is not set")))
				})

				Context("images section blank", func() {
					BeforeEach(func() {
						instance.Spec.Images = nil
					})

					It("normalizes fabric version and requests a requeue", func() {
						instance.Spec.FabricVersion = "1.4.9"
						requeue, err := peer.PreReconcileChecks(instance, update)
						Expect(err).NotTo(HaveOccurred())
						Expect(requeue).To(Equal(true))
						Expect(instance.Spec.FabricVersion).To(Equal("1.4.9-0"))
					})

					It("returns an error if fabric version not supported", func() {
						instance.Spec.FabricVersion = "0.0.1"
						_, err := peer.PreReconcileChecks(instance, update)
						Expect(err).To(MatchError(ContainSubstring("fabric version '0.0.1' is not supported")))
					})

					When("version is passed without hyphen", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = "1.4.9"
						})

						It("finds default version for release and updates images section", func() {
							requeue, err := peer.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(true))
							Expect(*instance.Spec.Images).To(Equal(current.PeerImages{
								PeerImage:     "peerimage",
								PeerTag:       "1.4.9",
								PeerInitImage: "peerinitimage",
								PeerInitTag:   "1.4.9",
							}))
						})
					})

					When("version is passed with hyphen", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = "1.4.9-0"
						})

						It("looks images and updates images section", func() {
							requeue, err := peer.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(true))
							Expect(*instance.Spec.Images).To(Equal(current.PeerImages{
								PeerImage:     "peerimage",
								PeerTag:       "1.4.9",
								PeerInitImage: "peerinitimage",
								PeerInitTag:   "1.4.9",
							}))
						})
					})
				})

				Context("images section passed", func() {
					BeforeEach(func() {
						instance.Spec.Images = &current.PeerImages{
							PeerImage:     "ghcr.io/ibm-blockchain/peerimage",
							PeerTag:       "2.0.0",
							PeerInitImage: "ghcr.io/ibm-blockchain/peerinitimage",
							PeerInitTag:   "2.0.0",
						}
					})

					When("version is not passed", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = ""
						})

						It("returns an error", func() {
							_, err := peer.PreReconcileChecks(instance, update)
							Expect(err).To(MatchError(ContainSubstring("fabric version is not set")))
						})
					})

					When("version is passed", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = "2.0.0-8"
						})

						It("persists current spec configuration", func() {
							requeue, err := peer.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("2.0.0-8"))
							Expect(*instance.Spec.Images).To(Equal(current.PeerImages{
								PeerImage:     "ghcr.io/ibm-blockchain/peerimage",
								PeerTag:       "2.0.0",
								PeerInitImage: "ghcr.io/ibm-blockchain/peerinitimage",
								PeerInitTag:   "2.0.0",
							}))
						})
					})
				})
			})

			Context("update CR", func() {
				BeforeEach(func() {
					instance.Spec.FabricVersion = "2.0.1-0"
					instance.Spec.Images = &current.PeerImages{
						PeerImage:     "ghcr.io/ibm-blockchain/peerimage",
						PeerTag:       "2.0.1",
						PeerInitImage: "ghcr.io/ibm-blockchain/peerinitimage",
						PeerInitTag:   "2.0.1",
					}
				})

				When("images updated", func() {
					BeforeEach(func() {
						update.ImagesUpdatedReturns(true)
						instance.Spec.Images = &current.PeerImages{
							PeerImage:     "ghcr.io/ibm-blockchain/peerimage",
							PeerTag:       "2.0.8",
							PeerInitImage: "ghcr.io/ibm-blockchain/peerinitimage",
							PeerInitTag:   "2.0.8",
						}
					})

					Context("and version updated", func() {
						BeforeEach(func() {
							update.FabricVersionUpdatedReturns(true)
							instance.Spec.FabricVersion = "2.0.1-8"
						})

						It("persists current spec configuration", func() {
							requeue, err := peer.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("2.0.1-8"))
							Expect(*instance.Spec.Images).To(Equal(current.PeerImages{
								PeerImage:     "ghcr.io/ibm-blockchain/peerimage",
								PeerTag:       "2.0.8",
								PeerInitImage: "ghcr.io/ibm-blockchain/peerinitimage",
								PeerInitTag:   "2.0.8",
							}))
						})
					})

					Context("and version not updated", func() {
						It("persists current spec configuration", func() {
							requeue, err := peer.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("2.0.1-0"))
							Expect(*instance.Spec.Images).To(Equal(current.PeerImages{
								PeerImage:     "ghcr.io/ibm-blockchain/peerimage",
								PeerTag:       "2.0.8",
								PeerInitImage: "ghcr.io/ibm-blockchain/peerinitimage",
								PeerInitTag:   "2.0.8",
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
							requeue, err := peer.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("unsupported"))
							Expect(*instance.Spec.Images).To(Equal(current.PeerImages{
								PeerImage:     "ghcr.io/ibm-blockchain/peerimage",
								PeerTag:       "2.0.1",
								PeerInitImage: "ghcr.io/ibm-blockchain/peerinitimage",
								PeerInitTag:   "2.0.1",
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
								requeue, err := peer.PreReconcileChecks(instance, update)
								Expect(err).NotTo(HaveOccurred())
								Expect(requeue).To(Equal(true))
								Expect(instance.Spec.FabricVersion).To(Equal("1.4.9-0"))
								Expect(*instance.Spec.Images).To(Equal(current.PeerImages{
									PeerImage:     "peerimage",
									PeerTag:       "1.4.9",
									PeerInitImage: "peerinitimage",
									PeerInitTag:   "1.4.9",
								}))
							})
						})

						When("using hyphenated version", func() {
							BeforeEach(func() {
								instance.Spec.FabricVersion = "1.4.9-0"
							})

							It("looks images and updates images section", func() {
								instance.Spec.RegistryURL = "test.cr"
								requeue, err := peer.PreReconcileChecks(instance, update)
								Expect(err).NotTo(HaveOccurred())
								Expect(requeue).To(Equal(true))
								Expect(instance.Spec.FabricVersion).To(Equal("1.4.9-0"))
								Expect(*instance.Spec.Images).To(Equal(current.PeerImages{
									PeerImage:     "test.cr/peerimage",
									PeerTag:       "1.4.9",
									PeerInitImage: "test.cr/peerinitimage",
									PeerInitTag:   "1.4.9",
								}))
							})
						})
					})
				})
			})
		})
	})

	Context("Reconciles", func() {
		It("returns nil and requeues request if instance version updated", func() {
			instance.Status.Version = ""
			_, err := peer.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockKubeClient.PatchStatusCallCount()).To(Equal(1))
		})
		It("returns a breaking error if initialization fails", func() {
			cfg.PeerInitConfig.CorePeerFile = "../../../../../defaultconfig/peer/badfile.yaml"
			peer.Initializer = peerinit.New(cfg.PeerInitConfig, nil, nil, nil, nil, enroller.HSMEnrollJobTimeouts{})
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Code: 22 - failed to initialize peer: open"))
			Expect(operatorerrors.IsBreakingError(err, "msg", nil)).NotTo(HaveOccurred())
		})

		It("returns an error for invalid HSM endpoint", func() {
			instance.Spec.HSM.PKCS11Endpoint = "tcp://:2347"
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("failed pre reconcile checks: invalid HSM endpoint for peer instance '%s': missing IP address", instance.Name)))
		})

		It("returns an error domain is not set", func() {
			instance.Spec.Domain = ""
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("failed pre reconcile checks: domain not set for peer instance '%s'", instance.Name)))
		})

		It("returns an error if both enroll and reenroll action for ecert set to true", func() {
			instance.Spec.Action.Enroll.Ecert = true
			instance.Spec.Action.Reenroll.Ecert = true
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed pre reconcile checks: both enroll and renenroll action requested for ecert, must only select one"))
		})

		It("returns an error if both enroll and reenroll action for TLS cert set to true", func() {
			instance.Spec.Action.Enroll.TLSCert = true
			instance.Spec.Action.Reenroll.TLSCert = true
			_, err := peer.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed pre reconcile checks: both enroll and renenroll action requested for TLS cert, must only select one"))
		})

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

		It("does not return an error on a successful reconcile", func() {
			_, err := peer.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("secret", func() {
		It("does not try to create secret if the get request returns an error other than 'not found'", func() {
			errMsg := "connection refused"
			mockKubeClient.GetReturns(errors.New(errMsg))
			err := peer.ReconcileSecret(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errMsg))
		})

		When("secret does not exist", func() {
			BeforeEach(func() {
				notFoundErr := &k8serror.StatusError{
					ErrStatus: metav1.Status{
						Reason: metav1.StatusReasonNotFound,
					},
				}
				mockKubeClient.GetReturns(notFoundErr)
			})

			It("returns an error if the creation of the Secret fails", func() {
				errMsg := "unable to create secret"
				mockKubeClient.CreateReturns(errors.New(errMsg))
				err := peer.ReconcileSecret(instance)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(errMsg))
			})

			It("does not return an error on a successfull secret creation", func() {
				err := peer.ReconcileSecret(instance)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("check csr hosts", func() {
		It("adds csr hosts if not present", func() {
			instance = &current.IBPPeer{
				Spec: current.IBPPeerSpec{
					Secret: &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{},
					},
				},
			}
			hosts := []string{"test.com", "127.0.0.1"}
			peer.CheckCSRHosts(instance, hosts)
			Expect(instance.Spec.Secret.Enrollment.TLS).NotTo(BeNil())
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR).NotTo(BeNil())
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts).To(Equal(hosts))
		})

		It("appends csr hosts if passed", func() {
			hostsCustom := []string{"custom.domain.com"}
			hosts := []string{"test.com", "127.0.0.1"}
			instance = &current.IBPPeer{
				Spec: current.IBPPeerSpec{
					Secret: &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{
							TLS: &current.Enrollment{
								CSR: &current.CSR{
									Hosts: hostsCustom,
								},
							},
						},
					},
				},
			}
			peer.CheckCSRHosts(instance, hosts)
			Expect(instance.Spec.Secret.Enrollment.TLS).NotTo(BeNil())
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR).NotTo(BeNil())
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts).To(ContainElement(hostsCustom[0]))
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts).To(ContainElement(hosts[0]))
			Expect(instance.Spec.Secret.Enrollment.TLS.CSR.Hosts).To(ContainElement(hosts[1]))
		})
	})
	Context("check certificates", func() {
		It("returns error if fails to get certificate expiry info", func() {
			certificateMgr.CheckCertificatesForExpireReturns("", "", errors.New("cert expiry error"))
			_, err := peer.CheckCertificates(instance)
			Expect(err).To(HaveOccurred())
		})

		It("sets cr status with certificate expiry info", func() {
			certificateMgr.CheckCertificatesForExpireReturns(current.Warning, "message", nil)
			status, err := peer.CheckCertificates(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Type).To(Equal(current.Warning))
			Expect(status.Message).To(Equal("message"))
			Expect(status.Reason).To(Equal("certRenewalRequired"))
		})
	})

	Context("set certificate timer", func() {
		BeforeEach(func() {
			instance.Spec.Secret = &current.SecretSpec{
				Enrollment: &current.EnrollmentSpec{
					TLS: &current.Enrollment{
						EnrollID: "enrollID",
					},
				},
			}
			mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *current.IBPPeer:
					o := obj.(*current.IBPPeer)
					o.Kind = "IBPPeer"
					o.Name = "peer1"
					o.Namespace = "random"
					o.Spec.Secret = &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{
							TLS: &current.Enrollment{
								EnrollID: "enrollID",
							},
						},
					}
				case *corev1.Secret:
					o := obj.(*corev1.Secret)
					switch types.Name {
					case "tls-" + instance.Name + "-signcert":
						o.Name = "tls-" + instance.Name + "-signcert"
						o.Namespace = instance.Namespace
						o.Data = map[string][]byte{"cert.pem": generateCertPemBytes(29)}
					case "tls-" + instance.Name + "-keystore":
						o.Name = "tls-" + instance.Name + "-keystore"
						o.Namespace = instance.Namespace
						o.Data = map[string][]byte{"key.pem": []byte("")}
					case instance.Name + "-crypto-backup":
						return k8serrors.NewNotFound(schema.GroupResource{}, "not found")
					}
				}
				return nil
			}
		})

		It("returns error if unable to get duration to next renewal", func() {
			certificateMgr.GetDurationToNextRenewalReturns(time.Duration(0), errors.New("failed to get duration"))
			err := peer.SetCertificateTimer(instance, "tls")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to get duration"))
		})

		Context("sets timer to renew TLS certificate", func() {
			BeforeEach(func() {
				certificateMgr.GetDurationToNextRenewalReturns(time.Duration(3*time.Second), nil)
				mockKubeClient.UpdateStatusReturns(nil)
				certificateMgr.RenewCertReturns(nil)
			})

			It("does not return error, but certificate fails to renew after timer", func() {
				certificateMgr.RenewCertReturns(errors.New("failed to renew cert"))
				err := peer.SetCertificateTimer(instance, "tls")
				Expect(err).NotTo(HaveOccurred())
				Expect(peer.RenewCertTimers["tls-peer1-signcert"]).NotTo(BeNil())

				By("certificate fails to be renewed", func() {
					Eventually(func() bool {
						return mockKubeClient.UpdateStatusCallCount() == 1 &&
							certificateMgr.RenewCertCallCount() == 1
					}, time.Duration(5*time.Second)).Should(Equal(true))
				})

				// timer.Stop() == false means that it already fired
				Expect(peer.RenewCertTimers["tls-peer1-signcert"].Stop()).To(Equal(false))
			})

			It("does not return error, and certificate is successfully renewed after timer", func() {
				err := peer.SetCertificateTimer(instance, "tls")
				Expect(err).NotTo(HaveOccurred())
				Expect(peer.RenewCertTimers["tls-peer1-signcert"]).NotTo(BeNil())

				By("certificate successfully renewed", func() {
					Eventually(func() bool {
						return mockKubeClient.UpdateStatusCallCount() == 1 &&
							certificateMgr.RenewCertCallCount() == 1
					}, time.Duration(5*time.Second)).Should(Equal(true))
				})

				// timer.Stop() == false means that it already fired
				Expect(peer.RenewCertTimers["tls-peer1-signcert"].Stop()).To(Equal(false))
			})

			It("does not return error, and timer is set to renew certificate at a later time", func() {
				// Set expiration date of certificate to be > 30 days from now
				certificateMgr.GetDurationToNextRenewalReturns(time.Duration(35*24*time.Hour), nil)

				err := peer.SetCertificateTimer(instance, "tls")
				Expect(err).NotTo(HaveOccurred())
				Expect(peer.RenewCertTimers["tls-peer1-signcert"]).NotTo(BeNil())

				// timer.Stop() == true means that it has not fired but is now stopped
				Expect(peer.RenewCertTimers["tls-peer1-signcert"].Stop()).To(Equal(true))
			})
		})

		Context("read certificate expiration date to set timer correctly", func() {
			BeforeEach(func() {
				peer.CertificateManager = &certificate.CertificateManager{
					Client: mockKubeClient,
					Scheme: &runtime.Scheme{},
				}

				// set to 30 days
				instance.Spec.NumSecondsWarningPeriod = 30 * basepeer.DaysToSecondsConversion
			})

			It("doesn't return error if timer is set correctly, but error in renewing certificate when timer goes off", func() {
				// Set tls signcert expiration date to be 29 days from now, cert is renewed if expires within 30 days
				mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
					switch obj.(type) {
					case *current.IBPPeer:
						o := obj.(*current.IBPPeer)
						o.Kind = "IBPPeer"
						instance = o

					case *corev1.Secret:
						o := obj.(*corev1.Secret)
						switch types.Name {
						case "tls-" + instance.Name + "-signcert":
							o.Name = "tls-" + instance.Name + "-signcert"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"cert.pem": generateCertPemBytes(29)}
						case "tls-" + instance.Name + "-keystore":
							o.Name = "tls-" + instance.Name + "-keystore"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"key.pem": []byte("")}
						case instance.Name + "-crypto-backup":
							return k8serrors.NewNotFound(schema.GroupResource{}, "not found")
						}
					}
					return nil
				}

				err := peer.SetCertificateTimer(instance, "tls")
				Expect(err).NotTo(HaveOccurred())
				Expect(peer.RenewCertTimers["tls-peer1-signcert"]).NotTo(BeNil())

				// Wait for timer to go off
				time.Sleep(5 * time.Second)

				// timer.Stop() == false means that it already fired
				Expect(peer.RenewCertTimers["tls-peer1-signcert"].Stop()).To(Equal(false))
			})

			It("doesn't return error if timer is set correctly, timer doesn't go off certificate isn't ready for renewal", func() {
				// Set tls signcert expiration date to be 50 days from now, cert is renewed if expires within 30 days
				mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
					switch obj.(type) {
					case *current.IBPPeer:
						o := obj.(*current.IBPPeer)
						o.Kind = "IBPPeer"
						instance = o

					case *corev1.Secret:
						o := obj.(*corev1.Secret)
						switch types.Name {
						case "tls-" + instance.Name + "-signcert":
							o.Name = "tls-" + instance.Name + "-signcert"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"cert.pem": generateCertPemBytes(50)}
						case "tls-" + instance.Name + "-keystore":
							o.Name = "tls-" + instance.Name + "-keystore"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"key.pem": []byte("")}
						case instance.Name + "-crypto-backup":
							return k8serrors.NewNotFound(schema.GroupResource{}, "not found")
						}
					}
					return nil
				}

				err := peer.SetCertificateTimer(instance, "tls")
				Expect(err).NotTo(HaveOccurred())

				// Timer shouldn't go off
				time.Sleep(5 * time.Second)

				Expect(peer.RenewCertTimers["tls-peer1-signcert"]).NotTo(BeNil())
				// timer.Stop() == true means that it has not fired but is now stopped
				Expect(peer.RenewCertTimers["tls-peer1-signcert"].Stop()).To(Equal(true))
			})
		})
	})

	Context("renew cert", func() {
		BeforeEach(func() {
			instance.Spec.Secret = &current.SecretSpec{
				Enrollment: &current.EnrollmentSpec{
					TLS: &current.Enrollment{},
				},
			}

			certificateMgr.RenewCertReturns(nil)
		})

		It("returns error if secret spec is missing", func() {
			instance.Spec.Secret = nil
			err := peer.RenewCert("tls", instance, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("missing secret spec for instance 'peer1'"))
		})

		It("returns error if certificate generated by MSP", func() {
			instance.Spec.Secret = &current.SecretSpec{
				MSP: &current.MSPSpec{},
			}
			err := peer.RenewCert("tls", instance, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("cannot auto-renew certificate created by MSP, force renewal required"))
		})

		It("returns error if certificate manager fails to renew certificate", func() {
			certificateMgr.RenewCertReturns(errors.New("failed to renew cert"))
			err := peer.RenewCert("tls", instance, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to renew cert"))
		})

		It("does not return error if certificate manager successfully renews cert", func() {
			err := peer.RenewCert("tls", instance, true)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("set cr status", func() {
		It("returns error if fails to get current instance", func() {
			mockKubeClient.GetReturns(errors.New("get error"))
			err := peer.UpdateCRStatus(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to get new instance: get error"))
		})

		It("returns error if fails to update instance status", func() {
			mockKubeClient.UpdateStatusReturns(errors.New("update status error"))
			certificateMgr.CheckCertificatesForExpireReturns(current.Warning, "cert renewal required", nil)
			err := peer.UpdateCRStatus(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to update status to Warning phase: update status error"))
		})

		It("sets instance CR status to Warning", func() {
			certificateMgr.CheckCertificatesForExpireReturns(current.Warning, "message", nil)
			err := peer.UpdateCRStatus(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(instance.Status.Type).To(Equal(current.Warning))
			Expect(instance.Status.Reason).To(Equal("certRenewalRequired"))
			Expect(instance.Status.Message).To(Equal("message"))
		})
	})

	Context("fabric peer migration", func() {
		BeforeEach(func() {
			overrides := &pconfig.Core{
				Core: v1.Core{
					Peer: v1.Peer{
						BCCSP: &commonapi.BCCSP{
							ProviderName: "pkcs11",
							PKCS11: &commonapi.PKCS11Opts{
								FileKeyStore: &commonapi.FileKeyStoreOpts{
									KeyStorePath: "msp/keystore",
								},
							},
						},
					},
				},
			}
			jmRaw, err := util.ConvertToJsonMessage(overrides)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.ConfigOverride = &runtime.RawExtension{Raw: *jmRaw}

			coreBytes, err := yaml.Marshal(overrides)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{
				BinaryData: map[string][]byte{
					"core.yaml": coreBytes,
				},
			}

			mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *corev1.ConfigMap:
					o := obj.(*corev1.ConfigMap)
					o.Name = "core-config"
					o.BinaryData = cm.BinaryData
				}
				return nil
			}
		})

		It("removes keystore path value", func() {
			peerConfig, err := peer.FabricPeerMigrationV1_4(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(peerConfig.Peer.BCCSP.PKCS11.FileKeyStore).To(BeNil())
		})

		When("fabric peer tag is less than 1.4.7", func() {
			BeforeEach(func() {
				instance.Spec.Images.PeerTag = "1.4.6-20200611"
			})

			It("returns without updating config", func() {
				peerConfig, err := peer.FabricPeerMigrationV1_4(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(peerConfig).To(BeNil())
			})
		})

		When("hsm is not enabled", func() {
			BeforeEach(func() {
				overrides := &pconfig.Core{
					Core: v1.Core{
						Peer: v1.Peer{
							BCCSP: &commonapi.BCCSP{
								ProviderName: "sw",
								SW: &commonapi.SwOpts{
									FileKeyStore: commonapi.FileKeyStoreOpts{
										KeyStorePath: "msp/keystore",
									},
								},
							},
						},
					},
				}
				jmRaw, err := util.ConvertToJsonMessage(overrides)
				Expect(err).NotTo(HaveOccurred())

				instance.Spec.ConfigOverride = &runtime.RawExtension{Raw: *jmRaw}
			})

			It("returns without updating config", func() {
				peerConfig, err := peer.FabricPeerMigrationV1_4(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(peerConfig).To(BeNil())
			})
		})
	})

	Context("images override", func() {
		var images *current.PeerImages

		Context("using registry url", func() {
			BeforeEach(func() {
				images = &current.PeerImages{
					PeerInitImage: "peerinitimage",
					PeerInitTag:   "2.0.0",
					PeerImage:     "peerimage",
					PeerTag:       "2.0.0",
					DindImage:     "dindimage",
					DindTag:       "2.0.0",
					CouchDBImage:  "couchimage",
					CouchDBTag:    "2.0.0",
					GRPCWebImage:  "grpcimage",
					GRPCWebTag:    "2.0.0",
					FluentdImage:  "fluentdimage",
					FluentdTag:    "2.0.0",
				}
			})

			It("overrides images based with registry url and does not append more value on each call", func() {
				images.Override(images, "ghcr.io/ibm-blockchain/", "amd64")
				Expect(images.PeerInitImage).To(Equal("ghcr.io/ibm-blockchain/peerinitimage"))
				Expect(images.PeerInitTag).To(Equal("2.0.0"))
				Expect(images.PeerImage).To(Equal("ghcr.io/ibm-blockchain/peerimage"))
				Expect(images.PeerTag).To(Equal("2.0.0"))
				Expect(images.DindImage).To(Equal("ghcr.io/ibm-blockchain/dindimage"))
				Expect(images.DindTag).To(Equal("2.0.0"))
				Expect(images.CouchDBImage).To(Equal("ghcr.io/ibm-blockchain/couchimage"))
				Expect(images.CouchDBTag).To(Equal("2.0.0"))
				Expect(images.GRPCWebImage).To(Equal("ghcr.io/ibm-blockchain/grpcimage"))
				Expect(images.GRPCWebTag).To(Equal("2.0.0"))
				Expect(images.FluentdImage).To(Equal("ghcr.io/ibm-blockchain/fluentdimage"))
				Expect(images.FluentdTag).To(Equal("2.0.0"))
			})

			It("overrides images based with registry url and does not append more value on each call", func() {
				images.Override(images, "ghcr.io/ibm-blockchain/images/", "s390")
				Expect(images.PeerInitImage).To(Equal("ghcr.io/ibm-blockchain/images/peerinitimage"))
				Expect(images.PeerInitTag).To(Equal("2.0.0"))
				Expect(images.PeerImage).To(Equal("ghcr.io/ibm-blockchain/images/peerimage"))
				Expect(images.PeerTag).To(Equal("2.0.0"))
				Expect(images.DindImage).To(Equal("ghcr.io/ibm-blockchain/images/dindimage"))
				Expect(images.DindTag).To(Equal("2.0.0"))
				Expect(images.CouchDBImage).To(Equal("ghcr.io/ibm-blockchain/images/couchimage"))
				Expect(images.CouchDBTag).To(Equal("2.0.0"))
				Expect(images.GRPCWebImage).To(Equal("ghcr.io/ibm-blockchain/images/grpcimage"))
				Expect(images.GRPCWebTag).To(Equal("2.0.0"))
				Expect(images.FluentdImage).To(Equal("ghcr.io/ibm-blockchain/images/fluentdimage"))
				Expect(images.FluentdTag).To(Equal("2.0.0"))
			})
		})

		Context("using fully qualified path", func() {
			BeforeEach(func() {
				images = &current.PeerImages{
					PeerInitImage: "ghcr.io/ibm-blockchain/peerinitimage",
					PeerInitTag:   "2.0.0",
					PeerImage:     "ghcr.io/ibm-blockchain/peerimage",
					PeerTag:       "2.0.0",
					DindImage:     "ghcr.io/ibm-blockchain/dindimage",
					DindTag:       "2.0.0",
					CouchDBImage:  "ghcr.io/ibm-blockchain/couchimage",
					CouchDBTag:    "2.0.0",
					GRPCWebImage:  "ghcr.io/ibm-blockchain/grpcimage",
					GRPCWebTag:    "2.0.0",
					FluentdImage:  "ghcr.io/ibm-blockchain/fluentdimage",
					FluentdTag:    "2.0.0",
				}
			})

			It("keeps images and adds arch to tag", func() {
				images.Override(images, "", "amd64")
				Expect(images.PeerInitImage).To(Equal("ghcr.io/ibm-blockchain/peerinitimage"))
				Expect(images.PeerInitTag).To(Equal("2.0.0"))
				Expect(images.PeerImage).To(Equal("ghcr.io/ibm-blockchain/peerimage"))
				Expect(images.PeerTag).To(Equal("2.0.0"))
				Expect(images.DindImage).To(Equal("ghcr.io/ibm-blockchain/dindimage"))
				Expect(images.DindTag).To(Equal("2.0.0"))
				Expect(images.CouchDBImage).To(Equal("ghcr.io/ibm-blockchain/couchimage"))
				Expect(images.CouchDBTag).To(Equal("2.0.0"))
				Expect(images.GRPCWebImage).To(Equal("ghcr.io/ibm-blockchain/grpcimage"))
				Expect(images.GRPCWebTag).To(Equal("2.0.0"))
				Expect(images.FluentdImage).To(Equal("ghcr.io/ibm-blockchain/fluentdimage"))
				Expect(images.FluentdTag).To(Equal("2.0.0"))
			})
		})
	})

	Context("update connection profile", func() {
		It("returns error if fails to get cert", func() {
			mockKubeClient.GetReturns(errors.New("get error"))
			err := peer.UpdateConnectionProfile(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get error"))
		})

		It("updates connection profile cm", func() {
			err := peer.UpdateConnectionProfile(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockKubeClient.GetCallCount()).To(Equal(7))
		})
	})

	Context("update msp certificates", func() {
		const testcert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNpVENDQWkrZ0F3SUJBZ0lVRkd3N0RjK0QvZUoyY08wOHd6d2tialIzK1M4d0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRBd09URTBNakF3TUZvWERUSXdNVEF3T0RFME1qQXdNRm93YnpFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1TQXdIZ1lEVlFRREV4ZFRZV0ZrY3kxTllXTkNiMjlyCkxWQnlieTVzYjJOaGJEQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJBK0JBRzhZakJvTllabGgKRjFrVHNUbHd6VERDQTJocDhZTXI5Ky8vbEd0NURoSGZVT1c3bkhuSW1USHlPRjJQVjFPcVRuUWhUbWpLYTdaQwpqeU9BUWxLamdhOHdnYXd3RGdZRFZSMFBBUUgvQkFRREFnT29NQjBHQTFVZEpRUVdNQlFHQ0NzR0FRVUZCd01CCkJnZ3JCZ0VGQlFjREFqQU1CZ05WSFJNQkFmOEVBakFBTUIwR0ExVWREZ1FXQkJTbHJjL0lNQkxvMzR0UktvWnEKNTQreDIyYWEyREFmQmdOVkhTTUVHREFXZ0JSWmpxT3RQZWJzSFI2UjBNQUhrNnd4ei85UFZqQXRCZ05WSFJFRQpKakFrZ2hkVFlXRmtjeTFOWVdOQ2IyOXJMVkJ5Ynk1c2IyTmhiSUlKYkc5allXeG9iM04wTUFvR0NDcUdTTTQ5CkJBTUNBMGdBTUVVQ0lRRGR0Y1QwUE9FQXJZKzgwdEhmWUwvcXBiWWoxMGU2eWlPWlpUQ29wY25mUVFJZ1FNQUQKaFc3T0NSUERNd3lqKzNhb015d2hFenFHYy9jRDJSU2V5ekRiRjFFPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="

		BeforeEach(func() {
			msp := &current.MSP{
				SignCerts: testcert,
				CACerts:   []string{testcert},
				KeyStore:  "keystore",
			}

			initializer.GetUpdatedPeerReturns(&peerinit.Peer{
				Cryptos: &commonconfig.Cryptos{
					TLS: &mspparser.MSPParser{
						Config: msp,
					},
				},
			}, nil)

		})

		It("returns error if fails to get update msp parsers", func() {
			initializer.GetUpdatedPeerReturns(nil, errors.New("get error"))
			err := peer.UpdateMSPCertificates(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get error"))
		})

		It("returns error if fails to generate crypto", func() {
			initializer.GetUpdatedPeerReturns(&peerinit.Peer{
				Cryptos: &commonconfig.Cryptos{
					TLS: &mspparser.MSPParser{
						Config: &current.MSP{
							SignCerts: "invalid",
						},
					},
				},
			}, nil)
			err := peer.UpdateMSPCertificates(instance)
			Expect(err).To(HaveOccurred())
		})

		It("returns error if fails to update secrets", func() {
			initializer.UpdateSecretsFromResponseReturns(errors.New("update error"))
			err := peer.UpdateMSPCertificates(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("update error"))
		})

		It("updates secrets of certificates passed through MSP spec", func() {
			err := peer.UpdateMSPCertificates(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(initializer.UpdateSecretsFromResponseCallCount()).To(Equal(1))
		})
	})

	Context("enroll for ecert", func() {
		It("returns error if no enrollment information provided", func() {
			err := peer.EnrollForEcert(instance)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("unable to enroll, no ecert enrollment information provided")))
		})

		It("returns error if enrollment with ca fails", func() {
			instance.Spec.Secret = &current.SecretSpec{
				Enrollment: &current.EnrollmentSpec{
					Component: &current.Enrollment{},
				},
			}
			err := peer.EnrollForEcert(instance)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to enroll for ecert")))
		})
	})

	Context("enroll for TLS cert", func() {
		It("returns error if no enrollment information provided", func() {
			err := peer.EnrollForTLSCert(instance)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("unable to enroll, no TLS enrollment information provided")))
		})

		It("returns error if enrollment with ca fails", func() {
			instance.Spec.Secret = &current.SecretSpec{
				Enrollment: &current.EnrollmentSpec{
					TLS: &current.Enrollment{},
				},
			}
			err := peer.EnrollForTLSCert(instance)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to enroll for TLS cert")))
		})
	})
})

func generateCertPemBytes(daysUntilExpired int) []byte {
	certtemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Duration(daysUntilExpired) * time.Hour * 24),
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())

	cert, err := x509.CreateCertificate(rand.Reader, &certtemplate, &certtemplate, &priv.PublicKey, priv)
	Expect(err).NotTo(HaveOccurred())

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}

	return pem.EncodeToMemory(block)
}
