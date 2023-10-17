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

package merge_test

import (
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/merge"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Merge", func() {

	var (
		dst *Test
		src *Test

		trueVal  = true
		falseVal = false
	)

	BeforeEach(func() {
		dst = &Test{
			String:  "string",
			Int:     1,
			Bool:    false,
			BoolPtr: &falseVal,
		}

		src = &Test{}
	})

	Context("WithOverride", func() {
		Context("string", func() {
			When("src is not an empty string", func() {
				It("merges string field by overwriting dst with src", func() {
					src.String = "test"

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(dst.String).To(Equal("test"))
				})
			})

			When("src is an empty string", func() {
				// NOTE: This is the expected behavior as defined by mergo.MergeWithOverwrite.
				// If we allow empty values to be merged, then all instances of empty src attributes
				// would overwrite both non-empty and empty dst attributes, which would possible
				// overwrite dst fields we didn't want set back to an empty value.
				It("does not merge string field", func() {
					src.String = ""

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(dst.String).To(Equal("string"))
				})
			})
		})

		Context("int", func() {
			When("src is not an empty value (0)", func() {
				It("merges int field by overwriting dst with src", func() {
					src.Int = 2

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(dst.Int).To(Equal(2))
				})
			})

			When("src is an empty value (0)", func() {
				// NOTE: This is the expected behavior as defined by mergo.MergeWithOverwrite.
				// If we allow empty values to be merged, then all instances of empty src attributes
				// would overwrite both non-empty and empty dst attributes, which would possible
				// overwrite dst fields we didn't want set back to an empty value.
				It("does not merge int field", func() {
					src.Int = 0

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(dst.Int).To(Equal(1))
				})
			})
		})

		Context("bool", func() {
			When("src is not an empty value (i.e. true)", func() {
				It("merges bool field by overwriting dst with src", func() {
					src.Bool = true

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(dst.Bool).To(Equal(true))
				})
			})

			When("src is an empty value (i.e. false)", func() {
				// NOTE: This is the expected behavior as defined by mergo.MergeWithOverwrite.
				// If we allow empty values to be merged, then all instances of empty src attributes
				// would overwrite both non-empty and empty dst attributes, which would possible
				// overwrite dst fields we didn't want set back to an empty value.
				It("does not merge bool field", func() {
					dst.Bool = true
					src.Bool = false

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(dst.Bool).To(Equal(true))
				})
			})
		})

		Context("bool pointer", func() {
			When("src is a pointer to 'true'", func() {
				BeforeEach(func() {
					// Reset dst and src to avoid issues with bool pointers
					// unintentially persisting through test suite
					dst = &Test{}
					src = &Test{}
				})

				It("merges bool pointer field by overwriting non-nil dst with src", func() {
					src.BoolPtr = &trueVal

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(*dst.BoolPtr).To(Equal(true))
				})
				It("merges bool pointer field by overwriting nil dst with src", func() {
					dst.BoolPtr = nil
					src.BoolPtr = &trueVal

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(*dst.BoolPtr).To(Equal(true))
				})
			})

			When("src is a pointer to 'false'", func() {
				BeforeEach(func() {
					// Reset dst and src to avoid issues with bool pointers
					// unintentially persisting through test suite
					dst = &Test{}
					src = &Test{}
				})

				It("merges bool pointer to field by overwriting non-nil dst with src", func() {
					dst.BoolPtr = &trueVal
					src.BoolPtr = &falseVal

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(*dst.BoolPtr).To(Equal(false))
				})

				It("merges bool pointer field by ovewriting nil dst with src", func() {
					dst.BoolPtr = nil
					src.BoolPtr = &falseVal

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(*dst.BoolPtr).To(Equal(false))
				})

				It("merges bool pointer field only if pointer is not nil in src", func() {
					dst = &Test{
						BoolPtr: &trueVal,
						BoolTest: BoolTest{
							BoolPtrA: &trueVal,
						},
					}
					src = &Test{
						BoolTest: BoolTest{
							BoolPtrA: &falseVal,
						},
					}

					err := merge.WithOverwrite(dst, src)
					Expect(err).NotTo(HaveOccurred())
					Expect(*dst.BoolTest.BoolPtrA).To(Equal(false))
					Expect(*dst.BoolPtr).To(Equal(true))
					Expect(dst.BoolTest.BoolPtrB).To(BeNil())
				})
			})
		})

	})
})

type Test struct {
	String   string
	Int      int
	Bool     bool
	BoolPtr  *bool
	BoolTest BoolTest
}

type BoolTest struct {
	BoolPtrA *bool
	BoolPtrB *bool
}
