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

package initializer_test

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	commonmocks "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/mocks"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testcert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNpVENDQWkrZ0F3SUJBZ0lVRkd3N0RjK0QvZUoyY08wOHd6d2tialIzK1M4d0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRBd09URTBNakF3TUZvWERUSXdNVEF3T0RFME1qQXdNRm93YnpFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1TQXdIZ1lEVlFRREV4ZFRZV0ZrY3kxTllXTkNiMjlyCkxWQnlieTVzYjJOaGJEQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJBK0JBRzhZakJvTllabGgKRjFrVHNUbHd6VERDQTJocDhZTXI5Ky8vbEd0NURoSGZVT1c3bkhuSW1USHlPRjJQVjFPcVRuUWhUbWpLYTdaQwpqeU9BUWxLamdhOHdnYXd3RGdZRFZSMFBBUUgvQkFRREFnT29NQjBHQTFVZEpRUVdNQlFHQ0NzR0FRVUZCd01CCkJnZ3JCZ0VGQlFjREFqQU1CZ05WSFJNQkFmOEVBakFBTUIwR0ExVWREZ1FXQkJTbHJjL0lNQkxvMzR0UktvWnEKNTQreDIyYWEyREFmQmdOVkhTTUVHREFXZ0JSWmpxT3RQZWJzSFI2UjBNQUhrNnd4ei85UFZqQXRCZ05WSFJFRQpKakFrZ2hkVFlXRmtjeTFOWVdOQ2IyOXJMVkJ5Ynk1c2IyTmhiSUlKYkc5allXeG9iM04wTUFvR0NDcUdTTTQ5CkJBTUNBMGdBTUVVQ0lRRGR0Y1QwUE9FQXJZKzgwdEhmWUwvcXBiWWoxMGU2eWlPWlpUQ29wY25mUVFJZ1FNQUQKaFc3T0NSUERNd3lqKzNhb015d2hFenFHYy9jRDJSU2V5ekRiRjFFPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	testkey  = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ3hRUXdSVFFpVUcwREo1UHoKQTJSclhIUEtCelkxMkxRa0MvbVlveWo1bEhDaFJBTkNBQVN5bE1YLzFqdDlmUGt1RTZ0anpvSTlQbGt4LzZuVQpCMHIvMU56TTdrYnBjUk8zQ3RIeXQ2TXlQR21FOUZUN29pYXphU3J1TW9JTDM0VGdBdUpIOU9ZWQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg=="
)

