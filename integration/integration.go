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

package integration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"
	"github.com/IBM-Blockchain/fabric-operator/pkg/command"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	TestAutomation1IngressDomain = "localho.st"
)

var (
	defaultConfigs = "../../defaultconfig"
	defaultDef     = "../../definitions"

	operatorCfg        *config.Config
	operatorContext    context.Context
	operatorCancelFunc context.CancelFunc
)

type Config struct {
	OperatorServiceAccount string
	OperatorRole           string
	OperatorRoleBinding    string
	OperatorDeployment     string
	OrdererSecret          string
	PeerSecret             string
	ConsoleTLSSecret       string
}

func SetupSignalHandler() context.Context {
	operatorContext, operatorCancelFunc = context.WithCancel(context.Background())
	return operatorContext
}

func Setup(ginkgoWriter io.Writer, cfg *Config, suffix, pathToDefaultDir string) (string, *kubernetes.Clientset, *ibpclient.IBPClient, error) {
	// Set up a signal handler Context to allow a graceful shutdown of the operator.
	SetupSignalHandler()

	var err error

	if pathToDefaultDir != "" {
		defaultConfigs = filepath.Join(pathToDefaultDir, "defaultconfig")
		defaultDef = filepath.Join(pathToDefaultDir, "definitions")
	}
	operatorCfg = getOperatorCfg()

	wd, err := os.Getwd()
	if err != nil {
		return "", nil, nil, err
	}
	fmt.Fprintf(ginkgoWriter, "Working directory: %s\n", wd)

	namespace := os.Getenv("OPERATOR_NAMESPACE")
	if namespace == "" {
		namespace = "operatortest"
	}
	if suffix != "" {
		namespace = fmt.Sprintf("%s%s", namespace, suffix)
	}

	fmt.Fprintf(ginkgoWriter, "Namespace set to '%s'\n", namespace)

	setupConfig, err := GetConfig()
	if err != nil {
		return "", nil, nil, err
	}

	fmt.Fprintf(ginkgoWriter, "Setup config %+v\n", setupConfig)

	kclient, ibpCRClient, err := InitClients(setupConfig)
	if err != nil {
		return "", nil, nil, err
	}

	err = os.Setenv("CLUSTERTYPE", "K8S")
	if err != nil {
		return "", nil, nil, err
	}
	err = os.Setenv("WATCH_NAMESPACE", namespace)
	if err != nil {
		return "", nil, nil, err
	}

	err = CleanupNamespace(ginkgoWriter, kclient, namespace)
	if err != nil {
		return "", nil, nil, err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err = kclient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		return "", nil, nil, err
	}
	fmt.Fprintf(ginkgoWriter, "Namespace '%s' created\n", namespace)

	// Set up an image pull secret if a docker config json has been specified
	if setupConfig.DockerConfigJson != "" {
		fmt.Fprintf(ginkgoWriter, "Creating 'regcred' image pull secret for DOCKERCONFIGJSON")

		err = CreatePullSecret(kclient, "regcred", namespace, setupConfig.DockerConfigJson)
		if err != nil {
			return "", nil, nil, err
		}
	}

	err = DeployOperator(ginkgoWriter, operatorContext, cfg, kclient, namespace)
	if err != nil {
		return "", nil, nil, err
	}

	return namespace, kclient, ibpCRClient, nil
}

func deleteNamespace(ginkgoWriter io.Writer, kclient *kubernetes.Clientset, namespace string) error {
	var zero int64 = 0
	policy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &zero,
		PropagationPolicy:  &policy,
	}
	fmt.Fprintf(ginkgoWriter, "Deleting namespace '%s' with options %s\n", namespace, &deleteOptions)
	return kclient.CoreV1().Namespaces().Delete(context.TODO(), namespace, deleteOptions)
}

type SetupConfig struct {
	DockerConfigJson string
	KubeConfig       string
}

func GetConfig() (*SetupConfig, error) {
	return &SetupConfig{
		DockerConfigJson: os.Getenv("DOCKERCONFIGJSON"),
		KubeConfig:       os.Getenv("KUBECONFIG_PATH"),
	}, nil
}

func InitClients(setupConfig *SetupConfig) (*kubernetes.Clientset, *ibpclient.IBPClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Not running in a cluster, get kube config from KUBECONFIG env var
		kubeConfigPath := setupConfig.KubeConfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			fmt.Println("error:", err)
			return nil, nil, err
		}
	}

	kclient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	client, err := ibpclient.New(config)
	if err != nil {
		return nil, nil, err
	}

	return kclient, client, nil
}

