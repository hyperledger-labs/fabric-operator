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

package init

import (
	"context"
	"os"

	apis "github.com/IBM-Blockchain/fabric-operator/api"
	"github.com/IBM-Blockchain/fabric-operator/pkg/global"
	"github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	cfconfig "github.com/cloudflare/cfssl/config"
	"github.com/hyperledger/fabric-ca/lib"
	"github.com/hyperledger/fabric-ca/lib/tls"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	rootDir  = "root-home"
	interDir = "inter-home"

	tlsCertFile = "../../../testdata/init/peer/tls-cert.pem"
	tlsKeyFile  = "../../../testdata/init/peer/tls-key.pem"
)

var (
	root      *lib.Server
	inter     *lib.Server
	client    controllerclient.Client
	scheme    *runtime.Scheme
	namespace string

	kclient *kubernetes.Clientset
)

var _ = BeforeSuite(func() {
	cfg, err := config.GetConfig()
	Expect(err).NotTo(HaveOccurred())

	namespace = os.Getenv("OPERATOR_NAMESPACE")
	if namespace == "" {
		namespace = "operator-test"
	}
	mgr, err := manager.New(cfg, manager.Options{
		Namespace: namespace,
	})
	Expect(err).NotTo(HaveOccurred())

	err = apis.AddToScheme(mgr.GetScheme())
	Expect(err).NotTo(HaveOccurred())
	go mgr.Start(signals.SetupSignalHandler())

	client = controllerclient.New(mgr.GetClient(), &global.ConfigSetter{})
	scheme = mgr.GetScheme()

	kclient, err = kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	ns := &corev1.Namespace{}
	ns.Name = namespace
	err = client.Create(context.TODO(), ns)
	Expect(err).NotTo(HaveOccurred())

	// Setup root server
	root = SetupServer(rootDir, "", 7054, nil)
	err = root.Start()
	Expect(err).NotTo(HaveOccurred())

	// Setup intermediate server
	tlsConfig := &tls.ServerTLSConfig{
		Enabled:  true,
		CertFile: tlsCertFile,
		KeyFile:  tlsKeyFile,
	}
	inter = SetupServer(interDir, "http://admin:adminpw@localhost:7054", 7055, tlsConfig)
	err = inter.Start()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := root.Stop()
	Expect(err).NotTo(HaveOccurred())

	err = inter.Stop()
	Expect(err).NotTo(HaveOccurred())

	err = os.RemoveAll(rootDir)
	Expect(err).NotTo(HaveOccurred())

	err = os.RemoveAll(interDir)
	Expect(err).NotTo(HaveOccurred())

	ns := &corev1.Namespace{}
	ns.Name = namespace
	err = client.Delete(context.TODO(), ns)
	Expect(err).NotTo(HaveOccurred())
})

func SetupServer(homeDir string, parentURL string, port int, tlsConfig *tls.ServerTLSConfig) *lib.Server {
	affiliations := map[string]interface{}{
		"hyperledger": map[string]interface{}{
			"fabric":    []string{"ledger", "orderer", "security"},
			"fabric-ca": nil,
			"sdk":       nil,
		},
		"org2":      []string{"dept1"},
		"org1":      nil,
		"org2dept1": nil,
	}
	profiles := map[string]*cfconfig.SigningProfile{
		"tls": &cfconfig.SigningProfile{
			Usage:        []string{"signing", "key encipherment", "server auth", "client auth", "key agreement"},
			ExpiryString: "8760h",
		},
		"ca": &cfconfig.SigningProfile{
			Usage:        []string{"cert sign", "crl sign"},
			ExpiryString: "8760h",
			CAConstraint: cfconfig.CAConstraint{
				IsCA:       true,
				MaxPathLen: 0,
			},
		},
	}
	defaultProfile := &cfconfig.SigningProfile{
		Usage:        []string{"cert sign"},
		ExpiryString: "8760h",
	}
	srv := &lib.Server{
		Config: &lib.ServerConfig{
			Port:  port,
			Debug: true,
		},
		CA: lib.CA{
			Config: &lib.CAConfig{
				Intermediate: lib.IntermediateCA{
					ParentServer: lib.ParentServer{
						URL: parentURL,
					},
				},
				Affiliations: affiliations,
				Registry: lib.CAConfigRegistry{
					MaxEnrollments: -1,
				},
				Signing: &cfconfig.Signing{
					Profiles: profiles,
					Default:  defaultProfile,
				},
				Version: "1.1.0", // The default test server/ca should use the latest version
			},
		},
		HomeDir: homeDir,
	}

	if tlsConfig != nil {
		srv.Config.TLS = *tlsConfig
	}
	// The bootstrap user's affiliation is the empty string, which
	// means the user is at the affiliation root
	err := srv.RegisterBootstrapUser("admin", "adminpw", "")
	Expect(err).NotTo(HaveOccurred())

	return srv
}
