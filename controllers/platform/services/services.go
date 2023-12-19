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

package services

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"

	"github.com/apache/incubator-kie-kogito-serverless-operator/version"
	"github.com/imdario/mergo"
)

const (
	DataIndexServiceName = "data-index-service"
	JobServiceName       = "jobs-service"
	imageNamePrefix      = "quay.io/kiegroup/kogito"
	DataIndexName        = "data-index"

	persistenceTypePostgreSQL = "postgresql"

	DataIndexServiceUrlProperty = "mp.messaging.outgoing.kogito-processinstances-events.url"

	DataIndexServiceUrlProtocol             = "http"
	jobServiceURLProperty                   = "mp.messaging.outgoing.kogito-job-service-job-request-events.url"
	jobServiceKafkaSinkInjectionHealthCheck = `quarkus.smallrye-health.check."org.kie.kogito.jobs.service.messaging.http.health.knative.KSinkInjectionHealthCheck".enabled`
	jobServiceStatusChangeEventsProperty    = "kogito.jobs-service.http.job-status-change-events"
	jobServiceStatusChangeEventsURL         = "mp.messaging.outgoing.kogito-job-service-job-status-events-http.url"
	jobServiceURLProtocol                   = "http"
	jobServiceDataSourceReactiveURLProperty = "quarkus.datasource.reactive.url"

	defaultDatabaseName   string = "sonataflow"
	defaultPostgreSQLPort int    = 5432
)

type PlatformService interface {
	// GetContainerName returns the name of the service's container in the deployment.
	GetContainerName() string
	// GetServiceImageName returns the image name of the service's container. It takes in the service and persistence types and returns a string
	// that contains the FQDN of the image, including the tag.
	GetServiceImageName(persistenceName string) string
	// GetServiceName returns the name of the kubernetes service prefixed with the platform name
	GetServiceName() string
	// GetServiceCmName returns the name of the configmap associated to the service
	GetServiceCmName() string
	// GetEnvironmentVariables returns the env variables to be injected to the service container
	GetEnvironmentVariables() []corev1.EnvVar
	// GetResourceLimits returns the pod's memory and CPU resource requirements
	// Values for job service taken from
	// https://github.com/parodos-dev/orchestrator-helm-chart/blob/52d09eda56fdbed3060782df29847c97f172600f/charts/orchestrator/values.yaml#L68-L72
	GetPodResourceRequirements() corev1.ResourceRequirements
	// GetReplicaCountForService Returns the default pod replica count for the given service
	GetReplicaCount() int32

	// MergeContainerSpec performs a merge with override using the containerSpec argument and the expected values based on the service's pod template specifications. The returning
	// object is the merged result
	MergeContainerSpec(containerSpec *corev1.Container) (*corev1.Container, error)

	//ConfigurePersistence sets the persistence's image and environment values when it is defined in the Persistence field of the service, overriding any existing value.
	ConfigurePersistence(containerSpec *corev1.Container) *corev1.Container

	//MergePodSpec performs a merge with override between the podSpec argument and the expected values based on the service's pod template specification. The returning
	// object is the result of the merge
	MergePodSpec(podSpec corev1.PodSpec) (corev1.PodSpec, error)
}

type DataIndex struct {
	platform *operatorapi.SonataFlowPlatform
}

func NewDataIndexService(platform *operatorapi.SonataFlowPlatform) DataIndex {
	return DataIndex{platform: platform}
}

func (d DataIndex) GetContainerName() string {
	return DataIndexServiceName
}

func (d DataIndex) GetServiceImageName(persistenceName string) string {
	var tag = version.GetMajorMinor()
	if version.IsSnapshot() {
		tag = "latest"
	}
	// returns "quay.io/kiegroup/kogito-data-index-<persistence_layer>:<tag>"
	return fmt.Sprintf("%s-%s-%s:%s", imageNamePrefix, DataIndexName, persistenceName, tag)
}

func (d DataIndex) GetServiceName() string {
	return fmt.Sprintf("%s-%s", d.platform.Name, DataIndexServiceName)
}

func (d DataIndex) GetEnvironmentVariables() []corev1.EnvVar {
	return []corev1.EnvVar{
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
	}
}

func (d DataIndex) GetPodResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}
}

func (d DataIndex) MergePodSpec(podSpec corev1.PodSpec) (corev1.PodSpec, error) {
	c := podSpec.DeepCopy()
	err := mergo.Merge(c, d.platform.Spec.Services.DataIndex.PodTemplate.PodSpec.ToPodSpec(), mergo.WithOverride)
	return *c, err
}

