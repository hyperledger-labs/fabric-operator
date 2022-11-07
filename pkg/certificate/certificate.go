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

package certificate

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	commonapi "github.com/IBM-Blockchain/fabric-operator/pkg/apis/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/certificate/reenroller"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("certificate_manager")

//go:generate counterfeiter -o mocks/reenroller.go -fake-name Reenroller . Reenroller
type Reenroller interface {
	Reenroll() (*config.Response, error)
}

type CertificateManager struct {
	Client k8sclient.Client
	Scheme *runtime.Scheme
}

func New(client k8sclient.Client, scheme *runtime.Scheme) *CertificateManager {
	return &CertificateManager{
		Client: client,
		Scheme: scheme,
	}
}

func (c *CertificateManager) GetExpireDate(pemBytes []byte) (time.Time, error) {
	cert, err := util.GetCertificateFromPEMBytes(pemBytes)
	if err != nil {
		return time.Time{}, errors.New("failed to get certificate from bytes")
	}

	return cert.NotAfter, nil
}

func (c *CertificateManager) GetDurationToNextRenewal(certType common.SecretType, instance v1.Object, numSecondsBeforeExpire int64) (time.Duration, error) {
	certName := fmt.Sprintf("%s-%s-signcert", certType, instance.GetName())
	cert, err := c.GetSignCert(certName, instance.GetNamespace())
	if err != nil {
		return time.Duration(0), err
	}

	return c.GetDurationToNextRenewalForCert(certName, cert, instance, numSecondsBeforeExpire)
}

func (c *CertificateManager) GetDurationToNextRenewalForCert(certName string, cert []byte, instance v1.Object, numSecondsBeforeExpire int64) (time.Duration, error) {
	expireDate, err := c.GetExpireDate(cert)
	if err != nil {
		return time.Duration(0), err
	}

	if expireDate.IsZero() {
		return time.Duration(0), errors.New("failed to get non-zero expiration date from certificate")
	}
	if expireDate.Before(time.Now()) {
		return time.Duration(0), fmt.Errorf("%s has expired", certName)
	}

	renewDate := expireDate.Add(-time.Duration(numSecondsBeforeExpire) * time.Second) // Subtract num seconds from expire date
	duration := time.Until(renewDate)                                                 // Get duration between now and the renew date (negative duration means renew date < time.Now())
	if duration < 0 {
		return time.Duration(0), nil
	}
	return duration, nil
}

func (c *CertificateManager) CertificateExpiring(certType common.SecretType, instance v1.Object, numSecondsBeforeExpire int64) (expiring bool, expireDate time.Time, err error) {
	certName := fmt.Sprintf("%s-%s-signcert", certType, instance.GetName())
	cert, err := c.GetSignCert(certName, instance.GetNamespace())
	if err != nil {
		return false, time.Time{}, err
	}

	return c.Expires(cert, numSecondsBeforeExpire)
}

func (c *CertificateManager) Expires(cert []byte, numSecondsBeforeExpire int64) (expiring bool, expireDate time.Time, err error) {
	expireDate, err = c.GetExpireDate(cert)
	if err != nil {
		return false, time.Time{}, err
	}

	// Checks if the duration between time.Now() and the expiration date is less than or equal to the numSecondsBeforeExpire
	if time.Until(expireDate) <= time.Duration(numSecondsBeforeExpire)*time.Second {
		return true, expireDate, nil
	}

	return false, time.Time{}, nil
}

