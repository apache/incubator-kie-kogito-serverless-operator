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

package properties

import (
	"context"
	"fmt"
	"testing"

	"github.com/apache/incubator-kie-kogito-serverless-operator/api/metadata"
	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/discovery"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/platform/services"

	"github.com/magiconair/properties"

	"github.com/stretchr/testify/assert"

	"github.com/apache/incubator-kie-kogito-serverless-operator/test"
)

const (
	defaultNamespace  = "default-namespace"
	namespace1        = "namespace1"
	myService1        = "my-service1"
	myService1Address = "http://10.110.90.1:80"
	myService2        = "my-service2"
	myService2Address = "http://10.110.90.2:80"
	myService3        = "my-service3"
	myService3Address = "http://10.110.90.3:80"

	jobServiceURLProperty                   = "mp.messaging.outgoing.kogito-job-service-job-request-events.url"
	jobServiceKafkaSinkInjectionHealthCheck = `quarkus.smallrye-health.check."org.kie.kogito.jobs.service.messaging.http.health.knative.KSinkInjectionHealthCheck".enabled`
	jobServiceStatusChangeEventsProperty    = "kogito.jobs-service.http.job-status-change-events"
	jobServiceStatusChangeEventsURL         = "mp.messaging.outgoing.kogito-job-service-job-status-events-http.url"
	jobServiceURLProtocol                   = "http"
	jobServiceDataSourceReactiveURLProperty = "quarkus.datasource.reactive.url"
)

type mockCatalogService struct {
}

func (c *mockCatalogService) Query(ctx context.Context, uri discovery.ResourceUri, outputFormat string) (string, error) {
	if uri.Scheme == discovery.KubernetesScheme && uri.Namespace == namespace1 && uri.Name == myService1 {
		return myService1Address, nil
	}
	if uri.Scheme == discovery.KubernetesScheme && uri.Name == myService2 && uri.Namespace == defaultNamespace {
		return myService2Address, nil
	}
	if uri.Scheme == discovery.KubernetesScheme && uri.Name == myService3 && uri.Namespace == defaultNamespace && uri.GetPort() == "http-port" {
		return myService3Address, nil
	}
	return "", nil
}

func Test_appPropertyHandler_WithKogitoServiceUrl(t *testing.T) {
	workflow := test.GetBaseSonataFlow("default")
	props := ImmutableApplicationProperties(workflow, nil)
	assert.Contains(t, props, kogitoServiceUrlProperty)
	assert.Contains(t, props, "http://"+workflow.Name+"."+workflow.Namespace)
}

func Test_appPropertyHandler_WithUserPropertiesWithNoUserOverrides(t *testing.T) {
	//just add some user provided properties, no overrides.
	userProperties := "property1=value1\nproperty2=value2"
	workflow := test.GetBaseSonataFlow("default")
	props := NewAppPropertyHandler(workflow, nil).WithUserProperties(userProperties).Build()
	generatedProps, propsErr := properties.LoadString(props)
	assert.NoError(t, propsErr)
	assert.Equal(t, 8, len(generatedProps.Keys()))
	assert.Equal(t, "value1", generatedProps.GetString("property1", ""))
	assert.Equal(t, "value2", generatedProps.GetString("property2", ""))
	assert.Equal(t, "http://greeting.default", generatedProps.GetString("kogito.service.url", ""))
	assert.Equal(t, "8080", generatedProps.GetString("quarkus.http.port", ""))
	assert.Equal(t, "0.0.0.0", generatedProps.GetString("quarkus.http.host", ""))
	assert.Equal(t, "false", generatedProps.GetString("org.kie.kogito.addons.knative.eventing.health-enabled", ""))
	assert.Equal(t, "false", generatedProps.GetString("quarkus.devservices.enabled", ""))
	assert.Equal(t, "false", generatedProps.GetString("quarkus.kogito.devservices.enabled", ""))
}

