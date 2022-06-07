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

package action

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/enroller"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate counterfeiter -o mocks/enrollinstance.go -fake-name EnrollInstance . EnrollInstance

type EnrollInstance interface {
	runtime.Object
	metav1.Object
	IsHSMEnabled() bool
	UsingHSMProxy() bool
	GetConfigOverride() (interface{}, error)
	EnrollerImage() string
	GetPullSecrets() []corev1.LocalObjectReference
	PVCName() string
	GetResource(current.Component) corev1.ResourceRequirements
}

func Enroll(instance EnrollInstance, enrollment *current.Enrollment, storagePath string, client k8sclient.Client, scheme *runtime.Scheme, ecert bool, timeouts enroller.HSMEnrollJobTimeouts) (*config.Response, error) {
	log.Info(fmt.Sprintf("Enroll action performing enrollment for identity: %s", enrollment.EnrollID))

	var err error
	defer os.RemoveAll(storagePath)

	bytes, err := enrollment.GetCATLSBytes()
	if err != nil {
		return nil, err
	}

	caClient := enroller.NewFabCAClient(enrollment, storagePath, nil, bytes)
	certEnroller := enroller.New(enroller.NewSWEnroller(caClient))

	// Only check if HSM enroller is needed if the request is for an ecert, TLS cert enrollment is not supported
	// via HSM
	if ecert {
		certEnroller, err = enroller.Factory(enrollment, client, instance, storagePath, scheme, bytes, timeouts)
		if err != nil {
			return nil, err
		}
	}

	crypto, err := config.GenerateCrypto(certEnroller)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate crypto")
	}

	return crypto, nil
}
