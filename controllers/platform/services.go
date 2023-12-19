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

package platform

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/container-builder/client"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles/common"
	"github.com/apache/incubator-kie-kogito-serverless-operator/log"
	"github.com/apache/incubator-kie-kogito-serverless-operator/utils"
	kubeutil "github.com/apache/incubator-kie-kogito-serverless-operator/utils/kubernetes"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj"
)

// NewServiceAction returns an action that deploys the services.
func NewServiceAction() Action {
	return &serviceAction{}
}

type serviceAction struct {
	baseAction
}

func (action *serviceAction) Name() string {
	return "service"
}

func (action *serviceAction) CanHandle(platform *operatorapi.SonataFlowPlatform) bool {
	return platform.Status.IsReady()
}

func (action *serviceAction) Handle(ctx context.Context, platform *operatorapi.SonataFlowPlatform) (*operatorapi.SonataFlowPlatform, error) {
	// Refresh applied configuration
	if err := ConfigureDefaults(ctx, action.client, platform, false); err != nil {
		return nil, err
	}

	if platform.Spec.Services.DataIndex != nil {
		if err := createServiceComponents(ctx, action.client, platform, common.NewDataIndexService(platform)); err != nil {
			return nil, err
		}
	}

	if platform.Spec.Services.JobService != nil {
		if err := createServiceComponents(ctx, action.client, platform, common.NewJobService(platform)); err != nil {
			return nil, err
		}
	}

	return platform, nil
}

func createServiceComponents(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, serviceType common.PlatformService) error {
	if err := createConfigMap(ctx, client, platform, serviceType); err != nil {
		return err
	}
	if err := createDeployment(ctx, client, platform, serviceType); err != nil {
		return err
	}
	return createService(ctx, client, platform, serviceType)
}

func createDeployment(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, svc common.PlatformService) error {
	readyProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   common.QuarkusHealthPathReady,
				Port:   common.DefaultHTTPWorkflowPortIntStr,
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: int32(45),
		TimeoutSeconds:      int32(10),
		PeriodSeconds:       int32(30),
		SuccessThreshold:    int32(1),
		FailureThreshold:    int32(4),
	}
	liveProbe := readyProbe.DeepCopy()
	liveProbe.ProbeHandler.HTTPGet.Path = common.QuarkusHealthPathLive
	dataDeployContainer := &corev1.Container{
		Image:          svc.GetServiceImageName(common.PersistenceTypeEphemeral),
		Env:            svc.GetEnvironmentVariables(),
		Resources:      svc.GetPodResourceRequirements(),
		ReadinessProbe: readyProbe,
		LivenessProbe:  liveProbe,
		Ports: []corev1.ContainerPort{
			{
				Name:          utils.HttpScheme,
				ContainerPort: int32(common.DefaultHTTPWorkflowPortInt),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "application-config",
				MountPath: "/home/kogito/config",
			},
		},
	}
	dataDeployContainer = svc.ConfigurePersistence(dataDeployContainer)
	dataDeployContainer, err := svc.MergeContainerSpec(dataDeployContainer)
	if err != nil {
		return err
	}

	// immutable
	dataDeployContainer.Name = svc.GetContainerName()

	replicas := svc.GetReplicaCount()
	lbl := map[string]string{
		workflowproj.LabelApp: platform.Name,
	}
	dataDeploySpec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: lbl,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: lbl,
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "application-config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: svc.GetServiceCmName(),
								},
							},
						},
					},
				},
			},
		},
	}

	dataDeploySpec.Template.Spec, err = svc.MergePodSpec(dataDeploySpec.Template.Spec)
	if err != nil {
		return err
	}
	kubeutil.AddOrReplaceContainer(dataDeployContainer.Name, *dataDeployContainer, &dataDeploySpec.Template.Spec)

	dataDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      svc.GetServiceName(),
			Labels:    lbl,
		}}
	if err := controllerutil.SetControllerReference(platform, dataDeploy, client.Scheme()); err != nil {
		return err
	}

	// Create or Update the deployment
	if op, err := controllerutil.CreateOrUpdate(ctx, client, dataDeploy, func() error {
		dataDeploy.Spec = dataDeploySpec

		return nil
	}); err != nil {
		return err
	} else {
		klog.V(log.I).InfoS("Deployment successfully reconciled", "operation", op)
	}

	return nil
}

func createService(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, svc common.PlatformService) error {
	lbl := map[string]string{
		workflowproj.LabelApp: platform.Name,
	}
	dataSvcSpec := corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:       utils.HttpScheme,
				Protocol:   corev1.ProtocolTCP,
				Port:       80,
				TargetPort: common.DefaultHTTPWorkflowPortIntStr,
			},
		},
		Selector: lbl,
	}
	dataSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      svc.GetServiceName(),
			Labels:    lbl,
		}}
	if err := controllerutil.SetControllerReference(platform, dataSvc, client.Scheme()); err != nil {
		return err
	}

	// Create or Update the service
	if op, err := controllerutil.CreateOrUpdate(ctx, client, dataSvc, func() error {
		dataSvc.Spec = dataSvcSpec

		return nil
	}); err != nil {
		return err
	} else {
		klog.V(log.I).InfoS("Service successfully reconciled", "operation", op)
	}

	return nil
}

func createConfigMap(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, svc common.PlatformService) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.GetServiceCmName(),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				workflowproj.LabelApp: platform.Name,
			},
		},
		Data: map[string]string{
			workflowproj.ApplicationPropertiesFileName: common.NewServiceAppPropertyHandler(platform).Build(),
		},
	}
	if err := controllerutil.SetControllerReference(platform, configMap, client.Scheme()); err != nil {
		return err
	}

	// Create or Update the service
	if op, err := controllerutil.CreateOrUpdate(ctx, client, configMap, func() error {
		configMap.Data[workflowproj.ApplicationPropertiesFileName] =
			common.NewServiceAppPropertyHandler(platform).
				WithUserProperties(configMap.Data[workflowproj.ApplicationPropertiesFileName]).
				Build()

		return nil
	}); err != nil {
		return err
	} else {
		klog.V(log.I).InfoS("ConfigMap successfully reconciled", "operation", op)
	}

	return nil
}
