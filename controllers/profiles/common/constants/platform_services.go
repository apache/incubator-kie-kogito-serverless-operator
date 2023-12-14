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

package constants

const (
	ConfigMapWorkflowPropsVolumeName = "workflow-properties"
	PersistenceTypePostgreSQL        = "postgresql"
	PersistenceTypeEphemeral         = "ephemeral"

	DataIndexServiceURLProperty = "mp.messaging.outgoing.kogito-processinstances-events.url"
	DataIndexServiceURLProtocol = "http"

	JobServiceURLProperty                   = "mp.messaging.outgoing.kogito-job-service-job-request-events.url"
	JobServiceKafkaSinkInjectionHealthCheck = `quarkus.smallrye-health.check."org.kie.kogito.jobs.service.messaging.http.health.knative.KSinkInjectionHealthCheck".enabled`
	JobServiceStatusChangeEventsProperty    = "kogito.jobs-service.http.job-status-change-events"
	JobServiceStatusChangeEventsURL         = "mp.messaging.outgoing.kogito-job-service-job-status-events-http.url"
	JobServiceURLProtocol                   = "http"
	JobServiceDataSourceReactiveURLProperty = "quarkus.datasource.reactive.url"

	KogitoServiceUrlProperty    = "kogito.service.url"
	KogitoServiceUrlProtocol    = "http"
	KafkaSmallRyeHealthProperty = "quarkus.smallrye-health.check.\"io.quarkus.kafka.client.health.KafkaHealthCheck\".enabled"

	MicroprofileServiceCatalogPropertyPrefix = "org.kie.kogito.addons.discovery."
	DiscoveryLikePropertyPattern             = "^\\${(kubernetes|knative|openshift):(.*)}$"

	DataIndexServiceName = "data-index-service"
	JobServiceName       = "jobs-service"
	ImageNamePrefix      = "quay.io/kiegroup/kogito"
	DataIndexName        = "data-index"

	DefaultDatabaseName   string = "sonataflow"
	DefaultPostgreSQLPort int    = 5432
)
