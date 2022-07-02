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
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	"github.com/IBM-Blockchain/fabric-operator/integration/helper"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("trigger CA actions", func() {
	AfterEach(func() {
		// Set flag if a test falls
		if CurrentGinkgoTestDescription().Failed {
			testFailed = true
		}
	})

	Context("renew TLS cert set to true", func() {
		var (
			expiringCA *helper.CA
			ibpca      *current.IBPCA
		)

		Context("TLS certificate", func() {
			var (
				err       error
				cert, key []byte
			)

			BeforeEach(func() {
				key, cert, err = GenSelfSignedCert(time.Hour * 48)
				Expect(err).NotTo(HaveOccurred())

				certB64 := util.BytesToBase64(cert)
				keyB64 := util.BytesToBase64(key)

				override := &v1.ServerConfig{
					TLS: v1.ServerTLSConfig{
						Enabled:  pointer.True(),
						CertFile: certB64,
						KeyFile:  keyB64,
					},
				}
				overrideBytes, err := json.Marshal(override)
				Expect(err).NotTo(HaveOccurred())

				expiringCA = CAWithOverrides(json.RawMessage(overrideBytes))
				helper.CreateCA(ibpCRClient, expiringCA.CR)

				Eventually(expiringCA.PodIsRunning).Should((Equal(true)))
			})

			When("TLS cert renew action is set to false", func() {
				BeforeEach(func() {
					patch := func(o client.Object) {
						ibpca = o.(*current.IBPCA)
						ibpca.Spec.Action.Renew.TLSCert = true
					}

					err := integration.ResilientPatch(ibpCRClient, expiringCA.Name, namespace, IBPCAS, 3, &current.IBPCA{}, patch)
					Expect(err).NotTo(HaveOccurred())

					Eventually(expiringCA.PodIsRunning).Should((Equal(true)))
				})

				It("renews", func() {
					By("backing up old crypto", func() {
						Eventually(func() bool {
							backup, err := GetBackup("tls", expiringCA.CR.Name)
							if err != nil {
								return false
							}

							if len(backup.List) > 0 {
								return backup.List[len(backup.List)-1].SignCerts == base64.StdEncoding.EncodeToString(cert)
							}

							return false
						}).Should(Equal(true))
					})

					By("updating crypto secret with new TLS Cert", func() {
						Eventually(func() bool {
							crypto, err := kclient.CoreV1().Secrets(namespace).
								Get(context.TODO(), fmt.Sprintf("%s-ca-crypto", expiringCA.CR.Name), metav1.GetOptions{})
							Expect(err).NotTo(HaveOccurred())

							return bytes.Equal(cert, crypto.Data["tls-cert.pem"])
						}).Should(Equal(false))
					})

					By("updating operations cert to match new TLS cert", func() {
						crypto, err := kclient.CoreV1().Secrets(namespace).
							Get(context.TODO(), fmt.Sprintf("%s-ca-crypto", expiringCA.CR.Name), metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						Expect(bytes.Equal(
							crypto.Data["operations-cert.pem"],
							crypto.Data["tls-cert.pem"],
						)).To(Equal(true))
					})

					By("refreshing the TLS certificate with expiration value of plus 10 years", func() {
						crypto, err := kclient.CoreV1().Secrets(namespace).
							Get(context.TODO(), fmt.Sprintf("%s-ca-crypto", expiringCA.CR.Name), metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						newTLSCert := crypto.Data["tls-cert.pem"]
						newCert, err := util.GetCertificateFromPEMBytes(newTLSCert)
						Expect(err).NotTo(HaveOccurred())
						Expect(newCert.NotAfter.Year()).To(Equal(time.Now().Add(time.Hour * 87600).Year()))
					})

					By("updating crypto secret with new TLS Key", func() {
						Eventually(func() bool {
							crypto, err := kclient.CoreV1().Secrets(namespace).
								Get(context.TODO(), fmt.Sprintf("%s-ca-crypto", expiringCA.CR.Name), metav1.GetOptions{})
							Expect(err).NotTo(HaveOccurred())

							return bytes.Equal(key, crypto.Data["tls-key.pem"])
						}).Should(Equal(false))
					})

					By("updating operations key to match new TLS Key", func() {
						crypto, err := kclient.CoreV1().Secrets(namespace).
							Get(context.TODO(), fmt.Sprintf("%s-ca-crypto", expiringCA.CR.Name), metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						Expect(bytes.Equal(
							crypto.Data["operations-key.pem"],
							crypto.Data["tls-key.pem"],
						)).To(Equal(true))
					})

					By("updating connection profile with new TLS cert", func() {
						Eventually(func() bool {
							cm, err := kclient.CoreV1().
								ConfigMaps(namespace).
								Get(context.TODO(),
									fmt.Sprintf("%s-connection-profile", expiringCA.CR.Name),
									metav1.GetOptions{},
								)
							Expect(err).NotTo(HaveOccurred())

							profileBytes := cm.BinaryData["profile.json"]
							connectionProfile := &current.CAConnectionProfile{}
							err = json.Unmarshal(profileBytes, connectionProfile)
							Expect(err).NotTo(HaveOccurred())

							crypto, err := kclient.CoreV1().Secrets(namespace).
								Get(context.TODO(), fmt.Sprintf("%s-ca-crypto", expiringCA.CR.Name), metav1.GetOptions{})
							Expect(err).NotTo(HaveOccurred())

							return bytes.Equal([]byte(connectionProfile.TLS.Cert), crypto.Data["tls-key.pem"])
						}).Should(Equal(false))
					})

					By("setting restart flag back to false after restart", func() {
						Eventually(func() bool {
							result := ibpCRClient.Get().Namespace(namespace).Resource(IBPCAS).Name(expiringCA.Name).Do(context.TODO())
							ibpca := &current.IBPCA{}
							result.Into(ibpca)

							return ibpca.Spec.Action.Renew.TLSCert
						}).Should(Equal(false))
					})
				})
			})
		})
	})

	Context("restart", func() {
		var (
			podName string
			ca      *current.IBPCA
		)

		BeforeEach(func() {
			Eventually(func() int {
				return len(org1ca.GetPods())
			}).Should(Equal(1))

			podName = org1ca.GetPods()[0].Name

			result := ibpCRClient.Get().Namespace(namespace).Resource(IBPCAS).Name(org1ca.Name).Do(context.TODO())
			Expect(result.Error()).NotTo(HaveOccurred())

			ca = &current.IBPCA{}
			result.Into(ca)
		})

		When("spec has restart flag set to true", func() {
			BeforeEach(func() {
				ca.Spec.Action.Restart = true
			})

			It("performs restart action", func() {
				bytes, err := json.Marshal(ca)
				Expect(err).NotTo(HaveOccurred())

				result := ibpCRClient.Put().Namespace(namespace).Resource(IBPCAS).Name(org1ca.Name).Body(bytes).Do(context.TODO())
				Expect(result.Error()).NotTo(HaveOccurred())

				Eventually(org1ca.PodIsRunning).Should((Equal(true)))

				By("restarting ca pod", func() {
					Eventually(func() bool {
						pods := org1ca.GetPods()
						if len(pods) == 0 {
							return false
						}

						newPodName := pods[0].Name
						if newPodName != podName {
							return true
						}

						return false
					}).Should(Equal(true))
				})

				By("setting restart flag back to false after restart", func() {
					Eventually(func() bool {
						result := ibpCRClient.Get().Namespace(namespace).Resource(IBPCAS).Name(org1ca.Name).Do(context.TODO())
						ca := &current.IBPCA{}
						result.Into(ca)

						return ca.Spec.Action.Restart
					}).Should(Equal(false))
				})
			})
		})
	})

})

func CAWithOverrides(rawMessage json.RawMessage) *helper.CA {
	cr := &current.IBPCA{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "org2ca",
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
			Zone:   "select",
			Region: "select",
			Domain: domain,
			ConfigOverride: &current.ConfigOverride{
				CA: &runtime.RawExtension{Raw: rawMessage},
			},
			FabricVersion: integration.FabricCAVersion,
		},
	}

	return &helper.CA{
		Domain:     domain,
		Name:       cr.Name,
		Namespace:  namespace,
		WorkingDir: wd,
		CR:         cr,
		CRClient:   ibpCRClient,
		KClient:    kclient,
		NativeResourcePoller: integration.NativeResourcePoller{
			Name:      cr.Name,
			Namespace: namespace,
			Client:    kclient,
		},
	}
}

// Generate TLS cert that is expires in the x days
func GenSelfSignedCert(expiresIn time.Duration) ([]byte, []byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate key")
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate serial number")
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(expiresIn)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Issuer: pkix.Name{
			Country:            []string{"US"},
			Province:           []string{"North Carolina"},
			Locality:           []string{"Durham"},
			Organization:       []string{"IBM"},
			OrganizationalUnit: []string{"Blockchain"},
		},
		Subject: pkix.Name{
			Country:            []string{"US"},
			Province:           []string{"North Carolina"},
			Locality:           []string{"Durham"},
			Organization:       []string{"IBM"},
			OrganizationalUnit: []string{"Blockchain"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create certificate")
	}

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal key")
	}

	certPEM := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}
	keyPEM := &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}

	certBytes := pem.EncodeToMemory(certPEM)
	keyBytes = pem.EncodeToMemory(keyPEM)

	return keyBytes, certBytes, nil
}

func GetBackup(certType, name string) (*common.Backup, error) {
	backupSecret, err := kclient.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf("%s-crypto-backup", name), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	backup := &common.Backup{}
	key := fmt.Sprintf("%s-backup.json", certType)
	err = json.Unmarshal(backupSecret.Data[key], backup)
	if err != nil {
		return nil, err
	}

	return backup, nil
}
