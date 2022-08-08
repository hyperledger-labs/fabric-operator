// Code generated by counterfeiter. DO NOT EDIT.
package mocks

import (
	"sync"

	"github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/common/config"
	initializer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/peer"
	basepeer "github.com/IBM-Blockchain/fabric-operator/pkg/offering/base/peer"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InitializeIBPPeer struct {
	CheckIfAdminCertsUpdatedStub        func(*v1beta1.IBPPeer) (bool, error)
	checkIfAdminCertsUpdatedMutex       sync.RWMutex
	checkIfAdminCertsUpdatedArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
	}
	checkIfAdminCertsUpdatedReturns struct {
		result1 bool
		result2 error
	}
	checkIfAdminCertsUpdatedReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	CoreConfigMapStub        func() *initializer.CoreConfigMap
	coreConfigMapMutex       sync.RWMutex
	coreConfigMapArgsForCall []struct {
	}
	coreConfigMapReturns struct {
		result1 *initializer.CoreConfigMap
	}
	coreConfigMapReturnsOnCall map[int]struct {
		result1 *initializer.CoreConfigMap
	}
	CreateStub        func(initializer.CoreConfig, initializer.IBPPeer, string) (*initializer.Response, error)
	createMutex       sync.RWMutex
	createArgsForCall []struct {
		arg1 initializer.CoreConfig
		arg2 initializer.IBPPeer
		arg3 string
	}
	createReturns struct {
		result1 *initializer.Response
		result2 error
	}
	createReturnsOnCall map[int]struct {
		result1 *initializer.Response
		result2 error
	}
	GenerateOrdererCACertsSecretStub        func(*v1beta1.IBPPeer, map[string][]byte) error
	generateOrdererCACertsSecretMutex       sync.RWMutex
	generateOrdererCACertsSecretArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
		arg2 map[string][]byte
	}
	generateOrdererCACertsSecretReturns struct {
		result1 error
	}
	generateOrdererCACertsSecretReturnsOnCall map[int]struct {
		result1 error
	}
	GenerateSecretsStub        func(common.SecretType, v1.Object, *config.Response) error
	generateSecretsMutex       sync.RWMutex
	generateSecretsArgsForCall []struct {
		arg1 common.SecretType
		arg2 v1.Object
		arg3 *config.Response
	}
	generateSecretsReturns struct {
		result1 error
	}
	generateSecretsReturnsOnCall map[int]struct {
		result1 error
	}
	GenerateSecretsFromResponseStub        func(*v1beta1.IBPPeer, *config.CryptoResponse) error
	generateSecretsFromResponseMutex       sync.RWMutex
	generateSecretsFromResponseArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
		arg2 *config.CryptoResponse
	}
	generateSecretsFromResponseReturns struct {
		result1 error
	}
	generateSecretsFromResponseReturnsOnCall map[int]struct {
		result1 error
	}
	GetCryptoStub        func(*v1beta1.IBPPeer) (*config.CryptoResponse, error)
	getCryptoMutex       sync.RWMutex
	getCryptoArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
	}
	getCryptoReturns struct {
		result1 *config.CryptoResponse
		result2 error
	}
	getCryptoReturnsOnCall map[int]struct {
		result1 *config.CryptoResponse
		result2 error
	}
	GetInitPeerStub        func(*v1beta1.IBPPeer, string) (*initializer.Peer, error)
	getInitPeerMutex       sync.RWMutex
	getInitPeerArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
		arg2 string
	}
	getInitPeerReturns struct {
		result1 *initializer.Peer
		result2 error
	}
	getInitPeerReturnsOnCall map[int]struct {
		result1 *initializer.Peer
		result2 error
	}
	GetUpdatedPeerStub        func(*v1beta1.IBPPeer) (*initializer.Peer, error)
	getUpdatedPeerMutex       sync.RWMutex
	getUpdatedPeerArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
	}
	getUpdatedPeerReturns struct {
		result1 *initializer.Peer
		result2 error
	}
	getUpdatedPeerReturnsOnCall map[int]struct {
		result1 *initializer.Peer
		result2 error
	}
	MissingCryptoStub        func(*v1beta1.IBPPeer) bool
	missingCryptoMutex       sync.RWMutex
	missingCryptoArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
	}
	missingCryptoReturns struct {
		result1 bool
	}
	missingCryptoReturnsOnCall map[int]struct {
		result1 bool
	}
	UpdateStub        func(initializer.CoreConfig, initializer.IBPPeer) (*initializer.Response, error)
	updateMutex       sync.RWMutex
	updateArgsForCall []struct {
		arg1 initializer.CoreConfig
		arg2 initializer.IBPPeer
	}
	updateReturns struct {
		result1 *initializer.Response
		result2 error
	}
	updateReturnsOnCall map[int]struct {
		result1 *initializer.Response
		result2 error
	}
	UpdateAdminSecretStub        func(*v1beta1.IBPPeer) error
	updateAdminSecretMutex       sync.RWMutex
	updateAdminSecretArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
	}
	updateAdminSecretReturns struct {
		result1 error
	}
	updateAdminSecretReturnsOnCall map[int]struct {
		result1 error
	}
	UpdateSecretsFromResponseStub        func(*v1beta1.IBPPeer, *config.CryptoResponse) error
	updateSecretsFromResponseMutex       sync.RWMutex
	updateSecretsFromResponseArgsForCall []struct {
		arg1 *v1beta1.IBPPeer
		arg2 *config.CryptoResponse
	}
	updateSecretsFromResponseReturns struct {
		result1 error
	}
	updateSecretsFromResponseReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *InitializeIBPPeer) CheckIfAdminCertsUpdated(arg1 *v1beta1.IBPPeer) (bool, error) {
	fake.checkIfAdminCertsUpdatedMutex.Lock()
	ret, specificReturn := fake.checkIfAdminCertsUpdatedReturnsOnCall[len(fake.checkIfAdminCertsUpdatedArgsForCall)]
	fake.checkIfAdminCertsUpdatedArgsForCall = append(fake.checkIfAdminCertsUpdatedArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
	}{arg1})
	stub := fake.CheckIfAdminCertsUpdatedStub
	fakeReturns := fake.checkIfAdminCertsUpdatedReturns
	fake.recordInvocation("CheckIfAdminCertsUpdated", []interface{}{arg1})
	fake.checkIfAdminCertsUpdatedMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *InitializeIBPPeer) CheckIfAdminCertsUpdatedCallCount() int {
	fake.checkIfAdminCertsUpdatedMutex.RLock()
	defer fake.checkIfAdminCertsUpdatedMutex.RUnlock()
	return len(fake.checkIfAdminCertsUpdatedArgsForCall)
}

