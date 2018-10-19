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
	"fmt"

	"github.com/spf13/viper"
	"github.com/AISphere/ffdl-commons/config"
	framework "github.com/AISphere/ffdl-commons/framework"
)

//Image ...
type Image struct {
	Framework, Version, Tag, Registry, Namespace string
}

var learnerConfigPath = "/etc/learner-config-json/learner-config.json"

//GetLearnerImageForFramework returns the full route for the learner image
func GetLearnerImageForFramework(image Image) string {
	var learnerImage string
	if image.Registry != "" && image.Namespace != "" {
		learnerImage = fmt.Sprintf("%s/%s/%s:%s", image.Registry, image.Namespace, image.Framework, image.Version)
	} else {
		learnerTag := getLearnerTag(image.Framework, image.Version, image.Tag)
		dockerRegistry := viper.GetString(config.LearnerRegistryKey)
		learnerImage = fmt.Sprintf("%s/%s_gpu_%s:%s", dockerRegistry, image.Framework, image.Version, learnerTag)
	}
	return learnerImage
}

func getLearnerTag(frameworkVersion, version, learnerTagFromRequest string) string {

	learnerTag := viper.GetString(config.LearnerTagKey)
	// Use any tag in the request (ie, specified in the manifest)
	learnerImageTagInManifest := learnerTagFromRequest
	if "" == learnerImageTagInManifest {
		// not in request; try looking up from configmap/learner-config
		imageBuildTag := framework.GetImageBuildTagForFramework(frameworkVersion, version, learnerConfigPath)
		if imageBuildTag != "" {
			return imageBuildTag
		}
	} else {
		learnerTag = learnerImageTagInManifest
	}
	return learnerTag
}
