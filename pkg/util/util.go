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

package util

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/pkg/k8s/clientset"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	yaml1 "sigs.k8s.io/yaml"
)

const (
	maximumCRNameLength = 32
)

func ConvertYamlFileToJson(file string) ([]byte, error) {
	absfilepath, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadFile(filepath.Clean(absfilepath))
	if err != nil {
		return nil, err
	}

	return yaml.ToJSON(bytes)
}

func GetContainerFromFile(file string) (*corev1.Container, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	cont := &corev1.Container{}
	err = json.Unmarshal(jsonBytes, &cont)
	if err != nil {
		return nil, err
	}

	return cont, nil
}

func GetPVCFromFile(file string) (*corev1.PersistentVolumeClaim, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	pvc := &corev1.PersistentVolumeClaim{}
	err = json.Unmarshal(jsonBytes, &pvc)
	if err != nil {
		return nil, err
	}

	return pvc, nil
}

func GetRoleFromFile(file string) (*rbacv1.Role, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	role := &rbacv1.Role{}
	err = json.Unmarshal(jsonBytes, &role)
	if err != nil {
		return nil, err
	}

	return role, nil
}

func GetClusterRoleFromFile(file string) (*rbacv1.ClusterRole, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	role := &rbacv1.ClusterRole{}
	err = json.Unmarshal(jsonBytes, &role)
	if err != nil {
		return nil, err
	}

	return role, nil
}

func GetRoleBindingFromFile(file string) (*rbacv1.RoleBinding, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	rolebinding := &rbacv1.RoleBinding{}
	err = json.Unmarshal(jsonBytes, &rolebinding)
	if err != nil {
		return nil, err
	}

	return rolebinding, nil
}

func GetClusterRoleBindingFromFile(file string) (*rbacv1.ClusterRoleBinding, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	rolebinding := &rbacv1.ClusterRoleBinding{}
	err = json.Unmarshal(jsonBytes, &rolebinding)
	if err != nil {
		return nil, err
	}

	return rolebinding, nil
}

func GetServiceAccountFromFile(file string) (*corev1.ServiceAccount, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	serviceaccount := &corev1.ServiceAccount{}
	err = json.Unmarshal(jsonBytes, &serviceaccount)
	if err != nil {
		return nil, err
	}

	return serviceaccount, nil
}

func GetDeploymentFromFile(file string) (*appsv1.Deployment, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	dep := &appsv1.Deployment{}
	err = json.Unmarshal(jsonBytes, &dep)
	if err != nil {
		return nil, err
	}

	return dep, nil
}

