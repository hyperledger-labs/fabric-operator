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

package reenroller_test

import (
	"fmt"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate/reenroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate/reenroller/mocks"
	"github.com/hyperledger/fabric-ca/lib"
	"github.com/hyperledger/fabric-ca/lib/client/credential"
	fabricx509 "github.com/hyperledger/fabric-ca/lib/client/credential/x509"
	"github.com/hyperledger/fabric-ca/lib/tls"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

const (
	homeDir = "test-reenroller-dir"
	testkey = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ3hRUXdSVFFpVUcwREo1UHoKQTJSclhIUEtCelkxMkxRa0MvbVlveWo1bEhDaFJBTkNBQVN5bE1YLzFqdDlmUGt1RTZ0anpvSTlQbGt4LzZuVQpCMHIvMU56TTdrYnBjUk8zQ3RIeXQ2TXlQR21FOUZUN29pYXphU3J1TW9JTDM0VGdBdUpIOU9ZWQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg=="
)

var _ = Describe("Reenroller", func() {
	BeforeEach(func() {
		mockIdentity = &mocks.Identity{}

		config = &current.Enrollment{
			CAHost:       serverUrlObj.Hostname(),
			CAPort:       serverUrlObj.Port(),
			EnrollID:     "admin",
			EnrollSecret: "adminpw",
			CATLS: &current.CATLS{
				CACert: serverCert,
			},
			CSR: &current.CSR{
				Hosts: []string{"csrhost"},
			},
		}

		client := &lib.Client{
			HomeDir: homeDir,
			Config: &lib.ClientConfig{
				TLS: tls.ClientTLSConfig{
					Enabled:   true,
					CertFiles: []string{"tlsCert.pem"},
				},
				URL: fmt.Sprintf("https://%s:%s", config.CAHost, config.CAPort),
			},
		}

		timeout, _ := time.ParseDuration("10s")
		testReenroller = &reenroller.Reenroller{
			Client:   client,
			Identity: mockIdentity,
			Config:   config,
			HomeDir:  homeDir,
			Timeout:  timeout,
		}

		signer := &fabricx509.Signer{}
		cred := &fabricx509.Credential{}
		cred.SetVal(signer)
		mockIdentity.ReenrollReturns(&lib.EnrollmentResponse{
			Identity: lib.NewIdentity(&lib.Client{}, "caIdentity", []credential.Credential{cred}),
		}, nil)
	})

	Context("Enrollment configuration validation", func() {
		It("returns an error if missing CA host", func() {
			config.CAHost = ""
			_, err = reenroller.New(config, homeDir, nil, "", true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to reenroll, CA host not specified"))
		})

		It("returns an error if missing CA Port", func() {
			config.CAPort = ""
			_, err = reenroller.New(config, homeDir, nil, "", true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to reenroll, CA port not specified"))
		})

		It("returns an error if missing enrollment ID", func() {
			config.EnrollID = ""
			_, err = reenroller.New(config, homeDir, nil, "", true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to reenroll, enrollment ID not specified"))
		})

		It("returns an error if missing TLS cert", func() {
			config.CATLS.CACert = ""
			_, err = reenroller.New(config, homeDir, nil, "", true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to reenroll, CA TLS certificate not specified"))
		})
	})

	Context("test avaialability of CA", func() {
		It("returns false if CA is not reachable", func() {
			timeout, _ := time.ParseDuration("0.5s")
			testReenroller.Timeout = timeout
			testReenroller.Config.CAHost = "unreachable.test"
			reachable := testReenroller.IsCAReachable()
			Expect(reachable).To(BeFalse())
		})
		It("returns true if CA is reachable", func() {
			timeout, _ := time.ParseDuration("0.5s")
			testReenroller.Timeout = timeout
			testReenroller.Config.CAHost = serverUrlObj.Hostname()
			testReenroller.Config.CAPort = serverUrlObj.Port()
			testReenroller.Config.CATLS.CACert = serverCert
			reachable := testReenroller.IsCAReachable()
			Expect(reachable).To(BeTrue())
		})
	})

	Context("init client", func() {
		It("returns an error if failed to initialize CA client", func() {
			testReenroller.Config.CATLS.CACert = ""
			err = testReenroller.InitClient()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to init client for re-enroll, CA is not reachable"))
		})

		It("returns initializes CA client", func() {
			err = testReenroller.InitClient()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("reenrolls with CA", func() {

		It("returns an error if reenrollment with CA fails", func() {
			mockIdentity.ReenrollReturns(nil, errors.New("bad reenrollment"))
			_, err = testReenroller.Reenroll()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to re-enroll with CA: bad reenrollment"))
		})

		It("reenrolls with CA for new certificate", func() {
			_, err = testReenroller.Reenroll()
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
