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

package common

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetNodeSelectorTerms(arch []string, zone, region string) []corev1.NodeSelectorTerm {
	nodeSelectorTerms := []corev1.NodeSelectorTerm{
		{
			MatchExpressions: []corev1.NodeSelectorRequirement{},
		},
		{
			MatchExpressions: []corev1.NodeSelectorRequirement{},
		},
	}

	AddArchSelector(arch, &nodeSelectorTerms)
	AddZoneSelector(zone, &nodeSelectorTerms)
	AddRegionSelector(region, &nodeSelectorTerms)

	return nodeSelectorTerms
}

func AddArchSelector(arch []string, nodeSelectorTerms *[]corev1.NodeSelectorTerm) {
	if len(arch) != 0 {
		archNode := corev1.NodeSelectorRequirement{
			Key:      "kubernetes.io/arch",
			Operator: corev1.NodeSelectorOpIn,
			Values:   arch,
		}
		(*nodeSelectorTerms)[0].MatchExpressions = append((*nodeSelectorTerms)[0].MatchExpressions, archNode)
	}
}

func AddZoneSelector(zone string, nodeSelectorTerms *[]corev1.NodeSelectorTerm) {
	zoneNode := corev1.NodeSelectorRequirement{
		Key:      "topology.kubernetes.io/zone",
		Operator: corev1.NodeSelectorOpIn,
	}
	zoneNodeOld := corev1.NodeSelectorRequirement{
		Key:      "failure-domain.beta.kubernetes.io/zone",
		Operator: corev1.NodeSelectorOpIn,
	}
	if zone != "" {
		zoneNode.Values = []string{zone}
		zoneNodeOld.Values = []string{zone}
		(*nodeSelectorTerms)[0].MatchExpressions = append((*nodeSelectorTerms)[0].MatchExpressions, zoneNode)
		(*nodeSelectorTerms)[1].MatchExpressions = append((*nodeSelectorTerms)[1].MatchExpressions, zoneNodeOld)
	}
}

func AddRegionSelector(region string, nodeSelectorTerms *[]corev1.NodeSelectorTerm) {
	regionNode := corev1.NodeSelectorRequirement{
		Key:      "topology.kubernetes.io/region",
		Operator: corev1.NodeSelectorOpIn,
	}
	regionNodeOld := corev1.NodeSelectorRequirement{
		Key:      "failure-domain.beta.kubernetes.io/region",
		Operator: corev1.NodeSelectorOpIn,
	}
	if region != "" {
		regionNode.Values = []string{region}
		regionNodeOld.Values = []string{region}
		(*nodeSelectorTerms)[0].MatchExpressions = append((*nodeSelectorTerms)[0].MatchExpressions, regionNode)
		(*nodeSelectorTerms)[1].MatchExpressions = append((*nodeSelectorTerms)[1].MatchExpressions, regionNodeOld)
	}
}

func GetPodAntiAffinity(orgName string) *corev1.PodAntiAffinity {
	return &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{
				Weight: 100,
				PodAffinityTerm: corev1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "orgname",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{orgName},
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}
}
