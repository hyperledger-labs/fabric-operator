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

package config_test

import (
	"os"

	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/ca/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util/pointer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	keyFile  = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBdFJBUDlMemUyZEc1cm1rbmcvdVVtREFZU0VwUElqRFdUUDhqUjMxcUJ5Yjc3YWUrCnk3UTRvRnZod1lDVUhsUWVTWjFKeTdUUHpEcitoUk5hdDJYNGdGYUpGYmVFbC9DSHJ3Rk1mNzNzQStWV1pHdnkKdXhtbjB2bEdYMW5zSEo5aUdIUS9qR2FvV1FJYzlVbnpHWi8yWStlZkpxOWd3cDBNemFzWWZkdXordXVBNlp4VAp5TTdDOWFlWmxYL2ZMYmVkSXVXTzVzaXhPSlZQeUVpcWpkd0RiY1AxYy9mRCtSMm1DbmM3VGovSnVLK1poTGxPCnhGcVlFRmtROHBmSi9LY1pabVF1QURZVFh6RGp6OENxcTRTRU5ySzI0b2hQQkN2SGgyanplWjhGdGR4MmpSSFQKaXdCZWZEYWlSWVBSOUM4enk4K1Z2Wmt6S0hQV3N5aENiNUMrN1FJREFRQUJBb0lCQUZROGhzL2IxdW9Mc3BFOApCdEJXaVVsTWh0K0xBc25yWXFncnd5UU5hdmlzNEdRdXVJdFk2MGRmdCtZb2hjQ2ViZ0RkbG1tWlUxdTJ6cGJtCjdEdUt5MVFaN21rV0dpLytEWUlUM3AxSHBMZ2pTRkFzRUorUFRnN1BQamc2UTZrRlZjUCt3Vm4yb0xmWVRkU28KZE5zbEdxSmNMaVQzVHRMNzhlcjFnTTE5RzN6T3J1ZndrSGJSYU1BRmtvZ1ExUlZLSWpnVGUvbmpIMHFHNW9JagoxNEJLeFFKTUZFTG1pQk50NUx5OVMxWWdxTDRjbmNtUDN5L1QyNEdodVhNckx0eTVOeVhnS0dFZ1pUTDMzZzZvCnYreDFFMFRURWRjMVQvWVBGWkdBSXhHdWRKNWZZZ2JtWU9LZ09mUHZFOE9TbEV6OW56aHNnckVZYjdQVThpZDUKTHFycVJRRUNnWUVBNjIyT3RIUmMxaVY1ZXQxdHQydTVTTTlTS2h2b0lPT3d2Q3NnTEI5dDJzNEhRUlRYN0RXcAo0VDNpUC9leEl5OXI3bTIxNFo5MEgzZlpVNElSUkdHSUxKUVMrYzRQNVA4cHJFTDcyd1dIWlpQTTM3QlZTQ1U3CkxOTXl4TkRjeVdjSUJIVFh4NUY2eXhLNVFXWTg5MVB0eDlDamJFSEcrNVJVdDA4UVlMWDlUQTBDZ1lFQXhPSmYKcXFjeThMOVZyYUFVZG9lbGdIU0NGSkJRR3hMRFNSQlJSTkRIOUJhaWlZOCtwZzd2TExTRXFMRFpsbkZPbFkrQQpiRENEQ0RtdHhwRXViY0x6b3FnOXhlQTZ0eXZZWkNWalY5dXVzNVh1Wmk1VDBBUHhCdm56OHNNa3dRY3RQWkRQCk8zQTN4WllkZzJBRmFrV1BmT1FFbjVaK3F4TU13SG9VZ1ZwQkptRUNnWUJ2Q2FjcTJVOEgrWGpJU0ROOU5TT1kKZ1ovaEdIUnRQcmFXcVVodFJ3MkxDMjFFZHM0NExEOUphdVNSQXdQYThuelhZWXROTk9XU0NmYkllaW9tdEZHRApwUHNtTXRnd1MyQ2VUS0Y0OWF5Y2JnOU0yVi8vdlAraDdxS2RUVjAwNkpGUmVNSms3K3FZYU9aVFFDTTFDN0swCmNXVUNwQ3R6Y014Y0FNQmF2THNRNlFLQmdHbXJMYmxEdjUxaXM3TmFKV0Z3Y0MwL1dzbDZvdVBFOERiNG9RV1UKSUowcXdOV2ZvZm95TGNBS3F1QjIrbkU2SXZrMmFiQ25ZTXc3V0w4b0VJa3NodUtYOVgrTVZ6Y1VPekdVdDNyaQpGeU9mcHJJRXowcm5zcWNSNUJJNUZqTGJqVFpyMEMyUWp2NW5FVFAvaHlpQWFRQ1l5THAyWlVtZ0Vjb0VPNWtwClBhcEJBb0dBZVV0WjE0SVp2cVorQnAxR1VqSG9PR0pQVnlJdzhSRUFETjRhZXRJTUlQRWFVaDdjZUtWdVN6VXMKci9WczA1Zjg0cFBVaStuUTUzaGo2ZFhhYTd1UE1aMFBnNFY4cS9UdzJMZ3BWWndVd0ltZUQrcXNsbldha3VWMQpMSnp3SkhOa3pOWE1OMmJWREFZTndSamNRSmhtbzF0V2xHYlpRQjNoSkEwR2thWGZPa2c9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg=="
	certFile = "../../../../testdata/tls/tls.crt"
)

var _ = Describe("config", func() {
	const (
		homeDir = "configtest"
	)

	var cfg *config.Config

	BeforeEach(func() {
		cfg = &config.Config{
			ServerConfig: &v1.ServerConfig{
				TLS: v1.ServerTLSConfig{
					Enabled:  pointer.True(),
					CertFile: certFile,
					KeyFile:  keyFile,
					ClientAuth: v1.ClientAuth{
						CertFiles: []string{"../../../../testdata/tls/tls.crt"},
					},
				},
			},
			HomeDir: homeDir,
		}
	})

	BeforeEach(func() {
		os.Mkdir(homeDir, 0777)
	})

	AfterEach(func() {
		err := os.RemoveAll(homeDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("get input type", func() {
		It("returns base64 type if filepath passed", func() {
			inputType := config.GetInputType(keyFile)
			Expect(inputType).To(Equal(config.Base64))
		})

		It("returns cert file type if filepath passed", func() {
			inputType := config.GetInputType(certFile)
			Expect(inputType).To(Equal(config.File))
		})

		It("returns unkown type if neither base64 or file passed in", func() {
			inputType := config.GetInputType("foo")
			Expect(inputType).To(Equal(config.Bccsp))
		})
	})

	Context("handle configuration", func() {
		var crypto map[string][]byte

		BeforeEach(func() {
			crypto = map[string][]byte{}
		})

		It("will convert cert to bytes and store in map", func() {
			err := cfg.HandleCertInput(certFile, "certname", crypto)
			Expect(err).NotTo(HaveOccurred())
			Expect(crypto).NotTo(BeNil())

			data, keyExists := crypto["certname"]
			Expect(keyExists).To(Equal(true))
			Expect(data).NotTo(BeNil())
		})

		It("will convert key to bytes and store in map", func() {
			err := cfg.HandleKeyInput(keyFile, "keyname", crypto)
			Expect(err).NotTo(HaveOccurred())
			Expect(crypto).NotTo(BeNil())

			data, keyExists := crypto["keyname"]
			Expect(keyExists).To(Equal(true))
			Expect(data).NotTo(BeNil())
		})
	})
})
