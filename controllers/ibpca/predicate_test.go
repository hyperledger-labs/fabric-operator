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

package ibpca

import (
	"context"
	"fmt"
	"sync"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/controllers/mocks"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	yaml "sigs.k8s.io/yaml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("predicates", func() {
	var (
		reconciler   *ReconcileIBPCA
		client       *mocks.Client
		oldCA, newCA *current.IBPCA
	)

	Context("create func predicate", func() {
		var (
			e event.CreateEvent
		)

		BeforeEach(func() {
			oldCA = &current.IBPCA{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ca1",
				},
				Spec: current.IBPCASpec{},
			}

			newCA = &current.IBPCA{
				ObjectMeta: metav1.ObjectMeta{
					Name: oldCA.GetName(),
				},
				Status: current.IBPCAStatus{
					CRStatus: current.CRStatus{
						Type: current.Deployed,
					},
				},
			}

			e = event.CreateEvent{
				Object: newCA,
			}

			client = &mocks.Client{
				GetStub: func(ctx context.Context, types types.NamespacedName, obj k8sclient.Object) error {
					switch obj.(type) {
					case *corev1.ConfigMap:
						cm := obj.(*corev1.ConfigMap)
						bytes, err := yaml.Marshal(oldCA.Spec)
						Expect(err).NotTo((HaveOccurred()))
						cm.BinaryData = map[string][]byte{
							"spec": bytes,
						}
					}

					return nil
				},
				ListStub: func(ctx context.Context, obj k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
					switch obj.(type) {
					case *current.IBPCAList:
						caList := obj.(*current.IBPCAList)
						caList.Items = []current.IBPCA{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-ca1"}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-ca2"}},
							{ObjectMeta: metav1.ObjectMeta{Name: "test-ca2"}},
						}
					case *current.IBPPeerList:
						peerList := obj.(*current.IBPPeerList)
						peerList.Items = []current.IBPPeer{
							{ObjectMeta: metav1.ObjectMeta{Name: "test-peer"}},
						}
					}
					return nil
				},
			}

			reconciler = &ReconcileIBPCA{
				update: map[string][]Update{},
				client: client,
				mutex:  &sync.Mutex{},
			}
		})

		It("sets update flags to false if instance has status type and a create event is detected but no spec changes detected", func() {
			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(true))

			Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{
				specUpdated:           false,
				caOverridesUpdated:    false,
				tlscaOverridesUpdated: false,
			}))
		})

		It("sets update flags to true if instance has status type and a create event is detected and spec changes detected", func() {
			jm, err := util.ConvertToJsonMessage(&v1.ServerConfig{})
			Expect(err).NotTo(HaveOccurred())

			spec := current.IBPCASpec{
				ImagePullSecrets: []string{"pullsecret1"},
				ConfigOverride: &current.ConfigOverride{
					CA:    &runtime.RawExtension{Raw: *jm},
					TLSCA: &runtime.RawExtension{Raw: *jm},
				},
			}
			binaryData, err := yaml.Marshal(spec)
			Expect(err).NotTo(HaveOccurred())

			client.GetStub = func(ctx context.Context, types types.NamespacedName, obj k8sclient.Object) error {
				switch obj.(type) {
				case *corev1.ConfigMap:
					o := obj.(*corev1.ConfigMap)
					o.BinaryData = map[string][]byte{
						"spec": binaryData,
					}
				}
				return nil
			}
			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(true))

			Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{
				specUpdated:           true,
				caOverridesUpdated:    true,
				tlscaOverridesUpdated: true,
			}))
		})

		It("does not trigger update if instance does not have status type and a create event is detected", func() {
			newCA.Status.Type = ""

			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(true))

			Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{}))
		})

		It("returns false if new instance's name already exists for another custom resource", func() {
			newCA.Status.Type = ""
			newCA.Name = "test-peer"

			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(false))
			Expect(newCA.Status.Type).To(Equal(current.Error))
		})

		It("returns false if new instance's name already exists for another IBPCA custom resource", func() {
			newCA.Status.Type = ""
			newCA.Name = "test-ca2"

			create := reconciler.CreateFunc(e)
			Expect(create).To(Equal(false))
			Expect(newCA.Status.Type).To(Equal(current.Error))
		})

		Context("fabric version", func() {
			It("returns no updates when fabric version is not changed", func() {
				reconciler.CreateFunc(e)
				Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{}))
			})

			When("fabric version updated", func() {
				BeforeEach(func() {
					newCA.Spec.FabricVersion = "2.2.1-1"
				})

				It("sets fabric version to true on version change", func() {
					reconciler.CreateFunc(e)
					Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{
						specUpdated:          true,
						fabricVersionUpdated: true,
					}))
				})
			})
		})

		Context("images", func() {
			It("returns no updates when images are not changed", func() {
				reconciler.CreateFunc(e)
				Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{}))
			})

			When("images updated", func() {
				BeforeEach(func() {
					newCA.Spec.Images = &current.CAImages{
						CAImage: "caimage2",
					}
				})

				It("sets imagesUpdated to true on image nil to non-nil update", func() {
					reconciler.CreateFunc(e)
					Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{
						specUpdated:   true,
						imagesUpdated: true,
					}))
				})

				It("sets imagesUpdated to true on image changes", func() {
					oldCA.Spec.Images = &current.CAImages{
						CAImage: "caimage1",
					}

					reconciler.CreateFunc(e)
					Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{
						specUpdated:   true,
						imagesUpdated: true,
					}))
				})
			})
		})
	})

	Context("update func", func() {
		var (
			e event.UpdateEvent
		)

		BeforeEach(func() {
			oldCA = &current.IBPCA{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ca1",
				},
			}

			newCA = &current.IBPCA{
				ObjectMeta: metav1.ObjectMeta{
					Name: oldCA.Name,
				},
			}

			e = event.UpdateEvent{
				ObjectOld: oldCA,
				ObjectNew: newCA,
			}

			client = &mocks.Client{}
			reconciler = &ReconcileIBPCA{
				update: map[string][]Update{},
				client: client,
				mutex:  &sync.Mutex{},
			}
		})

		It("returns false if zone being update", func() {
			oldCA.Spec.Zone = "old_zone"
			newCA.Spec.Zone = "new_zone"
			Expect(reconciler.UpdateFunc(e)).To(Equal(false))
		})

		It("returns false if region being update", func() {
			oldCA.Spec.Region = "old_region"
			newCA.Spec.Region = "new_region"
			Expect(reconciler.UpdateFunc(e)).To(Equal(false))
		})

		It("returns false old and new objects are equal", func() {
			Expect(reconciler.UpdateFunc(e)).To(Equal(false))
		})

		It("returns true if spec updated", func() {
			newCA.Spec.ImagePullSecrets = []string{"secret1"}
			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(newCA).SpecUpdated()).To(Equal(true))
		})

		It("returns true if ca overrides created for the first time", func() {
			newCA.Spec.ConfigOverride = &current.ConfigOverride{}
			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(newCA).CAOverridesUpdated()).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(newCA).TLSCAOverridesUpdated()).To(Equal(true))
		})

		It("returns true if enrollment ca overrides updated", func() {
			oldCA.Spec.ConfigOverride = &current.ConfigOverride{}
			newCA.Spec.ConfigOverride = &current.ConfigOverride{
				CA: &runtime.RawExtension{},
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(newCA).CAOverridesUpdated()).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(newCA).TLSCAOverridesUpdated()).To(Equal(false))
		})

		Context("ca crypto", func() {
			var (
				oldSecret *corev1.Secret
				newSecret *corev1.Secret
			)

			BeforeEach(func() {
				oldSecret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-ca-crypto", newCA.Name),
						OwnerReferences: []metav1.OwnerReference{
							{
								Name: newCA.Name,
								Kind: "IBPCA",
							},
						},
					},
				}
				newSecret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-ca-crypto", newCA.Name),
						OwnerReferences: []metav1.OwnerReference{
							{
								Name: newCA.Name,
								Kind: "IBPCA",
							},
						},
					},
				}
				e = event.UpdateEvent{
					ObjectOld: oldSecret,
					ObjectNew: newSecret,
				}
			})

			It("returns false if secret data not changed between old and new secret", func() {
				oldSecret.Data = map[string][]byte{
					"tls-cert.pem": []byte("cert"),
				}
				newSecret.Data = map[string][]byte{
					"tls-cert.pem": []byte("cert"),
				}
				Expect(reconciler.UpdateFunc(e)).To(Equal(false))
			})

			It("returns true if secret data changed between old and new secret", func() {
				oldSecret.Data = map[string][]byte{
					"tls-cert.pem": []byte("cert"),
				}
				newSecret.Data = map[string][]byte{
					"tls-cert.pem": []byte("newcert"),
				}
				Expect(reconciler.UpdateFunc(e)).To(Equal(true))
				Expect(reconciler.GetUpdateStatus(newCA).CACryptoUpdated()).To(Equal(true))
			})

			It("returns false if anything other than secret data changed between old and new secret", func() {
				oldSecret.APIVersion = "v1"
				newSecret.APIVersion = "v2"
				Expect(reconciler.UpdateFunc(e)).To(Equal(false))
			})
		})

		It("returns true if tls ca overrides updated", func() {
			caConfig := &v1.ServerConfig{
				CAConfig: v1.CAConfig{
					CA: v1.CAInfo{
						Name: "ca",
					},
				},
			}

			caJson, err := util.ConvertToJsonMessage(caConfig)
			Expect(err).NotTo(HaveOccurred())

			oldCA.Spec.ConfigOverride = &current.ConfigOverride{}
			newCA.Spec.ConfigOverride = &current.ConfigOverride{
				TLSCA: &runtime.RawExtension{Raw: *caJson},
			}

			Expect(reconciler.UpdateFunc(e)).To(Equal(true))
			Expect(reconciler.GetUpdateStatus(newCA).CAOverridesUpdated()).To(Equal(false))
			Expect(reconciler.GetUpdateStatus(newCA).TLSCAOverridesUpdated()).To(Equal(true))

		})

		Context("remove element", func() {
			BeforeEach(func() {
				reconciler.PushUpdate(newCA.Name, Update{
					caOverridesUpdated: true,
				})

				reconciler.PushUpdate(newCA.Name, Update{
					tlscaOverridesUpdated: true,
				})

				Expect(reconciler.GetUpdateStatus(newCA).CAOverridesUpdated()).To(Equal(true))
				Expect(reconciler.GetUpdateStatusAtElement(newCA, 1).TLSCAOverridesUpdated()).To(Equal(true))
			})

			It("removes top element", func() {
				reconciler.PopUpdate(newCA.Name)
				Expect(reconciler.GetUpdateStatus(newCA).CAOverridesUpdated()).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(newCA).TLSCAOverridesUpdated()).To(Equal(true))
			})

			It("removing more elements than in slice should not panic", func() {
				reconciler.PopUpdate(newCA.Name)
				reconciler.PopUpdate(newCA.Name)
				reconciler.PopUpdate(newCA.Name)
				Expect(reconciler.GetUpdateStatus(newCA).SpecUpdated()).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(newCA).CAOverridesUpdated()).To(Equal(false))
				Expect(reconciler.GetUpdateStatus(newCA).TLSCAOverridesUpdated()).To(Equal(false))
			})
		})

		Context("push update", func() {
			It("pushes update only if missing for certificate update", func() {
				reconciler.PushUpdate(newCA.Name, Update{specUpdated: true})
				Expect(len(reconciler.update[newCA.Name])).To(Equal(1))
				reconciler.PushUpdate(newCA.Name, Update{caOverridesUpdated: true})
				Expect(len(reconciler.update[newCA.Name])).To(Equal(2))
				reconciler.PushUpdate(newCA.Name, Update{tlscaOverridesUpdated: true})
				Expect(len(reconciler.update[newCA.Name])).To(Equal(3))
				reconciler.PushUpdate(newCA.Name, Update{tlscaOverridesUpdated: true})
				Expect(len(reconciler.update[newCA.Name])).To(Equal(3))
				reconciler.PushUpdate(newCA.Name, Update{restartNeeded: true, specUpdated: true})
				Expect(len(reconciler.update[newCA.Name])).To(Equal(4))
			})
		})

		Context("fabric version", func() {
			It("returns no updates when fabric version is not changed", func() {
				reconciler.UpdateFunc(e)
				Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{}))
			})

			When("fabric version updated", func() {
				BeforeEach(func() {
					newCA.Spec.FabricVersion = "2.2.1-1"
				})

				It("sets fabric version to true on version change", func() {
					reconciler.UpdateFunc(e)
					Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{
						specUpdated:          true,
						fabricVersionUpdated: true,
					}))
				})
			})
		})

		Context("images", func() {
			It("returns no updates when images are not changed", func() {
				reconciler.UpdateFunc(e)
				Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{}))
			})

			When("images updated", func() {
				BeforeEach(func() {
					newCA.Spec.Images = &current.CAImages{
						CAImage: "caimage2",
					}

					oldCA.Spec.Images = &current.CAImages{
						CAImage: "caimage1",
					}
				})

				It("sets imagesUpdated to true on image changes", func() {
					reconciler.UpdateFunc(e)
					Expect(reconciler.GetUpdateStatus(newCA)).To(Equal(&Update{
						specUpdated:   true,
						imagesUpdated: true,
					}))
				})
			})
		})
	})
})
