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
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/container-builder/client"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles/common"
	"github.com/apache/incubator-kie-kogito-serverless-operator/log"
	"github.com/apache/incubator-kie-kogito-serverless-operator/utils"
	kubeutil "github.com/apache/incubator-kie-kogito-serverless-operator/utils/kubernetes"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj"
	"github.com/imdario/mergo"
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
		if err := createServiceComponents(ctx, action.client, platform, common.DataIndexService); err != nil {
			return nil, err
		}
	}

	if platform.Spec.Services.JobService != nil {
		if err := createServiceComponents(ctx, action.client, platform, common.JobService); err != nil {
			return nil, err
		}
	}

	return platform, nil
}

// Values for job service taken from
// https://github.com/parodos-dev/orchestrator-helm-chart/blob/52d09eda56fdbed3060782df29847c97f172600f/charts/orchestrator/values.yaml#L68-L72
func getResourceLimits(stype common.ServiceType) corev1.ResourceRequirements {
	switch stype {
	case common.DataIndexService:
		return corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		}
	case common.JobService:
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("250m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}
	}
	return corev1.ResourceRequirements{}
}

func createServiceComponents(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, serviceType common.ServiceType) error {
	if err := createConfigMap(ctx, client, platform, serviceType); err != nil {
		return err
	}
	if err := createDeployment(ctx, client, platform, serviceType); err != nil {
		return err
	}
	return createService(ctx, client, platform, serviceType)
}

func getReplicaCountForService(serviceSpec operatorapi.ServicesPlatformSpec, serviceType common.ServiceType) int32 {
	var spec *operatorapi.ServiceSpec
	switch serviceType {
	case common.DataIndexService:
		spec = serviceSpec.DataIndex
	case common.JobService:
		spec = serviceSpec.JobService
	}
	if spec.PodTemplate.Replicas != nil {
		return *spec.PodTemplate.Replicas
	}
	return 1

}
func createDeployment(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, serviceType common.ServiceType) error {
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
		Image: common.GetServiceImageName(common.PersistenceTypeEphemeral, serviceType),
		Env: []corev1.EnvVar{
			{
				Name:  "KOGITO_DATA_INDEX_QUARKUS_PROFILE",
				Value: "http-events-support",
			},
			{
				Name:  "QUARKUS_HTTP_CORS",
				Value: "true",
			},
			{
				Name:  "QUARKUS_HTTP_CORS_ORIGINS",
				Value: "/.*/",
			},
		},
		Resources:      getResourceLimits(serviceType),
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
	configurePersistence(dataDeployContainer, types.NamespacedName{Namespace: platform.Namespace, Name: platform.Name}, platform.Spec.Services, serviceType)
	err := mergeContainerSpec(dataDeployContainer, platform.Spec.Services, serviceType)
	if err != nil {
		return err
	}

	// immutable
	dataDeployContainer.Name = common.GetContainerName(serviceType)

	replicas := getReplicaCountForService(platform.Spec.Services, serviceType)
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
									Name: common.GetServiceCmName(platform, serviceType),
								},
							},
						},
					},
				},
			},
		},
	}

	err = mergePodSpec(&dataDeploySpec.Template.Spec, platform.Spec.Services, serviceType)
	if err != nil {
		return err
	}
	kubeutil.AddOrReplaceContainer(dataDeployContainer.Name, *dataDeployContainer, &dataDeploySpec.Template.Spec)

	dataDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      common.GetServiceName(platform.Name, serviceType),
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

func mergeContainerSpec(containerSpec *corev1.Container, servicePlatformSpec operatorapi.ServicesPlatformSpec, serviceType common.ServiceType) error {
	switch serviceType {
	case common.DataIndexService:
		return mergo.Merge(containerSpec, servicePlatformSpec.DataIndex.PodTemplate.Container.ToContainer(), mergo.WithOverride)
	case common.JobService:
		return mergo.Merge(containerSpec, servicePlatformSpec.JobService.PodTemplate.Container.ToContainer(), mergo.WithOverride)
	}
	return fmt.Errorf("unknown service type %d", serviceType)
}

func mergePodSpec(podSpec *corev1.PodSpec, servicePlatformSpec operatorapi.ServicesPlatformSpec, serviceType common.ServiceType) error {
	switch serviceType {
	case common.DataIndexService:
		return mergo.Merge(podSpec, servicePlatformSpec.DataIndex.PodTemplate.PodSpec.ToPodSpec(), mergo.WithOverride)
	case common.JobService:
		return mergo.Merge(podSpec, servicePlatformSpec.JobService.PodTemplate.PodSpec.ToPodSpec(), mergo.WithOverride)
	}
	return fmt.Errorf("unknown service type %d", serviceType)
}

