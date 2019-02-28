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
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"github.com/AISphere/ffdl-lcm/service/lcm/policies"

	"github.com/AISphere/ffdl-lcm/service/lcm/certs"
	"github.com/AISphere/ffdl-lcm/service/lcm/helper"
	"github.com/AISphere/ffdl-lcm/service/lcm/learner"
	"github.com/sirupsen/logrus"

	"github.com/AISphere/ffdl-commons/config"
	"github.com/AISphere/ffdl-commons/logger"
	"github.com/AISphere/ffdl-lcm/service"

	"golang.org/x/net/context"

	"k8s.io/api/apps/v1beta1"
	v1core "k8s.io/api/core/v1"
	v1networking "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes"
)

// PodLevelJobDir represents the place to store the job state indicator files,
// as well as the $BREAK_FILE and $EXITCODE_FILE.
const PodLevelJobDir = "/job"

// tag specified in manifest and trainer when no result bucket is used
const noResultBucketTag = "none"

// PodLevelLogDir represents the place to store the per-learner logs.
const PodLevelLogDir = PodLevelJobDir + "/logs"

//Training ...
type Training interface {
	Start() error
}

type training struct {
	ctx        context.Context
	k8sClient  kubernetes.Interface
	req        *service.JobDeploymentRequest
	trainingID string
	learner    learnerDefinition
	helper     helperDefinition
	logr       *logger.LocLoggingEntry
}

type splitTrainingBOM struct {
	secrets              []*v1core.Secret
	networkPolicy        *v1networking.NetworkPolicy
	service              *v1core.Service
	sharedVolumeClaimBOM *v1core.PersistentVolumeClaim
	learnerBOM           *v1beta1.StatefulSet
	helperBOM            *v1beta1.Deployment
	numLearners          int
}

type nonSplitTrainingBOM struct {
	secrets       []*v1core.Secret
	networkPolicy *v1networking.NetworkPolicy
	service       *v1core.Service
	learnerBOM    *v1beta1.StatefulSet
	numLearners   int
}

type splitTraining struct {
	*training
}

type nonSplitTraining struct {
	*training
}

type learnerDefinition struct {
	secrets                                                                             []*v1core.Secret
	networkingPolicy                                                                    *v1networking.NetworkPolicy
	volumes                                                                             []v1core.Volume
	volumeMounts                                                                        []v1core.VolumeMount
	envVars                                                                             []v1core.EnvVar
	mountTrainingDataStoreInLearner, mountResultsStoreInLearner, mountSSHCertsInLearner bool
	numberOfLearners                                                                    int
	name                                                                                string
}

type helperDefinition struct {
	sharedVolume        v1core.Volume
	sslCertsVolume      v1core.Volume
	sslCertsVolumeMount v1core.VolumeMount
	etcdVolume          v1core.Volume
	etcdVolumeMount     v1core.VolumeMount
	sharedVolumeMount   v1core.VolumeMount
	sharedEnvVars       []v1core.EnvVar
	sharedVolumeClaim   *v1core.PersistentVolumeClaim
	name                string
}

