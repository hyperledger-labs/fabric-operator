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

package baseca_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	clientmocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	cav1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca"
	baseca "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca"
	basecamocks "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/ca/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Initialize CA", func() {
	var (
		instance        *current.IBPCA
		cainit          *baseca.Initialize
		mockinitializer *basecamocks.Initializer
		mockClient      *clientmocks.Client
		update          *basecamocks.Update
	)

	BeforeEach(func() {
		jm, err := util.ConvertToJsonMessage(&cav1.ServerConfig{})
		Expect(err).NotTo(HaveOccurred())

		instance = &current.IBPCA{
			Spec: current.IBPCASpec{
				ConfigOverride: &current.ConfigOverride{
					CA:    &runtime.RawExtension{Raw: *jm},
					TLSCA: &runtime.RawExtension{Raw: *jm},
				},
			},
		}

		config := &initializer.Config{
			CADefaultConfigPath:    "../../../../defaultconfig/ca/ca.yaml",
			TLSCADefaultConfigPath: "../../../../defaultconfig/ca/tlsca.yaml",
		}

		update = &basecamocks.Update{}
		mockClient = &clientmocks.Client{}
		scheme := &runtime.Scheme{}
		labels := func(v1.Object) map[string]string {
			return nil
		}

		mockinitializer = &basecamocks.Initializer{}

		cainit = &baseca.Initialize{
			Config:      config,
			Scheme:      scheme,
			Labels:      labels,
			Initializer: mockinitializer,
			Client:      mockClient,
		}
	})

	Context("handle enrollment ca's config", func() {
		Context("enrollment ca", func() {
			It("calls update enrollment when update detected", func() {
				update.CAOverridesUpdatedReturns(true)
				_, err := cainit.HandleEnrollmentCAInit(instance, update)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockinitializer.UpdateCallCount()).To(Equal(1))
			})

			It("calls create enrollment when update detected", func() {
				mockClient.GetReturns(errors.New("secret not found"))
				_, err := cainit.HandleEnrollmentCAInit(instance, update)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockinitializer.CreateCallCount()).To(Equal(1))
			})
		})

		Context("tls ca", func() {
			It("calls update enrollment when update detected", func() {
				update.TLSCAOverridesUpdatedReturns(true)
				_, err := cainit.HandleTLSCAInit(instance, update)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockinitializer.UpdateCallCount()).To(Equal(1))
			})

			It("calls create enrollment when update detected", func() {
				mockClient.GetReturns(errors.New("secret not found"))
				_, err := cainit.HandleTLSCAInit(instance, update)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockinitializer.CreateCallCount()).To(Equal(1))
			})
		})
	})

	Context("create enrollment ca's config", func() {
		It("returns an error if create fails", func() {
			msg := "failed to create"
			mockinitializer.CreateReturns(nil, errors.New(msg))
			_, err := cainit.CreateEnrollmentCAConfig(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("creates config", func() {
			_, err := cainit.CreateEnrollmentCAConfig(instance)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("update enrollment ca's config", func() {
		It("returns an error if update fails", func() {
			msg := "failed to update"
			mockinitializer.UpdateReturns(nil, errors.New(msg))
			_, err := cainit.UpdateEnrollmentCAConfig(instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(msg))
		})

		It("update config", func() {
			_, err := cainit.UpdateEnrollmentCAConfig(instance)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("create config resouces", func() {
		var resp *initializer.Response

		BeforeEach(func() {
			resp = &initializer.Response{
				CryptoMap: map[string][]byte{"cert.pem": []byte("cert.pem")},
				Config:    &cav1.ServerConfig{},
			}
		})

		It("returns an error if secret creation fails", func() {
			msg := "failed to create secret"
			mockClient.CreateOrUpdateReturns(errors.New(msg))
			err := cainit.CreateConfigResources("ibpca1", instance, resp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to create/update secret: " + msg))
		})

		It("returns an error if config map creation fails", func() {
			msg := "failed to create cm"
			mockClient.CreateOrUpdateReturnsOnCall(1, errors.New(msg))
			err := cainit.CreateConfigResources("ibpca1", instance, resp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to create/update config map: " + msg))
		})

		It("create secret and configmap", func() {
			err := cainit.CreateConfigResources("ibpca1", instance, resp)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("merge crypto", func() {
		var (
			oldCrypto = map[string][]byte{}
			newCrypto = map[string][]byte{}
		)

		BeforeEach(func() {
			oldCrypto = map[string][]byte{
				"key1.pem": []byte("key1"),
				"key2.pem": []byte("key2"),
				"cert.pem": []byte("cert"),
			}

			newCrypto = map[string][]byte{
				"key1.pem": []byte("newkey1"),
				"cert.pem": []byte("newcert"),
			}
		})

		It("only updates keys that have new values", func() {
			merged := cainit.MergeCryptoMaterial(oldCrypto, newCrypto)
			Expect(merged["key1.pem"]).To(Equal([]byte("newkey1")))
			Expect(merged["key2.pem"]).To(Equal([]byte("key2")))
			Expect(merged["cert.pem"]).To(Equal([]byte("newcert")))
		})
	})
})
