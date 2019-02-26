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

package lcm

import (
	"github.com/AISphere/ffdl-commons/config"
	"github.com/AISphere/ffdl-lcm/service/lcm/learner"
)

func (t nonSplitTraining) Start() error {

	gpus := make(map[string]string)
	if t.req.Resources.Gpus > 0 {
		gpus["ibm-cloud.kubernetes.io/gpu-type"] = t.req.Resources.GpuType
	}

	learnerDefn := t.learner
	helperDefn := t.helper

	helperAndLearnerVolumes := append(learnerDefn.volumes, helperDefn.etcdVolume, helperDefn.sharedVolume)
	helperContainers := t.constructAuxillaryContainers(false)

	//now create the learner container
	useLogCollector := useLogCollectors(t.k8sClient, t.logr)
	learnerContainer := constructLearnerContainer(t.req, learnerDefn.envVars, learnerDefn.volumeMounts, helperDefn.sharedVolumeMount, learnerDefn.mountTrainingDataStoreInLearner, learnerDefn.mountResultsStoreInLearner, learnerDefn.mountSSHCertsInLearner, t.logr, useLogCollector)
	helperContainers = append(helperContainers, learnerContainer)

	imagePullSecret, err := learner.GenerateImagePullSecret(t.k8sClient, t.req)
	if err != nil {
		t.logr.WithError(err).Errorf("Could not create pull secret for %s", t.learner.name)
		return err
	}

	//create pod, service, statefuleset spec
	labelsMap := map[string]string{
		"training_id": t.req.TrainingId,
		"user_id":     t.req.UserId,
		"deploy_zone": t.req.Labels["deploy_zone"],
		"framework":   t.req.Framework + t.req.Version,
		"gpu_type":    t.req.Resources.GpuType,
		"kube_major":  t.req.Labels["kube_major"],
		"kube_minor":  t.req.Labels["kube_minor"],
		"cluster_env": t.req.Labels["cluster_env"],
	}
	gpuTolerations := getTolerations(t.req.Resources.GpuType, 30)
	termGracePeriodSecs := getTermGracePeriodSecs(0)

	if isCPUOnly(t.req.Resources.GpuType) {
		gpus["gpu/nvidia"] = "NA"
	}
	nonSplitLearnerPodSpec := learner.CreatePodSpec(helperContainers, helperAndLearnerVolumes, labelsMap, gpus, imagePullSecret, nil, gpuTolerations, termGracePeriodSecs)
	serviceSpec := learner.CreateServiceSpec(learnerDefn.name, t.req.TrainingId)
	statefulSetSpec := learner.CreateStatefulSetSpecForLearner(learnerDefn.name, serviceSpec.Name, learnerDefn.numberOfLearners, nonSplitLearnerPodSpec)

	numLearners := int(t.req.GetResources().Learners)

	return t.CreateFromBOM(&nonSplitTrainingBOM{
		learnerDefn.secrets,
		learnerDefn.networkingPolicy,
		serviceSpec,
		statefulSetSpec,
		numLearners,
	})

}

//CreateFromBOM ... eventually use with controller and make this transactional
func (t nonSplitTraining) CreateFromBOM(bom *nonSplitTrainingBOM) error {
	logr := t.logr
	namespace := config.GetLearnerNamespace()

	if bom.networkPolicy != nil {
		logr.Infof("Applying network policy for training")
		if _, err := t.k8sClient.NetworkingV1().NetworkPolicies(namespace).Create(bom.networkPolicy); err != nil {
			logr.WithError(err).Errorf("Failed in creating policy %s while deploying for training ", bom.networkPolicy.Name)
			return err
		}
	}

	for _, secret := range bom.secrets {
		//create the secrets
		if _, err := t.k8sClient.CoreV1().Secrets(namespace).Create(secret); err != nil {
			logr.WithError(err).Errorf("Failed in creating secrets %s while deploying for training ", secret.Name)
			return err
		}
	}

	if bom.numLearners > 1 {
		//create service
		if _, err := t.k8sClient.CoreV1().Services(namespace).Create(bom.service); err != nil {
			logr.WithError(err).Errorf("Failed in creating service %s while deploying for training ", bom.service.Name)
			return err
		}
	}

	//create the stateful set
	if _, err := t.k8sClient.AppsV1beta1().StatefulSets(namespace).Create(bom.learnerBOM); err != nil {
		logr.WithError(err).Errorf("Failed in creating statefulsets %s while deploying for training ", bom.learnerBOM.Name)
		return err
	}

	return nil

}
