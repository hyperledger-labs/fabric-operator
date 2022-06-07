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
	"context"
	"fmt"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("action")

// By triggering a component restart by updating its annotations instead of deleting
// the deployment, components will be restarted with the rolling update strategy
// unless their deployments specify a recreate strategy. This will allow ibpconsole
// components with rolling update strategies to not have any downtime.
func Restart(client k8sclient.Client, name, namespace string) error {
	deployment := &appsv1.Deployment{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, deployment)
	if err != nil {
		return err
	}

	if deployment == nil {
		return fmt.Errorf("failed to get deployment %s", name)
	}

	if deployment.Spec.Template.ObjectMeta.Annotations == nil {
		deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	err = client.Patch(context.TODO(), deployment, nil, k8sclient.PatchOption{
		Resilient: &k8sclient.ResilientPatch{
			Retry:    3,
			Into:     &appsv1.Deployment{},
			Strategy: runtimeclient.MergeFrom,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

//go:generate counterfeiter -o mocks/reenroller.go -fake-name Reenroller . Reenroller

type Reenroller interface {
	RenewCert(certType common.SecretType, instance runtime.Object, newKey bool) error
}

//go:generate counterfeiter -o mocks/reenrollinstance.go -fake-name ReenrollInstance . ReenrollInstance

type ReenrollInstance interface {
	runtime.Object
	v1.Object
	ResetEcertReenroll()
	ResetTLSReenroll()
}

func Reenroll(reenroller Reenroller, client k8sclient.Client, certType common.SecretType, instance ReenrollInstance, newKey bool) error {
	err := reenroller.RenewCert(certType, instance, newKey)
	if err != nil {
		return err
	}

	return nil
}
