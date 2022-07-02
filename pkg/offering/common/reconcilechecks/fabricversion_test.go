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

package reconcilechecks_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common/reconcilechecks"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common/reconcilechecks/mocks"
)

var _ = Describe("fabric version", func() {
	var (
		instance *mocks.Instance
		update   *mocks.Update
		image    *mocks.Image
		fv       *mocks.Version
	)

	BeforeEach(func() {
		instance = &mocks.Instance{}
		update = &mocks.Update{}
		image = &mocks.Image{}
		fv = &mocks.Version{}
	})

	Context("create CR", func() {
		It("returns an error if fabric version is not set in spec", func() {
			_, err := reconcilechecks.FabricVersion(instance, update, image, fv)
			Expect(err).To(MatchError(ContainSubstring("fabric version is not set")))
		})

		Context("images section blank", func() {
			It("normalizes fabric version and requests a requeue", func() {
				instance.GetFabricVersionReturns("1.4.9")
				requeue, err := reconcilechecks.FabricVersion(instance, update, image, fv)
				Expect(err).NotTo(HaveOccurred())
				Expect(requeue).To(Equal(true))
				Expect(fv.NormalizeCallCount()).To(Equal(1))
				Expect(instance.SetFabricVersionCallCount()).To(Equal(1))
			})

			It("returns an error if fabric version not supported", func() {
				instance.GetFabricVersionReturns("0.0.1")
				fv.ValidateReturns(errors.New("not supported"))
				_, err := reconcilechecks.FabricVersion(instance, update, image, fv)
				Expect(err).To(MatchError(ContainSubstring("not supported")))
			})

			When("version is passed without hyphen", func() {
				BeforeEach(func() {
					instance.GetFabricVersionReturns("1.4.9")
				})

				It("finds default version for release and updates images section", func() {
					requeue, err := reconcilechecks.FabricVersion(instance, update, image, fv)
					Expect(err).NotTo(HaveOccurred())
					Expect(requeue).To(Equal(true))
					Expect(image.SetDefaultsCallCount()).To(Equal(1))
				})
			})

			When("version is passed with hyphen", func() {
				BeforeEach(func() {
					instance.GetFabricVersionReturns("1.4.9-0")
				})

				It("looks images and updates images section", func() {
					requeue, err := reconcilechecks.FabricVersion(instance, update, image, fv)
					Expect(err).NotTo(HaveOccurred())
					Expect(requeue).To(Equal(true))
					Expect(image.SetDefaultsCallCount()).To(Equal(1))
				})
			})
		})

		Context("images section passed", func() {
			BeforeEach(func() {
				instance.ImagesSetReturns(true)
			})

			When("version is not passed", func() {
				It("returns an error", func() {
					_, err := reconcilechecks.FabricVersion(instance, update, image, fv)
					Expect(err).To(MatchError(ContainSubstring("fabric version is not set")))
				})
			})

			When("version is passed", func() {
				BeforeEach(func() {
					instance.GetFabricVersionReturns("2.0.0-8")
				})

				It("persists current spec configuration", func() {
					requeue, err := reconcilechecks.FabricVersion(instance, update, image, fv)
					Expect(err).NotTo(HaveOccurred())
					Expect(requeue).To(Equal(false))
					Expect(instance.SetFabricVersionCallCount()).To(Equal(0))
					Expect(fv.NormalizeCallCount()).To(Equal(0))
					Expect(fv.ValidateCallCount()).To(Equal(0))
					Expect(image.SetDefaultsCallCount()).To(Equal(0))
				})
			})
		})
	})

	Context("update CR", func() {
		BeforeEach(func() {
			instance.GetFabricVersionReturns("2.0.1-0")
			instance.ImagesSetReturns(true)
		})

		When("images updated", func() {
			BeforeEach(func() {
				update.ImagesUpdatedReturns(true)
			})

			Context("and version updated", func() {
				BeforeEach(func() {
					update.FabricVersionUpdatedReturns(true)
				})

				It("persists current spec configuration", func() {
					requeue, err := reconcilechecks.FabricVersion(instance, update, image, fv)
					Expect(err).NotTo(HaveOccurred())
					Expect(requeue).To(Equal(false))
					Expect(fv.NormalizeCallCount()).To(Equal(0))
					Expect(instance.SetFabricVersionCallCount()).To(Equal(0))
					Expect(fv.ValidateCallCount()).To(Equal(0))
					Expect(image.SetDefaultsCallCount()).To(Equal(0))
				})
			})

			Context("and version not updated", func() {
				It("persists current spec configuration", func() {
					requeue, err := reconcilechecks.FabricVersion(instance, update, image, fv)
					Expect(err).NotTo(HaveOccurred())
					Expect(requeue).To(Equal(false))
					Expect(fv.NormalizeCallCount()).To(Equal(0))
					Expect(instance.SetFabricVersionCallCount()).To(Equal(0))
					Expect(fv.ValidateCallCount()).To(Equal(0))
					Expect(image.SetDefaultsCallCount()).To(Equal(0))
				})
			})
		})

		When("images not updated", func() {
			Context("and version updated during operator migration", func() {
				BeforeEach(func() {
					instance.GetFabricVersionReturns("unsupported")
					update.FabricVersionUpdatedReturns(true)
				})

				It("persists current spec configuration", func() {
					requeue, err := reconcilechecks.FabricVersion(instance, update, image, fv)
					Expect(err).NotTo(HaveOccurred())
					Expect(requeue).To(Equal(false))
					Expect(image.UpdateRequiredCallCount()).To(Equal(0))
					Expect(fv.NormalizeCallCount()).To(Equal(0))
					Expect(instance.SetFabricVersionCallCount()).To(Equal(0))
					Expect(fv.ValidateCallCount()).To(Equal(0))
					Expect(image.SetDefaultsCallCount()).To(Equal(0))
				})
			})

			Context("and version updated not due to operator migration", func() {
				BeforeEach(func() {
					image.UpdateRequiredReturns(true)
				})

				It("looks images and updates images section", func() {
					requeue, err := reconcilechecks.FabricVersion(instance, update, image, fv)
					Expect(err).NotTo(HaveOccurred())
					Expect(requeue).To(Equal(true))
					Expect(fv.ValidateCallCount()).To(Equal(1))
					Expect(image.SetDefaultsCallCount()).To(Equal(1))
				})
			})
		})
	})
})
