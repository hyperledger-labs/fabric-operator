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

package enroller

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/hyperledger/fabric-ca/lib"
	catls "github.com/hyperledger/fabric-ca/lib/tls"
	"github.com/pkg/errors"
)

func NewFabCAClient(cfg *current.Enrollment, homeDir string, bccsp *commonapi.BCCSP, cert []byte) *FabCAClient {
	client := &lib.Client{
		HomeDir: homeDir,
		Config: &lib.ClientConfig{
			TLS: catls.ClientTLSConfig{
				Enabled:   true,
				CertFiles: []string{"tlsCert.pem"},
			},
			URL: fmt.Sprintf("https://%s:%s", cfg.CAHost, cfg.CAPort),
		},
	}

	client = GetClient(client, bccsp)
	return &FabCAClient{
		Client:        client,
		EnrollmentCfg: cfg,
		CATLSCert:     cert,
		BCCSP:         bccsp,
	}
}

type FabCAClient struct {
	*lib.Client

	EnrollmentCfg *current.Enrollment
	BCCSP         *commonapi.BCCSP
	CATLSCert     []byte
}

func (c *FabCAClient) GetHomeDir() string {
	return c.HomeDir
}

func (c *FabCAClient) SetURL(url string) {
	c.Config.URL = url
}

func (c *FabCAClient) GetConfig() *lib.ClientConfig {
	return c.Config
}

func (c *FabCAClient) GetTLSCert() []byte {
	return c.CATLSCert
}

func (c *FabCAClient) GetEnrollmentRequest() *current.Enrollment {
	return c.EnrollmentCfg
}

func (c *FabCAClient) SetHSMLibrary(library string) {
	if c.BCCSP != nil {
		c.BCCSP.PKCS11.Library = library
		c.Client = GetClient(c.Client, c.BCCSP)
	}
}

func (c *FabCAClient) PingCA(timeout time.Duration) error {
	url := fmt.Sprintf("%s/cainfo", c.Client.Config.URL)
	log.Info(fmt.Sprintf("Pinging CA at '%s' with timeout value of %s", url, timeout.String()))

	rootCertPool := x509.NewCertPool()
	rootCertPool.AppendCertsFromPEM(c.CATLSCert)
	client := http.Client{
		Transport: &http.Transport{
			IdleConnTimeout: timeout,
			Dial: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: timeout,
			}).Dial,
			TLSHandshakeTimeout: timeout / 2,
			TLSClientConfig: &tls.Config{
				RootCAs:    rootCertPool,
				MinVersion: tls.VersionTLS12, // TLS 1.2 recommended, TLS 1.3 (current latest version) encouraged
			},
		},
		Timeout: timeout,
	}

	if err := c.healthCheck(client, url); err != nil {
		return errors.Wrapf(err, "pinging '%s' failed", url)
	}

	return nil
}

func (c *FabCAClient) healthCheck(client http.Client, healthURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return errors.Wrap(err, "invalid http request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "health check request failed")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrapf(err, "failed health check, ca is not running")
	}

	return nil
}