func Test_appPropertyHandler_WithUserPropertiesWithServiceDiscovery(t *testing.T) {
	//just add some user provided properties, no overrides.
	userProperties := "property1=value1\nproperty2=value2\n"
	//add some user properties that requires service discovery
	userProperties = userProperties + "service1=${kubernetes:services.v1/namespace1/my-service1}\n"
	userProperties = userProperties + "service2=${kubernetes:services.v1/my-service2}\n"

	workflow := test.GetBaseSonataFlow(defaultNamespace)
	props := NewAppPropertyHandler(workflow, nil).
		WithUserProperties(userProperties).
		WithServiceDiscovery(context.TODO(), &mockCatalogService{}).
		Build()
	generatedProps, propsErr := properties.LoadString(props)
	generatedProps.DisableExpansion = true
	assert.NoError(t, propsErr)
	assert.Equal(t, 12, len(generatedProps.Keys()))
	assertHasProperty(t, generatedProps, "property1", "value1")
	assertHasProperty(t, generatedProps, "property2", "value2")

	assertHasProperty(t, generatedProps, "service1", "${kubernetes:services.v1/namespace1/my-service1}")
	assertHasProperty(t, generatedProps, "service2", "${kubernetes:services.v1/my-service2}")
	//org.kie.kogito.addons.discovery.kubernetes\:services.v1\/usecase1º/my-service1 below we use the unescaped vale because the properties.LoadString removes them.
	assertHasProperty(t, generatedProps, "org.kie.kogito.addons.discovery.kubernetes:services.v1/namespace1/my-service1", myService1Address)
	//org.kie.kogito.addons.discovery.kubernetes\:services.v1\/my-service2 below we use the unescaped vale because the properties.LoadString removes them.
	assertHasProperty(t, generatedProps, "org.kie.kogito.addons.discovery.kubernetes:services.v1/my-service2", myService2Address)

	assertHasProperty(t, generatedProps, "kogito.service.url", fmt.Sprintf("http://greeting.%s", defaultNamespace))
	assertHasProperty(t, generatedProps, "quarkus.http.port", "8080")
	assertHasProperty(t, generatedProps, "quarkus.http.host", "0.0.0.0")
	assertHasProperty(t, generatedProps, "org.kie.kogito.addons.knative.eventing.health-enabled", "false")
	assertHasProperty(t, generatedProps, "quarkus.devservices.enabled", "false")
	assertHasProperty(t, generatedProps, "quarkus.kogito.devservices.enabled", "false")
}

func Test_generateDiscoveryProperties(t *testing.T) {

	catalogService := &mockCatalogService{}

	propertiesContent := "property1=value1\n"
	propertiesContent = propertiesContent + "property2=${value2}\n"
	propertiesContent = propertiesContent + "service1=${kubernetes:services.v1/namespace1/my-service1}\n"
	propertiesContent = propertiesContent + "service2=${kubernetes:services.v1/my-service2}\n"
	propertiesContent = propertiesContent + "service3=${kubernetes:services.v1/my-service3?port=http-port}\n"

	propertiesContent = propertiesContent + "non_service4=${kubernetes:--kaka}"

	props := properties.MustLoadString(propertiesContent)
	result := generateDiscoveryProperties(context.TODO(), catalogService, props, &operatorapi.SonataFlow{
		ObjectMeta: metav1.ObjectMeta{Name: "helloworld", Namespace: defaultNamespace},
	})

	assert.Equal(t, result.Len(), 3)
	assertHasProperty(t, result, "org.kie.kogito.addons.discovery.kubernetes\\:services.v1\\/namespace1\\/my-service1", myService1Address)
	assertHasProperty(t, result, "org.kie.kogito.addons.discovery.kubernetes\\:services.v1\\/my-service2", myService2Address)
	assertHasProperty(t, result, "org.kie.kogito.addons.discovery.kubernetes\\:services.v1\\/my-service3?port\\=http-port", myService3Address)
}

func assertHasProperty(t *testing.T, props *properties.Properties, expectedProperty string, expectedValue string) {
	value, ok := props.Get(expectedProperty)
	assert.True(t, ok, "Property %s, is not present as expected.", expectedProperty)
	assert.Equal(t, expectedValue, value, "Expected value for property: %s, is: %s but current value is: %s", expectedProperty, expectedValue, value)
}

func Test_generateMicroprofileServiceCatalogProperty(t *testing.T) {

	doTestGenerateMicroprofileServiceCatalogProperty(t, "kubernetes:services.v1/namespace1/financial-service",
		"org.kie.kogito.addons.discovery.kubernetes\\:services.v1\\/namespace1\\/financial-service")

	doTestGenerateMicroprofileServiceCatalogProperty(t, "kubernetes:services.v1/financial-service",
		"org.kie.kogito.addons.discovery.kubernetes\\:services.v1\\/financial-service")

	doTestGenerateMicroprofileServiceCatalogProperty(t, "kubernetes:pods.v1/namespace1/financial-service",
		"org.kie.kogito.addons.discovery.kubernetes\\:pods.v1\\/namespace1\\/financial-service")

	doTestGenerateMicroprofileServiceCatalogProperty(t, "kubernetes:pods.v1/financial-service",
		"org.kie.kogito.addons.discovery.kubernetes\\:pods.v1\\/financial-service")

	doTestGenerateMicroprofileServiceCatalogProperty(t, "kubernetes:deployments.v1.apps/namespace1/financial-service",
		"org.kie.kogito.addons.discovery.kubernetes\\:deployments.v1.apps\\/namespace1\\/financial-service")

	doTestGenerateMicroprofileServiceCatalogProperty(t, "kubernetes:deployments.v1.apps/financial-service",
		"org.kie.kogito.addons.discovery.kubernetes\\:deployments.v1.apps\\/financial-service")
}

