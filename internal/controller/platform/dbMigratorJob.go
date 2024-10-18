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

type QuarkusDataSource struct {
	JdbcUrl  string
	Username string
	Password string
	Schema   string
}

type DBMigratorJob struct {
	MigrateDBDataIndex    bool
	DataIndexDataSource   *QuarkusDataSource
	MigrateDBJobsService  bool
	JobsServiceDataSource *QuarkusDataSource
}

const (
	dbMigrationJobName       = "sonataflow-db-migrator-job"
	dbMigrationContainerName = "db-migration-container"
	dbMigratorToolImage      = "quay.io/rhkp/incubator-kie-kogito-service-db-migration-postgresql:latest"
	dbMigrationCmd           = "./migration.sh"
)

type DBMigrationJobCfg struct {
	JobName       string
	ContainerName string
	ToolImageName string
	MigrationCmd  string
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

func getNewQuarkusDataSource(jdbcURL string, userName string, password string, schema string) *QuarkusDataSource {
	return &QuarkusDataSource{
		JdbcUrl:  jdbcURL,
		Username: userName,
		Password: password,
		Schema:   schema,
	}
}

func getQuarkusDataSourceFromPersistence(ctx context.Context, platform *operatorapi.SonataFlowPlatform, persistence *operatorapi.PersistenceOptionsSpec, defaultSchemaName string) *QuarkusDataSource {
	quarkusDataSource := getNewQuarkusDataSource("", "", "", "")
	if persistence != nil && persistence.PostgreSQL != nil {
		quarkusDataSource.JdbcUrl = persistence.PostgreSQL.JdbcUrl
		quarkusDataSource.Username, _ = services.GetSecretKeyValueString(ctx, persistence.PostgreSQL.SecretRef.Name, persistence.PostgreSQL.SecretRef.UserKey, platform.Namespace)
		quarkusDataSource.Password, _ = services.GetSecretKeyValueString(ctx, persistence.PostgreSQL.SecretRef.Name, persistence.PostgreSQL.SecretRef.PasswordKey, platform.Namespace)
		quarkusDataSource.Schema = getDBSchemaName(persistence.PostgreSQL, defaultSchemaName)
	}
	return quarkusDataSource
}

func NewDBMigratorJobData(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform, pshDI services.PlatformServiceHandler, pshJS services.PlatformServiceHandler) *DBMigratorJob {

	diJobsBasedDBMigration := false
	jsJobsBasedDBMigration := false

	if pshDI.IsPersistenceEnabledtInSpec() {
		diJobsBasedDBMigration = services.IsJobsBasedDBMigration(platform.Spec.Services.DataIndex.Persistence)
	}
	if pshJS.IsPersistenceEnabledtInSpec() {
		jsJobsBasedDBMigration = services.IsJobsBasedDBMigration(platform.Spec.Services.JobService.Persistence)
	}

	if (pshDI.IsServiceSetInSpec() && diJobsBasedDBMigration) || (pshJS.IsServiceSetInSpec() && jsJobsBasedDBMigration) {
		quarkusDataSourceDataIndex := getNewQuarkusDataSource("", "", "", "")
		quarkusDataSourceJobService := getNewQuarkusDataSource("", "", "", "")

		if diJobsBasedDBMigration {
			quarkusDataSourceDataIndex = getQuarkusDataSourceFromPersistence(ctx, platform, platform.Spec.Services.DataIndex.Persistence, "defaultDi")
		}

		if jsJobsBasedDBMigration {
			quarkusDataSourceJobService = getQuarkusDataSourceFromPersistence(ctx, platform, platform.Spec.Services.JobService.Persistence, "defaultJs")
		}

		return &DBMigratorJob{
			MigrateDBDataIndex:    diJobsBasedDBMigration,
			DataIndexDataSource:   quarkusDataSourceDataIndex,
			MigrateDBJobsService:  jsJobsBasedDBMigration,
			JobsServiceDataSource: quarkusDataSourceJobService,
		}
	}
	return nil
}

func (dbmj DBMigratorJob) CreateJobDBMigration(platform *operatorapi.SonataFlowPlatform) *batchv1.Job {
	dbMigrationJobCfg := getDBMigrationJobCfg()
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbMigrationJobCfg.JobName,
			Namespace: platform.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  dbMigrationJobCfg.ContainerName,
							Image: dbMigrationJobCfg.ToolImageName,
							Env: []corev1.EnvVar{
								{
									Name:  "MIGRATE_DB_DATAINDEX",
									Value: strconv.FormatBool(dbmj.MigrateDBDataIndex),
								},
								{
									Name:  "QUARKUS_DATASOURCE_DATAINDEX_JDBC_URL",
									Value: dbmj.DataIndexDataSource.JdbcUrl,
								},
								{
									Name:  "QUARKUS_DATASOURCE_DATAINDEX_USERNAME",
									Value: dbmj.DataIndexDataSource.Username,
								},
								{
									Name:  "QUARKUS_DATASOURCE_DATAINDEX_PASSWORD",
									Value: dbmj.DataIndexDataSource.Password,
								},
								{
									Name:  "QUARKUS_FLYWAY_DATAINDEX_SCHEMAS",
									Value: dbmj.DataIndexDataSource.Schema,
								},
								{
									Name:  "MIGRATE_DB_JOBSSERVICE",
									Value: strconv.FormatBool(dbmj.MigrateDBJobsService),
								},
								{
									Name:  "QUARKUS_DATASOURCE_JOBSSERVICE_JDBC_URL",
									Value: dbmj.JobsServiceDataSource.JdbcUrl,
								},
								{
									Name:  "QUARKUS_DATASOURCE_JOBSSERVICE_USERNAME",
									Value: dbmj.JobsServiceDataSource.Username,
								},
								{
									Name:  "QUARKUS_DATASOURCE_JOBSSERVICE_PASSWORD",
									Value: dbmj.JobsServiceDataSource.Password,
								},
								{
									Name:  "QUARKUS_FLYWAY_JOBSSERVICE_SCHEMAS",
									Value: dbmj.JobsServiceDataSource.Schema,
								},
							},
							Command: []string{
								dbMigrationJobCfg.MigrationCmd,
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

// GetDBMigrationJobStatus Returns db migration job status
func (dbmj DBMigratorJob) GetDBMigrationJobStatus(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform) (*batchv1.JobStatus, error) {
	job, err := client.BatchV1().Jobs(platform.Namespace).Get(ctx, dbMigrationJobName, metav1.GetOptions{})
	if err != nil {
		klog.V(log.E).InfoS("Error getting DB migrator job while monitoring completion: ", "error", err)
		return nil, err
	}
	return &job.Status, nil
}

// NewSonataFlowPlatformDBMigrationPhase Returns a new DB migration phase for SonataFlowPlatform
func NewSonataFlowPlatformDBMigrationPhase(status operatorapi.DBMigrationStatus, message string, reason string) *operatorapi.SonataFlowPlatformDBMigrationPhase {
	return &operatorapi.SonataFlowPlatformDBMigrationPhase{
		Status:  status,
		Message: message,
		Reason:  reason,
	}
}

// UpdateSonataFlowPlatformDBMigrationPhase Updates a given SonataFlowPlatformDBMigrationPhase with the supplied values
func UpdateSonataFlowPlatformDBMigrationPhase(dbMigrationStatus *operatorapi.SonataFlowPlatformDBMigrationPhase, status operatorapi.DBMigrationStatus, message string, reason string) *operatorapi.SonataFlowPlatformDBMigrationPhase {
	if dbMigrationStatus != nil {
		dbMigrationStatus.Status = status
		dbMigrationStatus.Message = message
		dbMigrationStatus.Reason = reason
		return dbMigrationStatus
	}
	return nil
}

func getDBMigrationJobCfg() *DBMigrationJobCfg {
	return &DBMigrationJobCfg{
		JobName:       dbMigrationJobName,
		ContainerName: dbMigrationContainerName,
		ToolImageName: dbMigratorToolImage,
		MigrationCmd:  dbMigrationCmd,
	}
}

// ReconcileDBMigrationJob Check the status of running DB migration job and return status
func (dbmj DBMigratorJob) ReconcileDBMigrationJob(ctx context.Context, client client.Client, platform *operatorapi.SonataFlowPlatform) (*batchv1.JobStatus, error) {
	platform.Status.SonataFlowPlatformDBMigrationPhase = NewSonataFlowPlatformDBMigrationPhase(operatorapi.DBMigrationStatusStarted, operatorapi.MessageDBMigrationStatusStarted, operatorapi.ReasonDBMigrationStatusStarted)

	dbMigratorJobStatus, err := dbmj.GetDBMigrationJobStatus(ctx, client, platform)
	if err != nil {
		return nil, err
	}

	klog.V(log.I).InfoS("Db migration job status: ", "active", dbMigratorJobStatus.Active, "ready", dbMigratorJobStatus.Ready, "failed", dbMigratorJobStatus.Failed, "success", dbMigratorJobStatus.Succeeded, "CompletedIndexes", dbMigratorJobStatus.CompletedIndexes, "terminatedPods", dbMigratorJobStatus.UncountedTerminatedPods)

	if dbMigratorJobStatus.Failed == 1 {
		platform.Status.SonataFlowPlatformDBMigrationPhase = UpdateSonataFlowPlatformDBMigrationPhase(platform.Status.SonataFlowPlatformDBMigrationPhase, operatorapi.DBMigrationStatusFailed, operatorapi.MessageDBMigrationStatusFailed, operatorapi.ReasonDBMigrationStatusFailed)
		klog.V(log.I).InfoS("DB migration job failed")
		return dbMigratorJobStatus, errors.New("DB migration job failed")
	} else if dbMigratorJobStatus.Succeeded == 1 {
		platform.Status.SonataFlowPlatformDBMigrationPhase = UpdateSonataFlowPlatformDBMigrationPhase(platform.Status.SonataFlowPlatformDBMigrationPhase, operatorapi.DBMigrationStatusSucceeded, operatorapi.MessageDBMigrationStatusSucceeded, operatorapi.ReasonDBMigrationStatusSucceeded)
		klog.V(log.I).InfoS("DB migration job succeeded")
	} else {
		// DB migration is still running
		platform.Status.SonataFlowPlatformDBMigrationPhase = UpdateSonataFlowPlatformDBMigrationPhase(platform.Status.SonataFlowPlatformDBMigrationPhase, operatorapi.DBMigrationStatusInProgress, operatorapi.MessageDBMigrationStatusInProgress, operatorapi.ReasonDBMigrationStatusInProgress)
	}

	return dbMigratorJobStatus, nil
}
