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

function set_ecr_image_tag() {
  # converts local "/" separated image name to an appropriate ECR tag used in AWS_ECR_REPO
  # Example: fabric-samples/asset-transfer-basic/chaincode-java:latest -> asset-transfer-basic_java_latest

  local cc_local_image=$1
  ECR_IMAGE_TAG=$(python -c 'import sys; p=sys.argv[1]; p=p.split("/")[-3:]; cc=p[1]; lang=p[-1].split("-")[-1]; tag="latest"; print(f"{cc}_{lang}_{tag}")' ${cc_local_image})
}

function ecr_load_image() {
  local cc_local_image=$1

  ecr_login      ${AWS_PROFILE} ${AWS_ACCOUNT}

  local aws_ecr="${ECR_RESOURCE}/${AWS_ECR_REPO}"

  set_ecr_image_tag ${cc_local_image}

  CHAINCODE_IMAGE="${aws_ecr}:${ECR_IMAGE_TAG}"

  push_fn "Tag chaincode image for ECR"
  $CONTAINER_CLI tag ${cc_local_image} ${CHAINCODE_IMAGE}
  pop_fn

  push_fn "Load chaincode image into ECR"
  $CONTAINER_CLI push "${CHAINCODE_IMAGE}"
  pop_fn
}

# Convenience routine to "do everything" required to bring up a sample CC.
function deploy_chaincode() {
  local cc_name=$1
  local cc_label=$2
  local cc_folder=$(absolute_path $3)

  local temp_folder=$(mktemp -d)
  local cc_package=${temp_folder}/${cc_name}.tgz

  package_chaincode       ${cc_label} ${cc_name} ${cc_package}

  set_chaincode_id        ${cc_package}
  set_chaincode_image     ${cc_folder}

  build_chaincode_image   ${cc_folder} ${CHAINCODE_IMAGE}

  # push to container registry
  if [ "${CLUSTER_RUNTIME}" == "kind" ]; then
    kind_load_image       ${CHAINCODE_IMAGE}
  elif [ "${CLUSTER_RUNTIME}" == "k3s" ] && [ "${CHAINCODE_REGISTRY}" == "ecr" ]; then
    ecr_load_image        ${CHAINCODE_IMAGE}
  fi

  launch_chaincode        ${cc_name} ${CHAINCODE_ID} ${CHAINCODE_IMAGE}
  activate_chaincode      ${cc_name} ${cc_package}
}

