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

package migration_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	apis "github.com/IBM-Blockchain/fabric-operator/api"
	"github.com/IBM-Blockchain/fabric-operator/pkg/global"
	"github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestMigration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migration Suite")
}

var (
	kclient   *kubernetes.Clientset
	client    controllerclient.Client
	scheme    *runtime.Scheme
	namespace string
	mgr       manager.Manager
	killchan  context.Context
)

var _ = BeforeSuite(func() {
	var err error
	cfg, err := config.GetConfig()
	Expect(err).NotTo(HaveOccurred())

	namespace = os.Getenv("OPERATOR_NAMESPACE")
	if namespace == "" {
		namespace = "operator-test"
	}
	namespace = fmt.Sprintf("%s-migration", namespace)

	mgr, err = manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	err = apis.AddToScheme(mgr.GetScheme())
	Expect(err).NotTo(HaveOccurred())

	killchan = context.TODO()
	go mgr.Start(killchan)

	client = controllerclient.New(mgr.GetClient(), &global.ConfigSetter{})
	scheme = mgr.GetScheme()

	kclient, err = kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	cleanup()

	ns := &corev1.Namespace{}
	ns.Name = namespace
	err = client.Create(context.TODO(), ns)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := cleanup()
	Expect(err).NotTo(HaveOccurred())

	killchan.Done()
})

func cleanup() error {
	ns := &corev1.Namespace{}
	ns.Name = namespace

	err := client.Delete(context.TODO(), ns)
	if err != nil {
		return err
	}

	opts := metav1.ListOptions{}
	watchNamespace, err := kclient.CoreV1().Namespaces().Watch(context.TODO(), opts)
	if err != nil {
		return err
	}

	for {
		resultChan := <-watchNamespace.ResultChan()
		if resultChan.Type == watch.Deleted {
			ns := resultChan.Object.(*corev1.Namespace)
			if ns.Name == namespace {
				break
			}
		}
	}

	return nil
}