func GetServiceFromFile(file string) (*corev1.Service, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	svc := &corev1.Service{}
	err = json.Unmarshal(jsonBytes, &svc)
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func GetConfigMapFromFile(file string) (*corev1.ConfigMap, error) {
	absfilepath, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadFile(filepath.Clean(absfilepath))
	if err != nil {
		return nil, err
	}
	cm := &corev1.ConfigMap{}
	err = yaml1.Unmarshal(bytes, cm)
	if err != nil {
		return nil, err
	}

	return cm, nil
}

func GetRouteFromFile(file string) (*routev1.Route, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	route := &routev1.Route{}
	err = json.Unmarshal(jsonBytes, &route)
	if err != nil {
		return nil, err
	}

	return route, nil
}

func GetIngressFromFile(file string) (*networkingv1.Ingress, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	ingress := &networkingv1.Ingress{}
	err = json.Unmarshal(jsonBytes, &ingress)
	if err != nil {
		return nil, err
	}

	return ingress, nil
}

func GetIngressv1beta1FromFile(file string) (*networkingv1beta1.Ingress, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	ingress := &networkingv1beta1.Ingress{}
	err = json.Unmarshal(jsonBytes, &ingress)
	if err != nil {
		return nil, err
	}

	return ingress, nil
}

func GetSecretFromFile(file string) (*corev1.Secret, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{}
	err = json.Unmarshal(jsonBytes, &secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func GetCRDFromFile(file string) (*extv1.CustomResourceDefinition, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	crd := &extv1.CustomResourceDefinition{}
	err = json.Unmarshal(jsonBytes, &crd)
	if err != nil {
		return nil, err
	}

	return crd, nil
}

func GetPodFromFile(file string) (*corev1.Pod, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	pod := &corev1.Pod{}
	err = json.Unmarshal(jsonBytes, &pod)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

func GetResourcePatch(current, new *corev1.ResourceRequirements) (*corev1.ResourceRequirements, error) {
	currentBytes, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}

	newBytes, err := json.Marshal(new)
	if err != nil {
		return nil, err
	}

	patchBytes, err := strategicpatch.StrategicMergePatch(currentBytes, newBytes, corev1.ResourceRequirements{})
	if err != nil {
		return nil, err
	}

	update := &corev1.ResourceRequirements{}
	err = json.Unmarshal(patchBytes, update)
	if err != nil {
		return nil, err
	}

	return update, nil
}

func IgnoreAlreadyExistError(err error) error {
	if !strings.Contains(err.Error(), "already exists") {
		return err
	}
	return nil
}

// Ignore benign error
func IgnoreOutdatedResourceVersion(err error) error {
	if err == nil {
		return nil
	}

	if !strings.Contains(err.Error(), "please apply your changes to the latest version and try again") {
		return err
	}

	return nil
}

func EnvExists(envs []corev1.EnvVar, key string) bool {
	for _, ele := range envs {
		if ele.Name == key {
			return true
		}
	}
	return false
}

func GetEnvValue(envs []corev1.EnvVar, key string) string {
	for _, ele := range envs {
		if ele.Name == key {
			return ele.Value
		}
	}
	return ""
}

func ReplaceEnvIfDiff(envs []corev1.EnvVar, key, replace string) ([]corev1.EnvVar, bool) {
	var updated bool
	for _, ele := range envs {
		if ele.Name == key {
			oldValue := ele.Value
			if oldValue != replace {
				envs = UpdateEnvVar(ele.Name, replace, envs)
				updated = true
			}
		}
	}
	return envs, updated
}

func AppendStringIfMissing(array []string, newEle string) []string {
	for _, ele := range array {
		if ele == newEle {
			return array
		}
	}
	return append(array, newEle)
}

func AppendEnvIfMissing(envs []corev1.EnvVar, env corev1.EnvVar) []corev1.EnvVar {
	for _, ele := range envs {
		if ele.Name == env.Name {
			return envs
		}
	}
	return append(envs, env)
}

func AppendPullSecretIfMissing(pullSecrets []corev1.LocalObjectReference, pullSecret string) []corev1.LocalObjectReference {
	for _, ps := range pullSecrets {
		if ps.Name == pullSecret {
			return pullSecrets
		}
	}
	return append(pullSecrets, corev1.LocalObjectReference{Name: pullSecret})
}

func AppendEnvIfMissingOverrideIfPresent(envs []corev1.EnvVar, env corev1.EnvVar) []corev1.EnvVar {
	for index, ele := range envs {
		if ele.Name == env.Name {
			ele.Value = env.Value
			envs[index] = ele
			return envs
		}
	}
	return append(envs, env)
}

func AppendConfigMapFromSourceIfMissing(envFroms []corev1.EnvFromSource, envFrom corev1.EnvFromSource) []corev1.EnvFromSource {
	for _, ele := range envFroms {
		if ele.ConfigMapRef.Name == envFrom.ConfigMapRef.Name {
			return envFroms
		}
	}
	return append(envFroms, envFrom)
}

func AppendVolumeIfMissing(volumes []corev1.Volume, volume corev1.Volume) []corev1.Volume {
	for _, v := range volumes {
		if v.Name == volume.Name {
			return volumes
		}
	}
	return append(volumes, volume)
}

func AppendVolumeMountIfMissing(volumeMounts []corev1.VolumeMount, volumeMount corev1.VolumeMount) []corev1.VolumeMount {
	for _, v := range volumeMounts {
		if v.Name == volumeMount.Name {
			if v.MountPath == volumeMount.MountPath {
				return volumeMounts
			}
		}
	}
	return append(volumeMounts, volumeMount)
}

func AppendVolumeMountWithSubPathIfMissing(volumeMounts []corev1.VolumeMount, volumeMount corev1.VolumeMount) []corev1.VolumeMount {
	for _, v := range volumeMounts {
		if v.Name == volumeMount.Name {
			if v.SubPath == volumeMount.SubPath {
				return volumeMounts
			}
		}
	}
	return append(volumeMounts, volumeMount)
}

func AppendContainerIfMissing(containers []corev1.Container, container corev1.Container) []corev1.Container {
	for _, c := range containers {
		if c.Name == container.Name {
			return containers
		}
	}
	return append(containers, container)
}

func AppendImagePullSecretIfMissing(imagePullSecrets []corev1.LocalObjectReference, imagePullSecret corev1.LocalObjectReference) []corev1.LocalObjectReference {
	if imagePullSecret.Name == "" {
		return imagePullSecrets
	}
	for _, i := range imagePullSecrets {
		if i.Name == imagePullSecret.Name {
			return imagePullSecrets
		}
	}
	return append(imagePullSecrets, imagePullSecret)
}

func UpdateEnvVar(name string, value string, envs []corev1.EnvVar) []corev1.EnvVar {
	newEnvs := []corev1.EnvVar{}
	for _, e := range envs {
		if e.Name == name {
			e.Value = value
		}
		newEnvs = append(newEnvs, e)
	}
	return newEnvs
}

func ValidationChecks(typedata metav1.TypeMeta, metadata metav1.ObjectMeta, expectedKind string, maxNameLength *int) error {
	maxlength := maximumCRNameLength

	if maxNameLength != nil {
		maxlength = *maxNameLength
	}

	if len(metadata.Name) > maxlength {
		return fmt.Errorf("The instance name '%s' is too long, the name must be less than or equal to %d characters", metadata.Name, maxlength)
	}

	if typedata.Kind != "" {
		if typedata.Kind != expectedKind {
			return fmt.Errorf("The instance '%s' is of kind %s not an %s kind resource, please check to make sure there are no name collisions across resources", metadata.Name, typedata.Kind, expectedKind)
		}
	}

	return nil
}

func SelectRandomValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	randValue, _ := rand.Int(rand.Reader, big.NewInt(int64(len(values))))
	return values[randValue.Int64()]
}

type Client interface {
	Get(ctx context.Context, namespacedName types.NamespacedName, obj k8sclient.Object) error
	List(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error
}

func GetZone(client Client) string {
	nodeList := &corev1.NodeList{}
	err := client.List(context.TODO(), nodeList)
	if err != nil {
		return ""
	}

	zones := []string{}
	for _, node := range nodeList.Items {
		zone := node.ObjectMeta.Labels["topology.kubernetes.io/zone"]
		zones = append(zones, zone)
	}

	return SelectRandomValue(zones)
}

func GetRegion(client Client) string {
	nodeList := &corev1.NodeList{}
	err := client.List(context.TODO(), nodeList)
	if err != nil {
		return ""
	}

	regions := []string{}
	for _, node := range nodeList.Items {
		region := node.ObjectMeta.Labels["topology.kubernetes.io/region"]
		regions = append(regions, region)
	}

	return SelectRandomValue(regions)
}

func ContainsValue(find string, in []string) bool {
	for _, value := range in {
		if find == value {
			return true
		}
	}
	return false
}

func ValidateZone(client Client, requestedZone string) error {
	nodeList := &corev1.NodeList{}
	err := client.List(context.TODO(), nodeList)
	if err != nil {
		return nil
	}
	zones := []string{}
	for _, node := range nodeList.Items {
		zone := node.ObjectMeta.Labels["topology.kubernetes.io/zone"]
		zones = append(zones, zone)
		zone = node.ObjectMeta.Labels["failure-domain.beta.kubernetes.io/zone"]
		zones = append(zones, zone)
		zone = node.ObjectMeta.Labels["ibm-cloud.kubernetes.io/zone"]
		zones = append(zones, zone)
	}
	valueFound := ContainsValue(requestedZone, zones)
	if !valueFound {
		return errors.Errorf("Zone '%s' is not a valid zone", requestedZone)
	}
	return nil
}

func ValidateRegion(client Client, requestedRegion string) error {
	nodeList := &corev1.NodeList{}
	err := client.List(context.TODO(), nodeList)
	if err != nil {
		return nil
	}
	regions := []string{}
	for _, node := range nodeList.Items {
		region := node.ObjectMeta.Labels["topology.kubernetes.io/region"]
		regions = append(regions, region)
		region = node.ObjectMeta.Labels["failure-domain.beta.kubernetes.io/region"]
		regions = append(regions, region)
		region = node.ObjectMeta.Labels["ibm-cloud.kubernetes.io/region"]
		regions = append(regions, region)
	}
	valueFound := ContainsValue(requestedRegion, regions)
	if !valueFound {
		return errors.Errorf("Region '%s' is not a valid region", requestedRegion)
	}
	return nil
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func EnsureDir(dirName string) error {
	err := os.MkdirAll(dirName, 0750)

	if err == nil || os.IsExist(err) {
		return nil
	} else {
		return err
	}
}

func GetResourceVerFromSecret(client Client, name, namespace string) (string, error) {
	secret := &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, secret)
	if err != nil {
		return "", err
	}

	resourceVer := secret.ObjectMeta.ResourceVersion
	return resourceVer, nil
}

func JoinMaps(m1, m2 map[string][]byte) map[string][]byte {
	joined := map[string][]byte{}

	if m1 != nil {
		for k, v := range m1 {
			joined[k] = v
		}
	}

	if m2 != nil {
		for k, v := range m2 {
			joined[k] = v
		}
	}

	return joined
}

func PemStringToBytes(pem string) []byte {
	return []byte(pem)
}

func FileToBytes(file string) ([]byte, error) {
	data, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", file)
	}

	return data, nil
}

func Base64ToBytes(base64str string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(base64str)
	if err != nil {
		// If base64 encoded string is padded with too many '=' at the
		// end DecodeString will fail with error: "illegal base64 data at input byte ...".
		// Need to try stripping of '=' at the one at a time and trying again until no more '='
		// left at that point return err.

		if strings.HasSuffix(base64str, "=") {
			base64str = base64str[:len(base64str)-1]
			return Base64ToBytes(base64str)
		}
		return nil, errors.Wrapf(err, "failed to parse base64 string %s", base64str)
	}

	return data, nil
}

func BytesToBase64(b []byte) string {
	data := base64.StdEncoding.EncodeToString(b)

	return data
}

func GetCertificateFromPEMBytes(bytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM bytes")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse certificate")
	}

	return cert, nil
}

