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
	"fmt"
	"strconv"

	"regexp"
	"strings"

	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/discovery"
	"github.com/apache/incubator-kie-kogito-serverless-operator/version"
	"github.com/imdario/mergo"

	"github.com/magiconair/properties"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"k8s.io/klog/v2"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles"
	"github.com/apache/incubator-kie-kogito-serverless-operator/log"
)

const (
	ConfigMapWorkflowPropsVolumeName        = "workflow-properties"
	kogitoServiceUrlProperty                = "kogito.service.url"
	kogitoServiceUrlProtocol                = "http"
	dataIndexServiceUrlProperty             = "mp.messaging.outgoing.kogito-processinstances-events.url"
	kafkaSmallRyeHealthProperty             = "quarkus.smallrye-health.check.\"io.quarkus.kafka.client.health.KafkaHealthCheck\".enabled"
	dataIndexServiceUrlProtocol             = "http"
	jobServiceURLProperty                   = "mp.messaging.outgoing.kogito-job-service-job-request-events.url"
	jobServiceKafkaSinkInjectionHealthCheck = `quarkus.smallrye-health.check."org.kie.kogito.jobs.service.messaging.http.health.knative.KSinkInjectionHealthCheck".enabled`
	jobServiceStatusChangeEventsProperty    = "kogito.jobs-service.http.job-status-change-events"
	jobServiceStatusChangeEventsURL         = "mp.messaging.outgoing.kogito-job-service-job-status-events-http.url"
	jobServiceURLProtocol                   = "http"
	jobServiceDataSourceReactiveURLProperty = "quarkus.datasource.reactive.url"

	imageNamePrefix                          = "quay.io/kiegroup/kogito"
	DataIndexServiceName                     = "data-index-service"
	DataIndexName                            = "data-index"
	JobServiceName                           = "jobs-service"
	PersistenceTypeEphemeral                 = "ephemeral"
	PersistenceTypePostgressql               = "postgresql"
	microprofileServiceCatalogPropertyPrefix = "org.kie.kogito.addons.discovery."
	discoveryLikePropertyPattern             = "^\\${(kubernetes|knative|openshift):(.*)}$"

	defaultDatabaseName   string = "sonataflow"
	defaultPostgreSQLPort int    = 5432
)

