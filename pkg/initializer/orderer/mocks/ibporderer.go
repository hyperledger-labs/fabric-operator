// Code generated by counterfeiter. DO NOT EDIT.
package mocks

import (
	"sync"

	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer"
)

type IBPOrderer struct {
	GenerateCryptoStub        func() (*config.CryptoResponse, error)
	generateCryptoMutex       sync.RWMutex
	generateCryptoArgsForCall []struct {
	}
	generateCryptoReturns struct {
		result1 *config.CryptoResponse
		result2 error
	}
	generateCryptoReturnsOnCall map[int]struct {
		result1 *config.CryptoResponse
		result2 error
	}
	GetConfigStub        func() initializer.OrdererConfig
	getConfigMutex       sync.RWMutex
	getConfigArgsForCall []struct {
	}
	getConfigReturns struct {
		result1 initializer.OrdererConfig
	}
	getConfigReturnsOnCall map[int]struct {
		result1 initializer.OrdererConfig
	}
	OverrideConfigStub        func(initializer.OrdererConfig) error
	overrideConfigMutex       sync.RWMutex
	overrideConfigArgsForCall []struct {
		arg1 initializer.OrdererConfig
	}
	overrideConfigReturns struct {
		result1 error
	}
	overrideConfigReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *IBPOrderer) GenerateCrypto() (*config.CryptoResponse, error) {
	fake.generateCryptoMutex.Lock()
	ret, specificReturn := fake.generateCryptoReturnsOnCall[len(fake.generateCryptoArgsForCall)]
	fake.generateCryptoArgsForCall = append(fake.generateCryptoArgsForCall, struct {
	}{})
	fake.recordInvocation("GenerateCrypto", []interface{}{})
	fake.generateCryptoMutex.Unlock()
	if fake.GenerateCryptoStub != nil {
		return fake.GenerateCryptoStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.generateCryptoReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *IBPOrderer) GenerateCryptoCallCount() int {
	fake.generateCryptoMutex.RLock()
	defer fake.generateCryptoMutex.RUnlock()
	return len(fake.generateCryptoArgsForCall)
}

func (fake *IBPOrderer) GenerateCryptoCalls(stub func() (*config.CryptoResponse, error)) {
	fake.generateCryptoMutex.Lock()
	defer fake.generateCryptoMutex.Unlock()
	fake.GenerateCryptoStub = stub
}

func (fake *IBPOrderer) GenerateCryptoReturns(result1 *config.CryptoResponse, result2 error) {
	fake.generateCryptoMutex.Lock()
	defer fake.generateCryptoMutex.Unlock()
	fake.GenerateCryptoStub = nil
	fake.generateCryptoReturns = struct {
		result1 *config.CryptoResponse
		result2 error
	}{result1, result2}
}

func (fake *IBPOrderer) GenerateCryptoReturnsOnCall(i int, result1 *config.CryptoResponse, result2 error) {
	fake.generateCryptoMutex.Lock()
	defer fake.generateCryptoMutex.Unlock()
	fake.GenerateCryptoStub = nil
	if fake.generateCryptoReturnsOnCall == nil {
		fake.generateCryptoReturnsOnCall = make(map[int]struct {
			result1 *config.CryptoResponse
			result2 error
		})
	}
	fake.generateCryptoReturnsOnCall[i] = struct {
		result1 *config.CryptoResponse
		result2 error
	}{result1, result2}
}

func (fake *IBPOrderer) GetConfig() initializer.OrdererConfig {
	fake.getConfigMutex.Lock()
	ret, specificReturn := fake.getConfigReturnsOnCall[len(fake.getConfigArgsForCall)]
	fake.getConfigArgsForCall = append(fake.getConfigArgsForCall, struct {
	}{})
	fake.recordInvocation("GetConfig", []interface{}{})
	fake.getConfigMutex.Unlock()
	if fake.GetConfigStub != nil {
		return fake.GetConfigStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.getConfigReturns
	return fakeReturns.result1
}

func (fake *IBPOrderer) GetConfigCallCount() int {
	fake.getConfigMutex.RLock()
	defer fake.getConfigMutex.RUnlock()
	return len(fake.getConfigArgsForCall)
}

func (fake *IBPOrderer) GetConfigCalls(stub func() initializer.OrdererConfig) {
	fake.getConfigMutex.Lock()
	defer fake.getConfigMutex.Unlock()
	fake.GetConfigStub = stub
}

func (fake *IBPOrderer) GetConfigReturns(result1 initializer.OrdererConfig) {
	fake.getConfigMutex.Lock()
	defer fake.getConfigMutex.Unlock()
	fake.GetConfigStub = nil
	fake.getConfigReturns = struct {
		result1 initializer.OrdererConfig
	}{result1}
}

func (fake *IBPOrderer) GetConfigReturnsOnCall(i int, result1 initializer.OrdererConfig) {
	fake.getConfigMutex.Lock()
	defer fake.getConfigMutex.Unlock()
	fake.GetConfigStub = nil
	if fake.getConfigReturnsOnCall == nil {
		fake.getConfigReturnsOnCall = make(map[int]struct {
			result1 initializer.OrdererConfig
		})
	}
	fake.getConfigReturnsOnCall[i] = struct {
		result1 initializer.OrdererConfig
	}{result1}
}

func (fake *IBPOrderer) OverrideConfig(arg1 initializer.OrdererConfig) error {
	fake.overrideConfigMutex.Lock()
	ret, specificReturn := fake.overrideConfigReturnsOnCall[len(fake.overrideConfigArgsForCall)]
	fake.overrideConfigArgsForCall = append(fake.overrideConfigArgsForCall, struct {
		arg1 initializer.OrdererConfig
	}{arg1})
	fake.recordInvocation("OverrideConfig", []interface{}{arg1})
	fake.overrideConfigMutex.Unlock()
	if fake.OverrideConfigStub != nil {
		return fake.OverrideConfigStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.overrideConfigReturns
	return fakeReturns.result1
}

func (fake *IBPOrderer) OverrideConfigCallCount() int {
	fake.overrideConfigMutex.RLock()
	defer fake.overrideConfigMutex.RUnlock()
	return len(fake.overrideConfigArgsForCall)
}

func (fake *IBPOrderer) OverrideConfigCalls(stub func(initializer.OrdererConfig) error) {
	fake.overrideConfigMutex.Lock()
	defer fake.overrideConfigMutex.Unlock()
	fake.OverrideConfigStub = stub
}

func (fake *IBPOrderer) OverrideConfigArgsForCall(i int) initializer.OrdererConfig {
	fake.overrideConfigMutex.RLock()
	defer fake.overrideConfigMutex.RUnlock()
	argsForCall := fake.overrideConfigArgsForCall[i]
	return argsForCall.arg1
}

func (fake *IBPOrderer) OverrideConfigReturns(result1 error) {
	fake.overrideConfigMutex.Lock()
	defer fake.overrideConfigMutex.Unlock()
	fake.OverrideConfigStub = nil
	fake.overrideConfigReturns = struct {
		result1 error
	}{result1}
}

func (fake *IBPOrderer) OverrideConfigReturnsOnCall(i int, result1 error) {
	fake.overrideConfigMutex.Lock()
	defer fake.overrideConfigMutex.Unlock()
	fake.OverrideConfigStub = nil
	if fake.overrideConfigReturnsOnCall == nil {
		fake.overrideConfigReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.overrideConfigReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *IBPOrderer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.generateCryptoMutex.RLock()
	defer fake.generateCryptoMutex.RUnlock()
	fake.getConfigMutex.RLock()
	defer fake.getConfigMutex.RUnlock()
	fake.overrideConfigMutex.RLock()
	defer fake.overrideConfigMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *IBPOrderer) recordInvocation(key string, args []interface{}) {
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

var _ initializer.IBPOrderer = new(IBPOrderer)