func WriteFile(file string, buf []byte, perm os.FileMode) error {
	dir := path.Dir(file)
	// Create the directory if it doesn't exist
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0750)
		if err != nil {
			return errors.Wrapf(err, "Failed to create directory '%s' for file '%s'", dir, file)
		}
	}
	return ioutil.WriteFile(file, buf, perm)
}

func CheckIfZoneOrRegionUpdated(oldValue string, newValue string) bool {
	if (strings.ToLower(oldValue) != "select" && oldValue != "") && (strings.ToLower(newValue) != "select" && newValue != "") {
		if oldValue != newValue {
			return true
		}
	}

	return false
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

func ValidateHSMProxyURL(endpoint string) error {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	address := strings.Split(parsedURL.Host, ":")
	if len(address) < 2 {
		return errors.New("must specify both IP address and port")
	}

	if address[0] == "" {
		return errors.New("missing IP address")
	}

	if address[1] == "" {
		return errors.New("missing port")
	}

	scheme := parsedURL.Scheme
	if scheme != "tls" && scheme != "tcp" {
		return fmt.Errorf("unsupported scheme '%s', only tcp and tls are supported", scheme)
	}

	if !IsTCPReachable(parsedURL.Host) {
		return fmt.Errorf("Unable to reach HSM endpoint: %s", parsedURL.Host)
	}
	return nil
}

// func HealthCheck(caURL *url.URL, cert []byte) error {
func HealthCheck(healthURL string, cert []byte, timeout time.Duration) error {
	rootCertPool := x509.NewCertPool()
	rootCertPool.AppendCertsFromPEM(cert)

	transport := http.DefaultTransport
	transport.(*http.Transport).TLSClientConfig = &tls.Config{
		RootCAs:    rootCertPool,
		MinVersion: tls.VersionTLS12, // TLS 1.2 recommended, TLS 1.3 (current latest version) encouraged
	}

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
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

func IsTCPReachable(url string) bool {
	url = strings.Replace(url, "tcp://", "", -1)
	url = strings.Replace(url, "tls://", "", -1)

	conn, err := net.Dial("tcp", url)
	if err != nil {
		return false
	}

	defer func() {
		if err := conn.Close(); err != nil {
			return
		}
	}()

	return true
}

func IntermediateSecretExists(client Client, namespace, secretName string) bool {
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      secretName,
		Namespace: namespace}, &corev1.Secret{})
	if err != nil {
		return false
	}

	return true
}