//NewTraining ...
func NewTraining(ctx context.Context, k8sClient kubernetes.Interface, req *service.JobDeploymentRequest, log *logger.LocLoggingEntry) Training {
	const cosMountDriverName = "ibm/ibmc-s3fs"
	const cosMountType = "mount_cos"
	learnerName := fmt.Sprintf("learner-%s", req.Name)
	helperName := fmt.Sprintf("lhelper-%s", req.Name)
	numLearners := int(req.GetResources().Learners)
	if numLearners < 1 {
		numLearners = 1
	}

	mountTrainingDataStoreInLearner := req.EnvVars["DATA_STORE_TYPE"] == cosMountType
	mountResultsStoreInLearner := req.EnvVars["RESULT_STORE_TYPE"] == cosMountType
	mountSSHCertsInLearner := certs.NeedsMountedSSHCerts(req.Framework, req.Version)

	logr := log.WithFields(logrus.Fields{
		"learner_name": learnerName,
		"helper_name":  helperName,
		"mounted_cos":  mountResultsStoreInLearner && mountTrainingDataStoreInLearner,
	})

	if req.EnvVars["RESULT_STORE_OBJECTID"] == noResultBucketTag {
		mountResultsStoreInLearner = false
	}
	envVarsFromDeploymentRequest := extractEnvVarsFromDeploymentRequest(req) //shared across all containers of training
	envvarsForLearner := envVarsForDeployingLearner(envVarsFromDeploymentRequest, req.TrainingId,
		numLearners, learnerName, mountTrainingDataStoreInLearner, mountResultsStoreInLearner) //only for learner

	learnerVolumes := volumesForLearner(req, envvarsForLearner, mountTrainingDataStoreInLearner, mountResultsStoreInLearner, logr)
	if config.IsFfDLExtendedEnabled() {
		learnerVolumeSpecs := learnerVolumes.CreateVolumeForLearner()
		learnerVolumeSpecs = extendLearnerVolumes(learnerVolumeSpecs, logr)
	}
	learnerDefn := learnerDefinition{
		secrets:                         secretsForDeployingLearner(req, mountTrainingDataStoreInLearner, mountResultsStoreInLearner),
		networkingPolicy:                networkPoliciesForDistributedLearners(numLearners, req),
		volumes:                         learnerVolumes.CreateVolumeForLearner(),
		volumeMounts:                    learnerVolumes.CreateVolumeMountsForLearner(),
		envVars:                         envvarsForLearner,
		numberOfLearners:                numLearners,
		mountTrainingDataStoreInLearner: mountTrainingDataStoreInLearner,
		mountResultsStoreInLearner:      mountResultsStoreInLearner,
		mountSSHCertsInLearner:          mountSSHCertsInLearner,
		name:                            learnerName,
	}

	helperVolumes := volumesForHelper(req, logr)
	helperDefn := helperDefinition{
		sslCertsVolume:      helperVolumes.CreateSSLVolume(),
		sslCertsVolumeMount: helperVolumes.CreateSSLVolumeMount(),
		etcdVolume:          helperVolumes.CreateETCDVolume(),
		etcdVolumeMount:     helperVolumes.CreateETCDVolumeMount(),
		sharedEnvVars:       envVarsFromDeploymentRequest,
		sharedVolume:        helperVolumes.CreateDataVolume(req.Name),
		sharedVolumeMount:   helperVolumes.CreateDataVolumeMount(),
		sharedVolumeClaim:   helperVolumes.DynamicPVCReference(),
		name:                helperName,
	}

	if helperVolumes.SharedNonSplitLearnerHelperVolume != nil {
		//this should not be the default case, we should be running in split mode by default
		logr.Warnf("starting deploying learner infra for non split learning, this is not expected")
		return nonSplitTraining{&training{ctx, k8sClient, req, req.TrainingId, learnerDefn, helperDefn, logr}}
	}
	logr.Infof("starting deploying learner infra for split learning")
	return splitTraining{&training{ctx, k8sClient, req, req.TrainingId, learnerDefn, helperDefn, logr}}
}

///-------

func secretsForDeployingLearner(req *service.JobDeploymentRequest, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool) []*v1core.Secret {
	//irrespective of split/non split learners these secrets need to be created

	secretsStruct := learner.Secrets{}

	if mountTrainingDataStoreInLearner {
		trainingMountSecretName := "cossecretdata-" + req.Name
		secretsStruct.TrainingDataSecret = &learner.COSVolumeSecret{ID: trainingMountSecretName, TrainingID: req.TrainingId, Username: req.EnvVars["DATA_STORE_USERNAME"], APIKey: req.EnvVars["DATA_STORE_APIKEY"]}
	}

	if mountResultsStoreInLearner {
		resultsMountSecretName := "cossecretresults-" + req.Name
		secretsStruct.ResultsDirSecret = &learner.COSVolumeSecret{ID: resultsMountSecretName, TrainingID: req.TrainingId, Username: req.EnvVars["RESULT_STORE_USERNAME"], APIKey: req.EnvVars["RESULT_STORE_APIKEY"]}
	}

	if certs.NeedsMountedSSHCerts(req.Framework, req.Version) {
		sshSecretName := "jobsshcert-" + req.Name
		secretsStruct.SSHVolumeSecret = &learner.SSHVolumeSecret{ID: sshSecretName, TrainingID: req.TrainingId, Framework: req.Framework, Version: req.Version}
	}

	secretSpecs := learner.CreateVolumeSecretsSpec(secretsStruct)

	return secretSpecs
}