var _ = Describe("Initializing the Orderer", func() {
	var (
		ordererInitializer *initializer.Initializer
		instance           *current.IBPOrderer
		mockClient         *controllermocks.Client
		mockValidator      *commonmocks.CryptoValidator
	)

	BeforeEach(func() {
		testCertBytes, err := base64.StdEncoding.DecodeString(testcert)
		Expect(err).NotTo(HaveOccurred())

		mockValidator = &commonmocks.CryptoValidator{}

		mockClient = &controllermocks.Client{}
		mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			switch obj.(type) {
			case *corev1.Secret:
				s := obj.(*corev1.Secret)
				s.Data = map[string][]byte{"cert.pem": testCertBytes}
			}
			return nil
		}

		ordererInitializer = initializer.New(mockClient, &runtime.Scheme{}, nil, "", mockValidator)

		enrollment := &current.Enrollment{
			CAHost:       "localhost",
			CAPort:       "7054",
			EnrollID:     "admin",
			EnrollSecret: "adminpw",
			CATLS: &current.CATLS{
				CACert: testcert,
			},
		}
		tlsenrollment := enrollment.DeepCopy()

		msp := &current.MSP{
			KeyStore:   testkey,
			SignCerts:  testcert,
			AdminCerts: []string{testcert},
			CACerts:    []string{testcert},
		}
		tlsmsp := msp.DeepCopy()

		instance = &current.IBPOrderer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: current.IBPOrdererSpec{
				Secret: &current.SecretSpec{
					Enrollment: &current.EnrollmentSpec{
						Component: enrollment,
						TLS:       tlsenrollment,
						ClientAuth: &current.Enrollment{
							CAHost:       "host",
							CAPort:       "1234",
							EnrollID:     "admin",
							EnrollSecret: "adminpw",
							CATLS: &current.CATLS{
								CACert: "cert",
							},
						},
					},
					MSP: &current.MSPSpec{
						Component: msp,
						TLS:       tlsmsp,
						ClientAuth: &current.MSP{
							KeyStore:  "key",
							SignCerts: "cert",
							CACerts:   []string{"certs"},
						},
					},
				},
			},
		}
	})

	PContext("create", func() {
		// TODO
	})

	PContext("update", func() {
		// TODO
	})

	Context("check for missing crypto", func() {
		It("returns true, if missing any crypto", func() {
			mockValidator.CheckEcertCryptoReturns(errors.New("not found"))
			missing := ordererInitializer.MissingCrypto(instance)
			Expect(missing).To(Equal(true))
		})

		It("returns false, if all crypto found and is in proper format", func() {
			missing := ordererInitializer.MissingCrypto(instance)
			Expect(missing).To(Equal(false))
		})
	})

	Context("get init orderer", func() {
		It("returns empty init peer if neither MSP nor enrollment spec is passed", func() {
			instance.Spec.Secret.MSP.TLS = nil
			instance.Spec.Secret.Enrollment.TLS = nil
			initorderer, err := ordererInitializer.GetInitOrderer(instance, "foo")
			Expect(err).NotTo(HaveOccurred())
			Expect(initorderer.Cryptos).NotTo(BeNil())
			Expect(initorderer.Cryptos.TLS).To(BeNil())
		})

		It("returns init peer with ecert, tls, clientauth enrollers", func() {
			initorderer, err := ordererInitializer.GetInitOrderer(instance, "foo")
			Expect(err).NotTo(HaveOccurred())
			Expect(initorderer.Cryptos).NotTo(BeNil())
			Expect(initorderer.Cryptos.Enrollment).NotTo(BeNil())
			Expect(initorderer.Cryptos.TLS).NotTo(BeNil())
			Expect(initorderer.Cryptos.ClientAuth).NotTo(BeNil())
		})

		It("returns init peer with ecert, tls, clientauth msp parsers", func() {
			initorderer, err := ordererInitializer.GetInitOrderer(instance, "foo")
			Expect(err).NotTo(HaveOccurred())
			Expect(initorderer.Cryptos).NotTo(BeNil())
			Expect(initorderer.Cryptos.Enrollment).NotTo(BeNil())
			Expect(initorderer.Cryptos.TLS).NotTo(BeNil())
			Expect(initorderer.Cryptos.ClientAuth).NotTo(BeNil())
		})

		It("returns ecert msp parsers and tls enrollers", func() {
			instance.Spec.Secret.Enrollment.Component = nil
			instance.Spec.Secret.MSP.TLS = nil
			initorderer, err := ordererInitializer.GetInitOrderer(instance, "foo")
			Expect(err).NotTo(HaveOccurred())
			Expect(initorderer.Cryptos).NotTo(BeNil())
			Expect(initorderer.Cryptos.Enrollment).NotTo(BeNil())
			Expect(initorderer.Cryptos.TLS).NotTo(BeNil())
		})
	})

	Context("create or update config map", func() {
		BeforeEach(func() {
			wd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())

			ordererInitializer.Config = &initializer.Config{
				OUFile:      filepath.Join(wd, "../../../defaultconfig/orderer/ouconfig.yaml"),
				InterOUFile: filepath.Join(wd, "../../../defaultconfig/orderer/ouconfig-inter.yaml"),
			}

			// Trigger create config map logic
			mockClient.GetReturns(k8serrors.NewNotFound(schema.GroupResource{}, "not found"))
		})

		It("returns error if failed to create config map", func() {
			mockClient.CreateOrUpdateReturns(errors.New("update error"))
			err := ordererInitializer.CreateOrUpdateConfigMap(instance, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("update error"))
		})

		It("creates config map with node ou config", func() {
			err := ordererInitializer.CreateOrUpdateConfigMap(instance, nil)
			Expect(err).NotTo(HaveOccurred())

			_, obj, _ := mockClient.CreateOrUpdateArgsForCall(0)
			cm := obj.(*corev1.ConfigMap)
			Expect(cm.BinaryData["config.yaml"]).NotTo(BeNil())
			nodeOUs, err := commonconfig.NodeOUConfigFromBytes(cm.BinaryData["config.yaml"])
			Expect(nodeOUs.NodeOUs.Enable).To(Equal(true))
		})
	})

})
