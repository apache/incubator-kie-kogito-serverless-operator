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

package platform

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/container-builder/client"
	"github.com/apache/incubator-kie-kogito-serverless-operator/internal/controller/platform/services"

	"github.com/apache/incubator-kie-kogito-serverless-operator/log"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

const (
	jobName                  = "kogito-db-migrator-job"
	dbMigrationContainerName = "db-migration-container"
	dbMigratorToolImage      = "quay.io/rhkp/incubator-kie-kogito-service-db-migration-postgresql:latest"
	dbMigrationCmd           = "./migration.sh"
)

type DBMigratorJob struct {
	migrateDbDataindex                   bool
	quarkusDatasourceDataindexJdbcUrl    string
	quarkusDatasourceDataindexUsername   string
	quarkusDatasourceDataindexPassword   string
	quarkusFlywayDataindexSchemas        string
	migrateDbJobsservice                 bool
	quarkusDatasourceJobsserviceJdbcUrl  string
	quarkusDatasourceJobsserviceUsername string
	quarkusDatasourceJobsservicePassword string
	quarkusFlywayJobsserviceSchemas      string
}

func getDBSchemaName(persistencePostgreSQL *operatorapi.PersistencePostgreSQL, defaultSchemaName string) string {
	jdbcURL := persistencePostgreSQL.JdbcUrl

	if len(jdbcURL) == 0 {
		if persistencePostgreSQL.ServiceRef != nil && len(persistencePostgreSQL.ServiceRef.DatabaseSchema) > 0 {
			return persistencePostgreSQL.ServiceRef.DatabaseSchema
		}
	} else {
		_, a, found := strings.Cut(jdbcURL, "currentSchema=")

		if found {
			if strings.Contains(a, "&") {
				b, _, found := strings.Cut(a, "&")
				if found {
					return b
				}
			} else {
				return a
			}
		}
	}
	return defaultSchemaName
}

func NewDBMigratorJobData(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, pshDI services.PlatformServiceHandler, pshJS services.PlatformServiceHandler) (*DBMigratorJob, error) {
	quarkusDatasourceDataindexJdbcUrl := ""
	quarkusDatasourceDataindexUsername := ""
	quarkusDatasourceDataindexPassword := ""
	quarkusFlywayDataindexSchemas := ""
	quarkusDatasourceJobsserviceJdbcUrl := ""
	quarkusDatasourceJobsserviceUsername := ""
	quarkusDatasourceJobsservicePassword := ""
	quarkusFlywayJobsserviceSchemas := ""

	migrateDbDataindex := pshDI.IsServiceEnabledInSpec()
	if migrateDbDataindex {
		quarkusDatasourceDataindexJdbcUrl = platform.Spec.Services.DataIndex.Persistence.PostgreSQL.JdbcUrl
		quarkusDatasourceDataindexUsername, _ = services.GetSecretKeyValueString(ctx, client, platform.Spec.Services.DataIndex.Persistence.PostgreSQL.SecretRef.Name, platform.Spec.Services.DataIndex.Persistence.PostgreSQL.SecretRef.UserKey, platform)
		quarkusDatasourceDataindexPassword, _ = services.GetSecretKeyValueString(ctx, client, platform.Spec.Services.DataIndex.Persistence.PostgreSQL.SecretRef.Name, platform.Spec.Services.DataIndex.Persistence.PostgreSQL.SecretRef.PasswordKey, platform)
		quarkusFlywayDataindexSchemas = getDBSchemaName(platform.Spec.Services.DataIndex.Persistence.PostgreSQL, "defaultDi")
	}
	migrateDbJobsservice := pshJS.IsServiceEnabledInSpec()
	if migrateDbJobsservice {
		quarkusDatasourceJobsserviceJdbcUrl = platform.Spec.Services.JobService.Persistence.PostgreSQL.JdbcUrl
		quarkusDatasourceJobsserviceUsername, _ = services.GetSecretKeyValueString(ctx, client, platform.Spec.Services.JobService.Persistence.PostgreSQL.SecretRef.Name, platform.Spec.Services.JobService.Persistence.PostgreSQL.SecretRef.UserKey, platform)
		quarkusDatasourceJobsservicePassword, _ = services.GetSecretKeyValueString(ctx, client, platform.Spec.Services.JobService.Persistence.PostgreSQL.SecretRef.Name, platform.Spec.Services.JobService.Persistence.PostgreSQL.SecretRef.PasswordKey, platform)
		quarkusFlywayJobsserviceSchemas = getDBSchemaName(platform.Spec.Services.JobService.Persistence.PostgreSQL, "defaultJs")
	}

	return &DBMigratorJob{
		migrateDbDataindex:                   migrateDbDataindex,
		quarkusDatasourceDataindexJdbcUrl:    quarkusDatasourceDataindexJdbcUrl,
		quarkusDatasourceDataindexUsername:   quarkusDatasourceDataindexUsername,
		quarkusDatasourceDataindexPassword:   quarkusDatasourceDataindexPassword,
		quarkusFlywayDataindexSchemas:        quarkusFlywayDataindexSchemas,
		migrateDbJobsservice:                 migrateDbJobsservice,
		quarkusDatasourceJobsserviceJdbcUrl:  quarkusDatasourceJobsserviceJdbcUrl,
		quarkusDatasourceJobsserviceUsername: quarkusDatasourceJobsserviceUsername,
		quarkusDatasourceJobsservicePassword: quarkusDatasourceJobsservicePassword,
		quarkusFlywayJobsserviceSchemas:      quarkusFlywayJobsserviceSchemas,
	}, nil
}

