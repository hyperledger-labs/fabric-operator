#!/bin/bash

#
# Copyright contributors to the Hyperledger Fabric Operator project
#
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 {the "License"};
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

docker manifest inspect ${INIT_IMAGE}:${INIT_TAG}-${ARCH} >> /dev/null

if [[ $? -ne 0 ]]; then
    if [[ $1 = "build" ]]; then
        docker build --rm . -f sample-network/init/Dockerfile --build-arg K8S_BUILDER_TAG=${K8S_BUILDER_TAG} -t ${INIT_IMAGE}:${INIT_TAG}-${ARCH}
        docker tag ${INIT_IMAGE}:${INIT_TAG}-${ARCH} ${INIT_IMAGE}:latest-${ARCH}
    elif [[ $1 = "push" ]]; then
        docker push ${INIT_IMAGE}:latest-${ARCH}
        docker push ${INIT_IMAGE}:${INIT_TAG}-${ARCH}
    fi
fi
