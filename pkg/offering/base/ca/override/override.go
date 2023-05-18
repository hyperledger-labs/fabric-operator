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

package override

import (
	"encoding/json"
	"strings"

	current "github.com/IBM-Blockchain/fabric-operator/api/v1beta1"
	v1 "github.com/IBM-Blockchain/fabric-operator/pkg/apis/ca/v1"
	"github.com/IBM-Blockchain/fabric-operator/pkg/k8s/controllerclient"
	"github.com/IBM-Blockchain/fabric-operator/pkg/offering/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Override struct {
	Client controllerclient.Client
}

func (o *Override) IsPostgres(instance *current.IBPCA) bool {
	if instance.Spec.ConfigOverride != nil {
		if instance.Spec.ConfigOverride.CA != nil {
			caOverrides := &v1.ServerConfig{}
			err := json.Unmarshal(instance.Spec.ConfigOverride.CA.Raw, caOverrides)
			if err != nil {
				return false
			}

			if caOverrides.DB != nil {
				if strings.ToLower(caOverrides.DB.Type) == "postgres" {
					return true
				}
			}
		}

		if instance.Spec.ConfigOverride.TLSCA != nil {
			tlscaOverrides := &v1.ServerConfig{}
			err := json.Unmarshal(instance.Spec.ConfigOverride.TLSCA.Raw, tlscaOverrides)
			if err != nil {
				return false
			}

			if tlscaOverrides.DB != nil {
				if strings.ToLower(tlscaOverrides.DB.Type) == "postgres" {
					return true
				}
			}
		}
	}

	return false
}

func (o *Override) GetAffinity(instance *current.IBPCA) *corev1.Affinity {
	affinity := &corev1.Affinity{}

	affinity.NodeAffinity = o.GetNodeAffinity(instance)
	affinity.PodAntiAffinity = o.GetPodAntiAffinity(instance)

	return affinity
}

func (o *Override) GetNodeAffinity(instance *current.IBPCA) *corev1.NodeAffinity {
	arch := instance.Spec.Arch
	zone := instance.Spec.Zone
	region := instance.Spec.Region

	nodeSelectorTerms := []corev1.NodeSelectorTerm{
		{
			MatchExpressions: []corev1.NodeSelectorRequirement{},
		},
		{
			MatchExpressions: []corev1.NodeSelectorRequirement{},
		},
	}
	common.AddArchSelector(arch, &nodeSelectorTerms)

	if !o.IsPostgres(instance) {
		common.AddZoneSelector(zone, &nodeSelectorTerms)
		common.AddRegionSelector(region, &nodeSelectorTerms)
	}

	if len(nodeSelectorTerms[0].MatchExpressions) != 0 {
		return &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: nodeSelectorTerms,
			},
		}
	}

	return nil
}

func (o *Override) GetPodAntiAffinity(instance *current.IBPCA) *corev1.PodAntiAffinity {
	antiaffinity := &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{
				Weight: 100,
				PodAffinityTerm: corev1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{instance.GetName()},
							},
						},
					},
					TopologyKey: "topology.kubernetes.io/zone",
				},
			},
			{
				Weight: 100,
				PodAffinityTerm: corev1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{instance.GetName()},
							},
						},
					},
					TopologyKey: "failure-domain.beta.kubernetes.io/zone",
				},
			},
		},
	}

	if o.IsPostgres(instance) {
		term := corev1.WeightedPodAffinityTerm{
			Weight: 100,
			PodAffinityTerm: corev1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{instance.GetName()},
						},
					},
				},
				TopologyKey: "kubernetes.io/hostname",
			},
		}
		antiaffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(antiaffinity.PreferredDuringSchedulingIgnoredDuringExecution, term)
	}

	return antiaffinity
}
