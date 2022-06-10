# Sample Network

Create a sample network with CRDs, fabric-operator, and the Kube API server:

- Apply `kustomization` overlays to install the Operator
- Apply `kustomization` overlays to construct a Fabric Network
- Call `peer` CLI and channel participation SDKs to administer the network
- Deploy _Chaincode-as-a-Service_ smart contracts
- Develop _Gateway Client_ applications on a local workstation

Feedback, comments, questions, etc. at Discord : [#fabric-kubernetes](https://discord.gg/hyperledger)

![sample-network](../docs/images/fabric-operator-sample-network.png)

## Prerequisites:

### General

- Kubernetes - one of:
  - [KIND](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) + [Docker](https://www.docker.com) (resources: 8 CPU / 8 GRAM)
  - [Rancher Desktop](https://rancherdesktop.io) (resources: 8 CPU / 8GRAM, mobyd, and disable Traefik)
  - Cloud instance hosted at IKS or EKS
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [jq](https://stedolan.github.io/jq/)
- [envsubst](https://www.gnu.org/software/gettext/manual/html_node/envsubst-Invocation.html) (`brew install gettext` on OSX)
- Recommended: [k9s](https://k9scli.io)


### DNS Domain

Fabric-operator utilizes Kubernetes `Ingress` resources to expose services behind a common, unified
DNS wildcard domain (e.g. `*.my-blockchain.example.com`).  In typical cloud-based environments, a DNS administrator
is responsible for registering a public [DNS Wildcard Record](https://en.wikipedia.org/wiki/Wildcard_DNS_record),
associating a virtual domain name with the IP address of a load balancer or cluster Ingress controller.
For additional guidelines on public DNS, see [Considerations for Kubernetes Distributions](https://cloud.ibm.com/docs/blockchain-sw-252?topic=blockchain-sw-252-deploy-k8#console-deploy-k8-considerations).


For environments with a domain registered in public DNS, set the domain attribute and proceed to
[network setup](#test-network):

```shell
export TEST_NETWORK_COREDNS_DOMAIN_OVERRIDE=false
export TEST_NETWORK_INGRESS_DOMAIN=my-blockchain.example.com
```

To enable _local development_ without a public DNS resolver, the sample network has been bundled with
an Nginx ingress controller preconfigured in [ssl-passthrough](https://kubernetes.github.io/ingress-nginx/user-guide/tls/#ssl-passthrough) mode
for TLS termination directly at the Fabric nodes.  By default the ingress will expose fabric services at the
`*.localho.st` virtual domain, forcing all traffic from the host OS to be directed to the loopback interface.
(localho.st is a public domain with wildcard resolver to 127.0.0.1.)

For local development environments without write access to public DNS, set the ingress domain to the loopback interface:
```shell
export TEST_NETWORK_INGRESS_DOMAIN=localho.st
```

### Ingress IP Address

**Important**:

Before launching the network, you must determine an IP address where pods running in Kubernetes have a 
network route to the ingress controller.

While this will vary from system to system, the ingress IP can be determined with the following guidelines:


#### KIND + OSX
For KIND clusters running on OSX, determine the ingress IP by resolving `host.docker.internal` in a container:
```shell
docker run -it --rm alpine nslookup host.docker.internal
```
```
Non-authoritative answer:
Name:	host.docker.internal
Address: 192.168.65.2
```

```shell
export TEST_NETWORK_INGRESS_IPADDR=192.168.65.2
```

#### Windows / WSL2
Determine the host IP address according to [Microsoft guidelines](https://docs.microsoft.com/en-us/windows/wsl/networking):
```shell
export TEST_NETWORK_INGRESS_IPADDR=$(ip -json addr | jq -r '.[] | select(.ifname=="eth0") | .addr_info[] | select(.family=="inet") | .local')
```

#### Embedded VMs

For embedded virtual machines such as Vagrant, VirtualBox, VMWare, lima, etc., use the IP address of the
guest bridge interface:
```shell
export TEST_NETWORK_INGRESS_IPADDR=$(hostname -I | cut -d " " -f 1)
```

#### Rancher / k3s

On machines running [Rancher / k3s](https://rancherdesktop.io), use the host IP address assigned by DHCP 
(e.g. 192.168.1.42).

In addition to ingress IP, k3s clusters require the following settings:
```shell
export TEST_NETWORK_INGRESS_IPADDR=$(ipconfig getifaddr en0)

export TEST_NETWORK_CLUSTER_RUNTIME="k3s"
export TEST_NETWORK_STORAGE_CLASS="local-path"
export TEST_NETWORK_STAGE_DOCKER_IMAGES="false"
```


#### Amazon EKS

For deployments to public cloud, run `network cluster init` and use kubectl to determine the IP address of the
[Nginx ingress node](#amazon-kubernetes-service).

In addition to ingress IP, EKS clusters require the following settings: 

```shell
export TEST_NETWORK_CLUSTER_RUNTIME="k3s"
export TEST_NETWORK_STORAGE_CLASS="gp2"
export TEST_NETWORK_STAGE_DOCKER_IMAGES="false"
export TEST_NETWORK_COREDNS_DOMAIN_OVERRIDE=false
```


### Fabric Binaries

Fabric binaries (peer, osnadmin, etc.) will be installed into the local `bin` folder.  Add these to your PATH:

```shell
export PATH=$PWD:$PWD/bin:$PATH
```

In the examples below, the `peer` binary will be used to invoke smart contracts on the org1-peer1 ledger.  Set the CLI context with:
```shell
export FABRIC_CFG_PATH=${PWD}/temp/config
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_ADDRESS=test-network-org1-peer1-peer.${TEST_NETWORK_INGRESS_DOMAIN}:443
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_MSPCONFIGPATH=${PWD}/temp/enrollments/org1/users/org1admin/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/temp/channel-msp/peerOrganizations/org1/msp/tlscacerts/tlsca-signcert.pem
```


## Test Network

Create a KIND Kubernetes cluster (skip if using Rancher or a cloud-hosted instance):
```shell
network kind
```

Install the Nginx controller and Fabric CRDs:
```shell
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

Or use the native `peer` CLI to query the contract installed on org1 / peer1:
```shell
peer chaincode query -n asset-transfer-basic -C mychannel -c '{"Args":["org.hyperledger.fabric:GetMetadata"]}'
```


## K8s Chaincode Builder

The operator can also be configured for use with the [fabric-builder-k8s](https://github.com/hyperledgendary/fabric-builder-k8s)
chaincode builder, providing smooth and immediate _Chaincode Right Now!_ deployments.

Reconstruct the network with the "k8s-fabric-peer" image:
```shell
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
  --orderer       test-network-org0-orderersnode1-orderer.${TEST_NETWORK_INGRESS_DOMAIN}:443 \
  --tls --cafile  $PWD/temp/channel-msp/ordererOrganizations/org0/orderers/org0-orderersnode1/tls/signcerts/tls-cert.pem \
  --connTimeout   15s

peer lifecycle \
  chaincode commit \
  --channelID     mychannel \
  --name          conga-nft-contract \
  --version       1 \
  --sequence      1 \
  --orderer       test-network-org0-orderersnode1-orderer.${TEST_NETWORK_INGRESS_DOMAIN}:443 \
  --tls --cafile  $PWD/temp/channel-msp/ordererOrganizations/org0/orderers/org0-orderersnode1/tls/signcerts/tls-cert.pem \
  --connTimeout   15s

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

- open `https://test-network-hlf-console-console.localho.st`
- Accept the self-signed TLS certificate
- Log in as `admin:password`
- [Build a network](https://cloud.ibm.com/docs/blockchain?topic=blockchain-ibp-console-build-network)


## Troubleshooting Tips

#### Logs
- The `network` script prints output and progress to a `network-debug.log` file.  In a second shell:
```shell
tail -f network-debug.log
```

- Tail the operator logging output:
```shell
kubectl -n test-network logs -f deployment/fabric-operator
```

#### DNS

- KIND running under the vagrant based [Fabric developer environment](https://github.com/hyperledgendary/fabric-devenv)
  has problems resolving hosts on the kubernetes network domain `*.NAMESPACE.svc.cluster.local`, which is used for all
  of the cross node/peer/orderer traffic.  As a workaround, the host suffix can be shortened, forcing the k8s-based
  traffic to use the kube DNS:
```shell
export TEST_NETWORK_KUBE_DNS_DOMAIN=test-network
```


- On OSX, there is a bug in the Golang DNS resolver ([Fabric #3372](https://github.com/hyperledger/fabric/issues/3372) and [Golang #43398](https://github.com/golang/go/issues/43398)),
  causing the Fabric binaries to occasionally stall out when querying DNS.
  This issue can cause `osnadmin` / channel join to time out, throwing an error when joining the channel.
  Fix this by turning a build of [fabric](https://github.com/hyperledger/fabric) binaries and copying the build outputs
  from `fabric/build/bin/*` --> `sample-network/bin`


#### Amazon Kubernetes Service

- For deployments on EKS / Amazon cloud, it is possible to route traffic to the cluster ingress using the
  [Dead simple wildcard DNS for any IP Address](https://nip.io) service as an alternative to creating
  a public DNS record with Route 54.  For installation to EKS instances:

0. Set general properties for EKS:
```shell
declare -x TEST_NETWORK_CLUSTER_RUNTIME="k3s"
declare -x TEST_NETWORK_STAGE_DOCKER_IMAGES="false"
declare -x TEST_NETWORK_STORAGE_CLASS="gp2"
```

2. Initialize the Nginx ingress controller:
```shell
network cluster init
```
2. Determine the IP of the Ingress load balancer (EXTERNAL-IP)
```shell
$ kubectl -n ingress-nginx get svc/ingress-nginx-controller
NAME                       TYPE           CLUSTER-IP   EXTERNAL-IP                                PORT(S)
ingress-nginx-controller   LoadBalancer   10.1.1.10    xyzzy-abc123.us-east-1.elb.amazonaws.com   80:32545/TCP,443:30341/TCP

$ nslookup xyzzy.us-east-1.elb.amazonaws.com
Non-authoritative answer:
Name:	xyzzy-abc123.us-east-1.elb.amazonaws.com
Address: 55.1.1.101               # Not a real IP address - use this value
```

3. Deploy the network using a nip.io wildcard domain, directing hosts to the ingress EXTERNAL-IP:
```shell
export TEST_NETWORK_INGRESS_DOMAIN=55-1-1-101.nip.io
```