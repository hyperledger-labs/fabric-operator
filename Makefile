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

## license checks
check-license:
	@scripts/check-licenses.sh

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	@scripts/checks.sh

checks: fmt vet

# gosec checks
go-sec:
	@scripts/go-sec.sh