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

package merge

import (
	"reflect"

	"github.com/imdario/mergo"
)

// BoolTransformer will overwrite the behavior of merging boolean pointers such that
// a pointer to 'false' is not considered an empty value. Therefore, if the src's
// boolean pointer is not nil, it should overwrite the dst's boolean pointer value.
//
// This is required because the default behavior of mergo is to treat a pointer to 'false'
// as an empty value, which prevents boolean fields to be set from 'true' to 'false' if needed.
type BoolTransformer struct{}

func (t BoolTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	falseVal := false
	if typ == reflect.TypeOf(&falseVal) {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() && !src.IsNil() {
				dst.Set(src)
			}
			return nil
		}
	}
	return nil
}

// TODO: Can add transformers for other primitive types (i.e. int, string) if we run into
// issues setting non-empty primitive fields back to empty values - see unit tests for
// use cases.

// WithOverwrite encapsulates mergo's implementation of MergeWithOverwrite with our
// custom transformers.
func WithOverwrite(dst interface{}, src interface{}) error {
	err := mergo.MergeWithOverwrite(dst, src, mergo.WithTransformers(BoolTransformer{}))
	if err != nil {
		return err
	}

	return nil
}
