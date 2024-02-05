// Copyright 2024 Apache Software Foundation (ASF)
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

package v1alpha08

// PersistencePostgreSQL configure postgresql connection for service(s).
// +kubebuilder:validation:MinProperties=2
// +kubebuilder:validation:MaxProperties=2
type PersistencePostgreSQL struct {
	// Secret reference to the database user credentials
	SecretRef PostgreSQLSecretOptions `json:"secretRef"`
	// Service reference to postgresql datasource. Mutually exclusive to jdbcUrl.
	// +optional
	ServiceRef *PostgreSQLServiceOptions `json:"serviceRef,omitempty"`
	// PostgreSql JDBC URL. Mutually exclusive to serviceRef.
	// e.g. "jdbc:postgresql://host:port/database?currentSchema=data-index-service"
	// +optional
	JdbcUrl string `json:"jdbcUrl,omitempty"`
}

// PostgreSQLSecretOptions use credential secret for postgresql connection.
type PostgreSQLSecretOptions struct {
	// Name of the postgresql credentials secret.
	Name string `json:"name"`
	// Defaults to POSTGRESQL_USER
	// +optional
	UserKey string `json:"userKey,omitempty"`
	// Defaults to POSTGRESQL_PASSWORD
	// +optional
	PasswordKey string `json:"passwordKey,omitempty"`
}

// PostgreSQLServiceOptions use k8s service to configure postgresql jdbc url.
type PostgreSQLServiceOptions struct {
	// Name of the postgresql k8s service.
	Name string `json:"name"`
	// Namespace of the postgresql k8s service. Defaults to the SonataFlowPlatform's local namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Port to use when connecting to the postgresql k8s service. Defaults to 5432.
	// +optional
	Port *int `json:"port,omitempty"`
	// Name of postgresql database to be used. Defaults to "sonataflow"
	// +optional
	DatabaseName string `json:"databaseName,omitempty"`
	// Schema of postgresql database to be used. Defaults to "data-index-service"
	// +optional
	DatabaseSchema string `json:"databaseSchema,omitempty"`
}