func (fake *InitializeIBPPeer) CheckIfAdminCertsUpdatedCalls(stub func(*v1beta1.IBPPeer) (bool, error)) {
	fake.checkIfAdminCertsUpdatedMutex.Lock()
	defer fake.checkIfAdminCertsUpdatedMutex.Unlock()
	fake.CheckIfAdminCertsUpdatedStub = stub
}

func (fake *InitializeIBPPeer) CheckIfAdminCertsUpdatedArgsForCall(i int) *v1beta1.IBPPeer {
	fake.checkIfAdminCertsUpdatedMutex.RLock()
	defer fake.checkIfAdminCertsUpdatedMutex.RUnlock()
	argsForCall := fake.checkIfAdminCertsUpdatedArgsForCall[i]
	return argsForCall.arg1
}

func (fake *InitializeIBPPeer) CheckIfAdminCertsUpdatedReturns(result1 bool, result2 error) {
	fake.checkIfAdminCertsUpdatedMutex.Lock()
	defer fake.checkIfAdminCertsUpdatedMutex.Unlock()
	fake.CheckIfAdminCertsUpdatedStub = nil
	fake.checkIfAdminCertsUpdatedReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) CheckIfAdminCertsUpdatedReturnsOnCall(i int, result1 bool, result2 error) {
	fake.checkIfAdminCertsUpdatedMutex.Lock()
	defer fake.checkIfAdminCertsUpdatedMutex.Unlock()
	fake.CheckIfAdminCertsUpdatedStub = nil
	if fake.checkIfAdminCertsUpdatedReturnsOnCall == nil {
		fake.checkIfAdminCertsUpdatedReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.checkIfAdminCertsUpdatedReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) CoreConfigMap() *initializer.CoreConfigMap {
	fake.coreConfigMapMutex.Lock()
	ret, specificReturn := fake.coreConfigMapReturnsOnCall[len(fake.coreConfigMapArgsForCall)]
	fake.coreConfigMapArgsForCall = append(fake.coreConfigMapArgsForCall, struct {
	}{})
	stub := fake.CoreConfigMapStub
	fakeReturns := fake.coreConfigMapReturns
	fake.recordInvocation("CoreConfigMap", []interface{}{})
	fake.coreConfigMapMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *InitializeIBPPeer) CoreConfigMapCallCount() int {
	fake.coreConfigMapMutex.RLock()
	defer fake.coreConfigMapMutex.RUnlock()
	return len(fake.coreConfigMapArgsForCall)
}

func (fake *InitializeIBPPeer) CoreConfigMapCalls(stub func() *initializer.CoreConfigMap) {
	fake.coreConfigMapMutex.Lock()
	defer fake.coreConfigMapMutex.Unlock()
	fake.CoreConfigMapStub = stub
}

func (fake *InitializeIBPPeer) CoreConfigMapReturns(result1 *initializer.CoreConfigMap) {
	fake.coreConfigMapMutex.Lock()
	defer fake.coreConfigMapMutex.Unlock()
	fake.CoreConfigMapStub = nil
	fake.coreConfigMapReturns = struct {
		result1 *initializer.CoreConfigMap
	}{result1}
}

func (fake *InitializeIBPPeer) CoreConfigMapReturnsOnCall(i int, result1 *initializer.CoreConfigMap) {
	fake.coreConfigMapMutex.Lock()
	defer fake.coreConfigMapMutex.Unlock()
	fake.CoreConfigMapStub = nil
	if fake.coreConfigMapReturnsOnCall == nil {
		fake.coreConfigMapReturnsOnCall = make(map[int]struct {
			result1 *initializer.CoreConfigMap
		})
	}
	fake.coreConfigMapReturnsOnCall[i] = struct {
		result1 *initializer.CoreConfigMap
	}{result1}
}

func (fake *InitializeIBPPeer) Create(arg1 initializer.CoreConfig, arg2 initializer.IBPPeer, arg3 string) (*initializer.Response, error) {
	fake.createMutex.Lock()
	ret, specificReturn := fake.createReturnsOnCall[len(fake.createArgsForCall)]
	fake.createArgsForCall = append(fake.createArgsForCall, struct {
		arg1 initializer.CoreConfig
		arg2 initializer.IBPPeer
		arg3 string
	}{arg1, arg2, arg3})
	stub := fake.CreateStub
	fakeReturns := fake.createReturns
	fake.recordInvocation("Create", []interface{}{arg1, arg2, arg3})
	fake.createMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *InitializeIBPPeer) CreateCallCount() int {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return len(fake.createArgsForCall)
}

func (fake *InitializeIBPPeer) CreateCalls(stub func(initializer.CoreConfig, initializer.IBPPeer, string) (*initializer.Response, error)) {
	fake.createMutex.Lock()
	defer fake.createMutex.Unlock()
	fake.CreateStub = stub
}

func (fake *InitializeIBPPeer) CreateArgsForCall(i int) (initializer.CoreConfig, initializer.IBPPeer, string) {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	argsForCall := fake.createArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *InitializeIBPPeer) CreateReturns(result1 *initializer.Response, result2 error) {
	fake.createMutex.Lock()
	defer fake.createMutex.Unlock()
	fake.CreateStub = nil
	fake.createReturns = struct {
		result1 *initializer.Response
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) CreateReturnsOnCall(i int, result1 *initializer.Response, result2 error) {
	fake.createMutex.Lock()
	defer fake.createMutex.Unlock()
	fake.CreateStub = nil
	if fake.createReturnsOnCall == nil {
		fake.createReturnsOnCall = make(map[int]struct {
			result1 *initializer.Response
			result2 error
		})
	}
	fake.createReturnsOnCall[i] = struct {
		result1 *initializer.Response
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) GenerateOrdererCACertsSecret(arg1 *v1beta1.IBPPeer, arg2 map[string][]byte) error {
	fake.generateOrdererCACertsSecretMutex.Lock()
	ret, specificReturn := fake.generateOrdererCACertsSecretReturnsOnCall[len(fake.generateOrdererCACertsSecretArgsForCall)]
	fake.generateOrdererCACertsSecretArgsForCall = append(fake.generateOrdererCACertsSecretArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
		arg2 map[string][]byte
	}{arg1, arg2})
	stub := fake.GenerateOrdererCACertsSecretStub
	fakeReturns := fake.generateOrdererCACertsSecretReturns
	fake.recordInvocation("GenerateOrdererCACertsSecret", []interface{}{arg1, arg2})
	fake.generateOrdererCACertsSecretMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *InitializeIBPPeer) GenerateOrdererCACertsSecretCallCount() int {
	fake.generateOrdererCACertsSecretMutex.RLock()
	defer fake.generateOrdererCACertsSecretMutex.RUnlock()
	return len(fake.generateOrdererCACertsSecretArgsForCall)
}

func (fake *InitializeIBPPeer) GenerateOrdererCACertsSecretCalls(stub func(*v1beta1.IBPPeer, map[string][]byte) error) {
	fake.generateOrdererCACertsSecretMutex.Lock()
	defer fake.generateOrdererCACertsSecretMutex.Unlock()
	fake.GenerateOrdererCACertsSecretStub = stub
}

func (fake *InitializeIBPPeer) GenerateOrdererCACertsSecretArgsForCall(i int) (*v1beta1.IBPPeer, map[string][]byte) {
	fake.generateOrdererCACertsSecretMutex.RLock()
	defer fake.generateOrdererCACertsSecretMutex.RUnlock()
	argsForCall := fake.generateOrdererCACertsSecretArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *InitializeIBPPeer) GenerateOrdererCACertsSecretReturns(result1 error) {
	fake.generateOrdererCACertsSecretMutex.Lock()
	defer fake.generateOrdererCACertsSecretMutex.Unlock()
	fake.GenerateOrdererCACertsSecretStub = nil
	fake.generateOrdererCACertsSecretReturns = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) GenerateOrdererCACertsSecretReturnsOnCall(i int, result1 error) {
	fake.generateOrdererCACertsSecretMutex.Lock()
	defer fake.generateOrdererCACertsSecretMutex.Unlock()
	fake.GenerateOrdererCACertsSecretStub = nil
	if fake.generateOrdererCACertsSecretReturnsOnCall == nil {
		fake.generateOrdererCACertsSecretReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.generateOrdererCACertsSecretReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) GenerateSecrets(arg1 common.SecretType, arg2 v1.Object, arg3 *config.Response) error {
	fake.generateSecretsMutex.Lock()
	ret, specificReturn := fake.generateSecretsReturnsOnCall[len(fake.generateSecretsArgsForCall)]
	fake.generateSecretsArgsForCall = append(fake.generateSecretsArgsForCall, struct {
		arg1 common.SecretType
		arg2 v1.Object
		arg3 *config.Response
	}{arg1, arg2, arg3})
	stub := fake.GenerateSecretsStub
	fakeReturns := fake.generateSecretsReturns
	fake.recordInvocation("GenerateSecrets", []interface{}{arg1, arg2, arg3})
	fake.generateSecretsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *InitializeIBPPeer) GenerateSecretsCallCount() int {
	fake.generateSecretsMutex.RLock()
	defer fake.generateSecretsMutex.RUnlock()
	return len(fake.generateSecretsArgsForCall)
}

func (fake *InitializeIBPPeer) GenerateSecretsCalls(stub func(common.SecretType, v1.Object, *config.Response) error) {
	fake.generateSecretsMutex.Lock()
	defer fake.generateSecretsMutex.Unlock()
	fake.GenerateSecretsStub = stub
}

func (fake *InitializeIBPPeer) GenerateSecretsArgsForCall(i int) (common.SecretType, v1.Object, *config.Response) {
	fake.generateSecretsMutex.RLock()
	defer fake.generateSecretsMutex.RUnlock()
	argsForCall := fake.generateSecretsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *InitializeIBPPeer) GenerateSecretsReturns(result1 error) {
	fake.generateSecretsMutex.Lock()
	defer fake.generateSecretsMutex.Unlock()
	fake.GenerateSecretsStub = nil
	fake.generateSecretsReturns = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) GenerateSecretsReturnsOnCall(i int, result1 error) {
	fake.generateSecretsMutex.Lock()
	defer fake.generateSecretsMutex.Unlock()
	fake.GenerateSecretsStub = nil
	if fake.generateSecretsReturnsOnCall == nil {
		fake.generateSecretsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.generateSecretsReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) GenerateSecretsFromResponse(arg1 *v1beta1.IBPPeer, arg2 *config.CryptoResponse) error {
	fake.generateSecretsFromResponseMutex.Lock()
	ret, specificReturn := fake.generateSecretsFromResponseReturnsOnCall[len(fake.generateSecretsFromResponseArgsForCall)]
	fake.generateSecretsFromResponseArgsForCall = append(fake.generateSecretsFromResponseArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
		arg2 *config.CryptoResponse
	}{arg1, arg2})
	stub := fake.GenerateSecretsFromResponseStub
	fakeReturns := fake.generateSecretsFromResponseReturns
	fake.recordInvocation("GenerateSecretsFromResponse", []interface{}{arg1, arg2})
	fake.generateSecretsFromResponseMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *InitializeIBPPeer) GenerateSecretsFromResponseCallCount() int {
	fake.generateSecretsFromResponseMutex.RLock()
	defer fake.generateSecretsFromResponseMutex.RUnlock()
	return len(fake.generateSecretsFromResponseArgsForCall)
}

func (fake *InitializeIBPPeer) GenerateSecretsFromResponseCalls(stub func(*v1beta1.IBPPeer, *config.CryptoResponse) error) {
	fake.generateSecretsFromResponseMutex.Lock()
	defer fake.generateSecretsFromResponseMutex.Unlock()
	fake.GenerateSecretsFromResponseStub = stub
}

func (fake *InitializeIBPPeer) GenerateSecretsFromResponseArgsForCall(i int) (*v1beta1.IBPPeer, *config.CryptoResponse) {
	fake.generateSecretsFromResponseMutex.RLock()
	defer fake.generateSecretsFromResponseMutex.RUnlock()
	argsForCall := fake.generateSecretsFromResponseArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *InitializeIBPPeer) GenerateSecretsFromResponseReturns(result1 error) {
	fake.generateSecretsFromResponseMutex.Lock()
	defer fake.generateSecretsFromResponseMutex.Unlock()
	fake.GenerateSecretsFromResponseStub = nil
	fake.generateSecretsFromResponseReturns = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) GenerateSecretsFromResponseReturnsOnCall(i int, result1 error) {
	fake.generateSecretsFromResponseMutex.Lock()
	defer fake.generateSecretsFromResponseMutex.Unlock()
	fake.GenerateSecretsFromResponseStub = nil
	if fake.generateSecretsFromResponseReturnsOnCall == nil {
		fake.generateSecretsFromResponseReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.generateSecretsFromResponseReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) GetCrypto(arg1 *v1beta1.IBPPeer) (*config.CryptoResponse, error) {
	fake.getCryptoMutex.Lock()
	ret, specificReturn := fake.getCryptoReturnsOnCall[len(fake.getCryptoArgsForCall)]
	fake.getCryptoArgsForCall = append(fake.getCryptoArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
	}{arg1})
	stub := fake.GetCryptoStub
	fakeReturns := fake.getCryptoReturns
	fake.recordInvocation("GetCrypto", []interface{}{arg1})
	fake.getCryptoMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *InitializeIBPPeer) GetCryptoCallCount() int {
	fake.getCryptoMutex.RLock()
	defer fake.getCryptoMutex.RUnlock()
	return len(fake.getCryptoArgsForCall)
}

func (fake *InitializeIBPPeer) GetCryptoCalls(stub func(*v1beta1.IBPPeer) (*config.CryptoResponse, error)) {
	fake.getCryptoMutex.Lock()
	defer fake.getCryptoMutex.Unlock()
	fake.GetCryptoStub = stub
}

func (fake *InitializeIBPPeer) GetCryptoArgsForCall(i int) *v1beta1.IBPPeer {
	fake.getCryptoMutex.RLock()
	defer fake.getCryptoMutex.RUnlock()
	argsForCall := fake.getCryptoArgsForCall[i]
	return argsForCall.arg1
}

func (fake *InitializeIBPPeer) GetCryptoReturns(result1 *config.CryptoResponse, result2 error) {
	fake.getCryptoMutex.Lock()
	defer fake.getCryptoMutex.Unlock()
	fake.GetCryptoStub = nil
	fake.getCryptoReturns = struct {
		result1 *config.CryptoResponse
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) GetCryptoReturnsOnCall(i int, result1 *config.CryptoResponse, result2 error) {
	fake.getCryptoMutex.Lock()
	defer fake.getCryptoMutex.Unlock()
	fake.GetCryptoStub = nil
	if fake.getCryptoReturnsOnCall == nil {
		fake.getCryptoReturnsOnCall = make(map[int]struct {
			result1 *config.CryptoResponse
			result2 error
		})
	}
	fake.getCryptoReturnsOnCall[i] = struct {
		result1 *config.CryptoResponse
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) GetInitPeer(arg1 *v1beta1.IBPPeer, arg2 string) (*initializer.Peer, error) {
	fake.getInitPeerMutex.Lock()
	ret, specificReturn := fake.getInitPeerReturnsOnCall[len(fake.getInitPeerArgsForCall)]
	fake.getInitPeerArgsForCall = append(fake.getInitPeerArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
		arg2 string
	}{arg1, arg2})
	stub := fake.GetInitPeerStub
	fakeReturns := fake.getInitPeerReturns
	fake.recordInvocation("GetInitPeer", []interface{}{arg1, arg2})
	fake.getInitPeerMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *InitializeIBPPeer) GetInitPeerCallCount() int {
	fake.getInitPeerMutex.RLock()
	defer fake.getInitPeerMutex.RUnlock()
	return len(fake.getInitPeerArgsForCall)
}

func (fake *InitializeIBPPeer) GetInitPeerCalls(stub func(*v1beta1.IBPPeer, string) (*initializer.Peer, error)) {
	fake.getInitPeerMutex.Lock()
	defer fake.getInitPeerMutex.Unlock()
	fake.GetInitPeerStub = stub
}

func (fake *InitializeIBPPeer) GetInitPeerArgsForCall(i int) (*v1beta1.IBPPeer, string) {
	fake.getInitPeerMutex.RLock()
	defer fake.getInitPeerMutex.RUnlock()
	argsForCall := fake.getInitPeerArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *InitializeIBPPeer) GetInitPeerReturns(result1 *initializer.Peer, result2 error) {
	fake.getInitPeerMutex.Lock()
	defer fake.getInitPeerMutex.Unlock()
	fake.GetInitPeerStub = nil
	fake.getInitPeerReturns = struct {
		result1 *initializer.Peer
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) GetInitPeerReturnsOnCall(i int, result1 *initializer.Peer, result2 error) {
	fake.getInitPeerMutex.Lock()
	defer fake.getInitPeerMutex.Unlock()
	fake.GetInitPeerStub = nil
	if fake.getInitPeerReturnsOnCall == nil {
		fake.getInitPeerReturnsOnCall = make(map[int]struct {
			result1 *initializer.Peer
			result2 error
		})
	}
	fake.getInitPeerReturnsOnCall[i] = struct {
		result1 *initializer.Peer
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) GetUpdatedPeer(arg1 *v1beta1.IBPPeer) (*initializer.Peer, error) {
	fake.getUpdatedPeerMutex.Lock()
	ret, specificReturn := fake.getUpdatedPeerReturnsOnCall[len(fake.getUpdatedPeerArgsForCall)]
	fake.getUpdatedPeerArgsForCall = append(fake.getUpdatedPeerArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
	}{arg1})
	stub := fake.GetUpdatedPeerStub
	fakeReturns := fake.getUpdatedPeerReturns
	fake.recordInvocation("GetUpdatedPeer", []interface{}{arg1})
	fake.getUpdatedPeerMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *InitializeIBPPeer) GetUpdatedPeerCallCount() int {
	fake.getUpdatedPeerMutex.RLock()
	defer fake.getUpdatedPeerMutex.RUnlock()
	return len(fake.getUpdatedPeerArgsForCall)
}

func (fake *InitializeIBPPeer) GetUpdatedPeerCalls(stub func(*v1beta1.IBPPeer) (*initializer.Peer, error)) {
	fake.getUpdatedPeerMutex.Lock()
	defer fake.getUpdatedPeerMutex.Unlock()
	fake.GetUpdatedPeerStub = stub
}

func (fake *InitializeIBPPeer) GetUpdatedPeerArgsForCall(i int) *v1beta1.IBPPeer {
	fake.getUpdatedPeerMutex.RLock()
	defer fake.getUpdatedPeerMutex.RUnlock()
	argsForCall := fake.getUpdatedPeerArgsForCall[i]
	return argsForCall.arg1
}

func (fake *InitializeIBPPeer) GetUpdatedPeerReturns(result1 *initializer.Peer, result2 error) {
	fake.getUpdatedPeerMutex.Lock()
	defer fake.getUpdatedPeerMutex.Unlock()
	fake.GetUpdatedPeerStub = nil
	fake.getUpdatedPeerReturns = struct {
		result1 *initializer.Peer
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) GetUpdatedPeerReturnsOnCall(i int, result1 *initializer.Peer, result2 error) {
	fake.getUpdatedPeerMutex.Lock()
	defer fake.getUpdatedPeerMutex.Unlock()
	fake.GetUpdatedPeerStub = nil
	if fake.getUpdatedPeerReturnsOnCall == nil {
		fake.getUpdatedPeerReturnsOnCall = make(map[int]struct {
			result1 *initializer.Peer
			result2 error
		})
	}
	fake.getUpdatedPeerReturnsOnCall[i] = struct {
		result1 *initializer.Peer
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) MissingCrypto(arg1 *v1beta1.IBPPeer) bool {
	fake.missingCryptoMutex.Lock()
	ret, specificReturn := fake.missingCryptoReturnsOnCall[len(fake.missingCryptoArgsForCall)]
	fake.missingCryptoArgsForCall = append(fake.missingCryptoArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
	}{arg1})
	stub := fake.MissingCryptoStub
	fakeReturns := fake.missingCryptoReturns
	fake.recordInvocation("MissingCrypto", []interface{}{arg1})
	fake.missingCryptoMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *InitializeIBPPeer) MissingCryptoCallCount() int {
	fake.missingCryptoMutex.RLock()
	defer fake.missingCryptoMutex.RUnlock()
	return len(fake.missingCryptoArgsForCall)
}

func (fake *InitializeIBPPeer) MissingCryptoCalls(stub func(*v1beta1.IBPPeer) bool) {
	fake.missingCryptoMutex.Lock()
	defer fake.missingCryptoMutex.Unlock()
	fake.MissingCryptoStub = stub
}

func (fake *InitializeIBPPeer) MissingCryptoArgsForCall(i int) *v1beta1.IBPPeer {
	fake.missingCryptoMutex.RLock()
	defer fake.missingCryptoMutex.RUnlock()
	argsForCall := fake.missingCryptoArgsForCall[i]
	return argsForCall.arg1
}

func (fake *InitializeIBPPeer) MissingCryptoReturns(result1 bool) {
	fake.missingCryptoMutex.Lock()
	defer fake.missingCryptoMutex.Unlock()
	fake.MissingCryptoStub = nil
	fake.missingCryptoReturns = struct {
		result1 bool
	}{result1}
}

func (fake *InitializeIBPPeer) MissingCryptoReturnsOnCall(i int, result1 bool) {
	fake.missingCryptoMutex.Lock()
	defer fake.missingCryptoMutex.Unlock()
	fake.MissingCryptoStub = nil
	if fake.missingCryptoReturnsOnCall == nil {
		fake.missingCryptoReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.missingCryptoReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *InitializeIBPPeer) Update(arg1 initializer.CoreConfig, arg2 initializer.IBPPeer) (*initializer.Response, error) {
	fake.updateMutex.Lock()
	ret, specificReturn := fake.updateReturnsOnCall[len(fake.updateArgsForCall)]
	fake.updateArgsForCall = append(fake.updateArgsForCall, struct {
		arg1 initializer.CoreConfig
		arg2 initializer.IBPPeer
	}{arg1, arg2})
	stub := fake.UpdateStub
	fakeReturns := fake.updateReturns
	fake.recordInvocation("Update", []interface{}{arg1, arg2})
	fake.updateMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *InitializeIBPPeer) UpdateCallCount() int {
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	return len(fake.updateArgsForCall)
}

func (fake *InitializeIBPPeer) UpdateCalls(stub func(initializer.CoreConfig, initializer.IBPPeer) (*initializer.Response, error)) {
	fake.updateMutex.Lock()
	defer fake.updateMutex.Unlock()
	fake.UpdateStub = stub
}

func (fake *InitializeIBPPeer) UpdateArgsForCall(i int) (initializer.CoreConfig, initializer.IBPPeer) {
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	argsForCall := fake.updateArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *InitializeIBPPeer) UpdateReturns(result1 *initializer.Response, result2 error) {
	fake.updateMutex.Lock()
	defer fake.updateMutex.Unlock()
	fake.UpdateStub = nil
	fake.updateReturns = struct {
		result1 *initializer.Response
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) UpdateReturnsOnCall(i int, result1 *initializer.Response, result2 error) {
	fake.updateMutex.Lock()
	defer fake.updateMutex.Unlock()
	fake.UpdateStub = nil
	if fake.updateReturnsOnCall == nil {
		fake.updateReturnsOnCall = make(map[int]struct {
			result1 *initializer.Response
			result2 error
		})
	}
	fake.updateReturnsOnCall[i] = struct {
		result1 *initializer.Response
		result2 error
	}{result1, result2}
}

func (fake *InitializeIBPPeer) UpdateAdminSecret(arg1 *v1beta1.IBPPeer) error {
	fake.updateAdminSecretMutex.Lock()
	ret, specificReturn := fake.updateAdminSecretReturnsOnCall[len(fake.updateAdminSecretArgsForCall)]
	fake.updateAdminSecretArgsForCall = append(fake.updateAdminSecretArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
	}{arg1})
	stub := fake.UpdateAdminSecretStub
	fakeReturns := fake.updateAdminSecretReturns
	fake.recordInvocation("UpdateAdminSecret", []interface{}{arg1})
	fake.updateAdminSecretMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *InitializeIBPPeer) UpdateAdminSecretCallCount() int {
	fake.updateAdminSecretMutex.RLock()
	defer fake.updateAdminSecretMutex.RUnlock()
	return len(fake.updateAdminSecretArgsForCall)
}

func (fake *InitializeIBPPeer) UpdateAdminSecretCalls(stub func(*v1beta1.IBPPeer) error) {
	fake.updateAdminSecretMutex.Lock()
	defer fake.updateAdminSecretMutex.Unlock()
	fake.UpdateAdminSecretStub = stub
}

func (fake *InitializeIBPPeer) UpdateAdminSecretArgsForCall(i int) *v1beta1.IBPPeer {
	fake.updateAdminSecretMutex.RLock()
	defer fake.updateAdminSecretMutex.RUnlock()
	argsForCall := fake.updateAdminSecretArgsForCall[i]
	return argsForCall.arg1
}

func (fake *InitializeIBPPeer) UpdateAdminSecretReturns(result1 error) {
	fake.updateAdminSecretMutex.Lock()
	defer fake.updateAdminSecretMutex.Unlock()
	fake.UpdateAdminSecretStub = nil
	fake.updateAdminSecretReturns = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) UpdateAdminSecretReturnsOnCall(i int, result1 error) {
	fake.updateAdminSecretMutex.Lock()
	defer fake.updateAdminSecretMutex.Unlock()
	fake.UpdateAdminSecretStub = nil
	if fake.updateAdminSecretReturnsOnCall == nil {
		fake.updateAdminSecretReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateAdminSecretReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) UpdateSecretsFromResponse(arg1 *v1beta1.IBPPeer, arg2 *config.CryptoResponse) error {
	fake.updateSecretsFromResponseMutex.Lock()
	ret, specificReturn := fake.updateSecretsFromResponseReturnsOnCall[len(fake.updateSecretsFromResponseArgsForCall)]
	fake.updateSecretsFromResponseArgsForCall = append(fake.updateSecretsFromResponseArgsForCall, struct {
		arg1 *v1beta1.IBPPeer
		arg2 *config.CryptoResponse
	}{arg1, arg2})
	stub := fake.UpdateSecretsFromResponseStub
	fakeReturns := fake.updateSecretsFromResponseReturns
	fake.recordInvocation("UpdateSecretsFromResponse", []interface{}{arg1, arg2})
	fake.updateSecretsFromResponseMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *InitializeIBPPeer) UpdateSecretsFromResponseCallCount() int {
	fake.updateSecretsFromResponseMutex.RLock()
	defer fake.updateSecretsFromResponseMutex.RUnlock()
	return len(fake.updateSecretsFromResponseArgsForCall)
}

func (fake *InitializeIBPPeer) UpdateSecretsFromResponseCalls(stub func(*v1beta1.IBPPeer, *config.CryptoResponse) error) {
	fake.updateSecretsFromResponseMutex.Lock()
	defer fake.updateSecretsFromResponseMutex.Unlock()
	fake.UpdateSecretsFromResponseStub = stub
}

func (fake *InitializeIBPPeer) UpdateSecretsFromResponseArgsForCall(i int) (*v1beta1.IBPPeer, *config.CryptoResponse) {
	fake.updateSecretsFromResponseMutex.RLock()
	defer fake.updateSecretsFromResponseMutex.RUnlock()
	argsForCall := fake.updateSecretsFromResponseArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *InitializeIBPPeer) UpdateSecretsFromResponseReturns(result1 error) {
	fake.updateSecretsFromResponseMutex.Lock()
	defer fake.updateSecretsFromResponseMutex.Unlock()
	fake.UpdateSecretsFromResponseStub = nil
	fake.updateSecretsFromResponseReturns = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) UpdateSecretsFromResponseReturnsOnCall(i int, result1 error) {
	fake.updateSecretsFromResponseMutex.Lock()
	defer fake.updateSecretsFromResponseMutex.Unlock()
	fake.UpdateSecretsFromResponseStub = nil
	if fake.updateSecretsFromResponseReturnsOnCall == nil {
		fake.updateSecretsFromResponseReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateSecretsFromResponseReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *InitializeIBPPeer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.checkIfAdminCertsUpdatedMutex.RLock()
	defer fake.checkIfAdminCertsUpdatedMutex.RUnlock()
	fake.coreConfigMapMutex.RLock()
	defer fake.coreConfigMapMutex.RUnlock()
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	fake.generateOrdererCACertsSecretMutex.RLock()
	defer fake.generateOrdererCACertsSecretMutex.RUnlock()
	fake.generateSecretsMutex.RLock()
	defer fake.generateSecretsMutex.RUnlock()
	fake.generateSecretsFromResponseMutex.RLock()
	defer fake.generateSecretsFromResponseMutex.RUnlock()
	fake.getCryptoMutex.RLock()
	defer fake.getCryptoMutex.RUnlock()
	fake.getInitPeerMutex.RLock()
	defer fake.getInitPeerMutex.RUnlock()
	fake.getUpdatedPeerMutex.RLock()
	defer fake.getUpdatedPeerMutex.RUnlock()
	fake.missingCryptoMutex.RLock()
	defer fake.missingCryptoMutex.RUnlock()
	fake.updateMutex.RLock()
	defer fake.updateMutex.RUnlock()
	fake.updateAdminSecretMutex.RLock()
	defer fake.updateAdminSecretMutex.RUnlock()
	fake.updateSecretsFromResponseMutex.RLock()
	defer fake.updateSecretsFromResponseMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *InitializeIBPPeer) recordInvocation(key string, args []interface{}) {
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

var _ basepeer.InitializeIBPPeer = new(InitializeIBPPeer)