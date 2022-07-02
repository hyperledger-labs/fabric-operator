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

package validator_test

import (
	"context"
	"encoding/base64"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	initvalidator "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/validator"
)

const (
	testcert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNpVENDQWkrZ0F3SUJBZ0lVRkd3N0RjK0QvZUoyY08wOHd6d2tialIzK1M4d0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRBd09URTBNakF3TUZvWERUSXdNVEF3T0RFME1qQXdNRm93YnpFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1TQXdIZ1lEVlFRREV4ZFRZV0ZrY3kxTllXTkNiMjlyCkxWQnlieTVzYjJOaGJEQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJBK0JBRzhZakJvTllabGgKRjFrVHNUbHd6VERDQTJocDhZTXI5Ky8vbEd0NURoSGZVT1c3bkhuSW1USHlPRjJQVjFPcVRuUWhUbWpLYTdaQwpqeU9BUWxLamdhOHdnYXd3RGdZRFZSMFBBUUgvQkFRREFnT29NQjBHQTFVZEpRUVdNQlFHQ0NzR0FRVUZCd01CCkJnZ3JCZ0VGQlFjREFqQU1CZ05WSFJNQkFmOEVBakFBTUIwR0ExVWREZ1FXQkJTbHJjL0lNQkxvMzR0UktvWnEKNTQreDIyYWEyREFmQmdOVkhTTUVHREFXZ0JSWmpxT3RQZWJzSFI2UjBNQUhrNnd4ei85UFZqQXRCZ05WSFJFRQpKakFrZ2hkVFlXRmtjeTFOWVdOQ2IyOXJMVkJ5Ynk1c2IyTmhiSUlKYkc5allXeG9iM04wTUFvR0NDcUdTTTQ5CkJBTUNBMGdBTUVVQ0lRRGR0Y1QwUE9FQXJZKzgwdEhmWUwvcXBiWWoxMGU2eWlPWlpUQ29wY25mUVFJZ1FNQUQKaFc3T0NSUERNd3lqKzNhb015d2hFenFHYy9jRDJSU2V5ekRiRjFFPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	testkey  = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ3hRUXdSVFFpVUcwREo1UHoKQTJSclhIUEtCelkxMkxRa0MvbVlveWo1bEhDaFJBTkNBQVN5bE1YLzFqdDlmUGt1RTZ0anpvSTlQbGt4LzZuVQpCMHIvMU56TTdrYnBjUk8zQ3RIeXQ2TXlQR21FOUZUN29pYXphU3J1TW9JTDM0VGdBdUpIOU9ZWQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg=="
)

var _ = Describe("validator", func() {
	var (
		validator  *initvalidator.Validator
		instance   *current.IBPPeer
		mockClient *controllermocks.Client

		testCertBytes []byte
		testKeyBytes  []byte
	)

	BeforeEach(func() {
		var err error

		instance = &current.IBPPeer{}
		mockClient = &controllermocks.Client{}

		testCertBytes, err = base64.StdEncoding.DecodeString(testcert)
		Expect(err).NotTo(HaveOccurred())
		testKeyBytes, err = base64.StdEncoding.DecodeString(testkey)
		Expect(err).NotTo(HaveOccurred())

		mockClient.GetStub = func(ctx context.Context, t types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *corev1.Secret:
				if strings.Contains(t.Name, "keystore") {
					s := obj.(*corev1.Secret)
					s.Data = map[string][]byte{
						"key.pem": testKeyBytes,
					}
				} else {
					s := obj.(*corev1.Secret)
					s.Data = map[string][]byte{
						"cert.pem": testCertBytes,
					}
				}
			}
			return nil
		}

		validator = &initvalidator.Validator{
			Client: mockClient,
		}
	})

	Context("check ecert certs", func() {
		It("returns an error if secret contains no certs", func() {
			mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *corev1.Secret:
					s := obj.(*corev1.Secret)
					s.Data = nil
				}
				return nil
			}

			err := validator.CheckEcertCrypto(instance, instance.GetName())
			Expect(err).To(HaveOccurred())
		})

		It("returns no error if a valid cert found in secret", func() {
			err := validator.CheckEcertCrypto(instance, instance.GetName())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("check tls certs", func() {
		It("returns an error if secret contains no certs", func() {
			mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *corev1.Secret:
					s := obj.(*corev1.Secret)
					s.Data = nil
				}
				return nil
			}

			err := validator.CheckTLSCrypto(instance, instance.GetName())
			Expect(err).To(HaveOccurred())
		})

		It("returns no error if a valid cert found in secret", func() {
			err := validator.CheckTLSCrypto(instance, instance.GetName())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("check client auth certs", func() {
		It("returns an error if secret contains no certs", func() {
			mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *corev1.Secret:
					s := obj.(*corev1.Secret)
					s.Data = nil
				}
				return nil
			}

			err := validator.CheckClientAuthCrypto(instance, instance.GetName())
			Expect(err).To(HaveOccurred())
		})

		It("returns no error if a valid cert found in secret", func() {
			err := validator.CheckClientAuthCrypto(instance, instance.GetName())
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