func volumesForLearner(req *service.JobDeploymentRequest, learnerEnvVars []v1core.EnvVar, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool, logr *logger.LocLoggingEntry) learner.Volumes {
	volumesStruct := learner.Volumes{}
	if certs.NeedsMountedSSHCerts(req.Framework, req.Version) {
		volumesStruct.SSHVolume = &learner.SSHVolume{ID: "sshcertmount-" + req.Name, SecretName: "jobsshcert-" + req.Name,
			MountSpec: learner.VolumeMountSpec{MountPath: "/etc/ssh-certs", SubPath: ""}}
	}

	shmVolumeSize := getSHMVolumeSize(req.Framework, req.Version)
	if shmVolumeSize > 0 {
		volumesStruct.SHMVolume = &learner.SHMVolume{ID: "shmvolume-" + req.Name, Size: shmVolumeSize, MountSpec: learner.VolumeMountSpec{MountPath: "/dev/shm"}}
	}

	if mountTrainingDataStoreInLearner {
		region := req.EnvVars["DATA_STORE_REGION"]
		if region == "" {
			region = "us-standard"
		}
		configValStr := config.GetString("MOUNTCOS_GB_CACHE_PER_GPU")
		cacheSize, err := strconv.Atoi(configValStr)
		if err != nil {
			cacheSize = 6
			logr.Warnf("DLAAS_MOUNTCOS_GB_CACHE_PER_GPU value %s is not an integer.  Defaulting to %dGB/GPU", configValStr, cacheSize)
		}
		cacheSize = cacheSize * int(req.Resources.Gpus)
		// reserve 1/3 of cache for prefetching, up to a limit (diskFree is specified in MB, cache in GB)
		diskFree := (cacheSize * 1024) / 3
		if diskFree > 10000 {
			diskFree = 10000
		}

		buckets := getDatastoreBuckets(req.EnvVars)
		cacheSizePerBucket := cacheSize / len(buckets)
		for k, v := range buckets {
			if k != "DATA_STORE_OBJECTID" {
				bucketIdentifier := strings.TrimPrefix(k, "DATA_STORE_OBJECTID_")
				dataDirectory := "DATA_DIR_" + bucketIdentifier
				volumesStruct.TrainingDataVolumes = append(volumesStruct.TrainingDataVolumes, &learner.COSVolume{
					ID:        "cosinputmount-" + bucketIdentifier + "-" + req.Name,
					Region:    region,
					Bucket:    v,
					Endpoint:  req.EnvVars["DATA_STORE_AUTHURL"],
					SecretRef: "cossecretdata-" + req.Name,
					MountSpec: learner.VolumeMountSpec{
						MountPath: getValue(learnerEnvVars, dataDirectory),
						SubPath:   "",
					},
					CacheSize: strconv.Itoa(cacheSizePerBucket),
					DiskFree:  strconv.Itoa(diskFree),
				})
			}
		}
	}
	if mountResultsStoreInLearner {
		region := req.EnvVars["RESULT_STORE_REGION"]
		if region == "" {
			region = "us-standard"
		}
		resultBucketDir := getValue(learnerEnvVars, "RESULT_BUCKET_DIR")
		// drop the "/mnt/results" part of the path and only keep the bucket name
		_, resultBucketName := path.Split(resultBucketDir)
		volumesStruct.ResultsDir = &learner.COSVolume{
			ID:        "cosoutputmount-" + req.Name,
			Region:    region,
			Bucket:    resultBucketName,
			Endpoint:  req.EnvVars["RESULT_STORE_AUTHURL"],
			SecretRef: "cossecretresults-" + req.Name,
			MountSpec: learner.VolumeMountSpec{
				MountPath: resultBucketDir,
				SubPath:   "",
			},
			CacheSize: "0",
			DiskFree:  "2048",
		}
	}

	return volumesStruct
}