func DeployOperator(ginkgoWriter io.Writer, signal context.Context, cfg *Config, kclient *kubernetes.Clientset, namespace string) error {
	fmt.Fprintf(ginkgoWriter, "Deploying operator in namespace '%s'\n", namespace)
	// Create service account for operator
	sa, err := util.GetServiceAccountFromFile(cfg.OperatorServiceAccount)
	if err != nil {
		return err
	}
	_, err = kclient.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Create cluster role with permissions required by operator
	role, err := util.GetClusterRoleFromFile(cfg.OperatorRole)
	if err != nil {
		return err
	}
	_, err = kclient.RbacV1().ClusterRoles().Create(context.TODO(), role, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	// Create role binding for operator's cluster role
	roleBinding, err := util.GetClusterRoleBindingFromFile(cfg.OperatorRoleBinding)
	if err != nil {
		return err
	}

	roleBinding.Name = fmt.Sprintf("operator-%s", namespace)
	roleBinding.Subjects[0].Namespace = namespace

	_, err = kclient.RbacV1().ClusterRoleBindings().Create(context.TODO(), roleBinding, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	// Create resource secrets
	ordererSecret, err := util.GetSecretFromFile(cfg.OrdererSecret)
	if err != nil {
		return err
	}
	_, err = kclient.CoreV1().Secrets(namespace).Create(context.TODO(), ordererSecret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Peer 1 secret
	peerSecret, err := util.GetSecretFromFile(cfg.PeerSecret)
	if err != nil {
		return err
	}
	_, err = kclient.CoreV1().Secrets(namespace).Create(context.TODO(), peerSecret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Peer 2 secret
	peerSecret.Name = "ibppeer2-secret"
	_, err = kclient.CoreV1().Secrets(namespace).Create(context.TODO(), peerSecret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	consoleTLSSecret, err := util.GetSecretFromFile(cfg.ConsoleTLSSecret)
	if err != nil {
		return err
	}
	_, err = kclient.CoreV1().Secrets(namespace).Create(context.TODO(), consoleTLSSecret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	err = command.OperatorWithSignal(operatorCfg, signal, false, true)
	if err != nil {
		return err
	}

	fmt.Fprintf(ginkgoWriter, "Done deploying operator in namespace '%s'\n", namespace)

	return nil
}

func Cleanup(ginkgoWriter io.Writer, kclient *kubernetes.Clientset, namespace string) error {

	// The operator must halt before the namespace can be deleted in the foreground.
	ShutdownOperator(ginkgoWriter)

	err := CleanupNamespace(ginkgoWriter, kclient, namespace)
	if err != nil {
		return err
	}

	return nil
}

func ShutdownOperator(ginkgoWriter io.Writer) {
	if operatorContext != nil {
		fmt.Fprintf(ginkgoWriter, "Stopping operator\n")
		operatorContext.Done()
		operatorCancelFunc()
	}
}

func CleanupNamespace(ginkgoWriter io.Writer, kclient *kubernetes.Clientset, namespace string) error {
	err := deleteNamespace(ginkgoWriter, kclient, namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil // Namespace does not exist, don't need to wait for deletion to complete
		}
	}

	opts := metav1.ListOptions{}
	watchNamespace, err := kclient.CoreV1().Namespaces().Watch(context.TODO(), opts)
	if err != nil {
		return err
	}

	fmt.Fprintf(ginkgoWriter, "Waiting for namespace deletion\n")
	for {
		resultChan := <-watchNamespace.ResultChan()
		if resultChan.Type == watch.Deleted {
			ns := resultChan.Object.(*corev1.Namespace)
			if ns.Name == namespace {
				break
			}
		}
	}
	fmt.Fprintf(ginkgoWriter, "Done deleting namespace '%s'\n", namespace)
	return nil
}

func DeleteNamespace(ginkgoWriter io.Writer, kclient *kubernetes.Clientset, namespace string) error {
	err := deleteNamespace(ginkgoWriter, kclient, namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil // Namespace does not exist, don't need to wait for deletion to complete
		}
	}

	return nil
}

func CreatePullSecret(kclient *kubernetes.Clientset, name string, namespace string, dockerConfigJson string) error {
	b, err := base64.StdEncoding.DecodeString(dockerConfigJson)
	if err != nil {
		return err
	}

	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			".dockerconfigjson": b,
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}

	_, err = kclient.CoreV1().Secrets(namespace).Create(context.TODO(), pullSecret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func ClearOperatorConfig(kclient *kubernetes.Clientset, namespace string) error {
	err := kclient.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), "operator-config", *metav1.NewDeleteOptions(0))
	if !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}

func ResilientPatch(kclient *ibpclient.IBPClient, name, namespace, kind string, retry int, into client.Object, patch func(i client.Object)) error {

	for i := 0; i < retry; i++ {
		err := resilientPatch(kclient, name, namespace, kind, into, patch)
		if err != nil {
			if i == retry {
				return err
			}
			if k8serrors.IsConflict(err) {
				time.Sleep(2 * time.Second)
				continue
			}
			return err
		}
	}

	return nil
}

func resilientPatch(kclient *ibpclient.IBPClient, name, namespace, kind string, into client.Object, patch func(i client.Object)) error {
	result := kclient.Get().Namespace(namespace).Resource(kind).Name(name).Do(context.TODO())
	if result.Error() != nil {
		return result.Error()
	}

	err := result.Into(into)
	if err != nil {
		return err
	}

	patch(into)
	bytes, err := json.Marshal(into)
	if err != nil {
		return err
	}

	result = kclient.Patch(types.MergePatchType).Namespace(namespace).Resource(kind).Name(name).Body(bytes).Do(context.TODO())
	if result.Error() != nil {
		return result.Error()
	}

	return nil
}

func CreateOperatorConfigMapFromFile(namespace string, kclient *kubernetes.Clientset, file string) error {
	configData, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "operator",
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.yaml": string(configData),
		},
	}

	_, err = kclient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// CreateConfigMap creates config map
func CreateConfigMap(kclient *kubernetes.Clientset, config interface{}, key, name, namespace string) error {
	configBytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			key: string(configBytes),
		},
	}

	_, err = kclient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func OperatorCfg() *config.Config {
	return getOperatorCfg()
}

func getOperatorCfg() *config.Config {
	defaultPeerDef := filepath.Join(defaultDef, "peer")
	defaultCADef := filepath.Join(defaultDef, "ca")
	defaultOrdererDef := filepath.Join(defaultDef, "orderer")
	defaultConsoleDef := filepath.Join(defaultDef, "console")
	return GetOperatorConfig(defaultConfigs, defaultCADef, defaultPeerDef, defaultOrdererDef, defaultConsoleDef)
}
