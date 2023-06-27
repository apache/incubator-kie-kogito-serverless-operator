// Copyright 2023 Red Hat, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kiegroup/kogito-serverless-operator/container-builder/util/log"
)

const (
	envVarPodNamespaceName = "POD_NAMESPACE"
	// ConfigMapName is the default name for the Builder ConfigMap name
	ConfigMapName                       = "sonataflow-operator-builder-config"
	SonataPrefix                        = "sonataflow"
	configKeyDefaultExtension           = "DEFAULT_WORKFLOW_EXTENSION"
	configKeyDefaultBuilderResourceName = "DEFAULT_BUILDER_RESOURCE_NAME"
	ConfigDockerfile                    = "Dockerfile"
)

func GetNamespaceConfigMap(client client.Client, cmName string, namespace string) (*corev1.ConfigMap, error) {

	if len(namespace) == 0 {
		return nil, errors.Errorf("Can't find current context namespace, make sure that %s namespace exists", namespace)
	}

	existingConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
		},
		Data: map[string]string{},
	}

	err := client.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: namespace}, existingConfigMap)
	if err != nil {
		copyOperatorConfigMapIntoNamespaceConfigMap(client, existingConfigMap)
	}
	if len(existingConfigMap.Data) == 2 {
		commonConfig, _ := GetCommonConfigMap(client, "")
		existingConfigMap.Data[configKeyDefaultExtension] = commonConfig.Data[configKeyDefaultExtension]
		existingConfigMap.Data[configKeyDefaultBuilderResourceName] = commonConfig.Data[configKeyDefaultBuilderResourceName]
		err = client.Update(context.TODO(), existingConfigMap)
		if err != nil {
			log.Error(err, "Error adding fields in the configmap")
		}
	}

	err = isValidBuilderCommonConfigMap(existingConfigMap)
	if err != nil {
		log.Error(err, "configmap "+cmName+" is not valid")
		return existingConfigMap, err
	}

	return existingConfigMap, nil
}

func copyOperatorConfigMapIntoNamespaceConfigMap(client client.Client, existingConfigMap *corev1.ConfigMap) {
	commonConfig, _ := GetCommonConfigMap(client, "")
	existingConfigMap.Data[configKeyDefaultExtension] = commonConfig.Data[configKeyDefaultExtension]
	existingConfigMap.Data[configKeyDefaultBuilderResourceName] = commonConfig.Data[configKeyDefaultBuilderResourceName]
	existingConfigMap.Data[resourceDockerfile] = commonConfig.Data[resourceDockerfile]
	client.Create(context.TODO(), existingConfigMap)
}

// GetCommonConfigMap retrieves the config map with the builder common configuration information
func GetCommonConfigMap(client client.Client, fallbackNS string) (*corev1.ConfigMap, error) {
	namespace, found := os.LookupEnv(envVarPodNamespaceName)
	if !found {
		namespace = fallbackNS
	}
	return GetNamespaceConfigMap(client, ConfigMapName, namespace)
}

// isValidBuilderCommonConfigMap  function that will verify that in the builder config maps there are the required keys, and they aren't empty
func isValidBuilderCommonConfigMap(configMap *corev1.ConfigMap) error {

	// Verifying that the key to hold the extension for the workflow is there and not empty
	if len(configMap.Data[configKeyDefaultExtension]) == 0 {
		return fmt.Errorf("unable to find %s key into builder config map", configMap.Data[configKeyDefaultExtension])
	}

	// Verifying that the key to hold the name of the Dockerfile for building the workflow is there and not empty
	if len(configMap.Data[configKeyDefaultBuilderResourceName]) == 0 {
		return fmt.Errorf("unable to find %s key into builder config map", configMap.Data[configKeyDefaultBuilderResourceName])
	}

	// Verifying that the key to hold the content of the Dockerfile for building the workflow is there and not empty
	if len(configMap.Data[configMap.Data[configKeyDefaultBuilderResourceName]]) == 0 {
		return fmt.Errorf("unable to find %s key into builder config map", configMap.Data[configKeyDefaultBuilderResourceName])
	}
	return nil
}
