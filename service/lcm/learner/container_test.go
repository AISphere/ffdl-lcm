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

package learner

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	v1core "k8s.io/api/core/v1"
	v1resource "k8s.io/apimachinery/pkg/api/resource"
)

func TestContainerWithMountedCOS(t *testing.T) {

	var envars []v1core.EnvVar
	cpuCount := v1resource.NewMilliQuantity(int64(float64(1)*1000.0), v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(1), v1resource.DecimalSI)
	memCount := v1resource.NewQuantity(1024, v1resource.DecimalSI)

	container := Container{
		Image: Image{Framework: "tensorflow", Version: "1.5", Tag: "latest"},
		Resources: Resources{
			CPUs: *cpuCount, Memory: *memCount, GPUs: *gpuCount,
		},
		VolumeMounts: []v1core.VolumeMount{v1core.VolumeMount{MountPath: "/nfs"},
			v1core.VolumeMount{MountPath: "/nfs"}, v1core.VolumeMount{MountPath: "/cos/data"}, v1core.VolumeMount{MountPath: "/cos/results"}},
		Name:    "test-learner-container",
		EnvVars: envars,
		Command: "echo hello",
	}

	containerCreated := CreateContainerSpec(container, "1", "10")
	assert.Equal(t, 4, len(containerCreated.VolumeMounts))

}

func TestContainerWithNoMountedCOS(t *testing.T) {

	var envars []v1core.EnvVar
	cpuCount := v1resource.NewMilliQuantity(int64(float64(1)*1000.0), v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(1), v1resource.DecimalSI)
	memCount := v1resource.NewQuantity(1024, v1resource.DecimalSI)

	container := Container{
		Image: Image{Framework: "tensorflow", Version: "1.5", Tag: "latest"},
		Resources: Resources{
			CPUs: *cpuCount, Memory: *memCount, GPUs: *gpuCount,
		},
		VolumeMounts: []v1core.VolumeMount{v1core.VolumeMount{MountPath: "/nfs"},
			v1core.VolumeMount{MountPath: "/nfs"}},
		Name:    "test-learner-container",
		EnvVars: envars,
		Command: "echo hello",
	}

	containerCreated := CreateContainerSpec(container, "1", "10")
	assert.Equal(t, 2, len(containerCreated.VolumeMounts))

}

// TODO: Alter these tests when Armada integrates kube 1.10 `nvidia.com/gpu` labels
//       AND we `glide upgrade k8s.io/client-go
func TestGenerateResourceRequirementsKubeMinor10(t *testing.T) {

	cpuCount := v1resource.NewMilliQuantity(int64(float64(1)*1000.0), v1resource.DecimalSI)
	memCount := v1resource.NewQuantity(1024, v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(1), v1resource.DecimalSI)

	expectedRequirements := v1core.ResourceRequirements{
		Requests: v1core.ResourceList{
			v1core.ResourceCPU:               *cpuCount,
			v1core.ResourceMemory:            *memCount,
			"alpha.kubernetes.io/nvidia-gpu": *gpuCount,
		},
		Limits: v1core.ResourceList{
			v1core.ResourceCPU:               *cpuCount,
			v1core.ResourceMemory:            *memCount,
			"alpha.kubernetes.io/nvidia-gpu": *gpuCount,
		},
	}
	actualRequirements := generateResourceRequirements(*cpuCount, *memCount, *gpuCount, "1", "10")

	assert.Equal(t, expectedRequirements, actualRequirements)
}

func TestGenerateResourceRequirementsKubeMinor11(t *testing.T) {

	cpuCount := v1resource.NewMilliQuantity(int64(float64(1)*1000.0), v1resource.DecimalSI)
	memCount := v1resource.NewQuantity(1024, v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(1), v1resource.DecimalSI)

	expectedRequirements := v1core.ResourceRequirements{
		Requests: v1core.ResourceList{
			v1core.ResourceCPU:    *cpuCount,
			v1core.ResourceMemory: *memCount,
			"nvidia.com/gpu":      *gpuCount,
		},
		Limits: v1core.ResourceList{
			v1core.ResourceCPU:    *cpuCount,
			v1core.ResourceMemory: *memCount,
			"nvidia.com/gpu":      *gpuCount,
		},
	}
	actualRequirements := generateResourceRequirements(*cpuCount, *memCount, *gpuCount, "1", "11")

	assert.Equal(t, expectedRequirements, actualRequirements)
}

// END TODO

func TestGenerateResourceRequirementsKubeMinor8Plus(t *testing.T) {

	cpuCount := v1resource.NewMilliQuantity(int64(float64(1)*1000.0), v1resource.DecimalSI)
	memCount := v1resource.NewQuantity(1024, v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(1), v1resource.DecimalSI)

	expectedRequirements := v1core.ResourceRequirements{
		Requests: v1core.ResourceList{
			v1core.ResourceCPU:               *cpuCount,
			v1core.ResourceMemory:            *memCount,
			"alpha.kubernetes.io/nvidia-gpu": *gpuCount,
		},
		Limits: v1core.ResourceList{
			v1core.ResourceCPU:               *cpuCount,
			v1core.ResourceMemory:            *memCount,
			"alpha.kubernetes.io/nvidia-gpu": *gpuCount,
		},
	}
	// strings.Trim would be called in deployDistributedTrainingJob when the minor label is set
	actualRequirements := generateResourceRequirements(*cpuCount, *memCount, *gpuCount, "1", strings.Trim("8+", "+"))

	assert.Equal(t, expectedRequirements, actualRequirements)
}
