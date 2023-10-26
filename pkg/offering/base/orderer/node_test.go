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

package baseorderer_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	cmocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/apis/deployer"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v1"
	v2 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v2"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/mspparser"
	ordererinit "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer"
	oconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v1"
	v2config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v2"
	managermocks "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/mocks"
	baseorderer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/mocks"
	orderermocks "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/orderer/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Base Orderer Node", func() {
	var (
		node           *baseorderer.Node
		instance       *current.IBPOrderer
		mockKubeClient *cmocks.Client

		deploymentMgr *orderermocks.DeploymentManager
		serviceMgr    *managermocks.ResourceManager
		pvcMgr        *managermocks.ResourceManager
		configMapMgr  *managermocks.ResourceManager

		certificateMgr *orderermocks.CertificateManager
		initializer    *orderermocks.InitializeIBPOrderer
		update         *mocks.Update
		cfg            *config.Config
	)

	BeforeEach(func() {
		mockKubeClient = &cmocks.Client{}
		update = &mocks.Update{}

		replicas := int32(1)
		instance = &current.IBPOrderer{
			Spec: current.IBPOrdererSpec{
				ExternalAddress: "address",
				Domain:          "domain",
				HSM: &current.HSM{
					PKCS11Endpoint: "tcp://0.0.0.0:2346",
				},
				Images: &current.OrdererImages{
					OrdererTag: "1.4.9-20200611",
				},
				Replicas:      &replicas,
				FabricVersion: "1.4.9",
			},
		}
		instance.Kind = "IBPOrderer"
		instance.Name = "orderer1"
		instance.Namespace = "random"
		nodeNumber := 1
		instance.Spec.NodeNumber = &nodeNumber
		instance.Status.Version = version.Operator

		mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *current.IBPOrderer:
				o := obj.(*current.IBPOrderer)
				o.Kind = "IBPOrderer"
				instance = o
			case *corev1.Service:
				o := obj.(*corev1.Service)
				o.Spec.Type = corev1.ServiceTypeNodePort
				o.Spec.Ports = append(o.Spec.Ports, corev1.ServicePort{
					Name: "orderer-api",
					TargetPort: intstr.IntOrString{
						IntVal: 7051,
					},
					NodePort: int32(7051),
				})
			case *corev1.Secret:
				o := obj.(*corev1.Secret)
				switch types.Name {
				case "ecert-" + instance.Name + "-signcert":
					o.Name = "ecert-" + instance.Name + "-signcert"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"cert.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
				case "ecert-" + instance.Name + "-keystore":
					o.Name = "ecert-" + instance.Name + "-keystore"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"key.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
				case "tls-" + instance.Name + "-signcert":
					o.Name = "ecert-" + instance.Name + "-signcert"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"cert.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
				case "tls-" + instance.Name + "-keystore":
					o.Name = "ecert-" + instance.Name + "-keystore"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"key.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
				case "tls-" + instance.Name + "-cacerts":
					o.Name = "ecert-" + instance.Name + "-cacerts"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"key.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
				case "ecert-" + instance.Name + "-cacerts":
					o.Name = "ecert-" + instance.Name + "-cacerts"
					o.Namespace = instance.Namespace
					o.Data = map[string][]byte{"cacert-0.pem": []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwVENDQWtxZ0F3SUJBZ0lSQU1FeVZVcDRMdlYydEFUREhlWklldDh3Q2dZSUtvWkl6ajBFQXdJd2daVXgKQ3pBSkJnTlZCQVlUQWxWVE1SY3dGUVlEVlFRSUV3NU9iM0owYUNCRFlYSnZiR2x1WVRFUE1BMEdBMVVFQnhNRwpSSFZ5YUdGdE1Rd3dDZ1lEVlFRS0V3TkpRazB4RXpBUkJnTlZCQXNUQ2tKc2IyTnJZMmhoYVc0eE9UQTNCZ05WCkJBTVRNR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzFqWVM1aGNIQnpMbkIxYldGekxtOXpMbVo1Y21VdWFXSnQKTG1OdmJUQWVGdzB5TURBeE1qSXhPREExTURCYUZ3MHpNREF4TVRreE9EQTFNREJhTUlHVk1Rc3dDUVlEVlFRRwpFd0pWVXpFWE1CVUdBMVVFQ0JNT1RtOXlkR2dnUTJGeWIyeHBibUV4RHpBTkJnTlZCQWNUQmtSMWNtaGhiVEVNCk1Bb0dBMVVFQ2hNRFNVSk5NUk13RVFZRFZRUUxFd3BDYkc5amEyTm9ZV2x1TVRrd053WURWUVFERXpCcVlXNHkKTWkxdmNtUmxjbVZ5YjNKblkyRXRZMkV1WVhCd2N5NXdkVzFoY3k1dmN5NW1lWEpsTG1saWJTNWpiMjB3V1RBVApCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFTR0lHUFkvZC9tQVhMejM4SlROR3F5bldpOTJXUVB6cnN0Cm5vdEFWZlh0dHZ5QWJXdTRNbWNUMEh6UnBTWjNDcGdxYUNXcTg1MUwyV09LcnZ6L0JPREpvM2t3ZHpCMUJnTlYKSFJFRWJqQnNnakJxWVc0eU1pMXZjbVJsY21WeWIzSm5ZMkV0WTJFdVlYQndjeTV3ZFcxaGN5NXZjeTVtZVhKbApMbWxpYlM1amIyMkNPR3BoYmpJeUxXOXlaR1Z5WlhKdmNtZGpZUzF2Y0dWeVlYUnBiMjV6TG1Gd2NITXVjSFZ0CllYTXViM011Wm5seVpTNXBZbTB1WTI5dE1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRQzM3Y1pkNFY2RThPQ1IKaDloQXEyK0dyR21FVTFQU0I1eHo5RkdEWThkODZRSWhBT1crM3Urb2d4bFNWNUoyR3ZYbHRaQmpXRkpvYnJxeApwVVQ4cW4yMDA1b0wKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo")}
				}
			}
			return nil
		}

		deploymentMgr = &orderermocks.DeploymentManager{}
		serviceMgr = &managermocks.ResourceManager{}
		pvcMgr = &managermocks.ResourceManager{}
		configMapMgr = &managermocks.ResourceManager{}
		roleMgr := &managermocks.ResourceManager{}
		roleBindingMgr := &managermocks.ResourceManager{}
		serviceAccountMgr := &managermocks.ResourceManager{}

		initializer = &orderermocks.InitializeIBPOrderer{}
		initializer.GetInitOrdererReturns(&ordererinit.Orderer{}, nil)

		certificateMgr = &orderermocks.CertificateManager{}
		restartMgr := &orderermocks.RestartManager{}

		cfg = &config.Config{
			OrdererInitConfig: &ordererinit.Config{
				ConfigTxFile: "../../../../defaultconfig/orderer/configtx.yaml",
				OUFile:       "../../../../defaultconfig/orderer/ouconfig.yaml",
				OrdererFile:  "../../../../defaultconfig/orderer/orderer.yaml",
			},
			Operator: config.Operator{
				Versions: &deployer.Versions{
					Orderer: map[string]deployer.VersionOrderer{
						"1.4.9-0": {
							Default: true,
							Image: deployer.OrdererImages{
								OrdererImage:     "ordererimage",
								OrdererTag:       "1.4.9-amd64",
								OrdererInitImage: "ordererinitimage",
								OrdererInitTag:   "1.4.9-amd64",
							},
						},
					},
				},
			},
		}

		node = &baseorderer.Node{
			Client: mockKubeClient,
			Scheme: &runtime.Scheme{},
			Config: cfg,

			DeploymentManager:     deploymentMgr,
			ServiceManager:        serviceMgr,
			EnvConfigMapManager:   configMapMgr,
			PVCManager:            pvcMgr,
			RoleManager:           roleMgr,
			RoleBindingManager:    roleBindingMgr,
			ServiceAccountManager: serviceAccountMgr,

			CertificateManager: certificateMgr,
			RenewCertTimers:    make(map[string]*time.Timer),
			Initializer:        initializer,
			Restart:            restartMgr,
		}
	})

	Context("pre reconcile checks", func() {
		Context("version and images", func() {
			Context("create CR", func() {
				It("returns an error if fabric version is not set in spec", func() {
					instance.Spec.FabricVersion = ""
					_, err := node.PreReconcileChecks(instance, update)
					Expect(err).To(MatchError(ContainSubstring("fabric version is not set")))
				})

				Context("images section blank", func() {
					BeforeEach(func() {
						instance.Spec.Images = nil
					})

					It("normalizes fabric version and requests a requeue", func() {
						instance.Spec.FabricVersion = "1.4.9"
						requeue, err := node.PreReconcileChecks(instance, update)
						Expect(err).NotTo(HaveOccurred())
						Expect(requeue).To(Equal(true))
						Expect(instance.Spec.FabricVersion).To(Equal("1.4.9-0"))
					})

					It("returns an error if fabric version not supported", func() {
						instance.Spec.FabricVersion = "0.0.1"
						_, err := node.PreReconcileChecks(instance, update)
						Expect(err).To(MatchError(ContainSubstring("fabric version '0.0.1' is not supported")))
					})

					When("version is passed without hyphen", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = "1.4.9"
						})

						It("finds default version for release and updates images section", func() {
							requeue, err := node.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(true))
							Expect(*instance.Spec.Images).To(Equal(current.OrdererImages{
								OrdererImage:     "ordererimage",
								OrdererTag:       "1.4.9-amd64",
								OrdererInitImage: "ordererinitimage",
								OrdererInitTag:   "1.4.9-amd64",
							}))
						})
					})

					When("version is passed with hyphen", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = "1.4.9-0"
						})

						It("looks images and updates images section", func() {
							requeue, err := node.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(true))
							Expect(*instance.Spec.Images).To(Equal(current.OrdererImages{
								OrdererImage:     "ordererimage",
								OrdererTag:       "1.4.9-amd64",
								OrdererInitImage: "ordererinitimage",
								OrdererInitTag:   "1.4.9-amd64",
							}))
						})
					})
				})

				Context("images section passed", func() {
					BeforeEach(func() {
						instance.Spec.Images = &current.OrdererImages{
							OrdererImage:     "ghcr.io/ibm-blockchain/ordererimage",
							OrdererTag:       "2.0.0",
							OrdererInitImage: "ghcr.io/ibm-blockchain/ordererinitimage",
							OrdererInitTag:   "2.0.0",
						}
					})

					When("version is not passed", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = ""
						})

						It("returns an error", func() {
							_, err := node.PreReconcileChecks(instance, update)
							Expect(err).To(MatchError(ContainSubstring("fabric version is not set")))
						})
					})

					When("version is passed", func() {
						BeforeEach(func() {
							instance.Spec.FabricVersion = "2.0.0-8"
						})

						It("persists current spec configuration", func() {
							requeue, err := node.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("2.0.0-8"))
							Expect(*instance.Spec.Images).To(Equal(current.OrdererImages{
								OrdererImage:     "ghcr.io/ibm-blockchain/ordererimage",
								OrdererTag:       "2.0.0",
								OrdererInitImage: "ghcr.io/ibm-blockchain/ordererinitimage",
								OrdererInitTag:   "2.0.0",
							}))
						})
					})
				})
			})

			Context("update CR", func() {
				BeforeEach(func() {
					instance.Spec.FabricVersion = "2.0.1-0"
					instance.Spec.Images = &current.OrdererImages{
						OrdererImage:     "ghcr.io/ibm-blockchain/ordererimage",
						OrdererTag:       "2.0.1",
						OrdererInitImage: "ghcr.io/ibm-blockchain/ordererinitimage",
						OrdererInitTag:   "2.0.1",
					}
				})

				When("images updated", func() {
					BeforeEach(func() {
						update.ImagesUpdatedReturns(true)
						instance.Spec.Images = &current.OrdererImages{
							OrdererImage:     "ghcr.io/ibm-blockchain/ordererimage",
							OrdererTag:       "2.0.8",
							OrdererInitImage: "ghcr.io/ibm-blockchain/ordererinitimage",
							OrdererInitTag:   "2.0.8",
						}
					})

					Context("and version updated", func() {
						BeforeEach(func() {
							update.FabricVersionUpdatedReturns(true)
							instance.Spec.FabricVersion = "2.0.1-8"
						})

						It("persists current spec configuration", func() {
							requeue, err := node.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("2.0.1-8"))
							Expect(*instance.Spec.Images).To(Equal(current.OrdererImages{
								OrdererImage:     "ghcr.io/ibm-blockchain/ordererimage",
								OrdererTag:       "2.0.8",
								OrdererInitImage: "ghcr.io/ibm-blockchain/ordererinitimage",
								OrdererInitTag:   "2.0.8",
							}))
						})
					})

					Context("and version not updated", func() {
						It("persists current spec configuration", func() {
							requeue, err := node.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("2.0.1-0"))
							Expect(*instance.Spec.Images).To(Equal(current.OrdererImages{
								OrdererImage:     "ghcr.io/ibm-blockchain/ordererimage",
								OrdererTag:       "2.0.8",
								OrdererInitImage: "ghcr.io/ibm-blockchain/ordererinitimage",
								OrdererInitTag:   "2.0.8",
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
							requeue, err := node.PreReconcileChecks(instance, update)
							Expect(err).NotTo(HaveOccurred())
							Expect(requeue).To(Equal(false))
							Expect(instance.Spec.FabricVersion).To(Equal("unsupported"))
							Expect(*instance.Spec.Images).To(Equal(current.OrdererImages{
								OrdererImage:     "ghcr.io/ibm-blockchain/ordererimage",
								OrdererTag:       "2.0.1",
								OrdererInitImage: "ghcr.io/ibm-blockchain/ordererinitimage",
								OrdererInitTag:   "2.0.1",
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
								requeue, err := node.PreReconcileChecks(instance, update)
								Expect(err).NotTo(HaveOccurred())
								Expect(requeue).To(Equal(true))
								Expect(instance.Spec.FabricVersion).To(Equal("1.4.9-0"))
								Expect(*instance.Spec.Images).To(Equal(current.OrdererImages{
									OrdererImage:     "ordererimage",
									OrdererTag:       "1.4.9-amd64",
									OrdererInitImage: "ordererinitimage",
									OrdererInitTag:   "1.4.9-amd64",
								}))
							})
						})

						When("using hyphenated version", func() {
							BeforeEach(func() {
								instance.Spec.FabricVersion = "1.4.9-0"
							})

							It("looks images and updates images section", func() {
								requeue, err := node.PreReconcileChecks(instance, update)
								Expect(err).NotTo(HaveOccurred())
								Expect(requeue).To(Equal(true))
								Expect(instance.Spec.FabricVersion).To(Equal("1.4.9-0"))
								Expect(*instance.Spec.Images).To(Equal(current.OrdererImages{
									OrdererImage:     "ordererimage",
									OrdererTag:       "1.4.9-amd64",
									OrdererInitImage: "ordererinitimage",
									OrdererInitTag:   "1.4.9-amd64",
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
				updated, err := node.PreReconcileChecks(instance, update)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated).To(Equal(true))
				Expect(instance.Spec.Images.HSMImage).To(Equal("ghcr.io/ibm-blockchain/hsmimage"))
				Expect(instance.Spec.Images.HSMTag).To(Equal("1.0.0"))
			})

			It("doesn't update hsm image and tag if hsm update is disabled", func() {
				hsmConfig.Library.AutoUpdateDisabled = true

				updated, err := node.PreReconcileChecks(instance, update)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated).To(Equal(false))
				Expect(instance.Spec.Images.HSMImage).To(Equal(""))
				Expect(instance.Spec.Images.HSMTag).To(Equal(""))
			})
		})
	})

	Context("Reconciles", func() {
		It("returns nil and will requeue update request if instance version is updated", func() {
			instance.Status.Version = ""
			_, err := node.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockKubeClient.PatchStatusCallCount()).To(Equal(1))
		})
		It("returns a breaking error if initialization fails", func() {
			cfg.OrdererInitConfig.OrdererFile = "../../../../defaultconfig/orderer/badfile.yaml"
			node.Initializer = ordererinit.New(nil, nil, cfg.OrdererInitConfig, "", nil)
			_, err := node.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Code: 21 - failed to initialize orderer node"))
			Expect(operatorerrors.IsBreakingError(err, "msg", nil)).NotTo(HaveOccurred())
		})

		It("returns an error for invalid HSM endpoint", func() {
			instance.Spec.HSM.PKCS11Endpoint = "tcp://:2346"
			_, err := node.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("failed pre reconcile checks: invalid HSM endpoint for orderer instance '%s': missing IP address", instance.Name)))
		})

		It("returns an error domain is not set", func() {
			instance.Spec.Domain = ""
			_, err := node.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("failed pre reconcile checks: domain not set for orderer instance '%s'", instance.Name)))
		})

		It("returns an error if pvc manager fails to reconcile", func() {
			pvcMgr.ReconcileReturns(errors.New("failed to reconcile pvc"))
			_, err := node.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed PVC reconciliation: failed to reconcile pvc"))
		})

		It("returns an error if service manager fails to reconcile", func() {
			serviceMgr.ReconcileReturns(errors.New("failed to reconcile service"))
			_, err := node.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Service reconciliation: failed to reconcile service"))
		})

		It("returns an error if config map manager fails to reconcile", func() {
			configMapMgr.ReconcileReturns(errors.New("failed to reconcile config map"))
			_, err := node.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Env ConfigMap reconciliation: failed to reconcile config map"))
		})

		It("returns an error if deployment manager fails to reconcile", func() {
			deploymentMgr.ReconcileReturns(errors.New("failed to reconcile deployment"))
			_, err := node.Reconcile(instance, update)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to reconcile managers: failed Deployment reconciliation: failed to reconcile deployment"))
		})

		It("reconciles IBPOrderer", func() {
			_, err := node.Reconcile(instance, update)
			Expect(err).NotTo(HaveOccurred())
		})

	})
	Context("check certificates", func() {
		It("returns error if fails to get certificate expiry info", func() {
			certificateMgr.CheckCertificatesForExpireReturns("", "", errors.New("cert expiry error"))
			_, err := node.CheckCertificates(instance)
			Expect(err).To(HaveOccurred())
		})

		It("sets cr status with certificate expiry info", func() {
			certificateMgr.CheckCertificatesForExpireReturns(current.Warning, "cert renewal required", nil)
			status, err := node.CheckCertificates(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Type).To(Equal(current.Warning))
			Expect(status.Message).To(Equal("cert renewal required"))
		})
	})

	Context("set certificate timer", func() {
		BeforeEach(func() {
			mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *current.IBPOrderer:
					o := obj.(*current.IBPOrderer)
					o.Kind = "IBPOrderer"
					o.Name = "orderer1"
					o.Namespace = "random"
					o.Spec.Secret = &current.SecretSpec{
						Enrollment: &current.EnrollmentSpec{
							TLS: &current.Enrollment{
								EnrollID: "enrollID",
							},
						},
					}
					o.Status.Type = current.Deployed
				case *corev1.Secret:
					o := obj.(*corev1.Secret)
					if strings.Contains(o.Name, "crypto-backup") {
						return k8serrors.NewNotFound(schema.GroupResource{}, "not found")
					}
				}
				return nil
			}

			instance.Spec.Secret = &current.SecretSpec{
				Enrollment: &current.EnrollmentSpec{
					Component: &current.Enrollment{
						EnrollID: "enrollID",
					},
				},
			}
		})

		Context("sets timer to renew tls certificate", func() {
			BeforeEach(func() {
				certificateMgr.GetDurationToNextRenewalReturns(time.Duration(3*time.Second), nil)
			})

			It("does not renew certificate if disabled in config", func() {
				instance.Spec.FabricVersion = "1.4.9"
				node.Config.Operator.Orderer.Renewals.DisableTLScert = true
				err := node.SetCertificateTimer(instance, "tls")
				Expect(err).NotTo(HaveOccurred())
				Expect(node.RenewCertTimers["tls-orderer1-signcert"]).NotTo(BeNil())

				By("not renewing certificate", func() {
					Eventually(func() bool {
						return mockKubeClient.UpdateStatusCallCount() == 1 &&
							certificateMgr.RenewCertCallCount() == 0
					}, time.Duration(5*time.Second)).Should(Equal(true))

					// timer.Stop() == false means that it already fired
					Expect(node.RenewCertTimers["tls-orderer1-signcert"].Stop()).To(Equal(false))
				})
			})

			It("does not renew certificate if fabric version is less than 1.4.9 or 2.2.1", func() {
				instance.Spec.FabricVersion = "1.4.7"
				err := node.SetCertificateTimer(instance, "tls")
				Expect(err).NotTo(HaveOccurred())
				Expect(node.RenewCertTimers["tls-orderer1-signcert"]).NotTo(BeNil())

				By("not renewing certificate", func() {
					Eventually(func() bool {
						return mockKubeClient.UpdateStatusCallCount() == 1 &&
							certificateMgr.RenewCertCallCount() == 0
					}, time.Duration(5*time.Second)).Should(Equal(true))

					// timer.Stop() == false means that it already fired
					Expect(node.RenewCertTimers["tls-orderer1-signcert"].Stop()).To(Equal(false))
				})
			})

			It("renews certificate if fabric version is greater than or equal to 1.4.9 or 2.2.1", func() {
				instance.Spec.FabricVersion = "2.2.1"
				mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
					switch obj.(type) {
					case *current.IBPOrderer:
						o := obj.(*current.IBPOrderer)
						o.Kind = "IBPOrderer"
						o.Name = "orderer1"
						o.Namespace = "random"
						o.Spec.Secret = &current.SecretSpec{
							Enrollment: &current.EnrollmentSpec{
								TLS: &current.Enrollment{
									EnrollID: "enrollID",
								},
							},
						}
						o.Status.Type = current.Deployed
						o.Spec.FabricVersion = "2.2.1"
					case *corev1.Secret:
						o := obj.(*corev1.Secret)
						switch types.Name {
						case "ecert-" + instance.Name + "-signcert":
							o.Name = "ecert-" + instance.Name + "-signcert"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"cert.pem": generateCertPemBytes(29)}
						case "ecert-" + instance.Name + "-keystore":
							o.Name = "ecert-" + instance.Name + "-keystore"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"key.pem": []byte("")}
						case instance.Name + "-crypto-backup":
							return k8serrors.NewNotFound(schema.GroupResource{}, "not found")
						}
					}
					return nil
				}
				err := node.SetCertificateTimer(instance, "tls")
				Expect(err).NotTo(HaveOccurred())
				Expect(node.RenewCertTimers["tls-orderer1-signcert"]).NotTo(BeNil())

				By("renewing certificate", func() {
					Eventually(func() bool {
						return mockKubeClient.UpdateStatusCallCount() == 1 &&
							certificateMgr.RenewCertCallCount() == 1
					}, time.Duration(5*time.Second)).Should(Equal(true))

					// timer.Stop() == false means that it already fired
					Expect(node.RenewCertTimers["tls-orderer1-signcert"].Stop()).To(Equal(false))
				})
			})
		})

		Context("sets timer to renew ecert certificate", func() {
			BeforeEach(func() {
				certificateMgr.GetDurationToNextRenewalReturns(time.Duration(3*time.Second), nil)
				mockKubeClient.UpdateStatusReturns(nil)
				certificateMgr.RenewCertReturns(nil)
			})

			It("does not return error, but certificate fails to renew after timer", func() {
				certificateMgr.RenewCertReturns(errors.New("failed to renew cert"))
				err := node.SetCertificateTimer(instance, "ecert")
				Expect(err).NotTo(HaveOccurred())
				Expect(node.RenewCertTimers["ecert-orderer1-signcert"]).NotTo(BeNil())

				By("certificate fails to be renewed", func() {
					Eventually(func() bool {
						return mockKubeClient.UpdateStatusCallCount() == 1 &&
							certificateMgr.RenewCertCallCount() == 1
					}, time.Duration(5*time.Second)).Should(Equal(true))

					// timer.Stop() == false means that it already fired
					Expect(node.RenewCertTimers["ecert-orderer1-signcert"].Stop()).To(Equal(false))
				})
			})

			It("does not return error, and certificate is successfully renewed after timer", func() {
				err := node.SetCertificateTimer(instance, "ecert")
				Expect(err).NotTo(HaveOccurred())
				Expect(node.RenewCertTimers["ecert-orderer1-signcert"]).NotTo(BeNil())

				By("certificate successfully renewed", func() {
					Eventually(func() bool {
						return mockKubeClient.UpdateStatusCallCount() == 1 &&
							certificateMgr.RenewCertCallCount() == 1
					}, time.Duration(5*time.Second)).Should(Equal(true))

					// timer.Stop() == false means that it already fired
					Expect(node.RenewCertTimers["ecert-orderer1-signcert"].Stop()).To(Equal(false))
				})
			})

			It("does not return error, and timer is set to renew certificate at a later time", func() {
				// Set expiration date of certificate to be > 30 days from now
				certificateMgr.GetDurationToNextRenewalReturns(time.Duration(35*24*time.Hour), nil)

				err := node.SetCertificateTimer(instance, "ecert")
				Expect(err).NotTo(HaveOccurred())
				Expect(node.RenewCertTimers["ecert-orderer1-signcert"]).NotTo(BeNil())

				// timer.Stop() == true means that it has not fired but is now stopped
				Expect(node.RenewCertTimers["ecert-orderer1-signcert"].Stop()).To(Equal(true))
			})
		})

		Context("read certificate expiration date to set timer correctly", func() {
			BeforeEach(func() {
				node.CertificateManager = &certificate.CertificateManager{
					Client: mockKubeClient,
					Scheme: &runtime.Scheme{},
				}

				// set to 30 days
				instance.Spec.NumSecondsWarningPeriod = 30 * baseorderer.DaysToSecondsConversion
			})

			It("doesn't return error if timer is set correctly, but error in renewing certificate when timer goes off", func() {
				// Set ecert signcert expiration date to be 29 days from now, cert is renewed if expires within 30 days
				mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
					switch obj.(type) {
					case *current.IBPOrderer:
						o := obj.(*current.IBPOrderer)
						o.Kind = "IBPOrderer"
						instance = o

					case *corev1.Secret:
						o := obj.(*corev1.Secret)
						switch types.Name {
						case "ecert-" + instance.Name + "-signcert":
							o.Name = "ecert-" + instance.Name + "-signcert"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"cert.pem": generateCertPemBytes(29)}
						case "ecert-" + instance.Name + "-keystore":
							o.Name = "ecert-" + instance.Name + "-keystore"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"key.pem": []byte("")}
						case instance.Name + "-crypto-backup":
							return k8serrors.NewNotFound(schema.GroupResource{}, "not found")
						}
					}
					return nil
				}

				err := node.SetCertificateTimer(instance, "ecert")
				Expect(err).NotTo(HaveOccurred())
				Expect(node.RenewCertTimers["ecert-orderer1-signcert"]).NotTo(BeNil())

				// Wait for timer to go off
				time.Sleep(5 * time.Second)

				// timer.Stop() == false means that it already fired
				Expect(node.RenewCertTimers["ecert-orderer1-signcert"].Stop()).To(Equal(false))
			})

			It("doesn't return error if timer is set correctly, timer doesn't go off because certificate isn't ready for renewal", func() {
				// Set ecert signcert expiration date to be 50 days from now, cert is renewed if expires within 30 days
				mockKubeClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
					switch obj.(type) {
					case *current.IBPOrderer:
						o := obj.(*current.IBPOrderer)
						o.Kind = "IBPOrderer"
						instance = o

					case *corev1.Secret:
						o := obj.(*corev1.Secret)
						switch types.Name {
						case "ecert-" + instance.Name + "-signcert":
							o.Name = "ecert-" + instance.Name + "-signcert"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"cert.pem": generateCertPemBytes(50)}
						case "ecert-" + instance.Name + "-keystore":
							o.Name = "ecert-" + instance.Name + "-keystore"
							o.Namespace = instance.Namespace
							o.Data = map[string][]byte{"key.pem": []byte("")}
						case instance.Name + "-crypto-backup":
							return k8serrors.NewNotFound(schema.GroupResource{}, "not found")
						}
					}
					return nil
				}

				err := node.SetCertificateTimer(instance, "ecert")
				Expect(err).NotTo(HaveOccurred())

				// Timer shouldn't go off
				time.Sleep(5 * time.Second)

				Expect(node.RenewCertTimers["ecert-orderer1-signcert"]).NotTo(BeNil())
				// timer.Stop() == true means that it has not fired but is now stopped
				Expect(node.RenewCertTimers["ecert-orderer1-signcert"].Stop()).To(Equal(true))
			})
		})
	})

	Context("renew cert", func() {
		BeforeEach(func() {
			instance.Spec.Secret = &current.SecretSpec{
				Enrollment: &current.EnrollmentSpec{
					Component: &current.Enrollment{},
				},
			}

			certificateMgr.RenewCertReturns(nil)
		})

		It("returns error if secret spec is missing", func() {
			instance.Spec.Secret = nil
			err := node.RenewCert("ecert", instance, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("missing secret spec for instance 'orderer1'"))
		})

		It("returns error if certificate generated by MSP", func() {
			instance.Spec.Secret = &current.SecretSpec{
				MSP: &current.MSPSpec{},
			}
			err := node.RenewCert("ecert", instance, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("cannot auto-renew certificate created by MSP, force renewal required"))
		})

		It("returns error if certificate manager fails to renew certificate", func() {
			certificateMgr.RenewCertReturns(errors.New("failed to renew cert"))
			err := node.RenewCert("ecert", instance, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to renew cert"))
		})

		It("does not return error if certificate manager successfully renews cert", func() {
			err := node.RenewCert("ecert", instance, true)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("update cr status", func() {
		It("returns error if fails to get current instance", func() {
			mockKubeClient.GetReturns(errors.New("get error"))
			err := node.UpdateCRStatus(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to get new instance: get error"))
		})

		It("returns error if fails to update instance status", func() {
			mockKubeClient.UpdateStatusReturns(errors.New("update status error"))
			certificateMgr.CheckCertificatesForExpireReturns(current.Warning, "cert renewal required", nil)
			err := node.UpdateCRStatus(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to update status to Warning phase: update status error"))
		})

		It("sets instance CR status to Warning", func() {
			certificateMgr.CheckCertificatesForExpireReturns(current.Warning, "cert renewal required", nil)
			err := node.UpdateCRStatus(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(instance.Status.Type).To(Equal(current.Warning))
			Expect(instance.Status.Reason).To(Equal("certRenewalRequired"))
			Expect(instance.Status.Message).To(Equal("cert renewal required"))
		})
	})

	Context("fabric orderer migration", func() {
		BeforeEach(func() {
			overrides := &oconfig.Orderer{
				Orderer: v1.Orderer{
					General: v1.General{
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
			bytes, err := json.Marshal(overrides)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.ConfigOverride = &runtime.RawExtension{Raw: bytes}

			coreBytes, err := yaml.Marshal(overrides)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{
				BinaryData: map[string][]byte{
					"orderer.yaml": coreBytes,
				},
			}
			initializer.GetConfigFromConfigMapReturns(cm, nil)
		})

		When("fabric orderer tag is less than 1.4.7", func() {
			BeforeEach(func() {
				instance.Spec.Images.OrdererTag = "1.4.6-20200611"
			})

			It("returns without updating config", func() {
				ordererConfig, err := node.FabricOrdererMigration(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(ordererConfig).To(BeNil())
			})
		})

		When("hsm is not enabled", func() {
			BeforeEach(func() {
				overrides := &oconfig.Orderer{
					Orderer: v1.Orderer{
						General: v1.General{
							BCCSP: &commonapi.BCCSP{
								ProviderName: "sw",
								PKCS11: &commonapi.PKCS11Opts{
									FileKeyStore: &commonapi.FileKeyStoreOpts{
										KeyStorePath: "msp/keystore",
									},
								},
							},
						},
					},
				}
				bytes, err := json.Marshal(overrides)
				Expect(err).NotTo(HaveOccurred())

				instance.Spec.ConfigOverride = &runtime.RawExtension{Raw: bytes}
			})

			It("returns without updating config", func() {
				ordererConfig, err := node.FabricOrdererMigration(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(ordererConfig).To(BeNil())
			})
		})

		It("removes keystore path value", func() {
			ordererConfig, err := node.FabricOrdererMigration(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(ordererConfig.General.BCCSP.PKCS11.FileKeyStore).To(BeNil())
		})
	})

	Context("initialize", func() {
		BeforeEach(func() {
			config := v2config.Orderer{
				Orderer: v2.Orderer{
					General: v2.General{
						BCCSP: &commonapi.BCCSP{
							ProviderName: "PKCS11",
						},
					},
				},
			}

			bytes, err := json.Marshal(config)
			Expect(err).NotTo(HaveOccurred())

			instance.Spec.ConfigOverride = &runtime.RawExtension{Raw: bytes}
		})

		It("sets PKCS11_PROXY_SOCKET environment variable", func() {
			err := node.Initialize(instance, update)
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Getenv("PKCS11_PROXY_SOCKET")).To(Equal("tcp://0.0.0.0:2346"))
		})

	})

	Context("update connection profile", func() {
		It("returns error if fails to get cert", func() {
			mockKubeClient.GetReturns(errors.New("get error"))
			err := node.UpdateConnectionProfile(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get error"))
		})

		It("updates connection profile cm", func() {
			err := node.UpdateConnectionProfile(instance)
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

			initializer.GetUpdatedOrdererReturns(&ordererinit.Orderer{
				Cryptos: &commonconfig.Cryptos{
					TLS: &mspparser.MSPParser{
						Config: msp,
					},
				},
			}, nil)

		})

		It("returns error if fails to get update msp parsers", func() {
			initializer.GetUpdatedOrdererReturns(nil, errors.New("get error"))
			err := node.UpdateMSPCertificates(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get error"))
		})

		It("returns error if fails to generate crypto", func() {
			initializer.GetUpdatedOrdererReturns(&ordererinit.Orderer{
				Cryptos: &commonconfig.Cryptos{
					TLS: &mspparser.MSPParser{
						Config: &current.MSP{
							SignCerts: "invalid",
						},
					},
				},
			}, nil)
			err := node.UpdateMSPCertificates(instance)
			Expect(err).To(HaveOccurred())
		})

		It("returns error if fails to update tls secrets", func() {
			initializer.UpdateSecretsReturnsOnCall(1, errors.New("update error"))
			err := node.UpdateMSPCertificates(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to update tls secrets: update error"))
		})

		It("updates secrets of certificates passed through MSP spec", func() {
			err := node.UpdateMSPCertificates(instance)
			Expect(err).NotTo(HaveOccurred())
			Expect(initializer.UpdateSecretsCallCount()).To(Equal(3))
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
