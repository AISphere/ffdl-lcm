/*
 * Copyright 2017-2018 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package policies

import (
	k8sv1networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//DefineNetworkPoliciesForTrainingID ... network policies to only allow ingress egress from learner pods with same training id
func DefineNetworkPoliciesForTrainingID(name, trainingID string) *k8sv1networking.NetworkPolicy {
	networkPoliciesSpec := k8sv1networking.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"training_id": trainingID,
			},
		},
		Spec: k8sv1networking.NetworkPolicySpec{
			//This policy applies to pods with the following label
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"training_id": trainingID,
					"service":     "dlaas-learner",
				},
			},
			//We're adding ingress and egress polcies
			PolicyTypes: []k8sv1networking.PolicyType{
				k8sv1networking.PolicyTypeIngress,
				k8sv1networking.PolicyTypeEgress,
			},
			//Allow ingress only from peers (pods) that have the following label
			Ingress: []k8sv1networking.NetworkPolicyIngressRule{
				k8sv1networking.NetworkPolicyIngressRule{
					From: []k8sv1networking.NetworkPolicyPeer{
						k8sv1networking.NetworkPolicyPeer{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"training_id": trainingID,
									"service":     "dlaas-learner",
								},
							},
						},
					},
				},
			},
			//Allow egress only to peers (pods) that have the following label
			Egress: []k8sv1networking.NetworkPolicyEgressRule{
				k8sv1networking.NetworkPolicyEgressRule{
					To: []k8sv1networking.NetworkPolicyPeer{
						k8sv1networking.NetworkPolicyPeer{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"training_id": trainingID,
									"service":     "dlaas-learner",
								},
							},
						},
					},
				},
			},
		},
	}

	return &networkPoliciesSpec

}
