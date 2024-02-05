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

package baseorderer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	config "github.com/IBM-Blockchain/fabric-operator/operatorconfig"
	orderer "github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/config/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/initializer/orderer/configtx"
	k8sclient "github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources"
	resourcemanager "github.com/IBM-Blockchain/fabric-operator/pkg/manager/resources/manager"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	"github.com/IBM-Blockchain/fabric-operator/pkg/operatorerrors"
	"github.com/IBM-Blockchain/fabric-operator/pkg/restart"
	"github.com/IBM-Blockchain/fabric-operator/pkg/util"
	"github.com/IBM-Blockchain/fabric-operator/version"
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/orderer/etcdraft"
	"github.com/hyperledger/fabric/bccsp"
	fmsp "github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("base_orderer")

const (
	defaultOrdererNode = "./definitions/orderer/orderernode.yaml"
)

//go:generate counterfeiter -o mocks/node_manager.go -fake-name NodeManager . NodeManager

type NodeManager interface {
	GetNode(int, map[string]*time.Timer, RestartManager) *Node
}

var _ IBPOrderer = &Orderer{}

type Orderer struct {
	Client k8sclient.Client
	Scheme *runtime.Scheme
	Config *config.Config

	NodeManager        NodeManager
	OrdererNodeManager resources.Manager

	Override        Override
	RenewCertTimers map[string]*time.Timer
	RestartManager  *restart.RestartManager
}

func New(client k8sclient.Client, scheme *runtime.Scheme, config *config.Config, o Override) *Orderer {
	orderer := &Orderer{
		Client: client,
		Scheme: scheme,
		Config: config,
		NodeManager: &Manager{
			Client: client,
			Scheme: scheme,
			Config: config,
		},
		Override:        o,
		RenewCertTimers: make(map[string]*time.Timer),
		RestartManager:  restart.New(client, config.Operator.Restart.WaitTime.Get(), config.Operator.Restart.Timeout.Get()),
	}
	orderer.CreateManagers()
	return orderer
}

func (o *Orderer) CreateManagers() {
	resourceManager := resourcemanager.New(o.Client, o.Scheme)
	o.OrdererNodeManager = resourceManager.CreateOrderernodeManager("", o.Override.OrdererNode, o.GetLabels, defaultOrdererNode)
}

