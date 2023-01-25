/*
 * Copyright 2023 Red Hat, Inc. and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package utils

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/kiegroup/container-builder/util/log"
	"github.com/kiegroup/kogito-serverless-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ExternalResourceType string

const (
	//@TODO ask if we want to provide a fixed types or this are just the defaults and the user can add other types to handle
	CamelRoute   ExternalResourceType = constants.YAML_EXTENSION
	OpenAPI_YAML ExternalResourceType = constants.YAML_EXTENSION
	OpenAPI_JSON ExternalResourceType = constants.JSON_EXTENSION
	AsyncAPI     ExternalResourceType = constants.YAML_EXTENSION
)

type ExternalResource struct {
	ResourceType ExternalResourceType
	ContentText  string
}

func GetKogitoBuilderConfigMap() map[string]string {
	cmData := make(map[string]string)
	cmData[string(CamelRoute)] = ""
	cmData[string(OpenAPI_YAML)] = ""
	cmData[string(OpenAPI_JSON)] = ""
	cmData[string(AsyncAPI)] = ""
	return cmData
}

func InitExternalResourceConfigMap(client client.Client, namespace string, log logr.Logger) (cm corev1.ConfigMap, error error) {
	configMap, err := GetConfigMap(client, namespace)
	if err != nil {
		cmData := GetKogitoBuilderConfigMap()

		_, errx := CreateExternalResourcesConfigMap(client, namespace, cmData, log)
		if errx != nil {
			log.Error(err, "configmap error ")
		} else {
			log.Info(constants.SWF_EXTERNAL_RESOURCES_CM_NAME+" created", "")
		}
		return GetConfigMap(client, namespace)
	} else {
		return configMap, err
	}
}

func GetConfigMap(client client.Client, namespace string) (corev1.ConfigMap, error) {

	existingConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.SWF_EXTERNAL_RESOURCES_CM_NAME,
			Namespace: namespace,
		},
		Data: map[string]string{},
	}

	err := client.Get(context.TODO(), types.NamespacedName{Name: constants.SWF_EXTERNAL_RESOURCES_CM_NAME, Namespace: namespace}, &existingConfigMap)
	if err != nil {
		log.Error(err, "reading configmap")
		return corev1.ConfigMap{}, err
	} else {
		return existingConfigMap, nil
	}
}

func CreateExternalResourcesConfigMap(client client.Client, namespace string, cmData map[string]string, log logr.Logger) (cm corev1.ConfigMap, error error) {
	myDep := &appsv1.Deployment{
		//@TODO
	}
	error = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: constants.SWF_EXTERNAL_RESOURCES_CM_NAME}, myDep)

	cm = corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.SWF_EXTERNAL_RESOURCES_CM_NAME,
			Namespace: namespace,
		},
		Data: cmData,
	}
	cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
	cm.SetOwnerReferences(myDep.GetOwnerReferences()) // check the ownership to permit the edit
	error = client.Create(context.TODO(), &cm)
	if error != nil {
		log.Error(error, "configmap create error")
		return cm, error
	} else {
		return cm, nil
	}
}
