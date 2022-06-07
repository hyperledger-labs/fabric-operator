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

package clientset

import (
	"context"
	"fmt"
	"strings"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("clientset")

type Client struct {
	clientset.Clientset
}

func New(config *rest.Config) (*Client, error) {
	clientSet, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Client{*clientSet}, nil
}

func (c *Client) CreateCRD(crd *extv1.CustomResourceDefinition) (*extv1.CustomResourceDefinition, error) {
	log.Info(fmt.Sprintf("Creating CRD '%s'", crd.Name))
	result, err := c.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), crd, v1.CreateOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			existingcrd, err := c.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), crd.Name, v1.GetOptions{})
			if err != nil {
				return nil, err
			}

			log.Info(fmt.Sprintf("Updating CRD '%s'", crd.Name))
			existingcrd.Spec = crd.Spec
			result, err = c.ApiextensionsV1().CustomResourceDefinitions().Update(context.TODO(), existingcrd, v1.UpdateOptions{})
			if err != nil {
				return nil, err
			}
		} else {
			log.Error(err, "Error creating CRD", "CRD", crd.Name)
		}
	}
	return result, nil
}
