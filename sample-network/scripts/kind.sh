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

function kind_create() {
  push_fn  "Creating cluster \"${CLUSTER_NAME}\""

  # prevent the next kind cluster from using the previous Fabric network's enrollments.
  rm -rf $PWD/temp

  # todo: always delete?  Maybe return no-op if the cluster already exists?
  kind delete cluster --name $CLUSTER_NAME

  local reg_name=${LOCAL_REGISTRY_NAME}
  local reg_port=${LOCAL_REGISTRY_PORT}
  local ingress_http_port=${NGINX_HTTP_PORT}
  local ingress_https_port=${NGINX_HTTPS_PORT}

  # the 'ipvs'proxy mode permits better HA abilities

  cat <<EOF | kind create cluster --name $CLUSTER_NAME --config=-
---
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      - containerPort: 80
        hostPort: ${ingress_http_port}
        protocol: TCP
      - containerPort: 443
        hostPort: ${ingress_https_port}
        protocol: TCP
#networking:
#  kubeProxyMode: "ipvs"

# create a cluster with the local registry enabled in containerd
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:${reg_port}"]

EOF

  for node in $(kind get nodes);
  do
    docker exec "$node" sysctl net.ipv4.conf.all.route_localnet=1;
  done

  pop_fn
}

function launch_docker_registry() {
  push_fn "Launching container registry \"${LOCAL_REGISTRY_NAME}\" at localhost:${LOCAL_REGISTRY_PORT}"

  # create registry container unless it already exists
  local reg_name=${LOCAL_REGISTRY_NAME}
  local reg_port=${LOCAL_REGISTRY_PORT}
  local reg_interface=${LOCAL_REGISTRY_INTERFACE}

  running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
  if [ "${running}" != 'true' ]; then
    docker run  \
      --detach  \
      --restart always \
      --name    "${reg_name}" \
      --publish "${reg_interface}:${reg_port}:5000" \
      registry:2
  fi

  # connect the registry to the cluster network
  # (the network may already be connected)
  docker network connect "kind" "${reg_name}" || true

  # Document the local registry
  # https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
  cat <<EOF | kubectl apply -f -
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

  pop_fn
}

function stop_docker_registry() {
  push_fn "Deleting container registry \"${LOCAL_REGISTRY_NAME}\" at localhost:${LOCAL_REGISTRY_PORT}"

  docker kill kind-registry || true
  docker rm kind-registry   || true

  pop_fn
}

function kind_delete() {
  push_fn "Deleting KIND cluster ${CLUSTER_NAME}"

  kind delete cluster --name $CLUSTER_NAME

  pop_fn 2
}

function kind_init() {
  set -o errexit

  kind_create

  if [ "${USE_LOCAL_REGISTRY}" == true ]; then
    launch_docker_registry
  fi
}

function kind_unkind() {

  kind_delete

  if [ "${USE_LOCAL_REGISTRY}" == true ]; then
    stop_docker_registry
  fi
}