func (c *CertificateManager) CheckCertificatesForExpire(instance v1.Object, numSecondsBeforeExpire int64) (statusType current.IBPCRStatusType, message string, err error) {
	tlsExpiring, tlsExpireDate, err := c.CertificateExpiring(common.TLS, instance, numSecondsBeforeExpire)
	if err != nil {
		err = errors.Wrap(err, "failed to get tls signcert expiry info")
		return
	}

	ecertExpiring, ecertExpireDate, err := c.CertificateExpiring(common.ECERT, instance, numSecondsBeforeExpire)
	if err != nil {
		err = errors.Wrap(err, "failed to get ecert signcert expiry info")
		return
	}

	// If not certificate are expring, no further action is required
	if !tlsExpiring && !ecertExpiring {
		return current.Deployed, "", nil
	}

	statusType = current.Warning

	if tlsExpiring {
		// Check if tls cert's expiration date has already passed
		if tlsExpireDate.Before(time.Now()) {
			statusType = current.Error
			message += fmt.Sprintf("tls-%s-signcert has expired", instance.GetName())
		} else {
			message += fmt.Sprintf("tls-%s-signcert expires on %s", instance.GetName(), tlsExpireDate.String())
		}
	}

	if message != "" {
		message += ", "
	}

	if ecertExpiring {
		// Check if ecert's expiration date has already passed
		if ecertExpireDate.Before(time.Now()) {
			statusType = current.Error
			message += fmt.Sprintf("ecert-%s-signcert has expired", instance.GetName())
		} else {
			message += fmt.Sprintf("ecert-%s-signcert expires on %s", instance.GetName(), ecertExpireDate.String())
		}
	}

	return statusType, message, nil
}

func (c *CertificateManager) GetReenroller(certType common.SecretType, spec *current.EnrollmentSpec, bccsp *commonapi.BCCSP, storagePath string, certPemBytes, keyPemBytes []byte, hsmEnabled bool, newKey bool) (Reenroller, error) {
	storagePath = filepath.Join(storagePath, "reenroller", string(certType))

	var cfg *current.Enrollment
	if certType == common.TLS {
		cfg = spec.TLS
	} else {
		cfg = spec.Component
	}

	certReenroller, err := reenroller.New(cfg, storagePath, bccsp, "", newKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize reenroller")
	}

	err = certReenroller.InitClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize CA client for reenroller")
	}

	err = certReenroller.LoadIdentity(certPemBytes, keyPemBytes, hsmEnabled)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load Identity for reenroller")
	}

	return certReenroller, nil
}

func (c *CertificateManager) ReenrollCert(certType common.SecretType, reenroller Reenroller, instance v1.Object, hsmEnabled bool) error {
	if reenroller == nil {
		return errors.New("reenroller not passed")
	}

	resp, err := reenroller.Reenroll()
	if err != nil {
		return errors.Wrapf(err, "failed to renew %s certificate for instance '%s'", certType, instance.GetName())
	}

	err = c.UpdateSignCert(fmt.Sprintf("%s-%s-signcert", certType, instance.GetName()), resp.SignCert, instance)
	if err != nil {
		return errors.Wrapf(err, "failed to update signcert secret for instance '%s'", instance.GetName())
	}

	if !hsmEnabled {
		err = c.UpdateKey(fmt.Sprintf("%s-%s-keystore", certType, instance.GetName()), resp.Keystore, instance)
		if err != nil {
			return errors.Wrapf(err, "failed to update keystore secret for instance '%s'", instance.GetName())
		}
	}

	return nil
}

type Instance interface {
	v1.Object
	UsingHSMProxy() bool
	IsHSMEnabled() bool
	EnrollerImage() string
	GetPullSecrets() []corev1.LocalObjectReference
	GetResource(current.Component) corev1.ResourceRequirements
	PVCName() string
}

func (c *CertificateManager) RenewCert(certType common.SecretType, instance Instance, spec *current.EnrollmentSpec, bccsp *commonapi.BCCSP, storagePath string, hsmEnabled bool, newKey bool) error {
	cert, key, err := c.GetSignCertAndKey(certType, instance, hsmEnabled)
	if err != nil {
		return err
	}

	if certType == common.TLS && hsmEnabled && !instance.UsingHSMProxy() {
		bccsp = nil
	}

	var certReenroller Reenroller
	if certType == common.ECERT && hsmEnabled && !instance.UsingHSMProxy() {
		log.Info("Certificate manager renewing ecert, non-proxy HSM enabled")
		hsmConfig, err := config.ReadHSMConfig(c.Client, instance)
		if err != nil {
			return err
		}

		if hsmConfig.Daemon != nil {
			certReenroller, err = reenroller.NewHSMDaemonReenroller(spec.Component, storagePath, bccsp, "", hsmConfig, instance, c.Client, c.Scheme, newKey)
			if err != nil {
				return err
			}
		} else {
			certReenroller, err = reenroller.NewHSMReenroller(spec.Component, storagePath, bccsp, "", hsmConfig, instance, c.Client, c.Scheme, newKey)
			if err != nil {
				return err
			}
		}

		err = c.ReenrollCert(certType, certReenroller, instance, hsmEnabled)
		if err != nil {
			return err
		}

		return nil
	}

	// For TLS certificate, always use software enroller. We don't support HSM for TLS certificates
	if certType == common.TLS {
		log.Info("Certificate manager renewing TLS")
		bccsp = nil

		keySecretName := fmt.Sprintf("%s-%s-keystore", certType, instance.GetName())
		key, err = c.GetKey(keySecretName, instance.GetNamespace())
		if err != nil {
			return err
		}

		certReenroller, err = c.GetReenroller(certType, spec, bccsp, storagePath, cert, key, false, newKey)
		if err != nil {
			return err
		}

		err = c.ReenrollCert(certType, certReenroller, instance, false)
		if err != nil {
			return err
		}

		return nil
	}

	log.Info(fmt.Sprintf("Certificate manager renewing %s", certType))
	certReenroller, err = c.GetReenroller(certType, spec, bccsp, storagePath, cert, key, hsmEnabled, newKey)
	if err != nil {
		return err
	}

	err = c.ReenrollCert(certType, certReenroller, instance, hsmEnabled)
	if err != nil {
		return err
	}

	return nil
}

