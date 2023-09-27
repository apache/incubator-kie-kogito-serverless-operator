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

package dev

import (
	"path"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/controllers/profiles"
	"github.com/kiegroup/kogito-serverless-operator/controllers/profiles/common"
	"github.com/kiegroup/kogito-serverless-operator/controllers/workflowdef"
	"github.com/kiegroup/kogito-serverless-operator/utils"
	kubeutil "github.com/kiegroup/kogito-serverless-operator/utils/kubernetes"
	"github.com/kiegroup/kogito-serverless-operator/workflowproj"
)

const (
	// healthFailureThresholdDevMode exclusive threshold for devmode given that it might take longer than the immutable image to start/live/respond.
	healthFailureThresholdDevMode = 50
)

// serviceCreator is an objectCreator for a basic Service for a workflow using dev profile
// aiming a vanilla Kubernetes Deployment.
// It maps the default HTTP port (80) to the target Java application webserver on port 8080.
// It configures the Service as a NodePort type service, in this way it will be easier for a developer access the service
func serviceCreator(workflow *operatorapi.SonataFlow) (client.Object, error) {
	object, _ := common.ServiceCreator(workflow)
	service := object.(*corev1.Service)
	// Let's double-check that the workflow is using the Dev Profile we would like to expose it via NodePort
	if profiles.IsDevProfile(workflow) {
		service.Spec.Type = corev1.ServiceTypeNodePort
	}
	return service, nil
}

func deploymentCreator(workflow *operatorapi.SonataFlow) (client.Object, error) {
	obj, err := common.DeploymentCreator(workflow)
	if err != nil {
		return nil, err
	}
	deployment := obj.(*appsv1.Deployment)
	_, idx := kubeutil.GetContainerByName(operatorapi.DefaultContainerName, &deployment.Spec.Template.Spec)
	if workflow.Spec.PodTemplate.Container.StartupProbe == nil {
		deployment.Spec.Template.Spec.Containers[idx].StartupProbe.FailureThreshold = healthFailureThresholdDevMode
	}
	if workflow.Spec.PodTemplate.Container.LivenessProbe == nil {
		deployment.Spec.Template.Spec.Containers[idx].LivenessProbe.FailureThreshold = healthFailureThresholdDevMode
	}
	if workflow.Spec.PodTemplate.Container.ReadinessProbe == nil {
		deployment.Spec.Template.Spec.Containers[idx].ReadinessProbe.FailureThreshold = healthFailureThresholdDevMode
	}
	return deployment, nil
}

// workflowDefConfigMapCreator creates a new ConfigMap that holds the definition of a workflow specification.
func workflowDefConfigMapCreator(workflow *operatorapi.SonataFlow) (client.Object, error) {
	configMap, err := workflowdef.CreateNewConfigMap(workflow)
	if err != nil {
		return nil, err
	}
	workflowproj.SetDefaultLabels(workflow, configMap)
	return configMap, nil
}

// deploymentMutateVisitor guarantees the state of the default Deployment object
func deploymentMutateVisitor(workflow *operatorapi.SonataFlow) common.MutateVisitor {
	return func(object client.Object) controllerutil.MutateFn {
		return func() error {
			if kubeutil.IsObjectNew(object) {
				return nil
			}
			original, err := deploymentCreator(workflow)
			if err != nil {
				return err
			}
			common.EnsureDeployment(original.(*appsv1.Deployment), object.(*appsv1.Deployment))
			return nil
		}
	}
}

func ensureWorkflowDefConfigMapMutator(workflow *operatorapi.SonataFlow) common.MutateVisitor {
	return func(object client.Object) controllerutil.MutateFn {
		return func() error {
			if kubeutil.IsObjectNew(object) {
				return nil
			}
			original, err := workflowDefConfigMapCreator(workflow)
			if err != nil {
				return err
			}
			object.(*corev1.ConfigMap).Data = original.(*corev1.ConfigMap).Data
			object.(*corev1.ConfigMap).Labels = original.GetLabels()
			return nil
		}
	}
}

// mountDevConfigMapsMutateVisitor mounts the required configMaps in the Workflow Dev Deployment
func mountDevConfigMapsMutateVisitor(flowDefCM, propsCM *corev1.ConfigMap, workflowResCMs []operatorapi.ConfigMapWorkflowResource) common.MutateVisitor {
	return func(object client.Object) controllerutil.MutateFn {
		return func() error {
			deployment := object.(*appsv1.Deployment)

			volumeMounts := []corev1.VolumeMount{
				kubeutil.VolumeMount(configMapResourcesVolumeName, true, quarkusDevConfigMountPath),
			}

			// defaultResourcesVolume holds every ConfigMap mount required on src/main/resources
			defaultResourcesVolume := corev1.Volume{Name: configMapResourcesVolumeName, VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{}}}
			kubeutil.VolumeProjectionAddConfigMap(defaultResourcesVolume.Projected, propsCM.Name, corev1.KeyToPath{Key: workflowproj.ApplicationPropertiesFileName, Path: workflowproj.ApplicationPropertiesFileName})
			kubeutil.VolumeProjectionAddConfigMap(defaultResourcesVolume.Projected, flowDefCM.Name)

			// resourceVolumes holds every resource that needs to be mounted on src/main/resources/<specific_dir>
			resourceVolumes := make([]corev1.Volume, 0)

			for _, workflowResCM := range workflowResCMs {
				// if we need to mount at the root dir, we use the defaultResourcesVolume
				if len(workflowResCM.WorkflowPath) == 0 {
					kubeutil.VolumeProjectionAddConfigMap(defaultResourcesVolume.Projected, workflowResCM.ConfigMap.Name)
					continue
				}
				// the resource configMap needs a specific dir, inside the src/main/resources
				// to avoid clashing with other configMaps trying to mount on the same dir, we create one projected per path
				volumeMountName := configMapExternalResourcesVolumeNamePrefix + utils.PathToString(workflowResCM.WorkflowPath)
				volumeMounts = kubeutil.VolumeMountAdd(volumeMounts, volumeMountName, path.Join(quarkusDevConfigMountPath, workflowResCM.WorkflowPath))
				resourceVolumes = kubeutil.VolumeAddVolumeProjectionConfigMap(resourceVolumes, workflowResCM.ConfigMap.Name, volumeMountName)
			}

			if len(deployment.Spec.Template.Spec.Volumes) == 0 {
				deployment.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0, len(resourceVolumes)+2)
			}
			kubeutil.AddOrReplaceVolume(&deployment.Spec.Template.Spec, defaultResourcesVolume)
			kubeutil.AddOrReplaceVolume(&deployment.Spec.Template.Spec, resourceVolumes...)

			_, flowContainerIdx := kubeutil.GetContainerByName(operatorapi.DefaultContainerName, &deployment.Spec.Template.Spec)
			if len(deployment.Spec.Template.Spec.Containers[flowContainerIdx].VolumeMounts) == 0 {
				deployment.Spec.Template.Spec.Containers[flowContainerIdx].VolumeMounts = make([]corev1.VolumeMount, 0, len(volumeMounts))
			}
			kubeutil.AddOrReplaceVolumeMount(flowContainerIdx, &deployment.Spec.Template.Spec, volumeMounts...)

			return nil
		}
	}
}
