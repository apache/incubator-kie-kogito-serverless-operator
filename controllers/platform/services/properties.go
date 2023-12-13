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

package services

import (
	"fmt"
	"net/url"
	"strings"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles/common/constants"

	"github.com/magiconair/properties"
)

func generateReactiveURL(postgresSpec *operatorapi.PersistencePostgreSql, schema string, namespace string, dbName string, port int) (string, error) {
	if len(postgresSpec.JdbcUrl) > 0 {
		s := strings.TrimLeft(postgresSpec.JdbcUrl, "jdbc:")
		u, err := url.Parse(s)
		if err != nil {
			return "", err
		}
		ret := fmt.Sprintf("%s://", u.Scheme)
		if len(u.User.Username()) > 0 {
			p, ok := u.User.Password()
			if ok {
				ret = fmt.Sprintf("%s%s:%s@", ret, u.User.Username(), p)
			}
		}
		ret = fmt.Sprintf("%s%s%s?", ret, u.Host, u.Path)
		kv, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return "", err
		}
		spv := schema
		if v, ok := kv["search_path"]; ok {
			for _, val := range v {
				if len(val) != 0 {
					spv = v[0]
				}
			}
		} else if v, ok := kv["currentSchema"]; ok {
			for _, val := range v {
				if len(val) != 0 {
					spv = v[0]
				}
			}
		}
		return fmt.Sprintf("%ssearch_path=%s", ret, spv), nil
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
	return fmt.Sprintf("%s://%s:%d/%s?search_path=%s", constants.PersistenceTypePostgreSQL, postgresSpec.ServiceRef.Name+"."+databaseNamespace, dataSourcePort, databaseName, databaseSchema), nil
}

// GenerateDataIndexApplicationProperties returns the application properties required for the Data Index Service to work when deployed in a production profile
// and data index' service's spec field `Enabled` set to true
func GenerateDataIndexApplicationProperties(platform *operatorapi.SonataFlowPlatform) *properties.Properties {
	props := properties.NewProperties()
	if dataIndexEnabled(platform) {
		di := NewDataIndexService(platform)
		props.Set(
			constants.DataIndexServiceURLProperty,
			fmt.Sprintf("%s://%s.%s/processes", constants.DataIndexServiceURLProtocol, di.GetServiceName(), platform.Namespace),
		)
	}
	return props
}

// GenerateJobServiceApplicationProperties returns the application properties required for the Job Service to work in a production profile and job service's
// service's spec field `Enabled` set to true
func GenerateJobServiceApplicationProperties(platform *operatorapi.SonataFlowPlatform) (*properties.Properties, error) {
	props := properties.NewProperties()
	if jobServiceEnabled(platform) {
		js := JobService{platform: platform}
		props.Set(
			constants.JobServiceURLProperty, fmt.Sprintf("%s://%s.%s/v2/jobs/events", constants.JobServiceURLProtocol, js.GetServiceName(), platform.Namespace))
		// disable Kafka Sink for knative events until supported
		props.Set(constants.JobServiceKafkaSinkInjectionHealthCheck, "false")
		// add data source reactive URL
		jspec := platform.Spec.Services.JobService
		if jspec.Persistence != nil && jspec.Persistence.PostgreSql != nil {
			dataSourceReactiveURL, err := generateReactiveURL(jspec.Persistence.PostgreSql, js.GetServiceName(), platform.Namespace, constants.DefaultDatabaseName, constants.DefaultPostgreSQLPort)
			if err != nil {
				return nil, err
			}
			props.Set(constants.JobServiceDataSourceReactiveURLProperty, dataSourceReactiveURL)
		}
		if dataIndexEnabled(platform) {
			di := NewDataIndexService(platform)
			props.Set(constants.JobServiceStatusChangeEventsProperty, "true")
			props.Set(constants.JobServiceStatusChangeEventsURL, fmt.Sprintf("%s://%s.%s/jobs", constants.DataIndexServiceURLProtocol, di.GetServiceName(), platform.Namespace))
		}
	}
	return props, nil
}

func dataIndexEnabled(platform *operatorapi.SonataFlowPlatform) bool {
	return platform != nil && platform.Spec.Services.DataIndex != nil &&
		platform.Spec.Services.DataIndex.Enabled != nil && *platform.Spec.Services.DataIndex.Enabled
}

func jobServiceEnabled(platform *operatorapi.SonataFlowPlatform) bool {
	return platform != nil && platform.Spec.Services.JobService != nil && platform.Spec.Services.JobService.Enabled != nil && *platform.Spec.Services.JobService.Enabled
}
