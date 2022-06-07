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

package helper

import (
	"context"
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/integration"
	ibpclient "github.com/IBM-Blockchain/fabric-operator/pkg/client"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateOrderer(crClient *ibpclient.IBPClient, orderer *current.IBPOrderer) error {
	result := crClient.Post().Namespace(orderer.Namespace).Resource("ibporderers").Body(orderer).Do(context.TODO())
	err := result.Error()
	if !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

type Orderer struct {
	Domain     string
	Name       string
	Namespace  string
	NodeName   string
	Nodes      []Orderer
	WorkingDir string

	CR       *current.IBPOrderer
	CRClient *ibpclient.IBPClient
	KClient  *kubernetes.Clientset

	integration.NativeResourcePoller
}

func (o *Orderer) PollForParentCRStatus() current.IBPCRStatusType {
	crStatus := &current.IBPOrderer{}

	result := o.CRClient.Get().Namespace(o.Namespace).Resource("ibporderers").Name(o.Name).Do(context.TODO())
	// Not handling this as this is integration test
	_ = result.Into(crStatus)

	return crStatus.Status.Type
}

func (o *Orderer) PollForCRStatus() current.IBPCRStatusType {
	crStatus := &current.IBPOrderer{}

	result := o.CRClient.Get().Namespace(o.Namespace).Resource("ibporderers").Name(o.NodeName).Do(context.TODO())
	// Not handling this as this is integration test
	_ = result.Into(crStatus)

	return crStatus.Status.Type
}

func (o *Orderer) JobWithPrefixFound(prefix, namespace string) bool {
	jobs, err := o.KClient.BatchV1().Jobs(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false
	}

	for _, job := range jobs.Items {
		if strings.HasPrefix(job.GetName(), prefix) {
			return true
		}
	}

	return false
}
