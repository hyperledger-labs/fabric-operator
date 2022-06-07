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

# cluster "group" commands.  Like "main" for the fabric-cli "cluster" sub-command
function cluster_command_group() {

  # Default COMMAND is 'init' if not specified
  if [ "$#" -eq 0 ]; then
    COMMAND="init"

  else
    COMMAND=$1
    shift
  fi

  if [ "${COMMAND}" == "init" ]; then
    log "Initializing K8s cluster"
    cluster_init
    log "🏁 - Cluster is ready"

  elif [ "${COMMAND}" == "clean" ]; then
    log "Cleaning k8s cluster"
    cluster_clean
    log "🏁 - Cluster is cleaned"

  elif [ "${COMMAND}" == "load-images" ]; then
    log "Loading Docker images"
    pull_docker_images

    if [ "${CLUSTER_RUNTIME}" == "kind" ]; then
      kind_load_images
    fi

    log "🏁 - Images are ready"

  else
    print_help
    exit 1
  fi
}

function pull_docker_images() {
  push_fn "Pulling docker images for Fabric ${FABRIC_VERSION}"

  $CONTAINER_CLI pull ${CONTAINER_NAMESPACE} $FABRIC_OPERATOR_IMAGE
  $CONTAINER_CLI pull ${CONTAINER_NAMESPACE} $FABRIC_CONSOLE_IMAGE
  $CONTAINER_CLI pull ${CONTAINER_NAMESPACE} $FABRIC_DEPLOYER_IMAGE
  $CONTAINER_CLI pull ${CONTAINER_NAMESPACE} $FABRIC_CA_IMAGE
  $CONTAINER_CLI pull ${CONTAINER_NAMESPACE} $FABRIC_PEER_IMAGE
  $CONTAINER_CLI pull ${CONTAINER_NAMESPACE} $FABRIC_ORDERER_IMAGE
  $CONTAINER_CLI pull ${CONTAINER_NAMESPACE} $INIT_IMAGE
  $CONTAINER_CLI pull ${CONTAINER_NAMESPACE} $COUCHDB_IMAGE
  $CONTAINER_CLI pull ${CONTAINER_NAMESPACE} $GRPCWEB_IMAGE

  pop_fn
}

function kind_load_images() {
  push_fn "Loading docker images to KIND control plane"

  kind load docker-image $FABRIC_OPERATOR_IMAGE
  kind load docker-image $FABRIC_CONSOLE_IMAGE
  kind load docker-image $FABRIC_DEPLOYER_IMAGE
  kind load docker-image $FABRIC_CA_IMAGE
  kind load docker-image $FABRIC_PEER_IMAGE
  kind load docker-image $FABRIC_ORDERER_IMAGE
  kind load docker-image $INIT_IMAGE
  kind load docker-image $COUCHDB_IMAGE
  kind load docker-image $GRPCWEB_IMAGE

  pop_fn
}

function cluster_init() {
  apply_fabric_crds
  apply_nginx_ingress

  if [ "${STAGE_DOCKER_IMAGES}" == true ]; then
    pull_docker_images
    kind_load_images
  fi

  wait_for_nginx_ingress
}

function apply_fabric_crds() {
  push_fn "Applying Fabric CRDs"

  $KUSTOMIZE_BUILD ../config/crd | kubectl apply -f -

  pop_fn
}

function delete_fabric_crds() {
  push_fn "Deleting Fabric CRDs"

  $KUSTOMIZE_BUILD ../config/crd | kubectl delete -f -

  pop_fn
}

function apply_nginx_ingress() {
  push_fn "Applying ingress controller"

  $KUSTOMIZE_BUILD ../config/ingress/${CLUSTER_RUNTIME} | kubectl apply -f -

  sleep 5

  pop_fn
}

function delete_nginx_ingress() {
  push_fn "Deleting ${CLUSTER_RUNTIME} ingress controller"

  $KUSTOMIZE_BUILD ../config/ingress/${CLUSTER_RUNTIME} | kubectl delete -f -

  pop_fn
}

function wait_for_nginx_ingress() {
  push_fn "Waiting for ingress controller"

  kubectl wait --namespace ingress-nginx \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/component=controller \
    --timeout=2m

  pop_fn
}

function cluster_clean() {
  delete_fabric_crds
  delete_nginx_ingress
}