# Infer a reasonable name for the chaincode image based on the folder path conventions, or
# allow the user to override with TEST_NETWORK_CHAINCODE_IMAGE.
function set_chaincode_image() {
  local cc_folder=$1

  if [ -z "$TEST_NETWORK_CHAINCODE_IMAGE" ]; then
    # cc_folder path starting with first index of "fabric-samples"
    CHAINCODE_IMAGE=${cc_folder/*fabric-samples/fabric-samples}
  else
    CHAINCODE_IMAGE=${TEST_NETWORK_CHAINCODE_IMAGE}
  fi
}

# Convenience routine to "do everything other than package and launch" a sample CC.
# When debugging a chaincode server, the process must be launched prior to completing
# the chaincode lifecycle at the peer.  This routine provides a route for packaging
# and installing the chaincode out of band, and a single target to complete the peer
# chaincode lifecycle.
function activate_chaincode() {
  local cc_name=$1
  local cc_package=$2

  set_chaincode_id    ${cc_package}

  install_chaincode   ${cc_package}
  approve_chaincode   ${cc_name} ${CHAINCODE_ID}
  commit_chaincode    ${cc_name}
}

function query_chaincode() {
  local cc_name=$1
  shift

  set -x

  export_peer_context 1 1

  peer chaincode query \
    -n  $cc_name \
    -C  $CHANNEL_NAME \
    -c  $@
}

function query_chaincode_metadata() {
  local cc_name=$1
  shift

  set -x
  local args='{"Args":["org.hyperledger.fabric:GetMetadata"]}'

  log ''
  log 'Org1-Peer1:'
  export_peer_context 1 1
  peer chaincode query -n $cc_name -C $CHANNEL_NAME -c $args
#
#  log ''
#  log 'Org1-Peer2:'
#  export_peer_context 1 2
#  peer chaincode query -n $cc_name -C $CHANNEL_NAME -c $args
}

function invoke_chaincode() {
  local cc_name=$1
  shift

  export_peer_context 1 1

  peer chaincode invoke \
    -n              $cc_name \
    -C              $CHANNEL_NAME \
    -c              $@ \
    --orderer       ${NS}-org0-orderersnode1-orderer.${INGRESS_DOMAIN}:443 \
    --tls --cafile  ${TEMP_DIR}/channel-msp/ordererOrganizations/org0/orderers/org0-orderersnode1/tls/signcerts/tls-cert.pem \
    --connTimeout   ${ORDERER_TIMEOUT}

  sleep 2
}

function build_chaincode_image() {
  local cc_folder=$1
  local cc_image=$2

  push_fn "Building chaincode image ${cc_image}"

  $CONTAINER_CLI build ${CONTAINER_NAMESPACE} -t ${cc_image} ${cc_folder}

  pop_fn
}

function kind_load_image() {
  local cc_image=$1

  push_fn "Loading chaincode to kind image plane"

  kind load docker-image ${cc_image}

  pop_fn
}

function package_chaincode() {
  local cc_label=$1
  local cc_name=$2
  local cc_archive=$3

  local cc_folder=$(dirname $cc_archive)
  local archive_name=$(basename $cc_archive)

  push_fn "Packaging chaincode ${cc_label}"

  mkdir -p ${cc_folder}

  # Allow the user to override the service URL for the endpoint.  This allows, for instance,
  # local debugging at the 'host.docker.internal' DNS alias.
  local cc_default_address="{{.peername}}-ccaas-${cc_name}:9999"
  local cc_address=${TEST_NETWORK_CHAINCODE_ADDRESS:-$cc_default_address}

  cat << EOF > ${cc_folder}/connection.json
{
  "address": "${cc_address}",
  "dial_timeout": "10s",
  "tls_required": false
}
EOF

  cat << EOF > ${cc_folder}/metadata.json
{
  "type": "ccaas",
  "label": "${cc_label}"
}
EOF

  tar -C ${cc_folder} -zcf ${cc_folder}/code.tar.gz connection.json
  tar -C ${cc_folder} -zcf ${cc_archive} code.tar.gz metadata.json

  rm ${cc_folder}/code.tar.gz

  pop_fn
}

function launch_chaincode_service() {
  local org=$1
  local peer=$2
  local cc_name=$3
  local cc_id=$4
  local cc_image=$5
  push_fn "Launching chaincode container \"${cc_image}\""

  cat << EOF | envsubst | kubectl -n $NS apply -f -
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${org}-${peer}-ccaas-${cc_name}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${org}-${peer}-ccaas-${cc_name}
  template:
    metadata:
      labels:
        app: ${org}-${peer}-ccaas-${cc_name}
    spec:
      containers:
        - name: main
          image: ${cc_image}
          imagePullPolicy: IfNotPresent
          env:
            - name: CHAINCODE_SERVER_ADDRESS
              value: 0.0.0.0:9999
            - name: CHAINCODE_ID
              value: ${cc_id}
            - name: CORE_CHAINCODE_ID_NAME
              value: ${cc_id}
          ports:
            - containerPort: 9999

---
apiVersion: v1
kind: Service
metadata:
  name: ${org}-${peer}-ccaas-${cc_name}
spec:
  ports:
    - name: chaincode
      port: 9999
      protocol: TCP
  selector:
    app: ${org}-${peer}-ccaas-${cc_name}
EOF

  kubectl -n $NS rollout status deploy/${org}-${peer}-ccaas-${cc_name}

  pop_fn
}

function launch_chaincode() {
  local org=org1
  local cc_name=$1
  local cc_id=$2
  local cc_image=$3

  launch_chaincode_service ${org} peer1 ${cc_name} ${cc_id} ${cc_image}
#  launch_chaincode_service ${org} peer2 ${cc_name} ${cc_id} ${cc_image}
}

function install_chaincode_for() {
  local org=$1
  local peer=$2
  local cc_package=$3
  push_fn "Installing chaincode for org ${org} peer ${peer}"

  export_peer_context $org $peer

  peer lifecycle chaincode install $cc_package

  pop_fn
}

# Package and install the chaincode, but do not activate.
function install_chaincode() {
  local org=1
  local cc_package=$1

  install_chaincode_for ${org} 1 ${cc_package}
#  install_chaincode_for ${org} 2 ${cc_package}
}

# approve the chaincode package for an org and assign a name
function approve_chaincode() {
  local org=1
  local peer=1
  local cc_name=$1
  local cc_id=$2
  push_fn "Approving chaincode ${cc_name} with ID ${cc_id}"

  export_peer_context $org $peer

  peer lifecycle \
    chaincode approveformyorg \
    --channelID     ${CHANNEL_NAME} \
    --name          ${cc_name} \
    --version       1 \
    --package-id    ${cc_id} \
    --sequence      1 \
    --orderer       ${NS}-org0-orderersnode1-orderer.${INGRESS_DOMAIN}:443 \
    --tls --cafile  ${TEMP_DIR}/channel-msp/ordererOrganizations/org0/orderers/org0-orderersnode1/tls/signcerts/tls-cert.pem \
    --connTimeout   ${ORDERER_TIMEOUT}

  pop_fn
}

# commit the named chaincode for an org
function commit_chaincode() {
  local org=1
  local peer=1
  local cc_name=$1
  push_fn "Committing chaincode ${cc_name}"

  export_peer_context $org $peer

  peer lifecycle \
    chaincode commit \
    --channelID     ${CHANNEL_NAME} \
    --name          ${cc_name} \
    --version       1 \
    --sequence      1 \
    --orderer       ${NS}-org0-orderersnode1-orderer.${INGRESS_DOMAIN}:443 \
    --tls --cafile  ${TEMP_DIR}/channel-msp/ordererOrganizations/org0/orderers/org0-orderersnode1/tls/signcerts/tls-cert.pem \
    --connTimeout   ${ORDERER_TIMEOUT}

  pop_fn
}

function set_chaincode_id() {
  local cc_package=$1

  cc_sha256=$(shasum -a 256 ${cc_package} | tr -s ' ' | cut -d ' ' -f 1)
  cc_label=$(tar zxfO ${cc_package} metadata.json | jq -r '.label')

  CHAINCODE_ID=${cc_label}:${cc_sha256}
}

# chaincode "group" commands.  Like "main" for chaincode sub-command group.
function chaincode_command_group() {
  set -x

  COMMAND=$1
  shift

  if [ "${COMMAND}" == "deploy" ]; then
    log "Deploying chaincode"
    deploy_chaincode $@
    log "🏁 - Chaincode is ready."

  elif [ "${COMMAND}" == "activate" ]; then
    log "Activating chaincode"
    activate_chaincode $@
    log "🏁 - Chaincode is ready."

  elif [ "${COMMAND}" == "package" ]; then
    log "Packaging chaincode"
    package_chaincode $@
    log "🏁 - Chaincode package is ready."

  elif [ "${COMMAND}" == "id" ]; then
    set_chaincode_id $@
    log $CHAINCODE_ID

  elif [ "${COMMAND}" == "launch" ]; then
    log "Launching chaincode services"
    launch_chaincode $@
    log "🏁 - Chaincode services are ready"

  elif [ "${COMMAND}" == "install" ]; then
    log "Installing chaincode for org1"
    install_chaincode $@
    log "🏁 - Chaincode is installed"

  elif [ "${COMMAND}" == "approve" ]; then
    log "Approving chaincode for org1"
    approve_chaincode $@
    log "🏁 - Chaincode is approved"

  elif [ "${COMMAND}" == "commit" ]; then
    log "Committing chaincode for org1"
    commit_chaincode $@
    log "🏁 - Chaincode is committed"

  elif [ "${COMMAND}" == "invoke" ]; then
    invoke_chaincode $@ 2>> ${LOG_FILE}

  elif [ "${COMMAND}" == "query" ]; then
    query_chaincode $@ >> ${LOG_FILE}

  elif [ "${COMMAND}" == "metadata" ]; then
    query_chaincode_metadata $@ >> ${LOG_FILE}

  else
    print_help
    exit 1
  fi
}
