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

package jobmonitor

import (
	"reflect"
	"testing"

	"github.com/AISphere/ffdl-commons/config"
	"github.com/AISphere/ffdl-commons/metricsmon"
	"github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/assert"
)

func init() {
	config.InitViper()
}

func TestTransitions(t *testing.T) {

	jm := initJobMonitor()

	assert.EqualValues(t, true, jm.isTransitionAllowed("PENDING", "DOWNLOADING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "PROCESSING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "STORING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "COMPLETED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("PROCESSING", "STORING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("STORING", "COMPLETED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("PROCESSING", "COMPLETED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "FAILED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "HALTED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("PROCESSING", "FAILED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("PROCESSING", "PROCESSING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("STORING", "FAILED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("STORING", "HALTED"))

	assert.EqualValues(t, false, jm.isTransitionAllowed("STORING", "DOWNLOADING"))
	assert.EqualValues(t, false, jm.isTransitionAllowed("COMPLETED", "PROCESSING"))
	assert.EqualValues(t, false, jm.isTransitionAllowed("FAILED", "COMPLETED"))

}

func TestMetrics(t *testing.T) {

	jm := initJobMonitor()

	assert.NotNil(t, jm.metrics, "Metrics failed to be initialized")

	assert.NotPanics(t, func() {
		jm.metrics.FailedETCDConnectivityCounter.Add(1)
		jm.metrics.FailedETCDWatchCounter.Add(1)
		jm.metrics.FailedImagePullK8sErrorCounter.Add(1)
		jm.metrics.FailedK8sConnectivityCounter.Add(1)
		jm.metrics.FailedTrainerConnectivityCounter.Add(1)
		jm.metrics.InsufficientK8sResourcesErrorCounter.Add(1)
	}, "Metrics failed to be incremented")
}

func TestMetricsReflection(t *testing.T) {

	jm := initJobMonitor()

	assert.NotNil(t, jm.metrics, "Metrics failed to be initialized")

	allMetrics := reflect.ValueOf(jm.metrics).Elem()

	for i := 0; i < allMetrics.NumField(); i++ {
		metric := allMetrics.Field(i)
		metricName := allMetrics.Type().Field(i).Name
		assert.NotPanics(t, func() {
			// metric is of type metrics.Counter
			if metric.Type() == reflect.TypeOf((*metrics.Counter)(nil)).Elem() {
				counter := metric.Interface().(metrics.Counter)
				counter.Add(1)
			}
		}, "Metric failed to be incremented: "+metricName)
	}
}

func initJobMonitor() *JobMonitor {

	statsdClient := metricsmon.NewStatsdClient("jobmonitor")

	jm := &JobMonitor{
		TrainingID:  "unit-test-trainingId",
		UserID:      "unit-test-userId",
		NumLearners: 1,
		JobName:     "unit-test-jobName",
		trMap:       initTransitionMap(),
		metrics:     initMetrics(statsdClient),
	}
	return jm
}