func (dbmj DBMigratorJob) GetDBMigratorK8sJob(platform *operatorapi.SonataFlowPlatform) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: platform.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  dbMigrationContainerName,
							Image: dbMigratorToolImage,
							Env: []corev1.EnvVar{
								{
									Name:  "MIGRATE_DB_DATAINDEX",
									Value: strconv.FormatBool(dbmj.migrateDbDataindex),
								},
								{
									Name:  "QUARKUS_DATASOURCE_DATAINDEX_JDBC_URL",
									Value: dbmj.quarkusDatasourceDataindexJdbcUrl,
								},
								{
									Name:  "QUARKUS_DATASOURCE_DATAINDEX_USERNAME",
									Value: dbmj.quarkusDatasourceDataindexUsername,
								},
								{
									Name:  "QUARKUS_DATASOURCE_DATAINDEX_PASSWORD",
									Value: dbmj.quarkusDatasourceDataindexPassword,
								},
								{
									Name:  "QUARKUS_FLYWAY_DATAINDEX_SCHEMAS",
									Value: dbmj.quarkusFlywayDataindexSchemas,
								},
								{
									Name:  "MIGRATE_DB_JOBSSERVICE",
									Value: strconv.FormatBool(dbmj.migrateDbJobsservice),
								},
								{
									Name:  "QUARKUS_DATASOURCE_JOBSSERVICE_JDBC_URL",
									Value: dbmj.quarkusDatasourceJobsserviceJdbcUrl,
								},
								{
									Name:  "QUARKUS_DATASOURCE_JOBSSERVICE_USERNAME",
									Value: dbmj.quarkusDatasourceJobsserviceUsername,
								},
								{
									Name:  "QUARKUS_DATASOURCE_JOBSSERVICE_PASSWORD",
									Value: dbmj.quarkusDatasourceJobsservicePassword,
								},
								{
									Name:  "QUARKUS_FLYWAY_JOBSSERVICE_SCHEMAS",
									Value: dbmj.quarkusFlywayJobsserviceSchemas,
								},
							},
							Command: []string{
								dbMigrationCmd,
							},
						},
					},
					RestartPolicy: "Never",
				},
			},
			BackoffLimit: pointer.Int32(0),
		},
	}
	return job
}

func (dbmj DBMigratorJob) MonitorCompletionOfDBMigratorJob(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform) error {
	platform.Status.SonataFlowPlatformDBMigrationStatus = &operatorapi.SonataFlowPlatformDBMigrationStatus{
		Status: operatorapi.DBMigrationStatusStarted,
	}

	for {
		job, err := client.BatchV1().Jobs(platform.Namespace).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			klog.V(log.E).InfoS("Error getting DB migrator job while monitoring completion: ", "error", err)
			return err
		}

		klog.V(log.I).InfoS("Started to monitor the db migration job: ", "error", err)
		platform.Status.SonataFlowPlatformDBMigrationStatus.Status = operatorapi.DBMigrationStatusInProgress

		klog.V(log.I).InfoS("Db migration job status: ", "active", job.Status.Active, "ready", job.Status.Ready, "failed", job.Status.Failed, "success", job.Status.Succeeded, "CompletedIndexes", job.Status.CompletedIndexes, "terminatedPods", job.Status.UncountedTerminatedPods)
		if job.Status.Failed > 0 {
			platform.Status.SonataFlowPlatformDBMigrationStatus.Status = operatorapi.DBMigrationStatusFailed
			klog.V(log.E).InfoS("DB migrator job failed")
			return errors.New("DB migrator job failed and could not complete")
		} else if job.Status.Succeeded > 0 {
			platform.Status.SonataFlowPlatformDBMigrationStatus.Status = operatorapi.DBMigrationStatusSucceeded
			klog.V(log.E).InfoS("DB migrator job completed successful")
			return nil
		} else {
			time.Sleep(5 * time.Second)
			klog.V(log.E).InfoS("Continue to monitoring the DB migrator job")
		}
	}
}