func (o *Orderer) PreReconcileChecks(instance *current.IBPOrderer, update Update) (bool, error) {
	if strings.ToLower(instance.Spec.OrdererType) != "etcdraft" {
		return false, operatorerrors.New(operatorerrors.InvalidOrdererType, fmt.Sprintf("orderer type '%s' is not supported", instance.Spec.OrdererType))
	}

	size := instance.Spec.ClusterSize
	if instance.Spec.NodeNumber == nil && instance.Spec.ClusterLocation != nil && instance.Spec.ClusterSize != 0 && len(instance.Spec.ClusterLocation) != size {
		return false, operatorerrors.New(operatorerrors.InvalidOrdererType, "Number of Cluster Node Locations does not match cluster size")
	}

	if instance.Spec.NodeNumber == nil && instance.Spec.ClusterSecret == nil {
		return false, operatorerrors.New(operatorerrors.InvalidOrdererType, "Cluster MSP Secrets should be passed")
	}

	if instance.Spec.NodeNumber == nil && instance.Spec.ClusterSecret != nil && instance.Spec.ClusterSize != 0 && len(instance.Spec.ClusterSecret) != size {
		return false, operatorerrors.New(operatorerrors.InvalidOrdererType, "Number of Cluster MSP Secrets does not match cluster size")
	}

	if instance.Spec.NodeNumber == nil && instance.Spec.ClusterConfigOverride != nil && instance.Spec.ClusterSize != 0 && len(instance.Spec.ClusterConfigOverride) != size {
		return false, operatorerrors.New(operatorerrors.InvalidOrdererType, "Number of Cluster Override does not match cluster size")
	}

	var maxNameLength *int
	if instance.Spec.ConfigOverride != nil {
		override := &orderer.OrdererOverrides{}
		err := json.Unmarshal(instance.Spec.ConfigOverride.Raw, override)
		if err != nil {
			return false, err
		}
		maxNameLength = override.MaxNameLength
	}

	err := util.ValidationChecks(instance.TypeMeta, instance.ObjectMeta, "IBPOrderer", maxNameLength)
	if err != nil {
		return false, err
	}

	sizeUpdate := o.ClusterSizeUpdate(instance)
	if sizeUpdate {
		log.Info("Updating instance with default cluster size of 1")
		err = o.Client.Patch(context.TODO(), instance, nil, k8sclient.PatchOption{
			Resilient: &k8sclient.ResilientPatch{
				Retry:    3,
				Into:     &current.IBPOrderer{},
				Strategy: client.MergeFrom,
			},
		})
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (o *Orderer) ClusterSizeUpdate(instance *current.IBPOrderer) bool {
	size := instance.Spec.ClusterSize
	if size == 0 {
		instance.Spec.ClusterSize = 1
		return true
	}

	return false
}

func (o *Orderer) Initialize(instance *current.IBPOrderer, update Update) error {
	// NO-OP
	return nil
}

func (o *Orderer) ReconcileManagers(instance *current.IBPOrderer, update Update, genesisBlock []byte) error {
	var b64GenesisBlock string

	b64GenesisBlock = util.BytesToBase64(genesisBlock)

	for k := 0; k < instance.Spec.ClusterSize; k++ {
		nodenumber := k + 1
		nodeinstance := instance.DeepCopy()
		nodeinstance.Spec.NodeNumber = &nodenumber
		nodeinstance.Spec.ClusterSize = 1
		nodeinstance.Spec.GenesisBlock = b64GenesisBlock
		if len(instance.Spec.ClusterConfigOverride) != 0 {
			nodeinstance.Spec.ConfigOverride = instance.Spec.ClusterConfigOverride[k]
		}
		nodeinstance.Spec.Secret = instance.Spec.ClusterSecret[k]
		err := o.OrdererNodeManager.Reconcile(nodeinstance, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Orderer) UpdateNodesWithGenesisBlock(genesisBlock string, nodes []current.IBPOrderer) error {
	log.Info("Updating nodes with genesis block if missing")

	for _, node := range nodes {
		if node.Spec.GenesisBlock == "" {
			log.Info(fmt.Sprintf("Updating node '%s'", node.Name))

			node.Spec.GenesisBlock = genesisBlock
			nodeRef := node
			err := o.Client.Patch(context.TODO(), &nodeRef, nil, k8sclient.PatchOption{
				Resilient: &k8sclient.ResilientPatch{
					Retry:    3,
					Into:     &current.IBPOrderer{},
					Strategy: client.MergeFrom,
				},
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *Orderer) Reconcile(instance *current.IBPOrderer, update Update) (common.Result, error) {
	return common.Result{}, errors.New("base orderer reconcile not implemented, needs to be implemented by offering")
}

func (o *Orderer) ReconcileCluster(instance *current.IBPOrderer, update Update, addHostPortToProfile func(*configtx.Profile, *current.IBPOrderer) error) (common.Result, error) {
	log.Info(fmt.Sprintf("Reconciling Orderer Cluster %s", instance.GetName()))
	var err error

	size := instance.Spec.ClusterSize
	nodes, err := o.GetClusterNodes(instance)
	if err != nil {
		return common.Result{}, err
	}

	if len(nodes.Items) == size {
		if instance.Spec.IsPrecreateOrderer() {
			return common.Result{}, err
		}
	}

	for _, node := range nodes.Items {
		log.Info(fmt.Sprintf("GetClusterNodes returned node '%s'", node.Name))
	}

	log.Info(fmt.Sprintf("Size of cluster (number of nodes): %d", size))

	var genesisBlock []byte
	if len(nodes.Items) == size && !instance.Spec.IsUsingChannelLess() {
		// Wait till all nodes are in precreated state before generating genesis block.
		// Once in precreate state, the TLS certs and service should exists for genesis
		// block creation
		deployedNodes := 0
		for _, node := range nodes.Items {
			if node.Status.Type == current.Deployed {
				deployedNodes++
			} else {
				log.Info(fmt.Sprintf("Node '%s' hasn't deployed yet, checking if in precreated state", node.GetName()))
				if node.Status.Type != current.Precreated {
					log.Info(fmt.Sprintf("Node '%s' hasn't entered precreated state, requeue request, another check to be made at next reconcile", node.GetName()))
					return common.Result{
						Result: reconcile.Result{
							Requeue: true,
						},
					}, nil
				}
			}
		}

		// If all nodes are deployed state and parent hasn't deployed yet, ensure that
		// all genesis secrets are found. If all required genesis secrets are not present
		// continue with generating secrets
		genesisSecretsFound := 0
		for _, node := range nodes.Items {

			nn := types.NamespacedName{
				Name:      node.Name + "-genesis",
				Namespace: node.Namespace,
			}

			err := o.Client.Get(context.TODO(), nn, &corev1.Secret{})
			if err == nil {
				genesisSecretsFound++
			}
		}

		// If all genesis secrets found, nothing left to do by parent of cluster nodes
		if genesisSecretsFound == len(nodes.Items) {
			return common.Result{}, nil
		}

		log.Info(fmt.Sprintf("All nodes have been precreated by cluster reconcile for parent: %s", instance.GetName()))

		genesisBlock, err = o.GenerateGenesisBlock(instance, addHostPortToProfile)
		if err != nil {
			return common.Result{}, err
		}

		log.Info(fmt.Sprintf("Finished generating genesis block for cluster '%s'", instance.GetName()))

		b64GenesisBlock := util.BytesToBase64(genesisBlock)
		err = o.UpdateNodesWithGenesisBlock(b64GenesisBlock, nodes.Items)
		if err != nil {
			return common.Result{}, err
		}

		err = o.GenerateGenesisSecretForNodes(genesisBlock, nodes.Items)
		if err != nil {
			return common.Result{}, err
		}

		log.Info("Finished generating genesis secrets")

		return common.Result{}, err
	}

	if instance.Status.Type == "" || instance.Status.Type == current.Deploying {
		for i := 1; i <= size; i++ {
			err := o.CreateNodeCR(instance, i)
			if err != nil {
				return common.Result{}, err
			}
		}
	}

	if !version.String(instance.Status.Version).Equal(version.Operator) {
		log.Info(fmt.Sprintf("[Reconcile cluster] Setting version to %s for instance %s", version.Operator, instance.Name))
		instance.Status.Version = version.Operator
		err = o.PatchStatus(instance)
		if err != nil {
			return common.Result{}, err
		}
	}

	return common.Result{}, nil
}

func (o *Orderer) GenerateGenesisSecretForNodes(genesisBlock []byte, nodes []current.IBPOrderer) error {
	log.Info("Generating genesis secret for all nodes")

	for _, node := range nodes {
		log.Info(fmt.Sprintf("Processing node '%s' for genesis secret", node.Name))
		s := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      node.Name + "-genesis",
				Namespace: node.Namespace,
				Labels:    node.GetLabels(),
			},
			Data: map[string][]byte{
				"orderer.block": genesisBlock,
			},
		}

		log.Info(fmt.Sprintf("Creating secret '%s'", s.Name))
		nodeRef := node
		err := o.Client.Create(context.TODO(), s, k8sclient.CreateOption{Owner: &nodeRef, Scheme: o.Scheme})
		if err != nil {
			return errors.Wrap(err, "failed to create orderer node's genesis secret")
		}
	}

	return nil
}

func (o *Orderer) SetVersion(instance *current.IBPOrderer) (bool, error) {
	if instance.Status.Version == "" || !version.String(instance.Status.Version).Equal(version.Operator) {
		log.Info("Version of Operator: ", "version", version.Operator)
		log.Info(fmt.Sprintf("Version of CR '%s': %s", instance.GetName(), instance.Status.Version))
		log.Info(fmt.Sprintf("Setting '%s' to version '%s'", instance.Name, version.Operator))

		instance.Status.Version = version.Operator
		err := o.PatchStatus(instance)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (o *Orderer) GetClusterNodes(instance *current.IBPOrderer) (current.IBPOrdererList, error) {
	ordererList := current.IBPOrdererList{}

	labelSelector, err := labels.Parse(fmt.Sprintf("parent=%s", instance.GetName()))
	if err != nil {
		return ordererList, errors.Wrap(err, "failed to parse selector for parent name")
	}

	listOptions := &client.ListOptions{
		LabelSelector: labelSelector,
		Namespace:     instance.GetNamespace(),
	}

	err = o.Client.List(context.TODO(), &ordererList, listOptions)
	if err != nil {
		return ordererList, err
	}

	return ordererList, nil
}

func (o *Orderer) CreateNodeCR(instance *current.IBPOrderer, number int) error {
	if instance.Spec.NodeNumber != nil {
		return fmt.Errorf("only parent orderer can create nodes custom resources, instance '%s' is not a parent", instance.GetName())
	}

	if !instance.Spec.License.Accept {
		return errors.New("user must accept license before continuing")
	}

	label := os.Getenv("OPERATOR_LABEL_PREFIX")
	if label == "" {
		label = "fabric"
	}

	name := fmt.Sprintf("%snode%d", instance.GetName(), number)
	node := instance.DeepCopy()
	node.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: instance.GetNamespace(),
		Labels: map[string]string{
			"app":                          name,
			"creator":                      label,
			"parent":                       instance.GetName(),
			"app.kubernetes.io/name":       label,
			"app.kubernetes.io/instance":   label + "orderer",
			"app.kubernetes.io/managed-by": label + "-operator",
		},
	}

	log.Info(fmt.Sprintf("Cluster reconcile is precreating node '%s'", node.Name))

	if len(node.Spec.ClusterConfigOverride) >= number {
		node.Spec.ConfigOverride = node.Spec.ClusterConfigOverride[number-1]
	}

	if len(node.Spec.ClusterSecret) >= number {
		node.Spec.Secret = node.Spec.ClusterSecret[number-1]
	}

	if len(node.Spec.ClusterLocation) >= number {
		node.Spec.Zone = node.Spec.ClusterLocation[number-1].Zone
		node.Spec.Region = node.Spec.ClusterLocation[number-1].Region

		if node.Spec.Zone != "" && node.Spec.Region == "" {
			node.Spec.Region = "select"
		}
	}

	if instance.Spec.IsUsingChannelLess() {
		node.Spec.UseChannelLess = instance.Spec.UseChannelLess
	} else {
		node.Spec.IsPrecreate = pointer.Bool(true)
	}
	node.Spec.NodeNumber = &number
	node.Spec.ClusterSize = 1
	node.Spec.ClusterSecret = nil
	node.Spec.ClusterConfigOverride = nil
	node.Spec.ClusterLocation = nil

	err := o.Client.Create(context.TODO(), node)
	if err != nil {
		return err
	}

	if instance.Status.Version != version.Operator {
		log.Info(fmt.Sprintf("[Create Node CR] Setting version to %s for node %s", version.Operator, node.Name))
		node.Status.Version = version.Operator
		// Using Update instead of Patch status;error will be thrown when trying to get and merge instance during
		// Patch. Update status will work here because the node has just been created so its spec will not have updated
		// before setting its version.
		err = o.UpdateStatus(node)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Orderer) ReconcileNode(instance *current.IBPOrderer, update bool) (reconcile.Result, error) {
	return reconcile.Result{}, errors.New("base orderer reconcile node not implemented, needs to be implemented by offering")
}

func (o *Orderer) GenerateGenesisBlock(instance *current.IBPOrderer, addHostPortToProfile func(*configtx.Profile, *current.IBPOrderer) error) ([]byte, error) {
	log.Info("Generating genesis block")
	initProfile, err := o.LoadInitialProfile(instance)
	if err != nil {
		return nil, err
	}

	err = addHostPortToProfile(initProfile, instance)
	if err != nil {
		return nil, err
	}

	conf := initProfile.Orderer
	mspConfigs := map[string]*msp.MSPConfig{}
	for _, org := range conf.Organizations {
		var err error
		mspConfigs[org.Name], err = o.GetMSPConfig(instance, org.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create orderer org")
		}
	}

	genesisBlock, err := initProfile.GenerateBlock(instance.Spec.SystemChannelName, mspConfigs)
	if err != nil {
		return nil, err
	}

	return genesisBlock, nil
}

func (o *Orderer) LoadInitialProfile(instance *current.IBPOrderer) (*configtx.Profile, error) {
	profile := instance.Spec.GenesisProfile
	if profile == "" {
		profile = "Initial"
	}

	log.Info(fmt.Sprintf("Profile '%s' used for genesis creation", profile))

	configTx := configtx.New()
	initProfile, err := configTx.GetProfile(profile)
	if err != nil {
		return nil, err
	}

	org := &configtx.Organization{
		Name:           instance.Spec.OrgName,
		ID:             instance.Spec.MSPID,
		MSPType:        "bccsp",
		MSPDir:         "/certs/msp",
		AdminPrincipal: "Role.MEMBER",
	}
	err = initProfile.AddOrgToOrderer(org)
	if err != nil {
		return nil, err
	}

	return initProfile, nil
}

func (o *Orderer) AddHostPortToProfile(initProfile *configtx.Profile, instance *current.IBPOrderer) error {
	log.Info("Adding hosts to genesis block")

	nodes := o.GetNodes(instance)
	for _, node := range nodes {
		n := types.NamespacedName{
			Name:      fmt.Sprintf("tls-%s%s-signcert", instance.Name, node.Name),
			Namespace: instance.Namespace,
		}

		// To avoid the race condition of the TLS signcert secret not existing, need to poll for it's
		// existence before proceeding
		tlsSecret := &corev1.Secret{}
		err := wait.Poll(500*time.Millisecond, o.Config.Operator.Orderer.Timeouts.SecretPoll.Get(), func() (bool, error) {
			err := o.Client.Get(context.TODO(), n, tlsSecret)
			if err == nil {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			return errors.Wrapf(err, "failed to find secret '%s'", n.Name)
		}

		domain := instance.Spec.Domain
		fqdn := instance.Namespace + "-" + instance.Name + node.Name + "-orderer" + "." + domain

		log.Info(fmt.Sprintf("Adding consentor domain '%s' to genesis block", fqdn))

		initProfile.AddOrdererAddress(fmt.Sprintf("%s:%d", fqdn, 443))
		consentors := &etcdraft.Consenter{
			Host:          fqdn,
			Port:          443,
			ClientTlsCert: tlsSecret.Data["cert.pem"],
			ServerTlsCert: tlsSecret.Data["cert.pem"],
		}
		err = initProfile.AddRaftConsentingNode(consentors)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Orderer) GetMSPConfig(instance *current.IBPOrderer, ID string) (*msp.MSPConfig, error) {
	isIntermediate := false
	admincert := [][]byte{}
	n := types.NamespacedName{
		Name:      fmt.Sprintf("ecert-%s%s%d-admincerts", instance.Name, NODE, 1),
		Namespace: instance.Namespace,
	}
	adminCert := &corev1.Secret{}
	err := o.Client.Get(context.TODO(), n, adminCert)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
	}
	for _, cert := range adminCert.Data {
		admincert = append(admincert, cert)
	}

	cacerts := [][]byte{}
	n.Name = fmt.Sprintf("ecert-%s%s%d-cacerts", instance.Name, NODE, 1)
	caCerts := &corev1.Secret{}
	err = o.Client.Get(context.TODO(), n, caCerts)
	if err != nil {
		return nil, err
	}
	for _, cert := range caCerts.Data {
		cacerts = append(cacerts, cert)
	}

	intermediateCerts := [][]byte{}
	interCerts := &corev1.Secret{}
	n.Name = fmt.Sprintf("ecert-%s%s%d-intercerts", instance.Name, NODE, 1)
	err = o.Client.Get(context.TODO(), n, interCerts)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
	}
	for _, cert := range interCerts.Data {
		isIntermediate = true
		intermediateCerts = append(intermediateCerts, cert)
	}

	cryptoConfig := &msp.FabricCryptoConfig{
		SignatureHashFamily:            bccsp.SHA2,
		IdentityIdentifierHashFunction: bccsp.SHA256,
	}

	tlsCACerts := [][]byte{}
	n.Name = fmt.Sprintf("tls-%s%s%d-cacerts", instance.Name, NODE, 1)
	tlsCerts := &corev1.Secret{}
	err = o.Client.Get(context.TODO(), n, tlsCerts)
	if err != nil {
		return nil, err
	}
	for _, cert := range tlsCerts.Data {
		tlsCACerts = append(tlsCACerts, cert)
	}

	tlsIntermediateCerts := [][]byte{}
	tlsInterCerts := &corev1.Secret{}
	n.Name = fmt.Sprintf("tls-%s%s%d-intercerts", instance.Name, NODE, 1)
	err = o.Client.Get(context.TODO(), n, tlsInterCerts)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
	}
	for _, cert := range tlsInterCerts.Data {
		tlsIntermediateCerts = append(tlsIntermediateCerts, cert)
	}

	fmspconf := &msp.FabricMSPConfig{
		Admins:               admincert,
		RootCerts:            cacerts,
		IntermediateCerts:    intermediateCerts,
		Name:                 ID,
		CryptoConfig:         cryptoConfig,
		TlsRootCerts:         tlsCACerts,
		TlsIntermediateCerts: tlsIntermediateCerts,
		FabricNodeOus: &msp.FabricNodeOUs{
			Enable: true,
			ClientOuIdentifier: &msp.FabricOUIdentifier{
				OrganizationalUnitIdentifier: "client",
				Certificate:                  cacerts[0],
			},
			PeerOuIdentifier: &msp.FabricOUIdentifier{
				OrganizationalUnitIdentifier: "peer",
				Certificate:                  cacerts[0],
			},
			AdminOuIdentifier: &msp.FabricOUIdentifier{
				OrganizationalUnitIdentifier: "admin",
				Certificate:                  cacerts[0],
			},
			OrdererOuIdentifier: &msp.FabricOUIdentifier{
				OrganizationalUnitIdentifier: "orderer",
				Certificate:                  cacerts[0],
			},
		},
	}

	if isIntermediate {
		fmspconf.FabricNodeOus.ClientOuIdentifier.Certificate = intermediateCerts[0]
		fmspconf.FabricNodeOus.PeerOuIdentifier.Certificate = intermediateCerts[0]
		fmspconf.FabricNodeOus.AdminOuIdentifier.Certificate = intermediateCerts[0]
		fmspconf.FabricNodeOus.OrdererOuIdentifier.Certificate = intermediateCerts[0]
	}

	fmpsjs, err := proto.Marshal(fmspconf)
	if err != nil {
		return nil, err
	}

	mspconf := &msp.MSPConfig{Config: fmpsjs, Type: int32(fmsp.FABRIC)}

	return mspconf, nil
}

func (o *Orderer) GetLabels(instance v1.Object) map[string]string {
	label := os.Getenv("OPERATOR_LABEL_PREFIX")
	if label == "" {
		label = "fabric"
	}

	orderernode := instance.(*current.IBPOrderer)

	name := instance.GetName()

	if orderernode.Spec.NodeNumber != nil {
		nodename := fmt.Sprintf("%snode%d", name, *orderernode.Spec.NodeNumber)
		name = nodename
	}

	return map[string]string{
		"app":                          name,
		"creator":                      label,
		"parent":                       instance.GetName(),
		"app.kubernetes.io/name":       label,
		"app.kubernetes.io/instance":   label + "orderer",
		"app.kubernetes.io/managed-by": label + "-operator",
	}
}

func (o *Orderer) ReadOUConfigFile(instance *current.IBPOrderer, configFile string) ([]*msp.FabricOUIdentifier, *msp.FabricNodeOUs, error) {
	var ouis []*msp.FabricOUIdentifier
	var nodeOUs *msp.FabricNodeOUs
	// load the file, if there is a failure in loading it then
	// return an error
	raw, err := ioutil.ReadFile(filepath.Clean(configFile))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed loading configuration file at [%s]", configFile)
	}

	configuration := fmsp.Configuration{}
	err = yaml.Unmarshal(raw, &configuration)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed unmarshalling configuration file at [%s]", configFile)
	}

	n := types.NamespacedName{
		Name:      fmt.Sprintf("ecert-%s%s%d-cacerts", instance.Name, NODE, 1),
		Namespace: instance.Namespace,
	}
	caCerts := &corev1.Secret{}
	err = o.Client.Get(context.TODO(), n, caCerts)
	if err != nil {
		return nil, nil, err
	}
	rawCert := caCerts.Data["cacert-0.pem"]

	// Prepare OrganizationalUnitIdentifiers
	if len(configuration.OrganizationalUnitIdentifiers) > 0 {
		for _, ouID := range configuration.OrganizationalUnitIdentifiers {
			oui := &msp.FabricOUIdentifier{
				Certificate:                  rawCert,
				OrganizationalUnitIdentifier: ouID.OrganizationalUnitIdentifier,
			}
			ouis = append(ouis, oui)
		}
	}

	// Prepare NodeOUs
	if configuration.NodeOUs != nil && configuration.NodeOUs.Enable {
		nodeOUs = &msp.FabricNodeOUs{
			Enable: true,
		}
		if configuration.NodeOUs.ClientOUIdentifier != nil && len(configuration.NodeOUs.ClientOUIdentifier.OrganizationalUnitIdentifier) != 0 {
			nodeOUs.ClientOuIdentifier = &msp.FabricOUIdentifier{OrganizationalUnitIdentifier: configuration.NodeOUs.ClientOUIdentifier.OrganizationalUnitIdentifier}
		}
		if configuration.NodeOUs.PeerOUIdentifier != nil && len(configuration.NodeOUs.PeerOUIdentifier.OrganizationalUnitIdentifier) != 0 {
			nodeOUs.PeerOuIdentifier = &msp.FabricOUIdentifier{OrganizationalUnitIdentifier: configuration.NodeOUs.PeerOUIdentifier.OrganizationalUnitIdentifier}
		}
		if configuration.NodeOUs.AdminOUIdentifier != nil && len(configuration.NodeOUs.AdminOUIdentifier.OrganizationalUnitIdentifier) != 0 {
			nodeOUs.AdminOuIdentifier = &msp.FabricOUIdentifier{OrganizationalUnitIdentifier: configuration.NodeOUs.AdminOUIdentifier.OrganizationalUnitIdentifier}
		}
		if configuration.NodeOUs.OrdererOUIdentifier != nil && len(configuration.NodeOUs.OrdererOUIdentifier.OrganizationalUnitIdentifier) != 0 {
			nodeOUs.OrdererOuIdentifier = &msp.FabricOUIdentifier{OrganizationalUnitIdentifier: configuration.NodeOUs.OrdererOUIdentifier.OrganizationalUnitIdentifier}
		}

		// ClientOU
		if nodeOUs.ClientOuIdentifier != nil {
			nodeOUs.ClientOuIdentifier.Certificate = rawCert
		}
		// PeerOU
		if nodeOUs.PeerOuIdentifier != nil {
			nodeOUs.PeerOuIdentifier.Certificate = rawCert
		}
		// AdminOU
		if nodeOUs.AdminOuIdentifier != nil {
			nodeOUs.AdminOuIdentifier.Certificate = rawCert
		}
		// OrdererOU
		if nodeOUs.OrdererOuIdentifier != nil {
			nodeOUs.OrdererOuIdentifier.Certificate = rawCert
		}
	}

	return ouis, nodeOUs, nil
}

func (o *Orderer) DeleteNode(instance *current.IBPOrderer, nodes int) error {
	if nodes == 0 {
		return errors.New("no cluster nodes left to delete")
	}

	node := o.GetNode(nodes)
	err := node.Delete(instance)
	if err != nil {
		return errors.Wrapf(err, "failed to delete node '%s'", node.Name)
	}

	return nil
}

func (o *Orderer) GetNodes(instance *current.IBPOrderer) []*Node {
	size := instance.Spec.ClusterSize
	nodes := []*Node{}
	for i := 1; i <= size; i++ {
		node := o.GetNode(i)
		nodes = append(nodes, node)
	}
	return nodes
}

func (o *Orderer) GetNode(nodeNumber int) *Node {
	return o.NodeManager.GetNode(nodeNumber, o.RenewCertTimers, o.RestartManager)
}

func (o *Orderer) CheckCSRHosts(instance *current.IBPOrderer, hosts []string) {
	if instance.Spec.Secret != nil {
		if instance.Spec.Secret.Enrollment != nil {
			if instance.Spec.Secret.Enrollment.TLS == nil {
				instance.Spec.Secret.Enrollment.TLS = &current.Enrollment{}
			}
			if instance.Spec.Secret.Enrollment.TLS.CSR == nil {
				instance.Spec.Secret.Enrollment.TLS.CSR = &current.CSR{}
				instance.Spec.Secret.Enrollment.TLS.CSR.Hosts = hosts
			} else {
				for _, host := range instance.Spec.Secret.Enrollment.TLS.CSR.Hosts {
					hosts = util.AppendStringIfMissing(hosts, host)
				}
				instance.Spec.Secret.Enrollment.TLS.CSR.Hosts = hosts
			}
		}
	}
}

func GetDomainPort(address string) (string, string) {
	u := strings.Split(address, ":")
	return u[0], u[1]
}

func (o *Orderer) PatchStatus(instance *current.IBPOrderer) error {
	return o.Client.PatchStatus(context.TODO(), instance, nil, k8sclient.PatchOption{
		Resilient: &k8sclient.ResilientPatch{
			Retry:    3,
			Into:     &current.IBPOrderer{},
			Strategy: client.MergeFrom,
		},
	})
}

func (o *Orderer) UpdateStatus(instance *current.IBPOrderer) error {
	return o.Client.UpdateStatus(context.TODO(), instance)
}
