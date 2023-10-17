#!/bin/bash -e

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

# cd /tmp
# go install golang.org/x/tools/cmd/goimports@latest
# curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash
# sudo mv kustomize /usr/local/bin
# go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0
# cd -

## getOperatorSDK
sudo rm /usr/local/bin/operator-sdk || true

OPERATOR_SDK_VERSION="v1.24.1"
ARCH=$(go env GOARCH)
OS=$(go env GOOS)
URL="https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk_${OS}_${ARCH}"

echo "Installing operator-sdk version ${OPERATOR_SDK_VERSION} to /usr/local/bin/operator-sdk"
curl -sL $URL > operator-sdk
chmod +x operator-sdk
sudo mv operator-sdk /usr/local/bin
operator-sdk version