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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeAffinityTrue(t *testing.T) {
	naffinity := getNodeAffinity(map[string]string{"dummyval": "1", "dummyval2": "2"})
	e, err := json.Marshal(naffinity)
	if err != nil {
		t.Fail()
	} else {
		assert.Equal(t, string(e), "{\"requiredDuringSchedulingIgnoredDuringExecution\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"failure-domain.beta.kubernetes.io/zone\",\"operator\":\"In\",\"values\":[\"\"]}]}]}}")
	}
}

func TestGpuTolerations(t *testing.T) {
	gpuToleration, err := json.Marshal(getTolerations("nvidia-TeslaV100", 30))
	if err != nil {
		t.Fail()
	} else {
		assert.Equal(t, "[{\"key\":\"node.kubernetes.io/not-ready\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":30},{\"key\":\"node.kubernetes.io/unreachable\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":30},{\"key\":\"dedicated\",\"operator\":\"Equal\",\"value\":\"gpu-task\",\"effect\":\"NoSchedule\"}]", string(gpuToleration))
	}
}

func TestCpuTolerations(t *testing.T) {
	gpuToleration, err := json.Marshal(getTolerations("CPU", 30))
	if err != nil {
		t.Fail()
	} else {
		assert.Equal(t, "[{\"key\":\"node.kubernetes.io/not-ready\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":30},{\"key\":\"node.kubernetes.io/unreachable\",\"operator\":\"Exists\",\"effect\":\"NoExecute\",\"tolerationSeconds\":30}]", string(gpuToleration))
	}
}
