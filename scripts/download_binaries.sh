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

#!/bin/bash -e
if [ ! -f ${PWD}/bin/fabric-ca-client ] || [ ! -f ${PWD}/bin/peer ] ; then
    echo -e "\n\n======= Downloading Fabric & Fabric-CA Binaries  =========\n"
    curl -sSL http://bit.ly/2ysbOFE | bash -s ${FABRIC_VERSION} ${FABRIC_CA_VERSION} -d -s
else
    echo -e "\n\n======= Fabric Binaries already exists, Skipping download =========\n"
fi
