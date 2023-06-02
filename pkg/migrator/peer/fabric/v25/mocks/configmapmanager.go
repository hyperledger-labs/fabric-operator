// Code generated by counterfeiter. DO NOT EDIT.
package mocks

import (
	"sync"

	"github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	v25 "github.com/IBM-Blockchain/fabric-operator/pkg/migrator/peer/fabric/v25"
	v1 "k8s.io/api/core/v1"
)

type ConfigMapManager struct {
	CreateOrUpdateStub        func(*v1beta1.IBPPeer, initializer.CoreConfig) error
	createOrUpdateMutex       sync.RWMutex
	createOrUpdateArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
		arg2 initializer.CoreConfig
	}
	createOrUpdateReturns struct {
		result1 error
	}
	createOrUpdateReturnsOnCall map[int]struct {
		result1 error
	}
	GetCoreConfigStub        func(*v1beta1.IBPPeer) (*v1.ConfigMap, error)
	getCoreConfigMutex       sync.RWMutex
	getCoreConfigArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
	}
	getCoreConfigReturns struct {
		result1 *v1.ConfigMap
		result2 error
	}
	getCoreConfigReturnsOnCall map[int]struct {
		result1 *v1.ConfigMap
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *ConfigMapManager) CreateOrUpdate(arg1 *v1beta1.IBPPeer, arg2 initializer.CoreConfig) error {
	fake.createOrUpdateMutex.Lock()
	ret, specificReturn := fake.createOrUpdateReturnsOnCall[len(fake.createOrUpdateArgsForCall)]
	fake.createOrUpdateArgsForCall = append(fake.createOrUpdateArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
		arg2 initializer.CoreConfig
	}{arg1, arg2})
	stub := fake.CreateOrUpdateStub
	fakeReturns := fake.createOrUpdateReturns
	fake.recordInvocation("CreateOrUpdate", []interface{}{arg1, arg2})
	fake.createOrUpdateMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *ConfigMapManager) CreateOrUpdateCallCount() int {
	fake.createOrUpdateMutex.RLock()
	defer fake.createOrUpdateMutex.RUnlock()
	return len(fake.createOrUpdateArgsForCall)
}

func (fake *ConfigMapManager) CreateOrUpdateCalls(stub func(*v1beta1.IBPPeer, initializer.CoreConfig) error) {
	fake.createOrUpdateMutex.Lock()
	defer fake.createOrUpdateMutex.Unlock()
	fake.CreateOrUpdateStub = stub
}

func (fake *ConfigMapManager) CreateOrUpdateArgsForCall(i int) (*v1beta1.IBPPeer, initializer.CoreConfig) {
	fake.createOrUpdateMutex.RLock()
	defer fake.createOrUpdateMutex.RUnlock()
	argsForCall := fake.createOrUpdateArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *ConfigMapManager) CreateOrUpdateReturns(result1 error) {
	fake.createOrUpdateMutex.Lock()
	defer fake.createOrUpdateMutex.Unlock()
	fake.CreateOrUpdateStub = nil
	fake.createOrUpdateReturns = struct {
		result1 error
	}{result1}
}

func (fake *ConfigMapManager) CreateOrUpdateReturnsOnCall(i int, result1 error) {
	fake.createOrUpdateMutex.Lock()
	defer fake.createOrUpdateMutex.Unlock()
	fake.CreateOrUpdateStub = nil
	if fake.createOrUpdateReturnsOnCall == nil {
		fake.createOrUpdateReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.createOrUpdateReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *ConfigMapManager) GetCoreConfig(arg1 *v1beta1.IBPPeer) (*v1.ConfigMap, error) {
	fake.getCoreConfigMutex.Lock()
	ret, specificReturn := fake.getCoreConfigReturnsOnCall[len(fake.getCoreConfigArgsForCall)]
	fake.getCoreConfigArgsForCall = append(fake.getCoreConfigArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
	}{arg1})
	stub := fake.GetCoreConfigStub
	fakeReturns := fake.getCoreConfigReturns
	fake.recordInvocation("GetCoreConfig", []interface{}{arg1})
	fake.getCoreConfigMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *ConfigMapManager) GetCoreConfigCallCount() int {
	fake.getCoreConfigMutex.RLock()
	defer fake.getCoreConfigMutex.RUnlock()
	return len(fake.getCoreConfigArgsForCall)
}

func (fake *ConfigMapManager) GetCoreConfigCalls(stub func(*v1beta1.IBPPeer) (*v1.ConfigMap, error)) {
	fake.getCoreConfigMutex.Lock()
	defer fake.getCoreConfigMutex.Unlock()
	fake.GetCoreConfigStub = stub
}

func (fake *ConfigMapManager) GetCoreConfigArgsForCall(i int) *v1beta1.IBPPeer {
	fake.getCoreConfigMutex.RLock()
	defer fake.getCoreConfigMutex.RUnlock()
	argsForCall := fake.getCoreConfigArgsForCall[i]
	return argsForCall.arg1
}

func (fake *ConfigMapManager) GetCoreConfigReturns(result1 *v1.ConfigMap, result2 error) {
	fake.getCoreConfigMutex.Lock()
	defer fake.getCoreConfigMutex.Unlock()
	fake.GetCoreConfigStub = nil
	fake.getCoreConfigReturns = struct {
		result1 *v1.ConfigMap
		result2 error
	}{result1, result2}
}

func (fake *ConfigMapManager) GetCoreConfigReturnsOnCall(i int, result1 *v1.ConfigMap, result2 error) {
	fake.getCoreConfigMutex.Lock()
	defer fake.getCoreConfigMutex.Unlock()
	fake.GetCoreConfigStub = nil
	if fake.getCoreConfigReturnsOnCall == nil {
		fake.getCoreConfigReturnsOnCall = make(map[int]struct {
			result1 *v1.ConfigMap
			result2 error
		})
	}
	fake.getCoreConfigReturnsOnCall[i] = struct {
		result1 *v1.ConfigMap
		result2 error
	}{result1, result2}
}

func (fake *ConfigMapManager) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createOrUpdateMutex.RLock()
	defer fake.createOrUpdateMutex.RUnlock()
	fake.getCoreConfigMutex.RLock()
	defer fake.getCoreConfigMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *ConfigMapManager) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ v25.ConfigMapManager = new(ConfigMapManager)
