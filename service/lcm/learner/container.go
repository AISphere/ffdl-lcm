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
	"github.com/AISphere/ffdl-lcm/lcmconfig"

	v1core "k8s.io/api/core/v1"
	v1resource "k8s.io/apimachinery/pkg/api/resource"
)

//Container ...
type Container struct {
	Image
	Resources
	VolumeMounts  []v1core.VolumeMount
	Name, Command string //FIXME eventually get rid of command as well
	EnvVars       []v1core.EnvVar
}

//Resources ...
type Resources struct {
	CPUs, Memory, GPUs v1resource.Quantity
}

//CreateContainerSpec ...
func CreateContainerSpec(container Container) v1core.Container {
	image := GetLearnerImageForFramework(container.Image)
	resources := generateResourceRequirements(container.CPUs, container.Memory, container.GPUs)
	mounts := container.VolumeMounts
	return generateContainerSpec(container.Name, image, container.Command, container.EnvVars, resources, mounts)
}

func generateContainerSpec(name, image, cmd string, vars []v1core.EnvVar, resourceRequirements v1core.ResourceRequirements, mounts []v1core.VolumeMount) v1core.Container {

	caps := v1core.Capabilities{
		Drop: []v1core.Capability{
			"CHOWN",
			"DAC_OVERRIDE",
			"FOWNER",
			"FSETID",
			"KILL",
			"SETPCAP",
			"NET_RAW",
			"MKNOD",
			"SETFCAP",
			// The remaining capabilities below are necessary. Dropping these will break the containers.
			// "SETGID",
			// "SETUID",
			// "NET_BIND_SERVICE", // Needed for ssh
			// "SYS_CHROOT",
			// "AUDIT_WRITE", // Needed for ssh
		},
	}
	securityContext := v1core.SecurityContext{
		Capabilities: &caps,
	}

	return v1core.Container{
		Name:            name,
		Image:           image,
		ImagePullPolicy: lcmconfig.GetImagePullPolicy(),
		Command:         []string{"bash", "-c", cmd},
		Env:             vars,
		Ports: []v1core.ContainerPort{
			v1core.ContainerPort{ContainerPort: int32(22), Protocol: v1core.ProtocolTCP},
			v1core.ContainerPort{ContainerPort: int32(2222), Protocol: v1core.ProtocolTCP},
		},
		Resources:       resourceRequirements,
		VolumeMounts:    mounts,
		SecurityContext: &securityContext,
	}
}

func generateResourceRequirements(cpus, memory, gpus v1resource.Quantity) v1core.ResourceRequirements {

	return v1core.ResourceRequirements{
		Requests: v1core.ResourceList{
			v1core.ResourceCPU:       cpus,
			v1core.ResourceMemory:    memory,
			v1core.ResourceNvidiaGPU: gpus,
		},
		Limits: v1core.ResourceList{
			v1core.ResourceCPU:       cpus,
			v1core.ResourceMemory:    memory,
			v1core.ResourceNvidiaGPU: gpus,
		},
	}
}
