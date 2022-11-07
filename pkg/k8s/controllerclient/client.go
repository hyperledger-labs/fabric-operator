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

package controllerclient

import (
	"context"
	"time"

	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/pkg/errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:generate counterfeiter -o ../../controller/mocks/client.go -fake-name Client . Client

type Client interface {
	Get(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object) error
	List(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error
	Create(ctx context.Context, obj k8sclient.Object, opts ...CreateOption) error
	CreateOrUpdate(ctx context.Context, obj k8sclient.Object, opts ...CreateOrUpdateOption) error
	Delete(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error
	Patch(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...PatchOption) error
	PatchStatus(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...PatchOption) error
	Update(ctx context.Context, obj k8sclient.Object, opts ...UpdateOption) error
	UpdateStatus(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error
}

// GlobalConfig applies the global configuration defined in operator's config to appropriate
// kubernetes resources
type GlobalConfig interface {
	Apply(runtime.Object)
}

type ClientImpl struct {
	k8sClient    k8sclient.Client
	GlobalConfig GlobalConfig
}

func New(c k8sclient.Client, gc GlobalConfig) *ClientImpl {
	return &ClientImpl{
		k8sClient:    c,
		GlobalConfig: gc,
	}
}

func (c *ClientImpl) Get(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object) error {
	err := c.k8sClient.Get(ctx, key, obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClientImpl) List(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
	err := c.k8sClient.List(ctx, list, opts...)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClientImpl) Create(ctx context.Context, obj k8sclient.Object, opts ...CreateOption) error {
	var createOpts []k8sclient.CreateOption

	c.GlobalConfig.Apply(obj)

	if len(opts) > 0 {
		if err := setControllerReference(opts[0].Owner, obj, opts[0].Scheme); err != nil {
			return err
		}
		createOpts = opts[0].Opts
	}

	err := c.k8sClient.Create(ctx, obj, createOpts...)
	if err != nil {
		return util.IgnoreAlreadyExistError(err)
	}
	return nil
}

func (c *ClientImpl) Patch(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...PatchOption) error {
	var patchOpts []k8sclient.PatchOption

	c.GlobalConfig.Apply(obj)

	if len(opts) > 0 {
		if opts[0].Resilient != nil {
			return c.ResilientPatch(ctx, obj, opts[0].Resilient, opts[0].Opts...)
		}

		patchOpts = opts[0].Opts
	}

	err := c.k8sClient.Patch(ctx, obj, patch, patchOpts...)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClientImpl) ResilientPatch(ctx context.Context, obj k8sclient.Object, resilient *ResilientPatch, opts ...k8sclient.PatchOption) error {
	retry := resilient.Retry
	into := resilient.Into
	strategy := resilient.Strategy

	c.GlobalConfig.Apply(obj)

	for i := 0; i < retry; i++ {
		err := c.resilientPatch(ctx, obj, strategy, into, opts...)
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

func (c *ClientImpl) resilientPatch(ctx context.Context, obj k8sclient.Object, strategy func(k8sclient.Object) k8sclient.Patch, into k8sclient.Object, opts ...k8sclient.PatchOption) error {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	err := c.Get(ctx, key, into)
	if err != nil {
		return err
	}

	err = c.k8sClient.Patch(ctx, obj, strategy(into), opts...)
	if err != nil {
		return err
	}
	return nil
}

// If utilizing resilient option, nil can be passed for patch parameter
func (c *ClientImpl) PatchStatus(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...PatchOption) error {
	var patchOpts []k8sclient.PatchOption

	if len(opts) > 0 {
		if opts[0].Resilient != nil {
			return c.ResilientPatchStatus(ctx, obj, opts[0].Resilient, opts[0].Opts...)
		}

		patchOpts = opts[0].Opts
	}

	err := c.k8sClient.Status().Patch(ctx, obj, patch, patchOpts...)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClientImpl) ResilientPatchStatus(ctx context.Context, obj k8sclient.Object, resilient *ResilientPatch, opts ...k8sclient.PatchOption) error {
	retry := resilient.Retry
	into := resilient.Into
	strategy := resilient.Strategy

	for i := 0; i < retry; i++ {
		err := c.resilientPatchStatus(ctx, obj, strategy, into, opts...)
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

func (c *ClientImpl) resilientPatchStatus(ctx context.Context, obj k8sclient.Object, strategy func(k8sclient.Object) k8sclient.Patch, into k8sclient.Object, opts ...k8sclient.PatchOption) error {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	err := c.Get(ctx, key, into)
	if err != nil {
		return err
	}

	err = c.k8sClient.Status().Patch(ctx, obj, strategy(into), opts...)
	if err != nil {
		return err
	}
	return nil
}

// NOTE: Currently, Resilient Update is not supported as it requires more specific
// implementation based on scenario. When possible, should utilize resilient Patch.
func (c *ClientImpl) Update(ctx context.Context, obj k8sclient.Object, opts ...UpdateOption) error {
	var updateOpts []k8sclient.UpdateOption

	c.GlobalConfig.Apply(obj)

	if len(opts) > 0 {
		if err := setControllerReference(opts[0].Owner, obj, opts[0].Scheme); err != nil {
			return err
		}
		updateOpts = opts[0].Opts
	}

	err := c.k8sClient.Update(ctx, obj, updateOpts...)
	if err != nil {
		return err
	}
	return nil
}

// NOTE: Currently, Resilient UpdateStatus is not supported as it requires more specific
// implementation based on scenario. When possible, should utilize resilient PatchStatus.
func (c *ClientImpl) UpdateStatus(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
	err := c.k8sClient.Status().Update(ctx, obj, opts...)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClientImpl) Delete(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
	err := c.k8sClient.Delete(ctx, obj, opts...)
	if err != nil {
		return err
	}
	return nil
}

// CreateOrUpdate does not support k8sclient.CreateOption or k8sclient.UpdateOption being passed as variadic parameters,
// if want to use opts use Create or Update methods
func (c *ClientImpl) CreateOrUpdate(ctx context.Context, obj k8sclient.Object, opts ...CreateOrUpdateOption) error {
	if len(opts) > 0 {
		if err := setControllerReference(opts[0].Owner, obj, opts[0].Scheme); err != nil {
			return err
		}
	}

	c.GlobalConfig.Apply(obj)

	err := c.k8sClient.Create(ctx, obj)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return c.k8sClient.Update(ctx, obj)
		}
		return err
	}
	return nil
}

func setControllerReference(owner v1.Object, obj v1.Object, scheme *runtime.Scheme) error {
	if owner != nil && obj != nil && scheme != nil {
		err := controllerutil.SetControllerReference(owner, obj, scheme)
		if err != nil {
			if _, ok := err.(*controllerutil.AlreadyOwnedError); ok {
				return nil
			}
			return errors.Wrap(err, "controller reference error")
		}
	}

	return nil
}
