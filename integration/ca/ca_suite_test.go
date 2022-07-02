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

package ca_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestCa(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ca Suite")
}

const (
	// This TLS certificate is encoded for the DNS domain aliases 127.0.0.1, localhost, and *.vcap.me and is good for 5 years:
	//
	//   notAfter:    "2027-05-24T03:14:42Z"
	//   notBefore:   "2022-05-25T03:14:42Z"
	//   renewalTime: "2025-09-22T19:14:42Z"
	//
	// This certificate was generated with cert-manager.io using a self-signed issuer for the root CA.
	// If tests start to fail for TLS handshake errors, the certificate will need to be renewed or reissued.
	tlsCert            = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJqakNDQVRTZ0F3SUJBZ0lRVXRIS2NUTWNZS21KblVtbEJNZW94REFLQmdncWhrak9QUVFEQWpBbE1TTXcKSVFZRFZRUURFeHBtWVdKeWFXTXRZMkV0YVc1MFpXZHlZWFJwYjI0dGRHVnpkREFlRncweU1qQTFNalV3TXpFMApORGRhRncweU56QTFNalF3TXpFME5EZGFNQUF3V1RBVEJnY3Foa2pPUFFJQkJnZ3Foa2pPUFFNQkJ3TkNBQVRwCjN2d3RMZFlyUzFTNVFSUmFqRjJReHFIYWllMUo2dzlHM2RwQklLYWwwTTlYaUttR0Q4eFBvRkpkcENNZTZWdDIKeml1UjZrU2FNL3lXQmU4TGd5eExvMnN3YVRBT0JnTlZIUThCQWY4RUJBTUNCYUF3REFZRFZSMFRBUUgvQkFJdwpBREFmQmdOVkhTTUVHREFXZ0JRdkVBWWdjZEwwa0ljWEtDaGVmVzg3NW8vYnd6QW9CZ05WSFJFQkFmOEVIakFjCmdnbHNiMk5oYkdodmMzU0NDU291ZG1OaGNDNXRaWWNFZndBQUFUQUtCZ2dxaGtqT1BRUURBZ05JQURCRkFpQXUKMEpLY29lQmhYajJnbmQ1cjE5THUxeEVwdG1kelFoazh5OXFTRkZ2dkF3SWhBSWp5Z1VLY2tzQkk4a1dBeVNlbQp0VzJ4cVE3RVZkTmR6WDZYbWwrNVBQengKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="
	tlsKey             = "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSUhoWWFRbDViYXZVR3FJd2prK3YrODNmYzNIamZuRVdueEFQbjJ5OFRTUWRvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFNmQ3OExTM1dLMHRVdVVFVVdveGRrTWFoMm9udFNlc1BSdDNhUVNDbXBkRFBWNGlwaGcvTQpUNkJTWGFRakh1bGJkczRya2VwRW1qUDhsZ1h2QzRNc1N3PT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo="
	trustedRootTLSCert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJpekNDQVRDZ0F3SUJBZ0lRZXZWM2VUZmh3WlNHYVI4aXhTR1hRakFLQmdncWhrak9QUVFEQWpBbE1TTXcKSVFZRFZRUURFeHBtWVdKeWFXTXRZMkV0YVc1MFpXZHlZWFJwYjI0dGRHVnpkREFlRncweU1qQTFNalV3TXpFMApOREphRncweU56QTFNalF3TXpFME5ESmFNQ1V4SXpBaEJnTlZCQU1UR21aaFluSnBZeTFqWVMxcGJuUmxaM0poCmRHbHZiaTEwWlhOME1Ga3dFd1lIS29aSXpqMENBUVlJS29aSXpqMERBUWNEUWdBRXlzc2d3dFo2dlI3a2svbUsKYUFUZE45TEhmTWsrYXMxcm8rM24za1N2QTFuVEFCa1V6bVdGNlhCS1I5eUh6V3dwZTlHL0o3L3MrenZsME5GOApRZGdzenFOQ01FQXdEZ1lEVlIwUEFRSC9CQVFEQWdLa01BOEdBMVVkRXdFQi93UUZNQU1CQWY4d0hRWURWUjBPCkJCWUVGQzhRQmlCeDB2U1FoeGNvS0Y1OWJ6dm1qOXZETUFvR0NDcUdTTTQ5QkFNQ0Ewa0FNRVlDSVFEaXo1SnoKeGhKcjQ4SlpRRkpzd1dteTRCU21FWXp0NXFmUmsyMFhyRzI4M3dJaEFLaDBXMmkxcFpiY0lPODBXSmhlVkxzSQpDM0JGMk5McTBsVlhXanNGQVVndQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
)

var (
	namespace       string
	domain          string
	kclient         *kubernetes.Clientset
	ibpCRClient     *ibpclient.IBPClient
	namespaceSuffix = "ca"
	testFailed      bool
)

var (
	defaultRequests = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("50m"),
		corev1.ResourceMemory:           resource.MustParse("100M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("100M"),
	}

	defaultLimits = corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse("50m"),
		corev1.ResourceMemory:           resource.MustParse("100M"),
		corev1.ResourceEphemeralStorage: resource.MustParse("1G"),
	}
)

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(240 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)

	var err error
	domain = os.Getenv("DOMAIN")
	if domain == "" {
		domain = integration.TestAutomation1IngressDomain
	}

	cfg := &integration.Config{
		OperatorServiceAccount: "../../config/rbac/service_account.yaml",
		OperatorRole:           "../../config/rbac/role.yaml",
		OperatorRoleBinding:    "../../config/rbac/role_binding.yaml",
		OperatorDeployment:     "../../testdata/deploy/operator.yaml",
		OrdererSecret:          "../../testdata/deploy/orderer/secret.yaml",
		PeerSecret:             "../../testdata/deploy/peer/secret.yaml",
		ConsoleTLSSecret:       "../../testdata/deploy/console/tlssecret.yaml",
	}

	namespace, kclient, ibpCRClient, err = integration.Setup(GinkgoWriter, cfg, namespaceSuffix, "")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {

	if strings.ToLower(os.Getenv("SAVE_TEST")) == "true" {
		return
	}

	err := integration.Cleanup(GinkgoWriter, kclient, namespace)
	Expect(err).NotTo(HaveOccurred())
})

type CA struct {
	helper.CA

	expectedRequests corev1.ResourceList
	expectedLimits   corev1.ResourceList
}

func (ca *CA) resourcesRequestsUpdated() bool {
	dep, err := kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	updatedRequests := dep.Spec.Template.Spec.Containers[0].Resources.Requests
	if updatedRequests[corev1.ResourceCPU] == ca.expectedRequests[corev1.ResourceCPU] {
		if updatedRequests[corev1.ResourceMemory] == ca.expectedRequests[corev1.ResourceMemory] {
			return true
		}
	}
	return false
}

func (ca *CA) resourcesLimitsUpdated() bool {
	dep, err := kclient.AppsV1().Deployments(namespace).Get(context.TODO(), ca.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	updatedLimits := dep.Spec.Template.Spec.Containers[0].Resources.Limits
	if updatedLimits[corev1.ResourceCPU] == ca.expectedLimits[corev1.ResourceCPU] {
		if updatedLimits[corev1.ResourceMemory] == ca.expectedLimits[corev1.ResourceMemory] {
			return true
		}
	}
	return false
}
