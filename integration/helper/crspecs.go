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

package helper

import (
	"encoding/json"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	v1orderer "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v1"
	v2orderer "github.com/IBM-Blockchain/fabric-operator/pkg/apis/orderer/v2"
	v2peer "github.com/IBM-Blockchain/fabric-operator/pkg/apis/peer/v2"
	v2ordererconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Org1CACR(namespace, domain string) *current.IBPCA {
	return &current.IBPCA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "org1ca",
			Namespace: namespace,
		},
		Spec: current.IBPCASpec{
			License: current.License{
				Accept: true,
			},
			ImagePullSecrets: []string{"regcred"},
			Images: &current.CAImages{
				CAImage:     integration.CaImage,
				CATag:       integration.CaTag,
				CAInitImage: integration.InitImage,
				CAInitTag:   integration.InitTag,
			},
			Resources: &current.CAResources{
				CA: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("50m"),
						corev1.ResourceMemory:           resource.MustParse("100M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("50m"),
						corev1.ResourceMemory:           resource.MustParse("100M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					},
				},
			},
			Zone:          "select",
			Region:        "select",
			Domain:        domain,
			FabricVersion: integration.FabricVersion + "-1",
		},
	}
}

func Org1PeerCR(namespace, domain, peerUsername, tlsCert, caHost, adminCert string) (*current.IBPPeer, error) {
	resourceReq := &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:              resource.MustParse("100m"),
			corev1.ResourceMemory:           resource.MustParse("200M"),
			corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:              resource.MustParse("100m"),
			corev1.ResourceMemory:           resource.MustParse("200M"),
			corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
		},
	}

	configOverride := v2peer.Core{
		Peer: v2peer.Peer{
			ID: "testPeerID",
		},
	}
	configBytes, err := json.Marshal(configOverride)
	if err != nil {
		return nil, err
	}

	cr := &current.IBPPeer{
		TypeMeta: metav1.TypeMeta{
			Kind: "IBPPeer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "org1peer",
			Namespace: namespace,
		},
		Spec: current.IBPPeerSpec{
			License: current.License{
				Accept: true,
			},
			MSPID:            "Org1MSP",
			Region:           "select",
			Zone:             "select",
			ImagePullSecrets: []string{"regcred"},
			Images: &current.PeerImages{
				CouchDBImage:  integration.CouchdbImage,
				CouchDBTag:    integration.CouchdbTag,
				GRPCWebImage:  integration.GrpcwebImage,
				GRPCWebTag:    integration.GrpcwebTag,
				PeerImage:     integration.PeerImage,
				PeerTag:       integration.PeerTag,
				PeerInitImage: integration.InitImage,
				PeerInitTag:   integration.InitTag,
			},
			Domain: domain,
			Resources: &current.PeerResources{
				DinD: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("500m"),
						corev1.ResourceMemory:           resource.MustParse("1G"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("500m"),
						corev1.ResourceMemory:           resource.MustParse("1G"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					},
				},
				CouchDB: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("200m"),
						corev1.ResourceMemory:           resource.MustParse("400M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:              resource.MustParse("200m"),
						corev1.ResourceMemory:           resource.MustParse("400M"),
						corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
					},
				},
				FluentD:   resourceReq,
				GRPCProxy: resourceReq,
				Peer:      resourceReq,
			},
			Storage: &current.PeerStorages{
				Peer: &current.StorageSpec{
					Size: "150Mi",
				},
				StateDB: &current.StorageSpec{
					Size: "1Gi",
				},
			},
			Ingress: current.Ingress{
				TlsSecretName: "tlssecret",
			},
			Secret: &current.SecretSpec{
				Enrollment: &current.EnrollmentSpec{
					Component: &current.Enrollment{
						CAHost: caHost,
						CAPort: "443",
						CAName: "ca",
						CATLS: &current.CATLS{
							CACert: tlsCert,
						},
						EnrollID:     peerUsername,
						EnrollSecret: "peerpw",
						AdminCerts:   []string{adminCert, adminCert},
					},
					TLS: &current.Enrollment{
						CAHost: caHost,
						CAPort: "443",
						CAName: "tlsca",
						CATLS: &current.CATLS{
							CACert: tlsCert,
						},
						EnrollID:     peerUsername,
						EnrollSecret: "peerpw",
					},
				},
			},
			ConfigOverride: &runtime.RawExtension{Raw: configBytes},
			FabricVersion:  integration.FabricVersion + "-1",
		},
	}

	return cr, nil
}

func OrdererCR(namespace, domain, ordererUsername, tlsCert, caHost string) (*current.IBPOrderer, error) {
	resourceReq := &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:              resource.MustParse("200m"),
			corev1.ResourceMemory:           resource.MustParse("400M"),
			corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:              resource.MustParse("200m"),
			corev1.ResourceMemory:           resource.MustParse("400M"),
			corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
		},
	}

	configOverride := v2ordererconfig.Orderer{
		Orderer: v2orderer.Orderer{
			General: v2orderer.General{
				Keepalive: v1orderer.Keepalive{
					ServerMinInterval: commonapi.MustParseDuration("30h"),
				},
			},
		},
	}

	configBytes, err := json.Marshal(configOverride)
	if err != nil {
		return nil, err
	}

	enrollment := &current.EnrollmentSpec{
		Component: &current.Enrollment{
			CAHost: caHost,
			CAPort: "443",
			CAName: "ca",
			CATLS: &current.CATLS{
				CACert: tlsCert,
			},
			EnrollID:     ordererUsername,
			EnrollSecret: "ordererpw",
		},
		TLS: &current.Enrollment{
			CAHost: caHost,
			CAPort: "443",
			CAName: "tlsca",
			CATLS: &current.CATLS{
				CACert: tlsCert,
			},
			EnrollID:     ordererUsername,
			EnrollSecret: "ordererpw",
		},
	}

	cr := &current.IBPOrderer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ibporderer1",
			Namespace: namespace,
		},
		Spec: current.IBPOrdererSpec{
			License: current.License{
				Accept: true,
			},
			ClusterSize:       3,
			OrdererType:       "etcdraft",
			SystemChannelName: "testchainid",
			OrgName:           "orderermsp",
			MSPID:             "orderermsp",
			ImagePullSecrets:  []string{"regcred"},
			GenesisProfile:    "Initial",
			Domain:            domain,
			Images: &current.OrdererImages{
				GRPCWebImage:     integration.GrpcwebImage,
				GRPCWebTag:       integration.GrpcwebTag,
				OrdererImage:     integration.OrdererImage,
				OrdererTag:       integration.OrdererTag,
				OrdererInitImage: integration.InitImage,
				OrdererInitTag:   integration.InitTag,
			},
			ClusterSecret: []*current.SecretSpec{
				{
					Enrollment: enrollment,
				},
				{
					Enrollment: enrollment,
				},
				{
					Enrollment: enrollment,
				},
			},
			Resources: &current.OrdererResources{
				Orderer: resourceReq,
			},
			FabricVersion:  integration.FabricVersion + "-1",
			ConfigOverride: &runtime.RawExtension{Raw: configBytes},
		},
	}

	return cr, nil
}