func (d DataIndex) ConfigurePersistence(containerSpec *corev1.Container) *corev1.Container {

	if d.platform.Spec.Services.DataIndex.Persistence != nil && d.platform.Spec.Services.DataIndex.Persistence.PostgreSql != nil {
		c := containerSpec.DeepCopy()
		c.Image = d.GetServiceImageName(persistenceTypePostgreSQL)
		c.Env = append(c.Env, configurePostgreSqlEnv(d.platform.Spec.Services.DataIndex.Persistence.PostgreSql, d.GetServiceName(), d.platform.Namespace)...)
		return c
	}
	return containerSpec
}

func (d DataIndex) MergeContainerSpec(containerSpec *corev1.Container) (*corev1.Container, error) {
	c := containerSpec.DeepCopy()
	err := mergo.Merge(c, d.platform.Spec.Services.DataIndex.PodTemplate.Container.ToContainer(), mergo.WithOverride)
	return c, err
}

func (d DataIndex) GetReplicaCount() int32 {
	if d.platform.Spec.Services.DataIndex.PodTemplate.Replicas != nil {
		return *d.platform.Spec.Services.DataIndex.PodTemplate.Replicas
	}
	return 1
}

func (d DataIndex) GetServiceCmName() string {
	return fmt.Sprintf("%s-props", d.GetServiceName())
}

type JobService struct {
	platform *operatorapi.SonataFlowPlatform
}

func NewJobService(platform *operatorapi.SonataFlowPlatform) JobService {
	return JobService{platform: platform}
}

func (j JobService) GetContainerName() string {
	return JobServiceName
}

func (j JobService) GetServiceImageName(persistenceName string) string {
	var tag = version.GetMajorMinor()
	if version.IsSnapshot() {
		tag = "latest"
	}
	// returns "quay.io/kiegroup/kogito-jobs-service-<persistece_layer>:<tag>"
	return fmt.Sprintf("%s-%s-%s:%s", imageNamePrefix, JobServiceName, persistenceName, tag)
}

func (j JobService) GetServiceName() string {
	return fmt.Sprintf("%s-%s", j.platform.Name, JobServiceName)
}

func (j JobService) GetServiceCmName() string {
	return fmt.Sprintf("%s-props", j.GetServiceName())
}

func (j JobService) GetEnvironmentVariables() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "QUARKUS_HTTP_CORS",
			Value: "true",
		},
		{
			Name:  "QUARKUS_HTTP_CORS_ORIGINS",
			Value: "/.*/",
		},
	}
}

func (j JobService) GetPodResourceRequirements() corev1.ResourceRequirements {
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

func (j JobService) GetReplicaCount() int32 {
	if j.platform.Spec.Services.JobService.PodTemplate.Replicas != nil {
		return *j.platform.Spec.Services.JobService.PodTemplate.Replicas
	}
	return 1
}

func (j JobService) MergeContainerSpec(containerSpec *corev1.Container) (*corev1.Container, error) {
	c := containerSpec.DeepCopy()
	err := mergo.Merge(c, j.platform.Spec.Services.JobService.PodTemplate.Container.ToContainer(), mergo.WithOverride)
	return c, err
}

func (j JobService) ConfigurePersistence(containerSpec *corev1.Container) *corev1.Container {

	if j.platform.Spec.Services.JobService.Persistence != nil && j.platform.Spec.Services.JobService.Persistence.PostgreSql != nil {
		c := containerSpec.DeepCopy()
		c.Image = j.GetServiceImageName(persistenceTypePostgreSQL)
		c.Env = append(c.Env, configurePostgreSqlEnv(j.platform.Spec.Services.JobService.Persistence.PostgreSql, j.GetServiceName(), j.platform.Namespace)...)
		return c
	}
	return containerSpec
}

func (j JobService) MergePodSpec(podSpec corev1.PodSpec) (corev1.PodSpec, error) {
	c := podSpec.DeepCopy()
	err := mergo.Merge(c, j.platform.Spec.Services.JobService.PodTemplate.PodSpec.ToPodSpec(), mergo.WithOverride)
	return *c, err
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
	dataSourceUrl := "jdbc:" + persistenceTypePostgreSQL + "://" + postgresql.ServiceRef.Name + "." + databaseNamespace + ":" + strconv.Itoa(dataSourcePort) + "/" + databaseName + "?currentSchema=" + databaseSchema
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
			Value: persistenceTypePostgreSQL,
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
