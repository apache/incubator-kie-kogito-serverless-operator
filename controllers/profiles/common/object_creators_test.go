/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package common

import (
	"context"
	"testing"

	"github.com/magiconair/properties"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/apache/incubator-kie-kogito-serverless-operator/utils"
	kubeutil "github.com/apache/incubator-kie-kogito-serverless-operator/utils/kubernetes"

	"github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/test"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj"
)

func Test_ensureWorkflowPropertiesConfigMapMutator(t *testing.T) {
	workflow := test.GetBaseSonataFlowWithDevProfile(t.Name())
	platform := test.GetBasePlatform()
	// can't be new
	managedProps, _ := ManagedPropsConfigMapCreator(workflow, platform)
	managedProps.SetUID("1")
	managedProps.SetResourceVersion("1")
	managedPropsCM := managedProps.(*corev1.ConfigMap)

	userProps, _ := UserPropsConfigMapCreator(workflow, platform)
	userPropsCM := userProps.(*corev1.ConfigMap)
	visitor := UserPropertiesMutateVisitor(context.TODO(), nil, workflow, nil, userPropsCM)
	mutateFn := visitor(managedProps)

	assert.NoError(t, mutateFn())
	assert.Empty(t, managedPropsCM.Data[workflowproj.ApplicationPropertiesFileName])
	assert.NotEmpty(t, managedPropsCM.Data[workflowproj.GetManagedPropertiesFileName(workflow)])

	props := properties.MustLoadString(managedPropsCM.Data[workflowproj.GetManagedPropertiesFileName(workflow)])
	assert.Equal(t, "8080", props.GetString("quarkus.http.port", ""))

	// we change the properties to something different, we add ours and change the default
	userPropsCM.Data[workflowproj.ApplicationPropertiesFileName] = "quarkus.http.port=9090\nmy.new.prop=1"
	visitor(managedPropsCM)
	assert.NoError(t, mutateFn())

	// we should preserve the default, and still got ours
	props = properties.MustLoadString(managedPropsCM.Data[workflowproj.GetManagedPropertiesFileName(workflow)])
	assert.Equal(t, "8080", props.GetString("quarkus.http.port", ""))
	assert.Equal(t, "0.0.0.0", props.GetString("quarkus.http.host", ""))
	assert.Equal(t, "1", props.GetString("my.new.prop", ""))
}

func Test_ensureWorkflowPropertiesConfigMapMutator_DollarReplacement(t *testing.T) {
	workflow := test.GetBaseSonataFlowWithDevProfile(t.Name())
	platform := test.GetBasePlatform()
	managedProps, _ := ManagedPropsConfigMapCreator(workflow, platform)
	managedProps.SetName(workflow.Name)
	managedProps.SetNamespace(workflow.Namespace)
	managedProps.SetUID("0000-0001-0002-0003")
	managedPropsCM := managedProps.(*corev1.ConfigMap)

	userProps, _ := UserPropsConfigMapCreator(workflow, platform)
	userPropsCM := userProps.(*corev1.ConfigMap)
	userPropsCM.Data[workflowproj.ApplicationPropertiesFileName] = "mp.messaging.outgoing.kogito_outgoing_stream.url=${kubernetes:services.v1/event-listener}"

	mutateVisitorFn := UserPropertiesMutateVisitor(context.TODO(), nil, workflow, nil, userPropsCM)

	err := mutateVisitorFn(managedPropsCM)()
	assert.NoError(t, err)
	assert.Contains(t, managedPropsCM.Data[workflowproj.GetManagedPropertiesFileName(workflow)], "${kubernetes:services.v1/event-listener}")
}

func TestMergePodSpec(t *testing.T) {
	workflow := test.GetBaseSonataFlow(t.Name())
	platform := test.GetBasePlatform()
	workflow.Spec.PodTemplate = v1alpha08.PodTemplateSpec{
		Container: v1alpha08.ContainerSpec{
			// this one we can override
			Image: "quay.io/example/my-workflow:1.0.0",
			Ports: []corev1.ContainerPort{
				// let's override a immutable attribute
				{Name: utils.HttpScheme, ContainerPort: 9090},
			},
			Env: []corev1.EnvVar{
				// We should be able to override this too
				{Name: "ENV1", Value: "VALUE_CUSTOM"},
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "myvolume", ReadOnly: true, MountPath: "/tmp/any/path"},
			},
		},
		PodSpec: v1alpha08.PodSpec{
			ServiceAccountName: "superuser",
			Containers: []corev1.Container{
				{
					Name: "sidecar",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "myvolume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "customproperties"},
						},
					},
				},
			},
		},
	}

	object, err := DeploymentCreator(workflow, platform)
	assert.NoError(t, err)

	deployment := object.(*appsv1.Deployment)

	assert.Len(t, deployment.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, "superuser", deployment.Spec.Template.Spec.ServiceAccountName)
	assert.Len(t, deployment.Spec.Template.Spec.Volumes, 1)
	flowContainer, _ := kubeutil.GetContainerByName(v1alpha08.DefaultContainerName, &deployment.Spec.Template.Spec)
	assert.Equal(t, "quay.io/example/my-workflow:1.0.0", flowContainer.Image)
	assert.Equal(t, int32(8080), flowContainer.Ports[0].ContainerPort)
	assert.Equal(t, "VALUE_CUSTOM", flowContainer.Env[0].Value)
	assert.Len(t, flowContainer.VolumeMounts, 1)
}

func TestMergePodSpec_OverrideContainers(t *testing.T) {
	workflow := test.GetBaseSonataFlow(t.Name())
	platform := test.GetBasePlatform()
	workflow.Spec.PodTemplate = v1alpha08.PodTemplateSpec{
		PodSpec: v1alpha08.PodSpec{
			// Try to override the workflow container via the podspec
			Containers: []corev1.Container{
				{
					Name:  v1alpha08.DefaultContainerName,
					Image: "quay.io/example/my-workflow:1.0.0",
					Ports: []corev1.ContainerPort{
						{Name: utils.HttpScheme, ContainerPort: 9090},
					},
					Env: []corev1.EnvVar{
						{Name: "ENV1", Value: "VALUE_CUSTOM"},
					},
				},
			},
		},
	}

	object, err := DeploymentCreator(workflow, platform)
	assert.NoError(t, err)

	deployment := object.(*appsv1.Deployment)

	assert.Len(t, deployment.Spec.Template.Spec.Containers, 1)
	flowContainer, _ := kubeutil.GetContainerByName(v1alpha08.DefaultContainerName, &deployment.Spec.Template.Spec)
	assert.NotEqual(t, "quay.io/example/my-workflow:1.0.0", flowContainer.Image)
	assert.Equal(t, int32(8080), flowContainer.Ports[0].ContainerPort)
	assert.Empty(t, flowContainer.Env)
}
