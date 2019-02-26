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
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {

	learnerConfigPath = "../../../testdata/learner-config.json" //Uses configmap in testdata directory

}

func TestGetImageNameWithCustomRegistry(t *testing.T) {
	t.Skip("Skipping this test for right now")
	image := Image{
		Framework: "tensorflow",
		Version:   "1.5",
		Tag:       "latest",
		Registry:  "registry.ng.bluemix.net",
		Namespace: "custom_reg",
	}

	learnerImage := GetLearnerImageForFramework(image)
	assert.Equal(t, "registry.ng.bluemix.net/custom_reg/tensorflow:1.5", learnerImage)
}

func TestGetValidImageName(t *testing.T) {
	t.Skip("Skipping this test for right now")
	image := Image{
		Framework: "tensorflow",
		Version:   "1.5",
		Tag:       "latest",
	}
	learnerImage := GetLearnerImageForFramework(image)
	assert.Equal(t, "registry.ng.bluemix.net/dlaas_dev/tensorflow_gpu_1.5:latest", learnerImage)

}

func TestGetInvalidImageName(t *testing.T) {
	t.Skip("Skipping this test for right now")
	image := Image{
		Framework: "dlaas_config_test",
		Version:   "2.2",
		Tag:       "",
	}
	learnerImage := GetLearnerImageForFramework(image)
	assert.Equal(t, "registry.ng.bluemix.net/dlaas_dev/dlaas_config_test_gpu_2.2:master-2", learnerImage)

}
