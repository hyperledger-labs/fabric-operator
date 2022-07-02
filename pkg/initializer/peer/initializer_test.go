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
	"encoding/pem"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	commonconfig "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	commonmocks "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/mocks"
	peer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	config "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/config/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
)

const (
	testcert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNpVENDQWkrZ0F3SUJBZ0lVRkd3N0RjK0QvZUoyY08wOHd6d2tialIzK1M4d0NnWUlLb1pJemowRUF3SXcKYURFTE1Ba0dBMVVFQmhNQ1ZWTXhGekFWQmdOVkJBZ1REazV2Y25Sb0lFTmhjbTlzYVc1aE1SUXdFZ1lEVlFRSwpFd3RJZVhCbGNteGxaR2RsY2pFUE1BMEdBMVVFQ3hNR1JtRmljbWxqTVJrd0Z3WURWUVFERXhCbVlXSnlhV010ClkyRXRjMlZ5ZG1WeU1CNFhEVEU1TVRBd09URTBNakF3TUZvWERUSXdNVEF3T0RFME1qQXdNRm93YnpFTE1Ba0cKQTFVRUJoTUNWVk14RnpBVkJnTlZCQWdURGs1dmNuUm9JRU5oY205c2FXNWhNUlF3RWdZRFZRUUtFd3RJZVhCbApjbXhsWkdkbGNqRVBNQTBHQTFVRUN4TUdSbUZpY21sak1TQXdIZ1lEVlFRREV4ZFRZV0ZrY3kxTllXTkNiMjlyCkxWQnlieTVzYjJOaGJEQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJBK0JBRzhZakJvTllabGgKRjFrVHNUbHd6VERDQTJocDhZTXI5Ky8vbEd0NURoSGZVT1c3bkhuSW1USHlPRjJQVjFPcVRuUWhUbWpLYTdaQwpqeU9BUWxLamdhOHdnYXd3RGdZRFZSMFBBUUgvQkFRREFnT29NQjBHQTFVZEpRUVdNQlFHQ0NzR0FRVUZCd01CCkJnZ3JCZ0VGQlFjREFqQU1CZ05WSFJNQkFmOEVBakFBTUIwR0ExVWREZ1FXQkJTbHJjL0lNQkxvMzR0UktvWnEKNTQreDIyYWEyREFmQmdOVkhTTUVHREFXZ0JSWmpxT3RQZWJzSFI2UjBNQUhrNnd4ei85UFZqQXRCZ05WSFJFRQpKakFrZ2hkVFlXRmtjeTFOWVdOQ2IyOXJMVkJ5Ynk1c2IyTmhiSUlKYkc5allXeG9iM04wTUFvR0NDcUdTTTQ5CkJBTUNBMGdBTUVVQ0lRRGR0Y1QwUE9FQXJZKzgwdEhmWUwvcXBiWWoxMGU2eWlPWlpUQ29wY25mUVFJZ1FNQUQKaFc3T0NSUERNd3lqKzNhb015d2hFenFHYy9jRDJSU2V5ekRiRjFFPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
	testkey  = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ3hRUXdSVFFpVUcwREo1UHoKQTJSclhIUEtCelkxMkxRa0MvbVlveWo1bEhDaFJBTkNBQVN5bE1YLzFqdDlmUGt1RTZ0anpvSTlQbGt4LzZuVQpCMHIvMU56TTdrYnBjUk8zQ3RIeXQ2TXlQR21FOUZUN29pYXphU3J1TW9JTDM0VGdBdUpIOU9ZWQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg=="
)