func getDatastoreBuckets(envVars map[string]string) map[string]string {
	buckets := make(map[string]string)
	buckets["DATA_STORE_OBJECTID"] = envVars["DATA_STORE_OBJECTID"]
	for k, v := range envVars {
		if strings.HasPrefix(k, "DATA_STORE_OBJECTID_") {
			buckets[k] = v
		}
	}
	return buckets
}

func volumesForHelper(req *service.JobDeploymentRequest, logr *logger.LocLoggingEntry) helper.Volumes {
	volumesStruct := helper.Volumes{}

	volumesStruct.ETCDVolume = &helper.ETCDVolume{Name: "etcd-ssl-cert"}

	volumeSize := getStorageSize(req.Resources)
	logr.Debugf("Requested storage for job of size %d bytes", volumeSize)
	useDynamicExternalVolume := volumeSize > 0

	staticVolumeName := getStaticVolume(req.Labels["deploy_zone"], logr)
	logr.Debugf("Static volume for job: %s", staticVolumeName)
	useStaticExternalVolume := len(staticVolumeName) > 0

	logr.Infof("DLAAS_LCM_FLUENTD_EMETRICS_ENABLE is set to %v", viper.GetBool(config.LcmFluentdEmetricsEnable))

	useSplitLearner := useDynamicExternalVolume || useStaticExternalVolume

	if !useSplitLearner || viper.GetBool(config.LcmFluentdEmetricsEnable) {
		logr.Infof("Starting training %s with NON SPLIT MODE %d", req.TrainingId, volumeSize)
		volumesStruct.SharedNonSplitLearnerHelperVolume = &helper.LocalVolume{
			Name: "jobdata",
			MountSpec: helper.VolumeMountSpec{
				MountPath: PodLevelJobDir,
				SubPath:   req.TrainingId,
			},
		}
	} else {
		if useStaticExternalVolume {
			logr.Infof("Using static external volume for training %s with name %s", req.TrainingId, staticVolumeName)
			volumesStruct.SharedSplitLearnerHelperVolume = &helper.SharedNFSVolume{Name: "jobdata", PVCClaimName: staticVolumeName, PVC: nil,
				MountSpec: helper.VolumeMountSpec{MountPath: PodLevelJobDir, SubPath: req.TrainingId}}

		} else if useDynamicExternalVolume {
			sharedVolumeClaim := constructVolumeClaim(req.Name, config.GetLearnerNamespace(), volumeSize, map[string]string{"training_id": req.TrainingId})
			logr.Infof("Using dynamic external volume for Training %s with name %s", req.TrainingId, sharedVolumeClaim.Name)
			volumesStruct.SharedSplitLearnerHelperVolume = &helper.SharedNFSVolume{Name: "jobdata", PVCClaimName: sharedVolumeClaim.Name, PVC: sharedVolumeClaim,
				MountSpec: helper.VolumeMountSpec{MountPath: PodLevelJobDir, SubPath: req.TrainingId}}
		}
	}
	return volumesStruct
}

