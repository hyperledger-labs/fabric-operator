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

package common

import (
	"fmt"
	"strings"
	"time"
)

type Duration struct {
	time.Duration `json:",inline"`
}

// Decode is custom decoder for `envconfig` library, this
// method is used to handle reading in environment variables
// and converting them into the type that is expected in
// our structs
func (d *Duration) Decode(value string) error {
	dur, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	d.Duration = dur
	return nil
}

// Unmarshal is custom unmarshaler for github.com/kelseyhightower/envconfig
func (d *Duration) Unmarshal(s string) (err error) {
	if s == "" {
		return
	}

	d.Duration, err = time.ParseDuration(strings.Trim(string(s), `"`))
	return
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	if b == nil {
		return
	}
	if string(b) == "null" {
		return
	}
	d.Duration, err = time.ParseDuration(strings.Trim(string(b), `"`))
	return
}

func (d *Duration) Get() time.Duration {
	return d.Duration
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

func ParseDuration(d string) (Duration, error) {
	duration, err := time.ParseDuration(strings.Trim(string(d), `"`))
	if err != nil {
		return Duration{}, err
	}

	return Duration{duration}, nil
}

func MustParseDuration(d string) Duration {
	duration, err := time.ParseDuration(strings.Trim(string(d), `"`))
	if err != nil {
		return Duration{}
	}

	return Duration{duration}
}

func ConvertTimeDuration(d time.Duration) Duration {
	return Duration{d}
}

type BCCSP struct {
	Default string      `json:"default,omitempty"`
	SW      *SwOpts     `json:"SW,omitempty"`
	PKCS11  *PKCS11Opts `json:"PKCS11,omitempty"`
}

// SwOpts contains options for the SWFactory
type SwOpts struct {
	Security     int              `json:"security,omitempty"`
	Hash         string           `json:"hash,omitempty"`
	FileKeyStore FileKeyStoreOpts `json:"filekeystore,omitempty"`
}

type PKCS11Opts struct {
	Security       int               `json:"security,omitempty"`
	Hash           string            `json:"hash,omitempty"`
	Library        string            `json:"library,omitempty"`
	Label          string            `json:"label,omitempty"`
	Pin            string            `json:"pin,omitempty"`
	Ephemeral      bool              `json:"tempkeys,omitempty"`
	SoftwareVerify bool              `json:"softwareVerify,omitempty"`
	Immutable      bool              `json:"immutable,omitempty"`
	FileKeyStore   *FileKeyStoreOpts `json:"filekeystore,omitempty"`
}

type FileKeyStoreOpts struct {
	KeyStorePath string `json:"keystore,omitempty"`
}