var (
	immutableApplicationProperties = "quarkus.http.port=" + DefaultHTTPWorkflowPortIntStr.String() + "\n" +
		"quarkus.http.host=0.0.0.0\n" +
		// We disable the Knative health checks to not block the dev pod to run if Knative objects are not available
		// See: https://kiegroup.github.io/kogito-docs/serverlessworkflow/latest/eventing/consume-produce-events-with-knative-eventing.html#ref-knative-eventing-add-on-source-configuration
		"org.kie.kogito.addons.knative.eventing.health-enabled=false\n" +
		"quarkus.devservices.enabled=false\n" +
		"quarkus.kogito.devservices.enabled=false\n"

	discoveryLikePropertyExpr                    = regexp.MustCompile(discoveryLikePropertyPattern)
	_                         AppPropertyHandler = &appPropertyHandler{}
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
		c.Image = d.GetServiceImageName(PersistenceTypePostgressql)
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
		c.Image = j.GetServiceImageName(PersistenceTypePostgressql)
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
	dataSourceUrl := "jdbc:" + PersistenceTypePostgressql + "://" + postgresql.ServiceRef.Name + "." + databaseNamespace + ":" + strconv.Itoa(dataSourcePort) + "/" + databaseName + "?currentSchema=" + databaseSchema
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
			Value: PersistenceTypePostgressql,
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

type AppPropertyHandler interface {
	WithUserProperties(userProperties string) AppPropertyHandler
	WithServiceDiscovery(ctx context.Context, catalog discovery.ServiceCatalog) AppPropertyHandler
	Build() string
}

type appPropertyHandler struct {
	workflow                 *operatorapi.SonataFlow
	platform                 *operatorapi.SonataFlowPlatform
	catalog                  discovery.ServiceCatalog
	ctx                      context.Context
	userProperties           string
	defaultMutableProperties string
	isService                bool
}

func (a *appPropertyHandler) WithUserProperties(properties string) AppPropertyHandler {
	a.userProperties = properties
	return a
}

func (a *appPropertyHandler) WithServiceDiscovery(ctx context.Context, catalog discovery.ServiceCatalog) AppPropertyHandler {
	a.ctx = ctx
	a.catalog = catalog
	return a
}

func (a *appPropertyHandler) Build() string {
	var props *properties.Properties
	var propErr error = nil
	if len(a.userProperties) == 0 {
		props = properties.NewProperties()
	} else {
		props, propErr = properties.LoadString(a.userProperties)
	}
	if propErr != nil {
		// can't load user's properties, ignore it
		if a.isService && a.platform != nil {
			klog.V(log.D).InfoS("Can't load user's property", "platform", a.platform.Name, "namespace", a.platform.Namespace, "properties", a.userProperties)
		} else {
			klog.V(log.D).InfoS("Can't load user's property", "workflow", a.workflow.Name, "namespace", a.workflow.Namespace, "properties", a.userProperties)
		}
		props = properties.NewProperties()
	}
	// Disable expansions since it's not our responsibility
	// Property expansion means resolving ${} within the properties and environment context. Quarkus will do that in runtime.
	props.DisableExpansion = true

	removeDiscoveryProperties(props)
	if a.requireServiceDiscovery() {
		// produce the MicroProfileConfigServiceCatalog properties for the service discovery property values if any.
		discoveryProperties := generateDiscoveryProperties(a.ctx, a.catalog, props, a.workflow)
		if discoveryProperties.Len() > 0 {
			props.Merge(discoveryProperties)
		}
	}

	defaultMutableProps := properties.MustLoadString(a.defaultMutableProperties)
	for _, k := range defaultMutableProps.Keys() {
		if _, ok := props.Get(k); ok {
			defaultMutableProps.Delete(k)
		}
	}
	// overwrite with the default mutable properties provided by the operator that are not set by the user.
	props.Merge(defaultMutableProps)
	defaultImmutableProps := properties.MustLoadString(immutableApplicationProperties)
	// finally overwrite with the defaults immutable properties.
	props.Merge(defaultImmutableProps)
	return props.String()
}

// withKogitoServiceUrl adds the property kogitoServiceUrlProperty to the application properties.
// See Service Discovery https://kubernetes.io/docs/concepts/services-networking/service/#dns
func (a *appPropertyHandler) withKogitoServiceUrl() AppPropertyHandler {
	var kogitoServiceUrl string
	if len(a.workflow.Namespace) > 0 {
		kogitoServiceUrl = fmt.Sprintf("%s://%s.%s", kogitoServiceUrlProtocol, a.workflow.Name, a.workflow.Namespace)
	} else {
		kogitoServiceUrl = fmt.Sprintf("%s://%s", kogitoServiceUrlProtocol, a.workflow.Name)
	}
	return a.addDefaultMutableProperty(kogitoServiceUrlProperty, kogitoServiceUrl)
}

// withDataIndexServiceUrl adds the property dataIndexServiceUrlProperty to the application properties.
// See Service Discovery https://kubernetes.io/docs/concepts/services-networking/service/#dns
func (a *appPropertyHandler) withDataIndexServiceUrl() AppPropertyHandler {
	if profiles.IsProdProfile(a.workflow) && dataIndexEnabled(a.platform) {
		di := NewDataIndexService(a.platform)
		a.addDefaultMutableProperty(
			dataIndexServiceUrlProperty,
			fmt.Sprintf("%s://%s.%s/processes", dataIndexServiceUrlProtocol, di.GetServiceName(), a.platform.Namespace),
		)
	}

	return a
}

func generateReactiveURL(postgresSpec *operatorapi.PersistencePostgreSql, schema string, namespace string, dbName string, port int) string {
	if len(postgresSpec.JdbcUrl) > 0 && strings.Contains(postgresSpec.JdbcUrl, "currentSchema=") {
		return strings.Replace(strings.TrimPrefix(postgresSpec.JdbcUrl, "jdbc:"), "currentSchema=", "search_path=", -1)
	}
	databaseSchema := schema
	if len(postgresSpec.ServiceRef.DatabaseSchema) > 0 {
		databaseSchema = postgresSpec.ServiceRef.DatabaseSchema
	}
	databaseNamespace := namespace
	if len(postgresSpec.ServiceRef.Namespace) > 0 {
		databaseNamespace = postgresSpec.ServiceRef.Namespace
	}
	dataSourcePort := port
	if postgresSpec.ServiceRef.Port != nil {
		dataSourcePort = *postgresSpec.ServiceRef.Port
	}
	databaseName := dbName
	if len(postgresSpec.ServiceRef.DatabaseName) > 0 {
		databaseName = postgresSpec.ServiceRef.DatabaseName
	}
	return fmt.Sprintf("%s://%s:%d/%s?search_path=%s", PersistenceTypePostgressql, postgresSpec.ServiceRef.Name+"."+databaseNamespace, dataSourcePort, databaseName, databaseSchema)
}

// withJobServiceURL adds the property 'mp.messaging.outgoing.kogito-job-service-job-request-events.url' to the application property
func (a *appPropertyHandler) withJobServiceURL() AppPropertyHandler {
	if profiles.IsProdProfile(a.workflow) && jobServiceEnabled(a.platform) {
		js := JobService{platform: a.platform}
		a.addDefaultMutableProperty(
			jobServiceURLProperty, fmt.Sprintf("%s://%s.%s/v2/jobs/events", jobServiceURLProtocol, js.GetServiceName(), a.platform.Namespace))
		// disable Kafka Sink for knative events until supported
		a.addDefaultMutableProperty(jobServiceKafkaSinkInjectionHealthCheck, "false")
		// add data source reactive URL
		jspec := a.platform.Spec.Services.JobService
		if jspec.Persistence != nil && jspec.Persistence.PostgreSql != nil {
			dataSourceReactiveURL := generateReactiveURL(jspec.Persistence.PostgreSql, js.GetServiceName(), a.platform.Namespace, defaultDatabaseName, defaultPostgreSQLPort)
			a.addDefaultMutableProperty(jobServiceDataSourceReactiveURLProperty, dataSourceReactiveURL)
		}
		// a.addDefaultMutableProperty(jobServiceDataSourceReactiveURLProperty, dataSourceReactiveURL)
		if profiles.IsProdProfile(a.workflow) && dataIndexEnabled(a.platform) {
			di := NewDataIndexService(a.platform)
			a.addDefaultMutableProperty(jobServiceStatusChangeEventsProperty, "true")
			a.addDefaultMutableProperty(jobServiceStatusChangeEventsURL, fmt.Sprintf("%s://%s.%s/jobs", dataIndexServiceUrlProtocol, di.GetServiceName(), a.platform.Namespace))
		}
	}
	return a
}

// withKafkaHealthCheckDisabled adds the property kafkaSmallRyeHealthProperty to the application properties.
// See Service Discovery https://kubernetes.io/docs/concepts/services-networking/service/#dns
func (a *appPropertyHandler) withKafkaHealthCheckDisabled() AppPropertyHandler {
	a.addDefaultMutableProperty(
		kafkaSmallRyeHealthProperty,
		"false",
	)
	return a
}

func (a *appPropertyHandler) addDefaultMutableProperty(name string, value string) AppPropertyHandler {
	a.defaultMutableProperties = a.defaultMutableProperties + fmt.Sprintf("%s=%s\n", name, value)
	return a
}

func dataIndexEnabled(platform *operatorapi.SonataFlowPlatform) bool {
	return platform != nil && platform.Spec.Services.DataIndex != nil &&
		platform.Spec.Services.DataIndex.Enabled != nil && *platform.Spec.Services.DataIndex.Enabled
}

func jobServiceEnabled(platform *operatorapi.SonataFlowPlatform) bool {
	return platform != nil && platform.Spec.Services.JobService != nil && platform.Spec.Services.JobService.Enabled != nil && *platform.Spec.Services.JobService.Enabled
}

// NewAppPropertyHandler creates the default workflow configurations property handler
// The set of properties is initialized with the operator provided immutable properties.
// The set of defaultMutableProperties is initialized with the operator provided properties that the user might override.
func NewAppPropertyHandler(workflow *operatorapi.SonataFlow, platform *operatorapi.SonataFlowPlatform) AppPropertyHandler {
	handler := &appPropertyHandler{
		workflow: workflow,
		platform: platform,
	}
	handler.withDataIndexServiceUrl()
	handler.withJobServiceURL()
	return handler.withKogitoServiceUrl()
}

// NewServicePropertyHandler creates the default service configurations property handler
// The set of properties is initialized with the operator provided immutable properties.
// The set of defaultMutableProperties is initialized with the operator provided properties that the user might override.
func NewServiceAppPropertyHandler(platform *operatorapi.SonataFlowPlatform) AppPropertyHandler {
	handler := &appPropertyHandler{
		platform:  platform,
		isService: true,
	}
	return handler.withKafkaHealthCheckDisabled()
}

// ImmutableApplicationProperties immutable default application properties that can be used with any workflow based on Quarkus.
// Alias for NewAppPropertyHandler(workflow).Build()
func ImmutableApplicationProperties(workflow *operatorapi.SonataFlow, platform *operatorapi.SonataFlowPlatform) string {
	return NewAppPropertyHandler(workflow, platform).Build()
}

func (a *appPropertyHandler) requireServiceDiscovery() bool {
	return a.ctx != nil && a.catalog != nil
}

// generateDiscoveryProperties Given a user configured properties set, generates the MicroProfileConfigServiceCatalog
// required properties to resolve the corresponding service addresses base on these properties.
// e.g.
// Given a user configured property like this:
//
//	quarkus.rest-client.acme_financial_service_yml.url=${kubernetes:services.v1/usecase1/financial-service?port=http-port}
//
// generates the following property:
//
//	org.kie.kogito.addons.discovery.kubernetes\:services.v1\/usecase1\/financial-service?port\=http-port=http://10.5.9.1:8080
//
// where http://10.5.9.1:8080 is the corresponding k8s cloud address for the service financial-service in the namespace usecase1.
func generateDiscoveryProperties(ctx context.Context, catalog discovery.ServiceCatalog, props *properties.Properties,
	workflow *operatorapi.SonataFlow) *properties.Properties {
	klog.V(log.I).Infof("Generating service discovery properties for workflow: %s, and namespace: %s.", workflow.Name, workflow.Namespace)
	result := properties.NewProperties()
	props.DisableExpansion = true
	for _, k := range props.Keys() {
		value, _ := props.Get(k)
		klog.V(log.I).Infof("Scanning property %s=%s for service discovery configuration.", k, value)
		if !discoveryLikePropertyExpr.MatchString(value) {
			klog.V(log.I).Infof("Skipping property %s=%s since it does not look like a service discovery configuration.", k, value)
		} else {
			klog.V(log.I).Infof("Property %s=%s looks like a service discovery configuration.", k, value)
			plainUri := value[2 : len(value)-1]
			if uri, err := discovery.ParseUri(plainUri); err != nil {
				klog.V(log.I).Infof("Property %s=%s not correspond to a valid service discovery configuration, it will be excluded from service discovery.", k, value)
			} else {
				if len(uri.Namespace) == 0 {
					klog.V(log.I).Infof("Current service discovery configuration has no configured namespace, workflow namespace: %s will be used instead.", workflow.Namespace)
					uri.Namespace = workflow.Namespace
				}
				if address, err := catalog.Query(ctx, *uri, discovery.KubernetesDNSAddress); err != nil {
					klog.V(log.E).ErrorS(err, "An error was produced during service address resolution.", "serviceUri", plainUri)
				} else {
					klog.V(log.I).Infof("Service: %s was resolved into the following address: %s.", plainUri, address)
					mpProperty := generateMicroprofileServiceCatalogProperty(plainUri)
					klog.V(log.I).Infof("Generating microprofile service catalog property %s=%s.", mpProperty, address)
					result.MustSet(mpProperty, address)
				}
			}
		}
	}
	return result
}

func removeDiscoveryProperties(props *properties.Properties) {
	for _, k := range props.Keys() {
		if strings.HasPrefix(k, microprofileServiceCatalogPropertyPrefix) {
			props.Delete(k)
		}
	}
}

func generateMicroprofileServiceCatalogProperty(serviceUri string) string {
	escapedServiceUri := escapeValue(serviceUri, ":")
	escapedServiceUri = escapeValue(escapedServiceUri, "/")
	escapedServiceUri = escapeValue(escapedServiceUri, "=")
	property := microprofileServiceCatalogPropertyPrefix + escapedServiceUri
	return property
}

func escapeValue(unescaped string, value string) string {
	return strings.Replace(unescaped, value, fmt.Sprintf("\\%s", value), -1)
}
