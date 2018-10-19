/*
 * Copyright 2018. IBM Corporation
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

package learner

import (
	v1core "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
)

const cosMountDriverName = "ibm/ibmc-s3fs"

//COSVolume ...
type COSVolume struct {
	ID, Region, Bucket, Endpoint, SecretRef, CacheSize, DiskFree string
	MountSpec                                                    VolumeMountSpec
}

//SHMVolume ...
type SHMVolume struct {
	ID string
	Size int64
	MountSpec      VolumeMountSpec
}
//Volumes ...
type Volumes struct {
	TrainingData *COSVolume
	ResultsDir   *COSVolume
	SHMVolume    *SHMVolume
}

//VolumeMountSpec ...
type VolumeMountSpec struct {
	MountPath, SubPath string
}

//CreateVolumeForLearner ...
func (volumes Volumes) CreateVolumeForLearner() []v1core.Volume {

	var volumeSpecs []v1core.Volume
	
	if volumes.SHMVolume != nil {
		volumeSpecs = append(volumeSpecs, generateSHMVolume(volumes.SHMVolume.ID, volumes.SHMVolume.Size))
	}

	if volumes.TrainingData != nil {
		trainingDataParams := volumes.TrainingData
		volumeSpecs = append(volumeSpecs, generateCOSTrainingDataVolume(trainingDataParams.ID, trainingDataParams.Region, trainingDataParams.Bucket,
			trainingDataParams.Endpoint, trainingDataParams.SecretRef, trainingDataParams.CacheSize, trainingDataParams.DiskFree))
	}

	if volumes.ResultsDir != nil {
		resultDirParams := volumes.ResultsDir
		volumeSpecs = append(volumeSpecs, generateCOSResultsVolume(resultDirParams.ID, resultDirParams.Region, resultDirParams.Bucket,
			resultDirParams.Endpoint, resultDirParams.SecretRef, resultDirParams.CacheSize, resultDirParams.DiskFree))
	}

	return volumeSpecs
}

//CreateVolumeMountsForLearner ...
func (volumes Volumes) CreateVolumeMountsForLearner() []v1core.VolumeMount {

	var mounts []v1core.VolumeMount
	if volumes.SHMVolume != nil {
		mounts = append(mounts, generateVolumeMount(volumes.SHMVolume.ID, volumes.SHMVolume.MountSpec.MountPath))
	}
	if volumes.TrainingData != nil {
		mounts = append(mounts, generateVolumeMount(volumes.TrainingData.ID, volumes.TrainingData.MountSpec.MountPath))
	}
	if volumes.ResultsDir != nil {
		mounts = append(mounts, generateVolumeMount(volumes.ResultsDir.ID, volumes.ResultsDir.MountSpec.MountPath))
	}

	return mounts
}

func generateSHMVolume(id string, size int64) v1core.Volume {
	shmVolume := v1core.Volume{
		Name: id,
		VolumeSource: v1core.VolumeSource{
			EmptyDir: &v1core.EmptyDirVolumeSource{
				Medium:  v1core.StorageMediumMemory,
				SizeLimit: resource.NewQuantity(size, resource.DecimalSI),
			},
		},
	}
	return shmVolume
}

func generateCOSTrainingDataVolume(id, region, bucket, endpoint, secretRef, cacheSize, diskfree string) v1core.Volume {
	cosInputVolume := v1core.Volume{
		Name: id,
		VolumeSource: v1core.VolumeSource{
			FlexVolume: &v1core.FlexVolumeSource{
				Driver:    cosMountDriverName,
				FSType:    "",
				SecretRef: &v1core.LocalObjectReference{Name: secretRef},
				ReadOnly:  false,
				Options: map[string]string{
					"bucket":           bucket,
					"endpoint":         endpoint,
					"region":           region,
					"cache-size-gb":    cacheSize, // Amount of host memory to use for cache
					"chunk-size-mb":    "52",      // value suggested for cruiser10 by benchmarking with a dallas COS instance
					"parallel-count":   "20",      // should be at least expected file size / chunk size.  Extra threads will just sit idle
					"ensure-disk-free": diskfree,  // don't completely fill the cache, leave some buffer for parallel thread pulls
					"tls-cipher-suite": "DEFAULT",
					"multireq-max":     "20",
					"stat-cache-size":  "100000",
					"kernel-cache":     "true",
					"debug-level":      "warn",
					"curl-debug":       "false",
					"s3fs-fuse-retry-count": "30", // 4 second delay between retries * 30 = 2min
				},
			},
		},
	}

	return cosInputVolume
}

func generateCOSResultsVolume(id, region, bucket, endpoint, secretRef, cacheSize, diskfree string) v1core.Volume {
	cosOutputVolume := v1core.Volume{
		Name: id,
		VolumeSource: v1core.VolumeSource{
			FlexVolume: &v1core.FlexVolumeSource{
				Driver:    cosMountDriverName,
				FSType:    "",
				SecretRef: &v1core.LocalObjectReference{Name: secretRef},
				ReadOnly:  false,
				Options: map[string]string{
					"bucket":   bucket,
					"endpoint": endpoint,
					"region":   region,
					// tuning values suitable for writing checkpoints and logs
					"cache-size-gb":    cacheSize,
					"chunk-size-mb":    "52",
					"parallel-count":   "5",
					"ensure-disk-free": diskfree,
					"tls-cipher-suite": "DEFAULT",
					"multireq-max":     "20",
					"stat-cache-size":  "100000",
					"kernel-cache":     "false",
					"debug-level":      "warn",
					"curl-debug":       "false",
					"s3fs-fuse-retry-count": "30", // 4 second delay between retries * 30 = 2min
				},
			},
		},
	}

	return cosOutputVolume
}

func generateVolumeMount(id, bucket string) v1core.VolumeMount {
	return v1core.VolumeMount{
		Name:      id,
		MountPath: bucket,
	}
}