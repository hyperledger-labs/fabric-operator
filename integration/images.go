/*
 * Copyright contributors to the Hyperledger Fabric Operator project
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 * 	  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration

const (
	FabricCAVersion    = "1.5.3"
	FabricVersion      = "2.2.5"
	FabricVersion24    = "2.4.3"
	InitImage          = "registry.access.redhat.com/ubi8/ubi-minimal"
	InitTag            = "latest"
	CaImage            = "hyperledger/fabric-ca"
	CaTag              = FabricCAVersion
	PeerImage          = "hyperledger/fabric-peer"
	PeerTag            = FabricVersion24
	OrdererImage       = "hyperledger/fabric-orderer"
	OrdererTag         = FabricVersion24
	Orderer14Tag       = "1.4.12"
	Orderer24Tag       = FabricVersion24
	ConfigtxlatorImage = "hyperledger/fabric-tools"
	ConfigtxlatorTag   = FabricVersion24
	CouchdbImage       = "couchdb"
	CouchdbTag         = "3.2.2"
	GrpcwebImage       = "ghcr.io/hyperledger-labs/grpc-web"
	GrpcwebTag         = "latest"
	ConsoleImage       = "ghcr.io/hyperledger-labs/fabric-console"
	ConsoleTag         = "latest"
	DeployerImage      = "ghcr.io/ibm-blockchain/fabric-deployer"
	DeployerTag        = "latest-amd64"
)
