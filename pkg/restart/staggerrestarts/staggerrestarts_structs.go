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

package staggerrestarts

// RestartConfig defines <ca/peer/orderer>-restart-config.Data["restart-config.yaml"]
type RestartConfig struct {
	// key is mspid
	Queues map[string][]*Component
	// key is instance name
	Log map[string][]*Component
}

type Status string

const (
	Pending   Status = "pending"
	Waiting   Status = "waiting"
	Completed Status = "completed"
	Expired   Status = "expired"
	Deleted   Status = "deleted"

	Restarted Status = "restarted"
)

type Component struct {
	CRName               string
	Reason               string
	CheckUntilTimestamp  string
	LastCheckedTimestamp string
	Status               Status
	PodName              string
}

func (r *RestartConfig) AddToLog(component *Component) {
	if r.Log == nil {
		r.Log = map[string][]*Component{}
	}
	r.Log[component.CRName] = append(r.Log[component.CRName], component)
}

func (r *RestartConfig) AddToQueue(mspid string, component *Component) {
	if r.Queues == nil {
		r.Queues = map[string][]*Component{}
	}
	r.Queues[mspid] = append(r.Queues[mspid], component)
}

func (r *RestartConfig) PopFromQueue(mspid string) {
	r.Queues[mspid] = r.Queues[mspid][1:]
}
