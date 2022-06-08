# Sample Network 

This project uses the operator to launch a Fabric network on a local KIND or k3s cluster.

- Apply `kustomization` overlays to install the Operator
- Apply `kustomization` overlays to construct a Fabric Network 
- Call `peer` CLI and channel participation SDKs to administer the network
- Deploy _Chaincode-as-a-Service_ smart contracts
- Develop _Gateway Client_ applications on a local workstation 

Feedback, comments, questions, etc. at Discord : [#fabric-kubernetes](https://discord.gg/hyperledger)

![sample-network](../docs/images/fabric-operator-sample-network.png)

## Prerequisites:

- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [jq](https://stedolan.github.io/jq/)
- [envsubst](https://www.gnu.org/software/gettext/manual/html_node/envsubst-Invocation.html) (`brew install gettext` on OSX)

- K8s - either:
    - [KIND](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) + [Docker](https://www.docker.com) (resources: 8 CPU / 8 GRAM)
    - [Rancher Desktop](https://rancherdesktop.io) (resources: 8 CPU / 8GRAM, mobyd, and disable Traefik)
  

### Ingress and DNS 

Fabric-operator utilizes Kubernetes `Ingress` resources to expose services behind a common, unified 
DNS wildcard domain. In cloud-based environments, a network admin is typically responsible for 
registering a [DNS Wildcard Record](https://en.wikipedia.org/wiki/Wildcard_DNS_record), utilizing public 
DNS resolvers to associate a virtual domain name (e.g. `*.my-blockchain.example.com`) with the IP address 
of a load-balancing proxy or Layer 7 appliance.

To enable _local development_ and systems without a DNS wildcard resolver, the sample network has been bundled with 
an Nginx ingress controller preconfigured in [ssl-passthrough](https://kubernetes.github.io/ingress-nginx/user-guide/tls/#ssl-passthrough) mode 
for TLS termination directly at the Fabric nodes.  By default the ingress will expose fabric services at the 
`*.localho.st` virtual domain, forcing all traffic from the host OS to be directed to the localhost interface 
on 127.0.0.1.

However, on occasion, pods running in Kubernetes will also need to connect to services at the _ingress domain_, 
requiring a network route and [coredns override](todo-link) to guide traffic towards the ingress controller.

**Important**:

Before installing the sample network, you must determine an IP address for your system which is visible _both_ to pods running in Kubernetes _AND_ to programs running on the 
host OS.

While this IP address varies from system to system, here are some common guidelines:

- For KIND on OSX, find the host IP address by resolving `host.docker.internal` in a docker container.  E.g.: 
```shell
$ docker run -it --rm alpine nslookup host.docker.internal

Name:	host.docker.internal
Address: 192.168.65.2
```
- On machines running an embedded virtual machine (WSL, Virtualbox, VMWare, etc.), use the IP address of the
  bridge interface for the guest VM.
- On machines running Rancher / k3s, use the host IP address assigned by DHCP (e.g. 192.168.0.4)
- On environments with access to public DNS (e.g. IBM cloud, Fyre, EKS, etc.), use DNS: 
```shell
export TEST_NETWORK_COREDNS_DOMAIN_OVERRIDE=false
export TEST_NETWORK_INGRESS_DOMAIN=my-blockchain.example.com
```


After finding a suitable IP address, set the cluster ingress for the network.  E.g.: 
```shell
export TEST_NETWORK_INGRESS_IPADDR=192.168.65.2
```



### Fabric Binaries 

Fabric binaries (peer, osnadmin, etc.) will be installed into the local `bin` folder.  Add these to your PATH: 

```shell
export PATH=${PWD}:${PWD}/bin:$PATH
```

On OSX, there is a bug in the Golang DNS resolver, causing the Fabric binaries to stall out when resolving DNS.
See [Fabric #3372](https://github.com/hyperledger/fabric/issues/3372) and [Golang #43398](https://github.com/golang/go/issues/43398). 
Fix this by turning a build of [fabric](https://github.com/hyperledger/fabric) binaries and copying the build outputs
from `fabric/build/bin/*` --> `sample-network/bin`


## Test Network 

Create a Kubernetes cluster, Nginx ingress, and Fabric CRDs:
```shell
network kind
network cluster init
```

Launch the operator and `kustomize` a network of [CAs](config/cas), [peers](config/peers), and [orderers](config/orderers):
```shell
network up 
```

Explore Kubernetes `Pods`, `Deployments`, `Services`, `Ingress`, etc.:
```shell
kubectl -n test-network get all 
```

## Chaincode 

The operator is compatible with sample _Chaincode-as-a-Service_ smart contracts.

Clone the [fabric-samples](https://github.com/hyperledger/fabric-samples) git repository:
```shell
git clone git@github.com:hyperledger/fabric-samples.git /tmp/fabric-samples
```

Create a channel:
```shell
network channel create
```

Deploy a sample contract:
```shell
network cc deploy   asset-transfer-basic basic_1.0 /tmp/fabric-samples/asset-transfer-basic/chaincode-java

network cc metadata asset-transfer-basic 
network cc invoke   asset-transfer-basic '{"Args":["InitLedger"]}'
network cc query    asset-transfer-basic '{"Args":["ReadAsset","asset1"]}' | jq 
```

Or set the `peer` CLI context to org1 peer1:
```shell
export FABRIC_CFG_PATH=${PWD}/temp/config
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_ADDRESS=test-network-org1-peer1-peer.${TEST_NETWORK_DOMAIN}:443
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_MSPCONFIGPATH=${PWD}/temp/enrollments/org1/users/org1admin/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/temp/channel-msp/peerOrganizations/org1/msp/tlscacerts/tlsca-signcert.pem
```

and directly interact with the contract:
```shell
peer chaincode query -n asset-transfer-basic -C mychannel -c '{"Args":["org.hyperledger.fabric:GetMetadata"]}'
```

## K8s Chaincode Builder

The operator can also be configured for use with the [fabric-builder-k8s](https://github.com/hyperledgendary/fabric-builder-k8s) 
chaincode builder, providing smooth and immediate _Chaincode Right Now!_ deployments.

Reconstruct the network with the "k8s-fabric-peer" image: 
```yaml
network down

export TEST_NETWORK_PEER_IMAGE=ghcr.io/hyperledgendary/k8s-fabric-peer
export TEST_NETWORK_PEER_IMAGE_LABEL=v0.5.0

network up
network channel create
```

Download a "k8s" chaincode package:
```shell
curl -fsSL https://github.com/hyperledgendary/conga-nft-contract/releases/download/v0.1.1/conga-nft-contract-v0.1.1.tgz -o conga-nft-contract-v0.1.1.tgz
```

Install the smart contract:
```shell
peer lifecycle chaincode install conga-nft-contract-v0.1.1.tgz

export PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid conga-nft-contract-v0.1.1.tgz) && echo $PACKAGE_ID

peer lifecycle \
  chaincode approveformyorg \
  --channelID     mychannel \
  --name          conga-nft-contract \
  --version       1 \
  --package-id    ${PACKAGE_ID} \
  --sequence      1 \
  --orderer       test-network-org0-orderersnode1-orderer.${TEST_NETWORK_DOMAIN}:443 \
  --tls --cafile  $PWD/temp/channel-msp/ordererOrganizations/org0/orderers/org0-orderersnode1/tls/signcerts/tls-cert.pem  
  
peer lifecycle \
  chaincode commit \
  --channelID     mychannel \
  --name          conga-nft-contract \
  --version       1 \
  --sequence      1 \
  --orderer       test-network-org0-orderersnode1-orderer.${TEST_NETWORK_DOMAIN}:443 \
  --tls --cafile  $PWD/temp/channel-msp/ordererOrganizations/org0/orderers/org0-orderersnode1/tls/signcerts/tls-cert.pem  

```

Inspect chaincode pods:
```shell
kubectl -n test-network describe pods -l app.kubernetes.io/created-by=fabric-builder-k8s
```

Query the smart contract:
```shell
peer chaincode query -n conga-nft-contract -C mychannel -c '{"Args":["org.hyperledger.fabric:GetMetadata"]}'
```


## Teardown 

Invariably, something in the recipe above will go awry. Look for additional diagnostics in network-debug.log and
reset the stage with:

```shell
network down
```
or 
```shell
network unkind 
```


## Appendix: Operations Console

Launch the [Fabric Operations Console](https://github.com/hyperledger-labs/fabric-operations-console):
```shell
network console
```

- open `https://test-network-hlf-console-console.${TEST_NETWORK_DOMAIN}`
- Accept the self-signed TLS certificate
- Log in as `admin:password`
- [Build a network](https://cloud.ibm.com/docs/blockchain?topic=blockchain-ibp-console-build-network)



