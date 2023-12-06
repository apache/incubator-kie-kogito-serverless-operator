package services

import (
	"fmt"
	"strings"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles"

	"github.com/magiconair/properties"
)

const (
	PersistenceTypePostgreSQL = "postgresql"
)

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
	return fmt.Sprintf("%s://%s:%d/%s?search_path=%s", PersistenceTypePostgreSQL, postgresSpec.ServiceRef.Name+"."+databaseNamespace, dataSourcePort, databaseName, databaseSchema)
}

// withDataIndexServiceUrl adds the property dataIndexServiceUrlProperty to the application properties.
// See Service Discovery https://kubernetes.io/docs/concepts/services-networking/service/#dns
func GenerateDataIndexApplicationProperties(workflow *operatorapi.SonataFlow, platform *operatorapi.SonataFlowPlatform) *properties.Properties {
	props := properties.NewProperties()
	if profiles.IsProdProfile(workflow) && dataIndexEnabled(platform) {
		di := NewDataIndexService(platform)
		props.Set(
			DataIndexServiceUrlProperty,
			fmt.Sprintf("%s://%s.%s/processes", DataIndexServiceUrlProtocol, di.GetServiceName(), platform.Namespace),
		)
	}
	return props
}

// withJobServiceURL adds the property 'mp.messaging.outgoing.kogito-job-service-job-request-events.url' to the application property
func GenerateJobServiceApplicationProperties(workflow *operatorapi.SonataFlow, platform *operatorapi.SonataFlowPlatform) *properties.Properties {
	props := properties.NewProperties()
	if profiles.IsProdProfile(workflow) && jobServiceEnabled(platform) {
		js := JobService{platform: platform}
		props.Set(
			jobServiceURLProperty, fmt.Sprintf("%s://%s.%s/v2/jobs/events", jobServiceURLProtocol, js.GetServiceName(), platform.Namespace))
		// disable Kafka Sink for knative events until supported
		props.Set(jobServiceKafkaSinkInjectionHealthCheck, "false")
		// add data source reactive URL
		jspec := platform.Spec.Services.JobService
		if jspec.Persistence != nil && jspec.Persistence.PostgreSql != nil {
			dataSourceReactiveURL := generateReactiveURL(jspec.Persistence.PostgreSql, js.GetServiceName(), platform.Namespace, defaultDatabaseName, defaultPostgreSQLPort)
			props.Set(jobServiceDataSourceReactiveURLProperty, dataSourceReactiveURL)
		}
		// a.addDefaultMutableProperty(jobServiceDataSourceReactiveURLProperty, dataSourceReactiveURL)
		if profiles.IsProdProfile(workflow) && dataIndexEnabled(platform) {
			di := NewDataIndexService(platform)
			props.Set(jobServiceStatusChangeEventsProperty, "true")
			props.Set(jobServiceStatusChangeEventsURL, fmt.Sprintf("%s://%s.%s/jobs", DataIndexServiceUrlProtocol, di.GetServiceName(), platform.Namespace))
		}
	}
	return props
}

func dataIndexEnabled(platform *operatorapi.SonataFlowPlatform) bool {
	return platform != nil && platform.Spec.Services.DataIndex != nil &&
		platform.Spec.Services.DataIndex.Enabled != nil && *platform.Spec.Services.DataIndex.Enabled
}

func jobServiceEnabled(platform *operatorapi.SonataFlowPlatform) bool {
	return platform != nil && platform.Spec.Services.JobService != nil && platform.Spec.Services.JobService.Enabled != nil && *platform.Spec.Services.JobService.Enabled
}
