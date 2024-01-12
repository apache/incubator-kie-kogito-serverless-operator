// Copyright 2023 Apache Software Foundation (ASF)
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

package persistence

import (
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles/common/constants"
)

const (
	defaultSchemaName   = "default"
	defaultDatabaseName = "sonataflow"

	timeoutSeconds       = 3
	failureThreshold     = 5
	initialPeriodSeconds = 15
	initialDelaySeconds  = 10
	successThreshold     = 1

	postgreSQLCPULimit      = "500m"
	postgreSQLMemoryLimit   = "256Mi"
	postgreSQLMemoryRequest = "256Mi"
	postgreSQLCPURequest    = "100m"

	defaultPostgreSQLUsername  = "sonataflow"
	defaultPostgresSQLPassword = "sonataflow"
)

var (
	WorkflowConfig *syncConfig
)

func init() {
	WorkflowConfig = newSyncConfig()
}

type syncConfig struct {
	m      sync.Mutex
	config *operatorapi.PlatformPersistenceSpec
}

func newSyncConfig() *syncConfig {
	return &syncConfig{m: sync.Mutex{}}
}

func (s *syncConfig) SetConfig(config *operatorapi.PlatformPersistenceSpec) {
	s.m.Lock()
	defer s.m.Unlock()
	s.config = config
}

func (s *syncConfig) GetPostgreSQLConfiguration() *operatorapi.PostgreSQLPlatformSpec {
	s.m.Lock()
	defer s.m.Unlock()
	if s.config != nil && s.config.PostgreSQL != nil {
		return s.config.PostgreSQL
	}
	return nil
}

func ConfigurePostgreSQLEnvFromPlatformSpec(spec *operatorapi.PostgreSQLPlatformSpec, schema string) []corev1.EnvVar {

	var namespace string
	if len(spec.ServiceRef.Namespace) > 0 {
		namespace = fmt.Sprintf(".%s", spec.ServiceRef.Namespace)
	}
	dataSourceURL := fmt.Sprintf("jdbc:postgresql://%s%s:%d/%s?currentSchema=%s", spec.ServiceRef.Name, namespace, spec.ServiceRef.Port, spec.DatabaseName, schema)

	secretRef := corev1.LocalObjectReference{
		Name: spec.SecretRef.Name,
	}
	quarkusDatasourceUsername := spec.SecretRef.UserKey
	quarkusDatasourcePassword := spec.SecretRef.PasswordKey
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
			Value: constants.PersistenceTypePostgreSQL,
		},
		{
			Name:  "QUARKUS_DATASOURCE_JDBC_URL",
			Value: dataSourceURL,
		},
		{
			Name:  "KOGITO_PERSISTENCE_TYPE",
			Value: "jdbc",
		},
		{
			Name:  "KOGITO_PERSISTENCE_PROTO_MARSHALLER",
			Value: "false",
		},
		{
			Name:  "KOGITO_PERSISTENCE_QUERY_TIMEOUT_MILLIS",
			Value: "10000",
		},
	}
}

func ConfigurePostgreSQLEnv(postgresql *operatorapi.PersistencePostgreSql, databaseSchema, databaseNamespace string) []corev1.EnvVar {
	dataSourcePort := constants.DefaultPostgreSQLPort
	databaseName := defaultDatabaseName
	dataSourceURL := postgresql.JdbcUrl
	if postgresql.ServiceRef != nil {
		if len(postgresql.ServiceRef.DatabaseSchema) > 0 {
			databaseSchema = postgresql.ServiceRef.DatabaseSchema
		}
		if len(postgresql.ServiceRef.Namespace) > 0 {
			databaseNamespace = postgresql.ServiceRef.Namespace
		}
		if postgresql.ServiceRef.Port != nil {
			dataSourcePort = *postgresql.ServiceRef.Port
		}
		if len(postgresql.ServiceRef.DatabaseName) > 0 {
			databaseName = postgresql.ServiceRef.DatabaseName
		}
		dataSourceURL = fmt.Sprintf("jdbc:postgresql://%s.%s:%d/%s?currentSchema=%s", postgresql.ServiceRef.Name, databaseNamespace, dataSourcePort, databaseName, databaseSchema)
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
			Value: constants.PersistenceTypePostgreSQL,
		},
		{
			Name:  "QUARKUS_DATASOURCE_JDBC_URL",
			Value: dataSourceURL,
		},
		{
			Name:  "KOGITO_PERSISTENCE_TYPE",
			Value: "jdbc",
		},
		{
			Name:  "KOGITO_PERSISTENCE_PROTO_MARSHALLER",
			Value: "false",
		},
		{
			Name:  "KOGITO_PERSISTENCE_QUERY_TIMEOUT_MILLIS",
			Value: "10000",
		},
	}
}