var _ = Describe("Initializing the Peer", func() {
	var (
		peerinitializer *peer.Initializer
		instance        *current.IBPPeer
		mockClient      *controllermocks.Client
		mockValidator   *commonmocks.CryptoValidator
		serverURL       string
		serverCert      string
		serverUrlObj    *url.URL
	)

	BeforeEach(func() {
		serverURL = server.URL
		rawCert := server.Certificate().Raw
		pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rawCert})
		serverCert = string(util.BytesToBase64(pemCert))

		urlObj, err := url.Parse(serverURL)
		Expect(err).NotTo(HaveOccurred())
		serverUrlObj = urlObj

		mockClient = &controllermocks.Client{}
		mockValidator = &commonmocks.CryptoValidator{}
		getLabels := func(instance metav1.Object) map[string]string {
			return map[string]string{}
		}
		peerinitializer = peer.New(nil, &runtime.Scheme{}, mockClient, getLabels, mockValidator, enroller.HSMEnrollJobTimeouts{})

		instance = &current.IBPPeer{
			Spec: current.IBPPeerSpec{
				Secret: &current.SecretSpec{
					Enrollment: &current.EnrollmentSpec{
						Component: &current.Enrollment{
							CAHost:       serverUrlObj.Hostname(),
							CAPort:       serverUrlObj.Port(),
							EnrollID:     "admin",
							EnrollSecret: "adminpw",
							CATLS: &current.CATLS{
								CACert: serverCert,
							},
							AdminCerts: []string{testcert},
						},
						TLS: &current.Enrollment{
							CAHost:       serverUrlObj.Hostname(),
							CAPort:       serverUrlObj.Port(),
							EnrollID:     "admin",
							EnrollSecret: "adminpw",
							CATLS: &current.CATLS{
								CACert: serverCert,
							},
						},
						ClientAuth: &current.Enrollment{
							CAHost:       serverUrlObj.Hostname(),
							CAPort:       serverUrlObj.Port(),
							EnrollID:     "admin",
							EnrollSecret: "adminpw",
							CATLS: &current.CATLS{
								CACert: serverCert,
							},
						},
					},
					MSP: &current.MSPSpec{
						Component: &current.MSP{
							KeyStore:   "key",
							SignCerts:  "cert",
							CACerts:    []string{"certs"},
							AdminCerts: []string{testcert},
						},
						TLS: &current.MSP{
							KeyStore:  "key",
							SignCerts: "cert",
							CACerts:   []string{"certs"},
						},
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

	Context("create", func() {
		var peer *mocks.IBPPeer

		BeforeEach(func() {
			peer = &mocks.IBPPeer{}
		})

		It("returns an error if it fails to override peer's config", func() {
			msg := "failed to override"
			peer.OverrideConfigReturns(errors.New(msg))

			_, err := peerinitializer.Create(&config.Core{}, peer, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("returns an error if it fails to generate crypto", func() {
			msg := "failed to generate crypto"
			peer.GenerateCryptoReturns(nil, errors.New(msg))

			_, err := peerinitializer.Create(&config.Core{}, peer, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("creates and returns response containing config and crypto", func() {
			_, err := peerinitializer.Create(&config.Core{}, peer, "blah")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("update", func() {
		var peer *mocks.IBPPeer

		BeforeEach(func() {
			peer = &mocks.IBPPeer{}
		})

		It("returns an error if it fails to override peer's config", func() {
			msg := "failed to override"
			peer.OverrideConfigReturns(errors.New(msg))

			_, err := peerinitializer.Update(&config.Core{}, peer)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("creates and returns response containing config and crypto", func() {
			_, err := peerinitializer.Update(&config.Core{}, peer)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("get init peer", func() {
		It("returns empty init peer if neither MSP nor enrollment spec is passed", func() {
			instance.Spec.Secret.MSP.TLS = nil
			instance.Spec.Secret.Enrollment.TLS = nil
			initpeer, err := peerinitializer.GetInitPeer(instance, "foo")
			Expect(err).NotTo(HaveOccurred())
			Expect(initpeer.Cryptos).NotTo(BeNil())
			Expect(initpeer.Cryptos.TLS).To(BeNil())
		})

		It("returns init peer with ecert, tls, clientauth enrollers", func() {
			initpeer, err := peerinitializer.GetInitPeer(instance, "foo")
			Expect(err).NotTo(HaveOccurred())
			Expect(initpeer.Cryptos).NotTo(BeNil())
			Expect(initpeer.Cryptos.Enrollment).NotTo(BeNil())
			Expect(initpeer.Cryptos.TLS).NotTo(BeNil())
			Expect(initpeer.Cryptos.ClientAuth).NotTo(BeNil())
		})

		It("returns init peer with ecert, tls, clientauth msp parsers", func() {
			initpeer, err := peerinitializer.GetInitPeer(instance, "foo")
			Expect(err).NotTo(HaveOccurred())
			Expect(initpeer.Cryptos).NotTo(BeNil())
			Expect(initpeer.Cryptos.Enrollment).NotTo(BeNil())
			Expect(initpeer.Cryptos.TLS).NotTo(BeNil())
			Expect(initpeer.Cryptos.ClientAuth).NotTo(BeNil())
		})

		It("returns ecert msp parsers and tls enrollers", func() {
			instance.Spec.Secret.Enrollment.Component = nil
			instance.Spec.Secret.MSP.TLS = nil
			initpeer, err := peerinitializer.GetInitPeer(instance, "foo")
			Expect(err).NotTo(HaveOccurred())
			Expect(initpeer.Cryptos).NotTo(BeNil())
			Expect(initpeer.Cryptos.Enrollment).NotTo(BeNil())
			Expect(initpeer.Cryptos.TLS).NotTo(BeNil())
		})
	})

	Context("generate secrets", func() {
		var (
			resp *commonconfig.Response
		)

		BeforeEach(func() {
			resp = &commonconfig.Response{
				CACerts:           [][]byte{[]byte("cacert")},
				IntermediateCerts: [][]byte{[]byte("intercert")},
				AdminCerts:        [][]byte{[]byte("admincert")},
				SignCert:          []byte("signcert"),
				Keystore:          []byte("key"),
			}
		})

		It("returns an error if fails to create a secret", func() {
			msg := "admin certs error"
			mockClient.CreateOrUpdateReturnsOnCall(0, errors.New(msg))

			err := peerinitializer.GenerateSecrets("ecert", instance, resp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create admin certs secret: " + msg))
		})

		It("generates", func() {
			err := peerinitializer.GenerateSecrets("ecert", instance, resp)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("check for missing crypto", func() {
		It("returns true, if missing any crypto", func() {
			mockValidator.CheckEcertCryptoReturns(errors.New("not found"))
			missing := peerinitializer.MissingCrypto(instance)
			Expect(missing).To(Equal(true))
		})

		It("returns false, if all crypto found and is in proper format", func() {
			missing := peerinitializer.MissingCrypto(instance)
			Expect(missing).To(Equal(false))
		})
	})

	Context("check if admin certs need to be updated", func() {
		BeforeEach(func() {
			instance.Spec.Secret.Enrollment.Component.AdminCerts = []string{testcert}

			testCertBytes, err := base64.StdEncoding.DecodeString(testcert)
			Expect(err).NotTo(HaveOccurred())

			mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				switch obj.(type) {
				case *corev1.Secret:
					s := obj.(*corev1.Secret)
					s.Data = map[string][]byte{"cert.pem": testCertBytes}
				}
				return nil
			}
		})

		It("does not return an error if it fails to find admin secret", func() {
			errMsg := "failed to find admin certs secret"
			mockClient.GetReturns(errors.New(errMsg))
			_, err := peerinitializer.CheckIfAdminCertsUpdated(instance)
			Expect(err).NotTo(HaveOccurred())
		})

		When("admin certs updated as part of enrollment spec", func() {
			BeforeEach(func() {
				instance.Spec.Secret.MSP = nil
			})

			It("returns false when the same cert in spec as current admin certs secret", func() {
				needUpdating, err := peerinitializer.CheckIfAdminCertsUpdated(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(needUpdating).To(Equal(false))
			})

			It("returns an error if non-base64 encoded string passed as cert", func() {
				instance.Spec.Secret.Enrollment.Component.AdminCerts = []string{"foo"}
				_, err := peerinitializer.CheckIfAdminCertsUpdated(instance)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("illegal base64 data"))
			})

			It("returns true when the different cert in spec as current admin certs secret", func() {
				instance.Spec.Secret.Enrollment.Component.AdminCerts = []string{testkey}
				needUpdating, err := peerinitializer.CheckIfAdminCertsUpdated(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(needUpdating).To(Equal(true))
			})
		})

		When("admin certs updated as part of MSP spec", func() {
			BeforeEach(func() {
				instance.Spec.Secret.Enrollment = nil
			})

			It("returns false when the same cert in spec as current admin certs secret", func() {
				needUpdating, err := peerinitializer.CheckIfAdminCertsUpdated(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(needUpdating).To(Equal(false))
			})

			It("returns an error if non-base64 encoded string passed as cert", func() {
				instance.Spec.Secret.MSP.Component.AdminCerts = []string{"foo"}
				_, err := peerinitializer.CheckIfAdminCertsUpdated(instance)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("illegal base64 data"))
			})

			It("returns true when the diferent cert in spec as current admin certs secret", func() {
				instance.Spec.Secret.MSP.Component.AdminCerts = []string{testkey}
				needUpdating, err := peerinitializer.CheckIfAdminCertsUpdated(instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(needUpdating).To(Equal(true))
			})
		})
	})
})

// 	BeforeEach(func() {
// 		testCertBytes, err := base64.StdEncoding.DecodeString(testcert)
// 		Expect(err).NotTo(HaveOccurred())

// 		mockClient = &mocks.Client{}
// 		mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj runtime.Object) error {
// 			switch obj.(type) {
// 			case *corev1.Secret:
// 				s := obj.(*corev1.Secret)
// 				s.Data = map[string][]byte{"cert.pem": testCertBytes}
// 			}
// 			return nil
// 		}

// 		resp := &commonconfig.Response{
// 			CACerts:           [][]byte{[]byte("cacert")},
// 			IntermediateCerts: [][]byte{[]byte("intercert")},
// 			SignCert:          []byte("cert"),
// 			Keystore:          []byte("key"),
// 		}
// 		_ = resp

// 		peerInitializer = &initializer.Initializer{
// 			Client: mockClient,
// 		}

// 		enrollment := &current.Enrollment{
// 			CAHost:       "localhost",
// 			CAPort:       "7054",
// 			EnrollID:     "admin",
// 			EnrollSecret: "adminpw",
// 			CATLS: &current.CATLS{
// 				CACert: testcert,
// 			},
// 		}
// 		tlsenrollment := enrollment.DeepCopy()

// 		msp := &current.MSP{
// 			KeyStore:   testkey,
// 			SignCerts:  testcert,
// 			AdminCerts: []string{testcert},
// 			CACerts:    []string{testcert},
// 		}
// 		tlsmsp := msp.DeepCopy()

// 		instance = &current.IBPPeer{
// 			Spec: current.IBPPeerSpec{
// 				Secret: &current.SecretSpec{
// 					Enrollment: &current.EnrollmentSpec{
// 						Component: enrollment,
// 						TLS:       tlsenrollment,
// 					},
// 					MSP: &current.MSPSpec{
// 						Component: msp,
// 						TLS:       tlsmsp,
// 					},
// 				},
// 			},
// 		}
// 	})

// Context("check admin certs for existence and proper data", func() {
// 	It("returns error, if secret not found", func() {
// 		errMsg := "ecert admincerts secret not found"
// 		mockClient.GetReturns(errors.New(errMsg))
// 		err := peerInitializer.CheckAdminCerts(instance, "ecert")
// 		Expect(err).To(HaveOccurred())
// 		Expect(err.Error()).To(Equal(errMsg))
// 	})

// It("returns error, if secrets found but contains no data", func() {
// 	mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj runtime.Object) error {
// 		switch obj.(type) {
// 		case *corev1.Secret:
// 			s := obj.(*corev1.Secret)
// 			s.Data = nil
// 		}
// 		return nil
// 	}
// 	err := peerInitializer.CheckAdminCerts(instance, "ecert")
// 	Expect(err).To(HaveOccurred())
// 	Expect(err.Error()).To(Equal("no admin certificates found in admincerts secret"))
// })

// It("returns error, if secrets found but contains bad data", func() {
// 	mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj runtime.Object) error {
// 		switch obj.(type) {
// 		case *corev1.Secret:
// 			s := obj.(*corev1.Secret)
// 			s.Data = map[string][]byte{"cert.pem": []byte("foo")}
// 		}
// 		return nil
// 	}
// 	err := peerInitializer.CheckAdminCerts(instance, "ecert")
// 	Expect(err).To(HaveOccurred())
// 	Expect(err.Error()).To(Equal("not a proper admin cert: failed to get certificate block"))
// })

// It("returns no error, if secret found and contains proper data", func() {
// 	err := peerInitializer.CheckAdminCerts(instance, "ecert")
// 	Expect(err).NotTo(HaveOccurred())
// })
// })

// Context("check ca certs for existence and proper data", func() {
// 	It("returns error, if secret not found", func() {
// 		errMsg := "ecert cacerts secret not found"
// 		mockClient.GetReturns(errors.New(errMsg))
// 		err := peerInitializer.CheckCACerts(instance, "ecert")
// 		Expect(err).To(HaveOccurred())
// 		Expect(err.Error()).To(Equal(errMsg))
// 	})

// 	It("returns error, if secrets found but contains no data", func() {
// 		mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj runtime.Object) error {
// 			switch obj.(type) {
// 			case *corev1.Secret:
// 				s := obj.(*corev1.Secret)
// 				s.Data = nil
// 			}
// 			return nil
// 		}
// 		err := peerInitializer.CheckCACerts(instance, "ecert")
// 		Expect(err).To(HaveOccurred())
// 		Expect(err.Error()).To(Equal("no ca certificates found in cacerts secret"))
// 	})

// 	It("returns error, if secrets found but contains bad data", func() {
// 		mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj runtime.Object) error {
// 			switch obj.(type) {
// 			case *corev1.Secret:
// 				s := obj.(*corev1.Secret)
// 				s.Data = map[string][]byte{"cert.pem": []byte("foo")}
// 			}
// 			return nil
// 		}
// 		err := peerInitializer.CheckCACerts(instance, "ecert")
// 		Expect(err).To(HaveOccurred())
// 		Expect(err.Error()).To(Equal("not a proper ca cert: failed to get certificate block"))
// 	})

// 	It("returns no error, if secret found and contains proper data", func() {
// 		err := peerInitializer.CheckCACerts(instance, "ecert")
// 		Expect(err).NotTo(HaveOccurred())
// 	})
// })

// })