func configurePersistence(serviceContainer *corev1.Container, platformNamespacedName types.NamespacedName, serviceSpec operatorapi.ServicesPlatformSpec, serviceType common.ServiceType) {
	switch serviceType {
	case common.DataIndexService:
		if serviceSpec.DataIndex.Persistence != nil && serviceSpec.DataIndex.Persistence.PostgreSql != nil {
			serviceContainer.Image = common.GetServiceImageName(common.PersistenceTypePostgressql, serviceType)
			serviceContainer.Env = append(
				serviceContainer.Env,
				configurePostgreSqlEnv(serviceSpec.DataIndex.Persistence.PostgreSql, common.GetServiceName(platformNamespacedName.Name, serviceType), platformNamespacedName.Namespace)...)
		}
	case common.JobService:
		if serviceSpec.JobService.Persistence != nil && serviceSpec.JobService.Persistence.PostgreSql != nil {
			serviceContainer.Image = common.GetServiceImageName(common.PersistenceTypePostgressql, serviceType)
			serviceContainer.Env = append(
				serviceContainer.Env,
				configurePostgreSqlEnv(serviceSpec.JobService.Persistence.PostgreSql, common.GetServiceName(platformNamespacedName.Name, serviceType), platformNamespacedName.Namespace)...)
		}
	}
}

func configurePostgreSqlEnv(postgresql *operatorapi.PersistencePostgreSql, databaseSchema, databaseNamespace string) []corev1.EnvVar {
	if len(postgresql.ServiceRef.DatabaseSchema) > 0 {
		databaseSchema = postgresql.ServiceRef.DatabaseSchema
	}
	if len(postgresql.ServiceRef.Namespace) > 0 {
		databaseNamespace = postgresql.ServiceRef.Namespace
	}
	dataSourcePort := 5432
	if postgresql.ServiceRef.Port != nil {
		dataSourcePort = *postgresql.ServiceRef.Port
	}
	databaseName := "sonataflow"
	if len(postgresql.ServiceRef.DatabaseName) > 0 {
		databaseName = postgresql.ServiceRef.DatabaseName
	}
	dataSourceUrl := "jdbc:" + common.PersistenceTypePostgressql + "://" + postgresql.ServiceRef.Name + "." + databaseNamespace + ":" + strconv.Itoa(dataSourcePort) + "/" + databaseName + "?currentSchema=" + databaseSchema
	if len(postgresql.JdbcUrl) > 0 {
		dataSourceUrl = postgresql.JdbcUrl
	}
	secretRef := corev1.LocalObjectReference{
		Name: postgresql.SecretRef.Name,
	}
	quarkusDatasourceUsername := "POSTGRESQL_USER"
	if len(postgresql.SecretRef.UserKey) > 0 {
		quarkusDatasourceUsername = postgresql.SecretRef.UserKey
	}
	quarkusDatasourcePassword := "POSTGRESQL_PASSWORD"
	if len(postgresql.SecretRef.PasswordKey) > 0 {
		quarkusDatasourcePassword = postgresql.SecretRef.PasswordKey
	}
	return []corev1.EnvVar{
		{
			Name: "QUARKUS_DATASOURCE_USERNAME",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  quarkusDatasourceUsername,
					LocalObjectReference: secretRef,
				},
			},
		},
		{
			Name: "QUARKUS_DATASOURCE_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  quarkusDatasourcePassword,
					LocalObjectReference: secretRef,
				},
			},
		},
		{
			Name:  "QUARKUS_DATASOURCE_DB_KIND",
			Value: common.PersistenceTypePostgressql,
		},
		{
			Name:  "QUARKUS_HIBERNATE_ORM_DATABASE_GENERATION",
			Value: "update",
		},
		{
			Name:  "QUARKUS_FLYWAY_MIGRATE_AT_START",
			Value: "true",
		},
		{
			Name:  "QUARKUS_DATASOURCE_JDBC_URL",
			Value: dataSourceUrl,
		},
	}
}

func createService(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, serviceType common.ServiceType) error {
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
			Name:      common.GetServiceName(platform.Name, serviceType),
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

func createConfigMap(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, serviceType common.ServiceType) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.GetServiceCmName(platform, serviceType),
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