func doTestGenerateMicroprofileServiceCatalogProperty(t *testing.T, serviceUri string, expectedProperty string) {
	mpProperty := generateMicroprofileServiceCatalogProperty(serviceUri)
	assert.Equal(t, mpProperty, expectedProperty, "expected microprofile service catalog property for serviceUri: %s, is %s, but the returned value was: %s", serviceUri, expectedProperty, mpProperty)
}

func Test_appPropertyHandler_WithServicesWithUserOverrides(t *testing.T) {
	//try to override kogito.service.url and quarkus.http.port
	userProperties := "property1=value1\nproperty2=value2\nquarkus.http.port=9090\nkogito.service.url=http://myUrl.override.com\nquarkus.http.port=9090"
	ns := "default"
	workflow := test.GetBaseSonataFlow(ns)
	enabled := true
	platform := test.GetBasePlatform()
	platform.Namespace = ns
	platform.Spec = operatorapi.SonataFlowPlatformSpec{
		Services: operatorapi.ServicesPlatformSpec{
			DataIndex: &operatorapi.ServiceSpec{
				Enabled: &enabled,
			},
			JobService: &operatorapi.ServiceSpec{
				Enabled: &enabled,
			},
		},
	}

	di := services.NewDataIndexService(platform)

	props := NewAppPropertyHandler(workflow, platform).WithUserProperties(userProperties).Build()
	generatedProps, propsErr := properties.LoadString(props)
	assert.NoError(t, propsErr)
	assert.Equal(t, 8, len(generatedProps.Keys()))
	assert.Equal(t, "value1", generatedProps.GetString("property1", ""))
	assert.Equal(t, "value2", generatedProps.GetString("property2", ""))
	//kogito.service.url takes the user provided value since it's a default mutable property.
	assert.Equal(t, "http://myUrl.override.com", generatedProps.GetString("kogito.service.url", ""))
	//quarkus.http.port remains with the default value since it's immutable.
	assert.Equal(t, "8080", generatedProps.GetString("quarkus.http.port", ""))
	assert.Equal(t, "0.0.0.0", generatedProps.GetString("quarkus.http.host", ""))
	assert.Equal(t, "false", generatedProps.GetString("org.kie.kogito.addons.knative.eventing.health-enabled", ""))
	assert.Equal(t, "false", generatedProps.GetString("quarkus.devservices.enabled", ""))
	assert.Equal(t, "false", generatedProps.GetString("quarkus.kogito.devservices.enabled", ""))
	assert.Equal(t, "", generatedProps.GetString(services.DataIndexServiceUrlProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceURLProperty, ""))

	// prod profile enables config of outgoing events url
	workflow.SetAnnotations(map[string]string{metadata.Profile: string(metadata.ProdProfile)})
	props = NewAppPropertyHandler(workflow, platform).WithUserProperties(userProperties).Build()
	generatedProps, propsErr = properties.LoadString(props)
	assert.NoError(t, propsErr)
	assert.Equal(t, 13, len(generatedProps.Keys()))
	assert.Equal(t, "http://"+platform.Name+"-"+services.DataIndexServiceName+"."+platform.Namespace+"/processes", generatedProps.GetString(services.DataIndexServiceUrlProperty, ""))
	assert.Equal(t, "http://"+platform.Name+"-"+services.JobServiceName+"."+platform.Namespace+"/v2/jobs/events", generatedProps.GetString(jobServiceURLProperty, ""))
	assert.Equal(t, "false", generatedProps.GetString(jobServiceKafkaSinkInjectionHealthCheck, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceDataSourceReactiveURLProperty, ""))
	assert.Equal(t, "true", generatedProps.GetString(jobServiceStatusChangeEventsProperty, ""))
	assert.Equal(t, generatedProps.GetString(jobServiceStatusChangeEventsURL, ""), fmt.Sprintf("%s://%s.%s/jobs", services.DataIndexServiceUrlProtocol, di.GetServiceName(), platform.Namespace))

	// disabling data index bypasses config of outgoing events url
	platform.Spec.Services.DataIndex.Enabled = nil
	props = NewAppPropertyHandler(workflow, platform).WithUserProperties(userProperties).Build()
	generatedProps, propsErr = properties.LoadString(props)
	assert.NoError(t, propsErr)
	assert.Equal(t, 10, len(generatedProps.Keys()))
	assert.Equal(t, "", generatedProps.GetString(services.DataIndexServiceUrlProperty, ""))
	assert.Equal(t, "http://"+platform.Name+"-"+services.JobServiceName+"."+platform.Namespace+"/v2/jobs/events", generatedProps.GetString(jobServiceURLProperty, ""))
	assert.Equal(t, "false", generatedProps.GetString(jobServiceKafkaSinkInjectionHealthCheck, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceStatusChangeEventsProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceStatusChangeEventsURL, ""))

	// disabling job service bypasses config of outgoing events url
	platform.Spec.Services.JobService.Enabled = nil
	props = NewAppPropertyHandler(workflow, platform).WithUserProperties(userProperties).Build()
	generatedProps, propsErr = properties.LoadString(props)
	assert.NoError(t, propsErr)
	assert.Equal(t, 8, len(generatedProps.Keys()))
	assert.Equal(t, "", generatedProps.GetString(services.DataIndexServiceUrlProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceURLProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceKafkaSinkInjectionHealthCheck, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceDataSourceReactiveURLProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceStatusChangeEventsProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceStatusChangeEventsURL, ""))

	// check that service app properties are being properly set
	props = NewServiceAppPropertyHandler(platform).WithUserProperties(userProperties).Build()
	generatedProps, propsErr = properties.LoadString(props)
	assert.NoError(t, propsErr)
	assert.Equal(t, 9, len(generatedProps.Keys()))
	assert.Equal(t, "false", generatedProps.GetString(KafkaSmallRyeHealthProperty, ""))
	assert.Equal(t, "value1", generatedProps.GetString("property1", ""))
	assert.Equal(t, "value2", generatedProps.GetString("property2", ""))
	//quarkus.http.port remains with the default value since it's immutable.
	assert.Equal(t, "8080", generatedProps.GetString("quarkus.http.port", ""))

	// check that the reactive URL is generated from the postgreSQL JDBC URL when not provided
	platform.Spec.Services.JobService = &operatorapi.ServiceSpec{
		Enabled: &enabled,
		Persistence: &operatorapi.PersistenceOptions{
			PostgreSql: &operatorapi.PersistencePostgreSql{
				ServiceRef: operatorapi.PostgreSqlServiceOptions{
					Name: "jobs-service",
				},
			},
		},
	}
	props = NewAppPropertyHandler(workflow, platform).WithUserProperties(userProperties).Build()
	generatedProps, propsErr = properties.LoadString(props)
	assert.NoError(t, propsErr)
	assert.Equal(t, 11, len(generatedProps.Keys()))
	assert.Equal(t, "", generatedProps.GetString(services.DataIndexServiceUrlProperty, ""))
	assert.Equal(t, "http://"+platform.Name+"-"+services.JobServiceName+"."+platform.Namespace+"/v2/jobs/events", generatedProps.GetString(jobServiceURLProperty, ""))
	assert.Equal(t, "false", generatedProps.GetString(jobServiceKafkaSinkInjectionHealthCheck, ""))
	assert.Equal(t, "postgresql://jobs-service.default:5432/sonataflow?search_path=sonataflow-platform-jobs-service", generatedProps.GetString(jobServiceDataSourceReactiveURLProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceStatusChangeEventsProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceStatusChangeEventsURL, ""))

	// check that the reactive URL is generated from the postgreSQL JDBC URL when provided
	platform.Spec.Services.JobService = &operatorapi.ServiceSpec{
		Enabled: &enabled,
		Persistence: &operatorapi.PersistenceOptions{
			PostgreSql: &operatorapi.PersistencePostgreSql{
				JdbcUrl: "jdbc:postgresql://timeouts-showcase-database:5432/postgres?currentSchema=jobs-service",
			},
		},
	}
	props = NewAppPropertyHandler(workflow, platform).WithUserProperties(userProperties).Build()
	generatedProps, propsErr = properties.LoadString(props)
	assert.NoError(t, propsErr)
	assert.Equal(t, 11, len(generatedProps.Keys()))
	assert.Equal(t, "", generatedProps.GetString(services.DataIndexServiceUrlProperty, ""))
	assert.Equal(t, "http://"+platform.Name+"-"+services.JobServiceName+"."+platform.Namespace+"/v2/jobs/events", generatedProps.GetString(jobServiceURLProperty, ""))
	assert.Equal(t, "false", generatedProps.GetString(jobServiceKafkaSinkInjectionHealthCheck, ""))
	assert.Equal(t, "postgresql://timeouts-showcase-database:5432/postgres?search_path=jobs-service", generatedProps.GetString(jobServiceDataSourceReactiveURLProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceStatusChangeEventsProperty, ""))
	assert.Equal(t, "", generatedProps.GetString(jobServiceStatusChangeEventsURL, ""))

}
