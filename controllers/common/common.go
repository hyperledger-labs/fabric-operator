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

package common

import (
	"context"
	"fmt"
	"reflect"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IBPCA      = "IBPCA"
	IBPPEER    = "IBPPeer"
	IBPORDERER = "IBPOrderer"
	IBPCONSOLE = "IBPConsole"
)

type Client interface {
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

//  1. Only one existing instance (of the same type as 'instance') should have
//     the name 'instance.GetName()'; if more than one is present, return error
//  2. If any instance of a different type share the same name, return error
func ValidateCRName(k8sclient Client, name, namespace, kind string) error {
	listOptions := &client.ListOptions{
		Namespace: namespace,
	}

	count := 0

	caList := &current.IBPCAList{}
	err := k8sclient.List(context.TODO(), caList, listOptions)
	if err != nil {
		return err
	}
	for _, ca := range caList.Items {
		if name == ca.Name {
			if kind == IBPCA {
				count++
			} else {
				return fmt.Errorf("custom resource with name '%s' already exists", name)
			}
		}
	}

	ordererList := &current.IBPOrdererList{}
	err = k8sclient.List(context.TODO(), ordererList, listOptions)
	if err != nil {
		return err
	}
	for _, o := range ordererList.Items {
		if name == o.Name {
			if kind == IBPORDERER {
				count++
			} else {
				return fmt.Errorf("custom resource with name %s already exists", name)
			}
		}
	}

	peerList := &current.IBPPeerList{}
	err = k8sclient.List(context.TODO(), peerList, listOptions)
	if err != nil {
		return err
	}
	for _, p := range peerList.Items {
		if name == p.Name {
			if kind == IBPPEER {
				count++
			} else {
				return fmt.Errorf("custom resource with name %s already exists", name)
			}
		}
	}

	consoleList := &current.IBPConsoleList{}
	err = k8sclient.List(context.TODO(), consoleList, listOptions)
	if err != nil {
		return err
	}
	for _, c := range consoleList.Items {
		if name == c.Name {
			if kind == IBPCONSOLE {
				count++
			} else {
				return fmt.Errorf("custom resource with name %s already exists", name)
			}
		}
	}

	if count > 1 {
		return fmt.Errorf("custom resource with name %s already exists", name)
	}

	return nil
}

func MSPInfoUpdateDetected(oldSecret, newSecret *current.SecretSpec) bool {
	if newSecret == nil || newSecret.MSP == nil {
		return false
	}

	if oldSecret == nil || oldSecret.MSP == nil {
		if newSecret.MSP.Component != nil || newSecret.MSP.TLS != nil || newSecret.MSP.ClientAuth != nil {
			return true
		}
	} else {
		// For comparison purpose ignoring admin certs - admin cert updates
		// detected in Initialize() code
		if oldSecret.MSP.Component != nil && newSecret.MSP.Component != nil {
			oldSecret.MSP.Component.AdminCerts = newSecret.MSP.Component.AdminCerts
		}
		if oldSecret.MSP.TLS != nil && newSecret.MSP.TLS != nil {
			oldSecret.MSP.TLS.AdminCerts = newSecret.MSP.TLS.AdminCerts
		}
		if oldSecret.MSP.ClientAuth != nil && newSecret.MSP.ClientAuth != nil {
			oldSecret.MSP.ClientAuth.AdminCerts = newSecret.MSP.ClientAuth.AdminCerts
		}

		return !reflect.DeepEqual(oldSecret.MSP, newSecret.MSP)
	}

	return false
}
