
#!/bin/bash
#
# Copyright contributors to the Hyperledger Fabric Operations Console project
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

## delete the folder if already exists
rm -rf /tmp/fabric-operator

## clone the main repo to temp directory and copy the files
git clone https://github.com/hyperledger-labs/fabric-operator.git /tmp/fabric-operator >/dev/null 2>&1

# copy CA config
cp -r /tmp/fabric-operator/pkg/apis/ca/v1/ca.go ../pkg/apis/ca/v1/ca.go

# copy peer config
cp -r /tmp/fabric-operator/pkg/apis/peer/* ../pkg/apis/peer/.

# copy orderer config
cp -r /tmp/fabric-operator/pkg/apis/orderer/* ../pkg/apis/orderer/.

# copy console config
cp -r /tmp/fabric-operator/pkg/apis/console/* ../pkg/apis/console/.

# copy common config
cp -r /tmp/fabric-operator/pkg/apis/common/* ../pkg/apis/common/

# copy deployer config
cp -r /tmp/fabric-operator/pkg/apis/deployer/* ../pkg/apis/deployer/

# copy v1beta1 specs
cp /tmp/fabric-operator/api/v1beta1/common_struct.go ../api/v1beta1/
cp -r /tmp/fabric-operator/api/v1beta1/*_types.go ../api/v1beta1/.
cp /tmp/fabric-operator/api/v1beta1/zz_generated.deepcopy.go ../api/v1beta1/zz_generated.deepcopy.go