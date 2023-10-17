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

package version_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/IBM-Blockchain/fabric-operator/version"
)

var _ = Describe("Version", func() {

	Context("get fabric version from", func() {
		It("returns version from image tag", func() {
			fabricVersion := version.GetFabricVersionFrom("1.4.3-12345-amd64")
			Expect(fabricVersion).To(Equal("1.4.3"))
		})

		It("returns empty string if image tag is a sha256 digest", func() {
			fabricVersion := version.GetFabricVersionFrom("sha256:2037c532f6c823667baed5af248c01c941b2344c2a939e451b81ea0e03938243")
			Expect(fabricVersion).To(Equal(""))
		})
	})

	Context("get old fabric version from", func() {
		It("returns version from image tag", func() {
			fabricVersion := version.GetOldFabricVersionFrom("1.4.3-12345-amd64")
			Expect(fabricVersion).To(Equal("1.4.3"))
		})

		It("returns 'unsupported' if an old image tag with a version not found in the lookup table", func() {
			fabricVersion := version.GetOldFabricVersionFrom("1.4.1-12345-amd64")
			Expect(fabricVersion).To(Equal("unsupported"))
		})
	})

	Context("is migrated fabric version", func() {
		It("return true if version is found in old fabric versions lookup map", func() {
			migrated := version.IsMigratedFabricVersion("1.4.6")
			Expect(migrated).To(Equal(true))
		})

		It("returns true if version is 'unsupported''", func() {
			migrated := version.IsMigratedFabricVersion("unsupported")
			Expect(migrated).To(Equal(true))
		})

		It("returns false if version not found in old fabric versions lookup map", func() {
			migrated := version.IsMigratedFabricVersion("1.4.9-4")
			Expect(migrated).To(Equal(false))
		})
	})

	Context("version string", func() {
		var (
			V147   version.String
			V147_2 version.String
			V225_5 version.String
			V241_1 version.String
		)

		BeforeEach(func() {
			V147 = version.String("1.4.7")
			V147_2 = version.String("1.4.7-2")
			V225_5 = version.String("2.2.5-5")
			V241_1 = version.String("2.4.1-1")
		})

		Context("equal", func() {
			It("returns 1.4.7 == 1.4.7 as true", func() {
				equal := V147.Equal("1.4.7")
				Expect(equal).To(Equal(true))
			})

			It("returns 1.4.7 == 1.4.6 as false", func() {
				equal := V147.Equal("1.4.6")
				Expect(equal).To(Equal(false))
			})

			It("returns 1.4.7 == 1.4.7-1 as false", func() {
				equal := V147.Equal("1.4.7-1")
				Expect(equal).To(Equal(false))
			})
		})

		Context("greater than", func() {
			It("returns 1.4.7 > 1.4.7 as false", func() {
				equal := V147.GreaterThan("1.4.7")
				Expect(equal).To(Equal(false))
			})

			It("returns 1.4.7 > 1.4.6 as true", func() {
				equal := V147.GreaterThan("1.4.6")
				Expect(equal).To(Equal(true))
			})

			It("returns 1.4.7 > 1.4.7-1 as false", func() {
				equal := V147.GreaterThan("1.4.7-1")
				Expect(equal).To(Equal(false))
			})

			It("returns 1.4.7-2 > 1.4.7-1 as true", func() {
				equal := V147_2.GreaterThan("1.4.7-1")
				Expect(equal).To(Equal(true))
			})

			It("returns 2.2.5-5 > 2.4.1-1 as false", func() {
				equal := V225_5.GreaterThan("2.4.1-1")
				Expect(equal).To(Equal(false))
			})

			It("returns 2.4.1-1 > 2.2.5-5 as true", func() {
				equal := V241_1.GreaterThan("2.2.5-5")
				Expect(equal).To(Equal(true))
			})
		})

		Context("less than", func() {
			It("returns 1.4.7 < 1.4.7 as false", func() {
				equal := V147.LessThan("1.4.7")
				Expect(equal).To(Equal(false))
			})

			It("returns 1.4.7 < 1.4.6 as false", func() {
				equal := V147.LessThan("1.4.6")
				Expect(equal).To(Equal(false))
			})

			It("returns 1.4.7 < 1.4.7-1 as true", func() {
				equal := V147.LessThan("1.4.7-1")
				Expect(equal).To(Equal(true))
			})

			It("returns 1.4.7-2 < 1.4.7-1 as false", func() {
				equal := V147_2.LessThan("1.4.7-1")
				Expect(equal).To(Equal(false))
			})

			It("returns 2.4.1-1 < 2.2.5-5 as false", func() {
				equal := V241_1.LessThan("2.4.1-1")
				Expect(equal).To(Equal(false))
			})

			It("returns 2.2.5-5 < 2.4.1-1 as true", func() {
				equal := V225_5.LessThan("2.4.1-1")
				Expect(equal).To(Equal(true))
			})
		})
	})
})
