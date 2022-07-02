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

package certificate_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	controllermocks "github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate/mocks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Certificate", func() {
	var (
		certificateManager *certificate.CertificateManager
		mockClient         *controllermocks.Client
		mockEnroller       *mocks.Reenroller
		instance           v1.Object

		certBytes []byte
	)

	BeforeEach(func() {
		mockClient = &controllermocks.Client{}
		mockEnroller = &mocks.Reenroller{}

		certificateManager = certificate.New(mockClient, &runtime.Scheme{})

		instance = &current.IBPPeer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "peer-1",
				Namespace: "peer-namespace",
				Labels:    map[string]string{},
			},
		}

		certBytes = createCert(time.Now().Add(time.Hour * 24 * 30)) // expires in 30 days

		reenrollResponse := &config.Response{
			SignCert: []byte("cert"),
			Keystore: []byte("key"),
		}

		mockEnroller.ReenrollReturns(reenrollResponse, nil)
		mockClient.UpdateReturns(nil)

		mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
			o := obj.(*corev1.Secret)
			switch types.Name {
			case "tls-" + instance.GetName() + "-signcert":
				o.Name = "tls-" + instance.GetName() + "-signcert"
				o.Namespace = instance.GetNamespace()
				o.Data = map[string][]byte{"cert.pem": certBytes}
			case "tls-" + instance.GetName() + "-keystore":
				o.Name = "tls-" + instance.GetName() + "-keystore"
				o.Namespace = instance.GetNamespace()
				o.Data = map[string][]byte{"key.pem": []byte("key")}
			case "ecert-" + instance.GetName() + "-signcert":
				o.Name = "ecert-" + instance.GetName() + "-signcert"
				o.Namespace = instance.GetNamespace()
				o.Data = map[string][]byte{"cert.pem": certBytes}
			case "ecert-" + instance.GetName() + "-keystore":
				o.Name = "ecert-" + instance.GetName() + "-keystore"
				o.Namespace = instance.GetNamespace()
				o.Data = map[string][]byte{"key.pem": []byte("key")}
			}
			return nil
		}
	})

	Context("get expire date", func() {
		It("returns error if fails to read certificate", func() {
			certbytes := []byte("invalid")
			_, err := certificateManager.GetExpireDate(certbytes)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get certificate from bytes"))
		})

		It("returns expire date of certificate", func() {
			expectedtime := time.Now().Add(time.Hour * 24 * 30).UTC()
			expireDate, err := certificateManager.GetExpireDate(certBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(expireDate.Month()).To(Equal(expectedtime.Month()))
			Expect(expireDate.Day()).To(Equal(expectedtime.Day()))
			Expect(expireDate.Year()).To(Equal(expectedtime.Year()))
		})
	})

	Context("get duration to next renewal", func() {
		It("returns error if fails to get expire date", func() {
			mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				o := obj.(*corev1.Secret)
				o.Name = "tls-" + instance.GetName() + "-signcert"
				o.Namespace = instance.GetNamespace()
				o.Data = map[string][]byte{"cert.pem": []byte("invalid")}
				return nil
			}
			thirtyDaysToSeconds := int64(30 * 24 * 60 * 60)
			_, err := certificateManager.GetDurationToNextRenewal(common.TLS, instance, thirtyDaysToSeconds)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get certificate from bytes"))
		})

		It("gets duration until next renewal 10 days before expire", func() {
			tenDaysToSeconds := int64(10 * 24 * 60 * 60)
			duration, err := certificateManager.GetDurationToNextRenewal(common.TLS, instance, tenDaysToSeconds)
			Expect(err).NotTo(HaveOccurred())
			Expect(duration.Round(time.Hour)).To(Equal(time.Hour * 24 * 20)) // 10 days before cert that expires in 30 days = 20 days until next renewal
		})

		It("gets duration until next renewal 31 days before expire", func() {
			thiryOneDaysToSeconds := int64(31 * 24 * 60 * 60)
			duration, err := certificateManager.GetDurationToNextRenewal(common.TLS, instance, thiryOneDaysToSeconds)
			Expect(err).NotTo(HaveOccurred())
			Expect(duration.Round(time.Hour)).To(Equal(time.Duration(0))) // 31 days before cert that expires in 30 days = -1 days until next renewal, so should return 0
		})
	})

	Context("certificate expiring", func() {
		It("returns false if not expiring", func() {
			tenDaysToSeconds := int64(10 * 24 * 60 * 60)
			expiring, _, err := certificateManager.CertificateExpiring(common.TLS, instance, tenDaysToSeconds)
			Expect(err).NotTo(HaveOccurred())
			Expect(expiring).To(Equal(false))
		})

		It("returns true if expiring", func() {
			thirtyDaysToSeconds := int64(30 * 24 * 60 * 60)
			expiring, _, err := certificateManager.CertificateExpiring(common.TLS, instance, thirtyDaysToSeconds)
			Expect(err).NotTo(HaveOccurred())
			Expect(expiring).To(Equal(true))
		})
	})

	Context("check certificates for expire", func() {
		var (
			expiredCert []byte
		)
		BeforeEach(func() {
			expiredCert = createCert(time.Now().Add(-30 * time.Second)) // expired 30 seconds ago
		})

		It("returns error if fails to get tls signcert expiry info", func() {
			mockClient.GetReturns(errors.New("fake error"))
			_, _, err := certificateManager.CheckCertificatesForExpire(instance, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get tls signcert expiry info"))
		})

		It("returns deployed status if neither tls nor ecert signcerts are expiring", func() {
			tenDaysToSeconds := int64(10 * 24 * 60 * 60)
			status, message, err := certificateManager.CheckCertificatesForExpire(instance, tenDaysToSeconds)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(current.Deployed))
			Expect(message).To(Equal(""))
		})

		It("returns warning status if either tls or ecert signcert is expiring", func() {
			thirtyDaysToSeconds := int64(30 * 24 * 60 * 60)
			status, message, err := certificateManager.CheckCertificatesForExpire(instance, thirtyDaysToSeconds)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(current.Warning))
			Expect(message).To(ContainSubstring("tls-peer-1-signcert expires on"))
			Expect(message).To(ContainSubstring("ecert-peer-1-signcert expires on"))
		})

		It("returns error status if either tls or ecert signcert has expired", func() {
			mockClient.GetStub = func(ctx context.Context, types types.NamespacedName, obj client.Object) error {
				o := obj.(*corev1.Secret)
				switch types.Name {
				case "tls-" + instance.GetName() + "-signcert":
					o.Name = "tls-" + instance.GetName() + "-signcert"
					o.Namespace = instance.GetNamespace()
					o.Data = map[string][]byte{"cert.pem": expiredCert}
				case "ecert-" + instance.GetName() + "-signcert":
					o.Name = "ecert-" + instance.GetName() + "-signcert"
					o.Namespace = instance.GetNamespace()
					o.Data = map[string][]byte{"cert.pem": certBytes}
				}
				return nil
			}
			thirtyDaysToSeconds := int64(30 * 24 * 60 * 60)
			status, message, err := certificateManager.CheckCertificatesForExpire(instance, thirtyDaysToSeconds)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(current.Error))
			Expect(message).To(ContainSubstring("tls-peer-1-signcert has expired"))
			Expect(message).To(ContainSubstring("ecert-peer-1-signcert expires on"))
		})
	})

	Context("reenroll cert", func() {
		When("not using HSM", func() {
			It("returns error if enroller not passed", func() {
				err := certificateManager.ReenrollCert("tls", nil, instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("reenroller not passed"))
			})

			It("returns error if reenroll returns error", func() {
				mockEnroller.ReenrollReturns(nil, errors.New("fake error"))
				err := certificateManager.ReenrollCert("tls", mockEnroller, instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to renew tls certificate for instance 'peer-1': fake error"))
			})

			It("returns error if failed to update signcert secret", func() {
				mockClient.UpdateReturns(errors.New("fake error"))
				err := certificateManager.ReenrollCert("tls", mockEnroller, instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to update signcert secret for instance 'peer-1': fake error"))
			})

			It("returns error if failed to update keystore secret", func() {
				mockClient.UpdateReturnsOnCall(1, errors.New("fake error"))
				err := certificateManager.ReenrollCert("tls", mockEnroller, instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to update keystore secret for instance 'peer-1': fake error"))
			})

			It("renews certificate", func() {
				err := certificateManager.ReenrollCert("tls", mockEnroller, instance, false)
				Expect(err).NotTo(HaveOccurred())

				By("updating cert and key secret", func() {
					Expect(mockClient.UpdateCallCount()).To(Equal(2))
				})
			})
		})

		When("using HSM", func() {
			It("only updates cert secret", func() {
				err := certificateManager.ReenrollCert("tls", mockEnroller, instance, true)
				Expect(err).NotTo(HaveOccurred())
				Expect(mockClient.UpdateCallCount()).To(Equal(1))
			})
		})
	})

	Context("update signcert", func() {
		It("returns error if client fails to update secret", func() {
			mockClient.UpdateReturns(errors.New("fake error"))
			err := certificateManager.UpdateSignCert("secret-name", []byte("cert"), instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fake error"))
		})

		It("updates signcert secret", func() {
			err := certificateManager.UpdateSignCert("secret-name", []byte("cert"), instance)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("update key", func() {
		It("returns error if client fails to update secret", func() {
			mockClient.UpdateReturns(errors.New("fake error"))
			err := certificateManager.UpdateKey("secret-name", []byte("cert"), instance)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fake error"))
		})

		It("updates keystore secret", func() {
			err := certificateManager.UpdateKey("secret-name", []byte("cert"), instance)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("update secret", func() {
		It("returns error if client call for update fails", func() {
			mockClient.UpdateReturns(errors.New("fake error"))
			err := certificateManager.UpdateSecret(instance, "secret-name", map[string][]byte{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fake error"))
		})

		It("updates secret", func() {
			err := certificateManager.UpdateSecret(instance, "secret-name", map[string][]byte{})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("get signcert and key", func() {
		When("not using HSM", func() {
			It("returns an error if fails to get secret", func() {
				mockClient.GetReturns(errors.New("fake error"))
				_, _, err := certificateManager.GetSignCertAndKey("tls", instance, false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("fake error"))
			})

			It("gets signcert and key", func() {
				cert, key, err := certificateManager.GetSignCertAndKey("tls", instance, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(cert).NotTo(BeNil())
				Expect(key).NotTo(BeNil())
			})
		})

		When("using HSM", func() {
			It("gets signcert and empty key", func() {
				cert, key, err := certificateManager.GetSignCertAndKey("tls", instance, true)
				Expect(err).NotTo(HaveOccurred())
				Expect(cert).NotTo(BeNil())
				Expect(len(key)).To(Equal(0))
			})
		})
	})
})

func createCert(expireDate time.Time) []byte {
	certtemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotAfter:     expireDate,
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
