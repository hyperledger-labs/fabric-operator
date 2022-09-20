#!/bin/bash
#
# Copyright contributors to the Hyperledger Fabric Operator project
#
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at:
#
# 	  http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
set -o errexit

cd "$(dirname "$0")/.."

# Dump out the error log if we are exiting with a nonzero error code
function exit_fn() {
  rc=$?
  set +x

  if [ "0" -ne $rc ]; then
    cat network-debug.log
  fi
}

# Call the exit handler when we exit.
trap "exit_fn" EXIT

export PATH=$PWD:$PWD/bin:$PATH
export TEST_NETWORK_INGRESS_DOMAIN=localho.st

echo "Starting E2E Test.  Environment:"
printenv | sort

# Set up a cluster
network kind

# Set up ngress and CRDs
network cluster init

# Set up a Fabric Network
network up

kubectl -n test-network get all


# Chaincode: general

export FABRIC_CFG_PATH=${PWD}/temp/config
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_ADDRESS=test-network-org1-peer1-peer.${TEST_NETWORK_INGRESS_DOMAIN}:443
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_MSPCONFIGPATH=${PWD}/temp/enrollments/org1/users/org1admin/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/temp/channel-msp/peerOrganizations/org1/msp/tlscacerts/tlsca-signcert.pem


## ccaas

rm -rf /tmp/fabric-samples || true
git clone https://github.com/hyperledger/fabric-samples.git /tmp/fabric-samples

network channel create

network cc deploy   asset-transfer-basic basic_1.0 /tmp/fabric-samples/asset-transfer-basic/chaincode-java

network cc metadata asset-transfer-basic
network cc invoke   asset-transfer-basic '{"Args":["InitLedger"]}'
network cc query    asset-transfer-basic '{"Args":["ReadAsset","asset1"]}' | jq

peer chaincode query -n asset-transfer-basic -C mychannel -c '{"Args":["org.hyperledger.fabric:GetMetadata"]}'


## k8s-builder

network down

export TEST_NETWORK_PEER_IMAGE=ghcr.io/hyperledger-labs/k8s-fabric-peer
export TEST_NETWORK_PEER_IMAGE_LABEL=v0.7.2

network up
network channel create

curl -fsSL https://github.com/hyperledgendary/conga-nft-contract/releases/download/v0.1.1/conga-nft-contract-v0.1.1.tgz -o conga-nft-contract-v0.1.1.tgz

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

kubectl -n test-network describe pods -l app.kubernetes.io/created-by=fabric-builder-k8s

peer chaincode query -n conga-nft-contract -C mychannel -c '{"Args":["org.hyperledger.fabric:GetMetadata"]}'

network down