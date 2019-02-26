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
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/AISphere/ffdl-commons/config"
	"github.com/AISphere/ffdl-lcm/service"
	"github.com/spf13/viper"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type dockerConfigEntry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

// GenerateImagePullSecret ... creates secret only for custom images; otherwise returns default secret name
func GenerateImagePullSecret(k8sClient kubernetes.Interface, req *service.JobDeploymentRequest) ([]v1core.LocalObjectReference, error) {

	imagePullSecret := viper.GetString(config.LearnerImagePullSecretKey)

	// if no custom image, then use our default pull secret
	if req.ImageLocation == nil {
		return []v1core.LocalObjectReference{
			v1core.LocalObjectReference{
				Name: imagePullSecret,
			},
		}, nil
	}

	// if no token specified, then use ours
	if req.ImageLocation.AccessToken == "" {
		return []v1core.LocalObjectReference{}, errors.New("Custom image access token is missing")
	}

	// build a custom secret
	imagePullSecretCustom := "customimage-" + req.Name
	trainingID := req.TrainingId
	server := req.ImageLocation.Registry
	token := req.ImageLocation.AccessToken
	email := req.ImageLocation.Email
	// format of the .dockercfg entry
	entry := make(map[string]dockerConfigEntry)
	entry[server] = dockerConfigEntry{
		Username: "token",
		Password: token,
		Email:    email,
		Auth:     base64.StdEncoding.EncodeToString([]byte("token:" + token)),
	}
	dockerCfgContent, _ := json.Marshal(entry)
	// create Secret object
	secret := v1core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imagePullSecretCustom,
			Namespace: config.GetLearnerNamespace(),
			Labels:    map[string]string{"training_id": trainingID}, // this makes sure the secret is deleted with the other learner components
		},
		Type: v1core.SecretTypeDockercfg, // kubernetes.io/dockercfg
		Data: map[string][]byte{},
	}
	// add the dockercfg content (as binary)
	secret.Data[v1core.DockerConfigKey] = dockerCfgContent
	// create the secret
	if _, err := k8sClient.CoreV1().Secrets(secret.Namespace).Create(&secret); err != nil {
		return []v1core.LocalObjectReference{
			v1core.LocalObjectReference{
				Name: imagePullSecret,
			},
			v1core.LocalObjectReference{
				Name: imagePullSecretCustom,
			},
		}, err
	}

	// return its name
	return []v1core.LocalObjectReference{
		v1core.LocalObjectReference{
			Name: imagePullSecret,
		},
		v1core.LocalObjectReference{
			Name: imagePullSecretCustom,
		},
	}, nil
}