func (c *CertificateManager) UpdateSignCert(name string, cert []byte, instance v1.Object) error {
	// Cert might not be returned from reenroll call, for example if the reenroll happens in a job which handles
	// updating the secret when using HSM (non-proxy)
	if len(cert) == 0 {
		return nil
	}

	data := map[string][]byte{
		"cert.pem": cert,
	}

	err := c.UpdateSecret(instance, name, data)
	if err != nil {
		return err
	}

	return nil
}

func (c *CertificateManager) UpdateKey(name string, key []byte, instance v1.Object) error {
	// Need to ensure the value passed in for key is valid before updating.
	// Otherwise, an empty key will end up in the secret overriding a valid key, which
	// will cause runtime errors on nodes
	if len(key) == 0 {
		return nil
	}

	data := map[string][]byte{
		"key.pem": key,
	}

	err := c.UpdateSecret(instance, name, data)
	if err != nil {
		return err
	}

	return nil
}

func (c *CertificateManager) UpdateSecret(instance v1.Object, name string, data map[string][]byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: instance.GetNamespace(),
			Labels:    instance.GetLabels(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}

	err := c.Client.Update(context.TODO(), secret, k8sclient.UpdateOption{Owner: instance, Scheme: c.Scheme})
	if err != nil {
		return err
	}

	return nil
}

func (c *CertificateManager) GetSignCertAndKey(certType common.SecretType, instance v1.Object, hsmEnabled bool) ([]byte, []byte, error) {
	certSecretName := fmt.Sprintf("%s-%s-signcert", certType, instance.GetName())
	keySecretName := fmt.Sprintf("%s-%s-keystore", certType, instance.GetName())

	cert, err := c.GetSignCert(certSecretName, instance.GetNamespace())
	if err != nil {
		return nil, nil, err
	}

	key := []byte{}
	if !hsmEnabled {
		key, err = c.GetKey(keySecretName, instance.GetNamespace())
		if err != nil {
			return nil, nil, err
		}
	}

	return cert, key, nil
}

func (c *CertificateManager) GetSignCert(name, namespace string) ([]byte, error) {
	secret, err := c.GetSecret(name, namespace)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data) == 0 {
		return nil, errors.New(fmt.Sprintf("%s secret is blank", name))
	}

	if secret.Data["cert.pem"] != nil {
		return secret.Data["cert.pem"], nil
	}

	return nil, errors.New(fmt.Sprintf("cannot get %s", name))
}

func (c *CertificateManager) GetKey(name, namespace string) ([]byte, error) {
	secret, err := c.GetSecret(name, namespace)
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data) == 0 {
		return nil, errors.New(fmt.Sprintf("%s secret is blank", name))
	}

	if secret.Data["key.pem"] != nil {
		return secret.Data["key.pem"], nil
	}

	return nil, errors.New(fmt.Sprintf("cannot get %s", name))
}

func (c *CertificateManager) GetSecret(name, namespace string) (*corev1.Secret, error) {
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}

	secret := &corev1.Secret{}
	err := c.Client.Get(context.TODO(), namespacedName, secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}
