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
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	if err := createDataIndexComponents(ctx, action.client, platform); err != nil {
		return nil, err
	}

	return platform, nil
}

func createDataIndexComponents(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform) error {
	if platform.Spec.Services.DataIndex != nil {
		if err := createDataIndexConfigMap(ctx, client, platform); err != nil {
			return err
		}
		if err := createDataIndexDeployment(ctx, client, platform); err != nil {
			return err
		}
		if err := createDataIndexService(ctx, client, platform); err != nil {
			return err
		}
	}

	return nil
}

func createDataIndexDeployment(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform) error {
	readyProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   common.QuarkusHealthPathReady,
				Port:   common.DefaultHTTPWorkflowPortIntStr,
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: int32(15),
		TimeoutSeconds:      int32(10),
		PeriodSeconds:       int32(30),
		SuccessThreshold:    int32(1),
		FailureThreshold:    int32(3),
	}
	liveProbe := readyProbe.DeepCopy()
	liveProbe.ProbeHandler.HTTPGet.Path = common.QuarkusHealthPathLive
	dataDeployContainer := &corev1.Container{
		Image: common.DataIndexImageBase + common.PersistenceTypeEphemeral,
		Env: []corev1.EnvVar{
			{
				Name:  "KOGITO_DATA_INDEX_QUARKUS_PROFILE",
				Value: "http-events-support",
			},
			{
				Name:  "QUARKUS_HTTP_PORT",
				Value: "8080",
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
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
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
	configurePersistence(ctx, client, dataDeployContainer, platform)
	if err := mergo.Merge(dataDeployContainer, platform.Spec.Services.DataIndex.Container.ToContainer(), mergo.WithOverride); err != nil {
		return err
	}
	// immutable
	dataDeployContainer.Name = common.DataIndexName

	var replicas int32 = 1
	if platform.Spec.Services.DataIndex.Replicas != nil {
		replicas = *platform.Spec.Services.DataIndex.Replicas
	}
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
									Name: getDataIndexName(platform),
								},
							},
						},
					},
				},
			},
		},
	}
	if err := mergo.Merge(&dataDeploySpec.Template.Spec, platform.Spec.Services.DataIndex.PodSpec.ToPodSpec(), mergo.WithOverride); err != nil {
		return err
	}
	kubeutil.AddOrReplaceContainer(common.DataIndexName, *dataDeployContainer, &dataDeploySpec.Template.Spec)

	dataDeploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      getDataIndexName(platform),
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

func configurePersistence(ctx context.Context, client client.Client, dataDeployContainer *corev1.Container, platform *operatorapi.SonataFlowPlatform) {
	if platform.Spec.Services.Persistence != nil {
		if platform.Spec.Services.Persistence.PostgreSql != nil {
			configurePostgreSql(ctx, client, dataDeployContainer, platform)
		}
	}
}

func configurePostgreSql(ctx context.Context, client client.Client, dataDeployContainer *corev1.Container, platform *operatorapi.SonataFlowPlatform) {
	persistenceType := common.PersistenceTypePostgressql
	dataDeployContainer.Image = common.DataIndexImageBase + persistenceType

	databaseNamespace := platform.Namespace
	if len(platform.Spec.Services.Persistence.PostgreSql.ServiceRef.Namespace) > 0 {
		databaseNamespace = platform.Spec.Services.Persistence.PostgreSql.ServiceRef.Namespace
	}
	dataSourcePort := 5432
	if platform.Spec.Services.Persistence.PostgreSql.ServiceRef.Port != nil {
		dataSourcePort = *platform.Spec.Services.Persistence.PostgreSql.ServiceRef.Port
	}
	databaseName := "sonataflow"
	if len(platform.Spec.Services.Persistence.PostgreSql.ServiceRef.DatabaseName) > 0 {
		databaseName = platform.Spec.Services.Persistence.PostgreSql.ServiceRef.DatabaseName
	}
	databaseSchema := "data-index-service"
	if len(platform.Spec.Services.Persistence.PostgreSql.ServiceRef.DatabaseSchema) > 0 {
		databaseSchema = platform.Spec.Services.Persistence.PostgreSql.ServiceRef.DatabaseSchema
	}
	dataSourceUrl := "jdbc:" + persistenceType + "://" + platform.Spec.Services.Persistence.PostgreSql.ServiceRef.Name + "." + databaseNamespace + ":" + strconv.Itoa(dataSourcePort) + "/" + databaseName + "?currentSchema=" + databaseSchema
	if len(platform.Spec.Services.Persistence.PostgreSql.JdbcUrl) > 0 {
		dataSourceUrl = platform.Spec.Services.Persistence.PostgreSql.JdbcUrl
	}

	secretRef := corev1.LocalObjectReference{
		Name: platform.Spec.Services.Persistence.PostgreSql.SecretRef.Name,
	}
	quarkusDatasourceUsername := "POSTGRESQL_USER"
	if len(platform.Spec.Services.Persistence.PostgreSql.SecretRef.UserKey) > 0 {
		quarkusDatasourceUsername = platform.Spec.Services.Persistence.PostgreSql.SecretRef.UserKey
	}
	quarkusDatasourcePassword := "POSTGRESQL_PASSWORD"
	if len(platform.Spec.Services.Persistence.PostgreSql.SecretRef.PasswordKey) > 0 {
		quarkusDatasourcePassword = platform.Spec.Services.Persistence.PostgreSql.SecretRef.PasswordKey
	}
	persistEnvVars := []corev1.EnvVar{
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
			Value: persistenceType,
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
	dataDeployContainer.Env = append(dataDeployContainer.Env, persistEnvVars...)
}

func createDataIndexService(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform) error {
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
			Name:      getDataIndexName(platform),
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

// createDataIndexConfigMap ...
func createDataIndexConfigMap(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getDataIndexName(platform),
			Namespace: platform.Namespace,
			Labels: map[string]string{
				workflowproj.LabelApp: platform.Name,
			},
		},
	}
	if err := controllerutil.SetControllerReference(platform, configMap, client.Scheme()); err != nil {
		return err
	}

	// Create or Update the service
	if op, err := controllerutil.CreateOrUpdate(ctx, client, configMap, func() error {
		configMap.Data = map[string]string{
			"application.properties": "quarkus.smallrye-health.check.\"io.quarkus.kafka.client.health.KafkaHealthCheck\".enabled=false\n",
		}

		return nil
	}); err != nil {
		return err
	} else {
		klog.V(log.I).InfoS("ConfigMap successfully reconciled", "operation", op)
	}

	return nil
}

func getDataIndexName(platform *operatorapi.SonataFlowPlatform) string {
	return platform.Name + "-" + common.DataIndexName
}