func IsSecretTLSCert(secretName string) bool {
	if strings.HasSuffix(secretName, "-signcert") {
		return strings.HasPrefix(secretName, "tls")
	} else if strings.HasSuffix(secretName, "-ca-crypto") {
		return true
	}

	return false
}

func IsSecretEcert(secretName string) bool {
	if strings.HasSuffix(secretName, "-signcert") {
		return strings.HasPrefix(secretName, "ecert")
	}

	return false
}

func ConvertSpec(in interface{}, out interface{}) error {
	jsonBytes, err := yaml1.Marshal(in)
	if err != nil {
		return err
	}

	err = yaml1.Unmarshal(jsonBytes, out)
	if err != nil {
		return err
	}
	return nil
}

func FindStringInArray(str string, slice []string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func ConvertToJsonMessage(in interface{}) (*json.RawMessage, error) {
	bytes, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	jm := json.RawMessage(bytes)
	return &jm,

		nil
}

func GetNetworkPolicyFromFile(file string) (*networkingv1.NetworkPolicy, error) {
	jsonBytes, err := ConvertYamlFileToJson(file)
	if err != nil {
		return nil, err
	}

	policy := &networkingv1.NetworkPolicy{}
	err = json.Unmarshal(jsonBytes, &policy)
	if err != nil {
		return nil, err
	}

	return policy, nil
}

func GetServerVersion() (*version.Info, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientSet, err := clientset.New(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client")
	}

	version, err := clientSet.DiscoveryClient.ServerVersion()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get version")
	}
	return version, nil
}