//list of shared env vars shared by all containers in helper and learner pod
func extractEnvVarsFromDeploymentRequest(req *service.JobDeploymentRequest) []v1core.EnvVar {
	var envVars []v1core.EnvVar
	for k, v := range req.EnvVars {
		envVars = append(envVars, v1core.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	return envVars
}

//needs to happen before the volume creation since we muck around with the value and change the paths of data/result dir
func envVarsForDeployingLearner(existingEnvVars []v1core.EnvVar, trainingID string, numLearners int, statefulsetName string, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool) []v1core.EnvVar {
	return learner.PopulateLearnerEnvVariablesAndLabels(existingEnvVars, trainingID, numLearners, statefulsetName, mountTrainingDataStoreInLearner, mountResultsStoreInLearner)

}

func networkPoliciesForDistributedLearners(numberOfLearners int, req *service.JobDeploymentRequest) *v1networking.NetworkPolicy {
	if numberOfLearners > 1 { //network policies are only applicable for distributed learners
		return policies.DefineNetworkPoliciesForTrainingID(req.Name, req.TrainingId)
	}
	return nil

}

func (t *training) constructAuxillaryContainers(isSplit bool) []v1core.Container {
	learnerDefn := t.learner
	helperDefn := t.helper
	skipStoreResults := learnerDefn.mountResultsStoreInLearner || getValue(learnerDefn.envVars, "RESULT_STORE_OBJECTID") == noResultBucketTag
	helperContainers := []v1core.Container{
		constructControllerContainer(t.req.TrainingId, helperDefn.etcdVolumeMount, helperDefn.sharedVolumeMount, learnerDefn.mountTrainingDataStoreInLearner, skipStoreResults),
	}
	if useLogCollectors(t.k8sClient, t.logr) {
		var sslCertsVolumeMount *v1core.VolumeMount = nil
		sslCertsVolumeMount = &helperDefn.sslCertsVolumeMount
		helperContainers = append(helperContainers,
			constructLogCollector(
				sslCertsVolumeMount,
				helperDefn.sharedVolumeMount,
				t.k8sClient, t.req, helperDefn.sharedEnvVars, t.logr))
	}

	if !learnerDefn.mountTrainingDataStoreInLearner {
		helperContainers = append(helperContainers, constructLoadTrainingDataContainer(helperDefn.sharedVolumeMount, helperDefn.sharedEnvVars))
	}
	if !learnerDefn.mountResultsStoreInLearner && getValue(learnerDefn.envVars, "RESULT_STORE_OBJECTID") != noResultBucketTag {
		helperContainers = append(helperContainers, constructLoadModelContainer(helperDefn.sharedVolumeMount, helperDefn.sharedEnvVars))
		helperContainers = append(helperContainers, constructStoreResultsContainer(helperDefn.sharedVolumeMount, helperDefn.sharedEnvVars))
		helperContainers = append(helperContainers, constructStoreLogsContainer(helperDefn.sharedVolumeMount, helperDefn.sharedEnvVars))
	}
	return helperContainers
}

// Returns the amount of shared memory to give to the learner, in bytes.  A return value of 0 indicates that the default amount of memory should be used.
func getSHMVolumeSize(framework, version string) int64 {
	var result int64 // defaults to 0
	if strings.EqualFold(framework, "pytorch") {
		result = 4194304 // 4GB
	}
	return result
}

func getTolerations(gpuType string, tolerationSeconds int) []v1core.Toleration {
	tolSecs := int64(tolerationSeconds)
	notReadyToleration := v1core.Toleration{
		Key:               "node.kubernetes.io/not-ready",
		Operator:          v1core.TolerationOpExists,
		Effect:            v1core.TaintEffectNoExecute,
		TolerationSeconds: &tolSecs,
	}
	unreachableToleration := v1core.Toleration{
		Key:               "node.kubernetes.io/unreachable",
		Operator:          v1core.TolerationOpExists,
		Effect:            v1core.TaintEffectNoExecute,
		TolerationSeconds: &tolSecs,
	}

	tolerations := []v1core.Toleration{
		notReadyToleration,
		unreachableToleration,
	}

	if gpuType != "CPU" {
		gpuToleration := v1core.Toleration{
			Key:      "dedicated",
			Operator: v1core.TolerationOpEqual,
			Value:    "gpu-task",
			Effect:   v1core.TaintEffectNoSchedule,
		}
		gpuTolerations := append(tolerations, gpuToleration)
		return gpuTolerations
	}

	return tolerations
}

// By setting terminationGracePeriod to 0, this allows the master to spin up a replacement statefulset learner pod in the event that the learner's node goes AWOL on the network.
func getTermGracePeriodSecs(secs int) int64 {
	return int64(secs)
}

func isCPUOnly(gpuType string) bool {
	if strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(gpuType)) == "CPU" {
		return true
	}
	return false
}
