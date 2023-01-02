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
	// This TLS certificate is encoded for the DNS domain aliases 127.0.0.1, localhost, and *.localho.st and is good for 5 years:
	//
	//    Validity
	//        Not Before: Jan  2 12:37:24 2023 GMT
	//        Not After : Jan  1 12:37:24 2028 GMT
	//
	// This certificate was generated with cert-manager.io using a self-signed issuer for the root CA.
	// If tests start to fail for TLS handshake errors, the certificate will need to be renewed or reissued.
	tlsCert            = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURzRENDQXBpZ0F3SUJBZ0lSQVBuR09XY1NyL3ZLdVlIaW43VUFZbG93RFFZSktvWklodmNOQVFFTEJRQXcKYXpFMU1ETUdBMVVFQ2hNc1NXNTBaWEp1WVhScGIyNWhiQ0JDZFhOcGJtVnpjeUJOWVdOb2FXNWxjeUJKYm1OdgpjbkJ2Y21GMFpXUXhNakF3QmdOVkJBTU1LU291Ykc5allXeG9ieTV6ZENCcGJuUmxaM0poZEdsdmJpQjBaWE4wCklHTmxjblJwWm1sallYUmxNQjRYRFRJek1ERXdNakV5TXpjeU5Gb1hEVEk0TURFd01URXlNemN5TkZvd2F6RTEKTURNR0ExVUVDaE1zU1c1MFpYSnVZWFJwYjI1aGJDQkNkWE5wYm1WemN5Qk5ZV05vYVc1bGN5QkpibU52Y25CdgpjbUYwWldReE1qQXdCZ05WQkFNTUtTb3ViRzlqWVd4b2J5NXpkQ0JwYm5SbFozSmhkR2x2YmlCMFpYTjBJR05sCmNuUnBabWxqWVhSbE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBMDRwbTl1WWYKT0g2SFRTWUk4WW5XSGJZb2xWcDdhL0lKVnYvNDR5Wm5YZFJLNXJwZys2TG5TazBBS1p2OHRpa0JrZXZRRTVzWApKYzVtYldhZjFtYmhvbVY0U2RObzRuNkw4aUdTWERjR3FocTBLWUJ2ZjFrOUJ6SGZxKzQ0OEQxaG1nL0ZkTlQwCmFJWUN3akNhZytWT0Jtcm9rY1pjSXFWT3VHL1NTWXd0Q3FiRU1YeExkczUrd0U2NnNYeWx5Si82MGZ5aThOdFoKaDdwcXNvTU9WS0Zla2FwWHFIWXgzTTlVd3ZjUWszYkEyUFZkUXZsZ0RBc3VIZWpqb0xBU0pEN1YrclZ6NVRIMgpsZUhxYzR1cVlScmkvT2o2UzlaVUFHc1AvTnRtMjc3eld2OWlCTkxlR1padkZ6SlQ3Tm5udmpDeGR5YmppUmtVCmp2OFVVTDdxOE82T1h3SURBUUFCbzA4d1RUQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBVEFNQmdOVkhSTUIKQWY4RUFqQUFNQ2dHQTFVZEVRUWhNQitDQ1d4dlkyRnNhRzl6ZElJTUtpNXNiMk5oYkdodkxuTjBod1IvQUFBQgpNQTBHQ1NxR1NJYjNEUUVCQ3dVQUE0SUJBUUM1bDJ6UzY0K3RBYmprVU05MkxFNC82dDdtTHlRNkVqSWxWTHBjCk53WXVlOU9LbHdLU0xGajFlaTdLVW0yRVFWNVpzdmhMejZXTGZndWtsd0NlSWVsMGNxM3RiSEJ1VFFnR0N3QkEKb0VVUjB3Nm82NFp6dmhNWWRRWENzVkJGYjNXWkdTeURnMFZ1b0NZdkJRQ1IxTmFOU2Z5UlF5bXZGSWtBTHJ3bgoxbmNPNnNOVVBaKzRUSW5UYnRTRWY3QUU2eXF3T0F3VXVIcW81amk3bFpuaVhDdW5oM1h4WGhoK29xT01URjZwCmFoNzI1cmdCNWdhbldyL3JmS2RkZUdrc2xTUlVDY2tXTWVEYXJETllRRXFJd1N0enorZC9OS1ZuZUtKaERtMjAKWFRBTUFhNGRQcmhrK1ZMZitwcEZjZVlMa0hYcE1YdjMxV3pRWWN6VHV3QnorUklRCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
	tlsKey             = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcFFJQkFBS0NBUUVBMDRwbTl1WWZPSDZIVFNZSThZbldIYllvbFZwN2EvSUpWdi80NHlablhkUks1cnBnCis2TG5TazBBS1p2OHRpa0JrZXZRRTVzWEpjNW1iV2FmMW1iaG9tVjRTZE5vNG42TDhpR1NYRGNHcWhxMEtZQnYKZjFrOUJ6SGZxKzQ0OEQxaG1nL0ZkTlQwYUlZQ3dqQ2FnK1ZPQm1yb2tjWmNJcVZPdUcvU1NZd3RDcWJFTVh4TApkczUrd0U2NnNYeWx5Si82MGZ5aThOdFpoN3Bxc29NT1ZLRmVrYXBYcUhZeDNNOVV3dmNRazNiQTJQVmRRdmxnCkRBc3VIZWpqb0xBU0pEN1YrclZ6NVRIMmxlSHFjNHVxWVJyaS9PajZTOVpVQUdzUC9OdG0yNzd6V3Y5aUJOTGUKR1padkZ6SlQ3Tm5udmpDeGR5YmppUmtVanY4VVVMN3E4TzZPWHdJREFRQUJBb0lCQVFDdktwL3dPc1lIaGQ2TAo1NzdvSTNjRnkxejNyNkViMWFRZVFuL1p1R2RIcnc4RzE3YVBLR25WZ01WdHJ4a256ZlRhM0NYRTFsdm9sbTBDCmtrUXd5YWgxVFFpNk9URlV1KzB1WnRaSFBkbHE2Z25kZzlqUDN4bEY1K3FLK0F4MkFwM2JjTXZVM3JJMEN5UWwKb1JHUnZrTkoxU1VYOE9WQ1d4aEFhWGY4SnZMMUtYa3ZGMjF1ZEZ3VnlmVmhJTk9NT2NyeUNkdERwb0NRd3FlcQoxUTJGeTlvdjZMVVArRXZzdzgwaUVVTDMrVVBmcEVLV1Z6SG9DYXpGRHpwOUNBRFBwbGJ4amd4LzBhWDFhbkIrCnh3M0RmbElNQUdNaUxuWUs2TmIzR2JNOWhLRjJrUVAyUkJEQ3JxVE9LMEJVQ0ttVW1qQ0NnRjY5a3VUWE94b3QKU0RLSWRaekJBb0dCQU9MdldKeTkwSTViNHNEVHRxQjZoRVJiajgwOTlYUlVCd3plR2ZRRlUrRndUOForMlVJZApQc09BQjlJQXFxaFVrbVd4RkJUYkZvWTZsM09pc28wTUU2anJ4bkI5WFZiNlZYNkFJRzl3ZGIyKzUrTHVpTjY5ClgwMTJ0cTVTUWt3bWNlSGtHRUxvaE5ER3dtK0tUZmhTcWdTMmhmSDdCdkdyMGlROGFMSlg3bkU1QW9HQkFPNmkKVURzQjBhMnlycHVHSnlOTloxMGQvKzhOVkpBV0FRZlYwb1lpYk1zR0lXMFpuOUx5M3lZd0tzcUZkSjVMMm41Zwovek03RnluNm55bU5RTlpBd3JRSDhvTm4vWUZBVlJaOGFZZUpsSnZUTEhBcmt3d0tSZ0FteGxnb0FqY0g3eGw3CmJ3QmY5ajNRRE13dDBqSnJRMU15QjNSbmpRVlBKV0lwV0pTam1yUlhBb0dCQUxPK3lsd1VDSTNKZjlnbG1PQ1IKU2hSdXhYN1dWWVZYVE9KSFJSMC8zd21RRU0vekJ4aFQyN096dy8zMUl6Y0REWlhZWlVTRHA5cVhyQUFlWFBoVgpHWGxSanJMb3lUYXNQMjFjQk5UZnFaS3FGRGR0b2lGeXMzckN6YjFUVUVuS3BhYzdLSEJPaFd4c0VmT1JBMkx0Cjd0YWV6NGN6d25OSEdjSXp5dVYvdWxBWkFvR0JBSzdYQzdPQURMR1lOaWhLN1VnSFFWRlBWcUkrZ1JPa201S3oKRGpFcTdjeitxK1QwbmszL2xwR3pQdGJ0V3RsVU9EemFNb0RGclo0Yk94eEZteGlma0VnNWZtemE5emtJK282awpEdW00V3NLa3dXMVo3NzRsbE00dG1xc2lmU1QyMGk4NGFjYTdpSDRYZmhqbkJaZmRVUkdXbVRHbllRSmZ6OE1SCkNnNjFvL2EzQW9HQVJyR1pLUzRXcWtKazlZa3BsNmxxMlZOeXA5bVNQQS80dWFTcFhTdWR6RzhoeHFvSm5RSFIKV1E5VWUvSGY5Mm5Hd284RXkwZURzTXJrU0xNOVQvajV0QkpQbUZicHhwandGbjhFYkhjUmNsZkhZOFNkQ0k1TApHeDhrM1MxZ3VWRW9QYmo2YnVyOFRwSFdyVjFtR2lDOENqTXh5bi93V2dXcjNpSG9jV1VSaElNPQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo="
	trustedRootTLSCert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURzRENDQXBpZ0F3SUJBZ0lSQVBuR09XY1NyL3ZLdVlIaW43VUFZbG93RFFZSktvWklodmNOQVFFTEJRQXcKYXpFMU1ETUdBMVVFQ2hNc1NXNTBaWEp1WVhScGIyNWhiQ0JDZFhOcGJtVnpjeUJOWVdOb2FXNWxjeUJKYm1OdgpjbkJ2Y21GMFpXUXhNakF3QmdOVkJBTU1LU291Ykc5allXeG9ieTV6ZENCcGJuUmxaM0poZEdsdmJpQjBaWE4wCklHTmxjblJwWm1sallYUmxNQjRYRFRJek1ERXdNakV5TXpjeU5Gb1hEVEk0TURFd01URXlNemN5TkZvd2F6RTEKTURNR0ExVUVDaE1zU1c1MFpYSnVZWFJwYjI1aGJDQkNkWE5wYm1WemN5Qk5ZV05vYVc1bGN5QkpibU52Y25CdgpjbUYwWldReE1qQXdCZ05WQkFNTUtTb3ViRzlqWVd4b2J5NXpkQ0JwYm5SbFozSmhkR2x2YmlCMFpYTjBJR05sCmNuUnBabWxqWVhSbE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBMDRwbTl1WWYKT0g2SFRTWUk4WW5XSGJZb2xWcDdhL0lKVnYvNDR5Wm5YZFJLNXJwZys2TG5TazBBS1p2OHRpa0JrZXZRRTVzWApKYzVtYldhZjFtYmhvbVY0U2RObzRuNkw4aUdTWERjR3FocTBLWUJ2ZjFrOUJ6SGZxKzQ0OEQxaG1nL0ZkTlQwCmFJWUN3akNhZytWT0Jtcm9rY1pjSXFWT3VHL1NTWXd0Q3FiRU1YeExkczUrd0U2NnNYeWx5Si82MGZ5aThOdFoKaDdwcXNvTU9WS0Zla2FwWHFIWXgzTTlVd3ZjUWszYkEyUFZkUXZsZ0RBc3VIZWpqb0xBU0pEN1YrclZ6NVRIMgpsZUhxYzR1cVlScmkvT2o2UzlaVUFHc1AvTnRtMjc3eld2OWlCTkxlR1padkZ6SlQ3Tm5udmpDeGR5YmppUmtVCmp2OFVVTDdxOE82T1h3SURBUUFCbzA4d1RUQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBVEFNQmdOVkhSTUIKQWY4RUFqQUFNQ2dHQTFVZEVRUWhNQitDQ1d4dlkyRnNhRzl6ZElJTUtpNXNiMk5oYkdodkxuTjBod1IvQUFBQgpNQTBHQ1NxR1NJYjNEUUVCQ3dVQUE0SUJBUUM1bDJ6UzY0K3RBYmprVU05MkxFNC82dDdtTHlRNkVqSWxWTHBjCk53WXVlOU9LbHdLU0xGajFlaTdLVW0yRVFWNVpzdmhMejZXTGZndWtsd0NlSWVsMGNxM3RiSEJ1VFFnR0N3QkEKb0VVUjB3Nm82NFp6dmhNWWRRWENzVkJGYjNXWkdTeURnMFZ1b0NZdkJRQ1IxTmFOU2Z5UlF5bXZGSWtBTHJ3bgoxbmNPNnNOVVBaKzRUSW5UYnRTRWY3QUU2eXF3T0F3VXVIcW81amk3bFpuaVhDdW5oM1h4WGhoK29xT01URjZwCmFoNzI1cmdCNWdhbldyL3JmS2RkZUdrc2xTUlVDY2tXTWVEYXJETllRRXFJd1N0enorZC9OS1ZuZUtKaERtMjAKWFRBTUFhNGRQcmhrK1ZMZitwcEZjZVlMa0hYcE1YdjMxV3pRWWN6VHV3QnorUklRCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
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
