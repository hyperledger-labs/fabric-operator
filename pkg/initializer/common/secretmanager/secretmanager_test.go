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

package secretmanager_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/secretmanager"
)

var _ = Describe("Secretmanager", func() {
	Context("generate secrets", func() {
		var (
			resp          *config.Response
			mockClient    *controllermocks.Client
			instance      v1.Object
			secretManager *secretmanager.SecretManager
		)

		BeforeEach(func() {
			mockClient = &controllermocks.Client{}
			instance = &current.IBPPeer{}

			getLabels := func(instance v1.Object) map[string]string {
				return map[string]string{}
			}
			secretManager = secretmanager.New(mockClient, runtime.NewScheme(), getLabels)

			resp = &config.Response{
				CACerts:           [][]byte{[]byte("cacert")},
				IntermediateCerts: [][]byte{[]byte("intercert")},
				AdminCerts:        [][]byte{[]byte("admincert")},
				SignCert:          []byte("signcert"),
				Keystore:          []byte("key"),
			}

			mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				o := obj.(*corev1.Secret)
				switch types.Name {
				case "ecert-" + instance.GetName() + "-signcert":
					o.Name = "ecert-" + instance.GetName() + "-signcert"
					o.Namespace = instance.GetNamespace()
					o.Data = map[string][]byte{"cert.pem": []byte("signcert")}
				case "ecert-" + instance.GetName() + "-keystore":
					o.Name = "ecert-" + instance.GetName() + "-keystore"
					o.Namespace = instance.GetNamespace()
					o.Data = map[string][]byte{"key.pem": []byte("key")}
				case "ecert-" + instance.GetName() + "-admincerts":
					o.Name = "ecert-" + instance.GetName() + "-admincerts"
					o.Namespace = instance.GetNamespace()
					o.Data = map[string][]byte{
						"admincert-0.pem": []byte("admincert"),
						"admincert-1.pem": []byte("admincert"),
					}
				}
				return nil
			}
		})

		Context("admin certs secret", func() {
			It("returns an error on failure", func() {
				msg := "admin certs error"
				mockClient.CreateOrUpdateReturnsOnCall(0, errors.New(msg))

				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create admin certs secret: " + msg))
			})

			It("generates ecert admin cert secret", func() {
				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockClient.CreateOrUpdateCallCount()).To(Equal(5))
			})

			It("does not generate tls admin cert secret", func() {
				err := secretManager.GenerateSecrets("tls", instance, resp)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockClient.CreateOrUpdateCallCount()).To(Equal(4))
			})
		})

		Context("ca certs secret", func() {
			It("returns an error on failure", func() {
				msg := "ca certs error"
				mockClient.CreateOrUpdateReturnsOnCall(1, errors.New(msg))

				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create ca certs secret: " + msg))
			})

			It("generates", func() {
				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("intermediate certs secret", func() {
			It("returns an error on failure", func() {
				msg := "intermediate certs error"
				mockClient.CreateOrUpdateReturnsOnCall(2, errors.New(msg))

				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create intermediate ca certs secret: " + msg))
			})

			It("generates", func() {
				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("sign certs secret", func() {
			It("returns an error on failure", func() {
				msg := "sign certs error"
				mockClient.CreateOrUpdateReturnsOnCall(3, errors.New(msg))

				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create signing cert secret: " + msg))
			})

			It("generates", func() {
				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("key secret", func() {
			It("returns an error on failure", func() {
				msg := "key error"
				mockClient.CreateOrUpdateReturnsOnCall(4, errors.New(msg))

				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create key secret: " + msg))
			})

			It("generates", func() {
				err := secretManager.GenerateSecrets("ecert", instance, resp)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("get crypto from secret", func() {
			It("returns an error if failed to get secret", func() {
				mockClient.GetReturns(errors.New("get error"))
				_, err := secretManager.GetCryptoFromSecrets("ecert", instance)
				Expect(err).To(HaveOccurred())
			})

			It("returns crypto response from tls cert secrets", func() {
				tlscrypto, err := secretManager.GetCryptoFromSecrets("ecert", instance)
				Expect(err).NotTo(HaveOccurred())
				Expect(tlscrypto).NotTo(BeNil())

				Expect(tlscrypto.AdminCerts).To(Equal([][]byte{[]byte("admincert"), []byte("admincert")}))
				Expect(tlscrypto.SignCert).To(Equal([]byte("signcert")))
				Expect(tlscrypto.Keystore).To(Equal([]byte("key")))
			})
		})
	})

})
