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
	"github.com/magiconair/properties"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/incubator-kie-kogito-serverless-operator/api/metadata"
	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles/common/constants"
)

const (
	defaultSchema = "schema"
)

var (
	enabled  = true
	disabled = false
)

var _ = Describe("Platform properties", func() {

	var _ = Context("PostgreSQL properties", func() {
		var _ = DescribeTable("Generate a reactive URL", func(spec *operatorapi.PersistencePostgreSql, expectedReactiveURL string, expectedError bool) {
			res, err := generateReactiveURL(spec, defaultSchema, "default", constants.DefaultDatabaseName, constants.DefaultPostgreSQLPort)
			if expectedError {
				Expect(err).NotTo(BeNil())
			} else {
				Expect(res).To(BeIdenticalTo(expectedReactiveURL))
			}
		},
			Entry("With an invalid URL", generatePostgreSQLOptions(setJDBC("jdbc:\\postgress://url to fail/fail?here&and&here")), "", true),
			Entry("Empty JDBC string in spec", generatePostgreSQLOptions(setServiceName("svcName")), "postgresql://svcName.default:5432/sonataflow?search_path=schema", false),
			Entry("JDBC in spec with duplicated jdbc prefix and no currentSchema in URL parameter",
				generatePostgreSQLOptions(setJDBC("jdbc:jdbc:postgres://host.com:5432/path?k=v#f")), "postgres://host.com:5432/path", false),
			Entry("JDBC in spec with username and password and no currentSchema in URL parameter",
				generatePostgreSQLOptions(setJDBC("jdbc:postgres://user:pass@host.com:5432/dbName?k=v#f")), "postgres://user:pass@host.com:5432/dbName", false),
			Entry("JDBC in spec without currentSchema in URL parameter",
				generatePostgreSQLOptions(setJDBC("jdbc:postgresql://postgres:5432/sonataflow")), "postgresql://postgres:5432/sonataflow", false),
			Entry("JDBC in spec with duplicated currentSchema in URL parameter",
				generatePostgreSQLOptions(setJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema&currentSchema=myschema2")), "postgresql://postgres:5432/sonataflow?search_path=myschema", false),
			Entry("JDBC in spec with currentSchema first and search_path later in URL parameter",
				generatePostgreSQLOptions(setJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema&search_path=myschema2")), "postgresql://postgres:5432/sonataflow?search_path=myschema2", false),
			Entry("JDBC in spec with search_path first and currentSchema later in URL parameter",
				generatePostgreSQLOptions(setJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema&search_path=myschema2")), "postgresql://postgres:5432/sonataflow?search_path=myschema2", false),
			Entry("JDBC in spec with empty value in currentSchema parameter",
				generatePostgreSQLOptions(setJDBC("jdbc:postgresql://postgres:342/sonataflow?currentSchema")), "postgresql://postgres:342/sonataflow", false),
			Entry("JDBC in spec with currentSchema in URL parameter",
				generatePostgreSQLOptions(setJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema")), "postgresql://postgres:5432/sonataflow?search_path=myschema", false),
			Entry("With only database service namespace defined",
				generatePostgreSQLOptions(setServiceName("svc"), setServiceNamespace("test")), "postgresql://svc.test:5432/sonataflow?search_path=schema", false),
			Entry("With only database schema defined",
				generatePostgreSQLOptions(setServiceName("svc"), setDatabaseSchemaName("myschema")), "postgresql://svc.default:5432/sonataflow?search_path=myschema", false),
			Entry("With only database port defined",
				generatePostgreSQLOptions(setServiceName("svc"), setDBPort(3432)), "postgresql://svc.default:3432/sonataflow?search_path=schema", false),
			Entry("With only database name defined",
				generatePostgreSQLOptions(setServiceName("svc"), setDatabaseName("foo")), "postgresql://svc.default:5432/foo?search_path=schema", false),
		)
	})

	var _ = Context("Platform service properties", func() {
		var (
			emptyProperties = properties.NewProperties()
		)
		DescribeTable("Generate Data Index application properties", func(sf *operatorapi.SonataFlow, plfm *operatorapi.SonataFlowPlatform, expectedProperties *properties.Properties) {
			props, err := GenerateDataIndexApplicationProperties(sf, plfm)
			Expect(err).NotTo(HaveOccurred())
			Expect(props).To(Equal(expectedProperties))
		},
			Entry("Data index enabled in production and workflow with production profile", generateFlow(setProfileInFlow(metadata.ProdProfile)), generatePlatform(setDataIndexEnabledValue(true), setPlatformNamespace("default"), setPlatformName("foo")), func() *properties.Properties {
				props := properties.NewProperties()
				props.Set(constants.DataIndexServiceURLProperty, "http://foo-data-index-service.default/processes")
				return props
			}()),
			Entry("Data index enabled in production and workflow with dev profile", generateFlow(setProfileInFlow(metadata.DevProfile)), generatePlatform(setDataIndexEnabledValue(true), setPlatformNamespace("default"), setPlatformName("foo")), emptyProperties),
			Entry("Data index enabled field undefined and workflow with dev profile", generateFlow(setProfileInFlow(metadata.DevProfile)), generatePlatform(setPlatformNamespace("default"), setPlatformName("foo")), emptyProperties),
			Entry("Data index disabled in production and workflow has production profile", generateFlow(setProfileInFlow(metadata.ProdProfile)), generatePlatform(setDataIndexEnabledValue(false)), emptyProperties),
			Entry("Data index enabled field undefined and workflow with production profile", generateFlow(setProfileInFlow(metadata.ProdProfile)), generatePlatform(), emptyProperties),
		)

		var _ = Context("Job service application properties", func() {

			DescribeTable("for deployment of the service",
				func(plfm *operatorapi.SonataFlowPlatform, expectedProperties *properties.Properties) {
					js := NewJobService(plfm)
					props, err := js.GenerateServiceProperties()
					Expect(err).NotTo(HaveOccurred())
					Expect(props).To(Equal(expectedProperties))
				},
				Entry("without Data Index and ephemeral persistence", generatePlatform(setJobServiceEnabledValue(&enabled)), emptyProperties),
				Entry("without Data Index and postgreSQL persistence",
					generatePlatform(setJobServiceEnabledValue(&enabled), setPlatformName("foo"), setPlatformNamespace("default"), setJobServiceJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema")),
					func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.JobServiceDataSourceReactiveURL, "postgresql://postgres:5432/sonataflow?search_path=myschema")
						return p
					}()),
				Entry("with Data Index and ephemeral persistence", generatePlatform(setJobServiceEnabledValue(&enabled), setDataIndexEnabledValue(true), setPlatformName("foo"), setPlatformNamespace("default")),
					func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.JobServiceStatusChangeEvents, "true")
						p.Set(constants.JobServiceStatusChangeEventsURL, "http://foo-data-index-service.default/jobs")
						p.Sort()
						return p
					}()),
				Entry("with Data Index and Data Index and Job Service with PostgreSQL persistence", generatePlatform(setJobServiceEnabledValue(&enabled), setDataIndexEnabledValue(true), setPlatformName("foo"), setPlatformNamespace("default"), setJobServiceJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema"), setDataIndexJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema")),
					func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.JobServiceDataSourceReactiveURL, "postgresql://postgres:5432/sonataflow?search_path=myschema")
						p.Set(constants.JobServiceStatusChangeEvents, "true")
						p.Set(constants.JobServiceStatusChangeEventsURL, "http://foo-data-index-service.default/jobs")
						p.Sort()
						return p
					}()),
			)

			DescribeTable("for workflows when",
				func(wf *operatorapi.SonataFlow, plfm *operatorapi.SonataFlowPlatform, expectedProperties *properties.Properties) {
					props, err := GenerateJobServiceApplicationProperties(wf, plfm)
					Expect(err).NotTo(HaveOccurred())
					Expect(props).To(Equal(expectedProperties))
				},
				Entry("Job service disabled in production and workflow with dev profile", generateFlow(setProfileInFlow(metadata.DevProfile)), generatePlatform(setJobServiceEnabledValue(&disabled), setPlatformNamespace("default"), setPlatformName("foo")),
					func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.JobServiceRequestEventsConnector, constants.QuarkusHTTP)
						p.Set(constants.JobServiceRequestEventsURL, "http://foo-jobs-service.default/v2/jobs/events")
						p.Sort()
						return p
					}()),
				Entry("Job service enabled field undefined and workflow with dev profile", generateFlow(setProfileInFlow(metadata.DevProfile)), generatePlatform(setJobServiceEnabledValue(nil), setPlatformNamespace("default"), setPlatformName("foo")),
					func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.JobServiceRequestEventsConnector, constants.QuarkusHTTP)
						p.Set(constants.JobServiceRequestEventsURL, "http://foo-jobs-service.default/v2/jobs/events")
						p.Sort()
						return p
					}()),
				Entry("Job service enabled in production and workflow with dev profile", generateFlow(setProfileInFlow(metadata.DevProfile)), generatePlatform(setJobServiceEnabledValue(&enabled), setPlatformNamespace("default"), setPlatformName("foo")), func() *properties.Properties {
					p := properties.NewProperties()
					p.Set(constants.JobServiceRequestEventsConnector, constants.QuarkusHTTP)
					p.Set(constants.JobServiceRequestEventsURL, "http://foo-jobs-service.default/v2/jobs/events")
					p.Sort()
					return p
				}()),
				Entry("Job service disabled in production and workflow with production profile", generateFlow(setProfileInFlow(metadata.ProdProfile)), generatePlatform(setJobServiceEnabledValue(&disabled), setPlatformNamespace("default"), setPlatformName("foo")), func() *properties.Properties {
					p := properties.NewProperties()
					p.Set(constants.JobServiceRequestEventsConnector, constants.QuarkusHTTP)
					p.Set(constants.JobServiceRequestEventsURL, "http://foo-jobs-service.default/v2/jobs/events")
					p.Sort()
					return p
				}()),
				Entry("Job service enabled in production and workflow with production profile with ephemeral persistence and data index not enabled for production",
					generateFlow(setProfileInFlow(metadata.ProdProfile)), generatePlatform(setJobServiceEnabledValue(&enabled), setPlatformName("foo"), setPlatformNamespace("default"), setPlatformNamespace("default"), setPlatformName("foo")),
					func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.JobServiceRequestEventsConnector, constants.QuarkusHTTP)
						p.Set(constants.JobServiceRequestEventsURL, "http://foo-jobs-service.default/v2/jobs/events")
						p.Set(constants.KogitoProcessInstancesEnabled, "false")
						p.Sort()
						return p
					}()),
				Entry("Job service enabled in production and workflow with production profile with postgreSQL persistence and data index not enabled for production",
					generateFlow(setProfileInFlow(metadata.ProdProfile)), generatePlatform(setJobServiceEnabledValue(&enabled), setPlatformName("foo"), setPlatformNamespace("default"), setJobServiceJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema")),
					func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.KogitoProcessInstancesEnabled, "false")
						p.Set(constants.JobServiceDataSourceReactiveURL, "postgresql://postgres:5432/sonataflow?search_path=myschema")
						p.Set(constants.JobServiceRequestEventsConnector, constants.QuarkusHTTP)
						p.Set(constants.JobServiceRequestEventsURL, "http://foo-jobs-service.default/v2/jobs/events")
						p.Sort()
						return p
					}()),
				Entry("Job service enabled in production and workflow with production profile with ephemeral persistence and data index enabled for production",
					generateFlow(setProfileInFlow(metadata.ProdProfile)), generatePlatform(setJobServiceEnabledValue(&enabled), setPlatformName("foo"), setPlatformNamespace("default"), setDataIndexEnabledValue(true)),
					func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.KogitoProcessInstancesEnabled, "true")
						p.Set(constants.DataIndexServiceURLProperty, "http://foo-data-index-service.default/processes")
						p.Set(constants.JobServiceRequestEventsConnector, constants.QuarkusHTTP)
						p.Set(constants.JobServiceRequestEventsURL, "http://foo-jobs-service.default/v2/jobs/events")
						p.Sort()
						return p
					}()),
				Entry("Job service enabled in production and workflow with production profile with postgreSQL persistence and data index enabled for production",
					generateFlow(setProfileInFlow(metadata.ProdProfile)), generatePlatform(setJobServiceEnabledValue(&enabled), setPlatformName("foo"), setPlatformNamespace("default"), setJobServiceJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema"), setDataIndexEnabledValue(true)),
					func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.JobServiceDataSourceReactiveURL, "postgresql://postgres:5432/sonataflow?search_path=myschema")
						p.Set(constants.JobServiceRequestEventsConnector, constants.QuarkusHTTP)
						p.Set(constants.JobServiceRequestEventsURL, "http://foo-jobs-service.default/v2/jobs/events")
						p.Set(constants.KogitoProcessInstancesEnabled, "true")
						p.Set(constants.DataIndexServiceURLProperty, "http://foo-data-index-service.default/processes")
						p.Sort()
						return p
					}()),
				Entry("Job service disabled in production and workflow with dev profile defined with postgreSQL persistence and data index enabled for production",
					generateFlow(setProfileInFlow(metadata.ProdProfile)), generatePlatform(setJobServiceEnabledValue(&disabled), setPlatformName("foo"), setPlatformNamespace("default"), setJobServiceJDBC("jdbc:postgresql://postgres:5432/sonataflow?currentSchema=myschema"), setDataIndexEnabledValue(true)), func() *properties.Properties {
						p := properties.NewProperties()
						p.Set(constants.JobServiceRequestEventsConnector, constants.QuarkusHTTP)
						p.Set(constants.JobServiceRequestEventsURL, "http://foo-jobs-service.default/v2/jobs/events")
						p.Sort()
						return p
					}()),
			)

		})
	})

})

type wfOptionFn func(wf *operatorapi.SonataFlow)

func generateFlow(opts ...wfOptionFn) *operatorapi.SonataFlow {
	wf := &operatorapi.SonataFlow{}
	for _, f := range opts {
		f(wf)
	}
	return wf
}

func setProfileInFlow(p metadata.ProfileType) wfOptionFn {
	return func(wf *operatorapi.SonataFlow) {
		if wf.Annotations == nil {
			wf.Annotations = make(map[string]string)
		}
		wf.Annotations[metadata.Profile] = p.String()
	}
}

type plfmOptionFn func(p *operatorapi.SonataFlowPlatform)

func generatePlatform(opts ...plfmOptionFn) *operatorapi.SonataFlowPlatform {
	plfm := &operatorapi.SonataFlowPlatform{}
	for _, f := range opts {
		f(plfm)
	}
	return plfm
}

func setJobServiceEnabledValue(v *bool) plfmOptionFn {
	return func(p *operatorapi.SonataFlowPlatform) {
		if p.Spec.Services.JobService == nil {
			p.Spec.Services.JobService = &operatorapi.ServiceSpec{}
		}
		p.Spec.Services.JobService.Enabled = v
	}
}

func setDataIndexEnabledValue(v bool) plfmOptionFn {
	return func(p *operatorapi.SonataFlowPlatform) {
		if p.Spec.Services.DataIndex == nil {
			p.Spec.Services.DataIndex = &operatorapi.ServiceSpec{}
		}
		p.Spec.Services.DataIndex.Enabled = &v
	}
}

func setPlatformNamespace(namespace string) plfmOptionFn {
	return func(p *operatorapi.SonataFlowPlatform) {
		p.Namespace = namespace
	}
}

func setPlatformName(name string) plfmOptionFn {
	return func(p *operatorapi.SonataFlowPlatform) {
		p.Name = name
	}
}

func setJobServiceJDBC(jdbc string) plfmOptionFn {
	return func(p *operatorapi.SonataFlowPlatform) {
		if p.Spec.Services.JobService == nil {
			p.Spec.Services.JobService = &operatorapi.ServiceSpec{}
		}
		if p.Spec.Services.JobService.Persistence == nil {
			p.Spec.Services.JobService.Persistence = &operatorapi.PersistenceOptions{}
		}
		if p.Spec.Services.JobService.Persistence.PostgreSql == nil {
			p.Spec.Services.JobService.Persistence.PostgreSql = &operatorapi.PersistencePostgreSql{}
		}
		p.Spec.Services.JobService.Persistence.PostgreSql.JdbcUrl = jdbc
	}
}

func setDataIndexJDBC(jdbc string) plfmOptionFn {
	return func(p *operatorapi.SonataFlowPlatform) {
		if p.Spec.Services.DataIndex == nil {
			p.Spec.Services.DataIndex = &operatorapi.ServiceSpec{}
		}
		if p.Spec.Services.DataIndex.Persistence == nil {
			p.Spec.Services.DataIndex.Persistence = &operatorapi.PersistenceOptions{}
		}
		if p.Spec.Services.DataIndex.Persistence.PostgreSql == nil {
			p.Spec.Services.DataIndex.Persistence.PostgreSql = &operatorapi.PersistencePostgreSql{}
		}
		p.Spec.Services.DataIndex.Persistence.PostgreSql.JdbcUrl = jdbc
	}
}

type optionFn func(*operatorapi.PersistencePostgreSql)

func generatePostgreSQLOptions(options ...optionFn) *operatorapi.PersistencePostgreSql {
	p := &operatorapi.PersistencePostgreSql{}
	for _, f := range options {
		f(p)
	}
	return p
}

func setJDBC(url string) optionFn {
	return func(o *operatorapi.PersistencePostgreSql) {
		o.JdbcUrl = url
	}
}

func setServiceName(svcName string) optionFn {
	return func(o *operatorapi.PersistencePostgreSql) {
		if o.ServiceRef == nil {
			o.ServiceRef = &operatorapi.PostgreSqlServiceOptions{}
		}
		o.ServiceRef.Name = svcName
	}
}

func setDatabaseSchemaName(dbSchemaName string) optionFn {
	return func(o *operatorapi.PersistencePostgreSql) {
		if o.ServiceRef == nil {
			o.ServiceRef = &operatorapi.PostgreSqlServiceOptions{}
		}
		o.ServiceRef.DatabaseSchema = dbSchemaName
	}
}

func setDatabaseName(dbName string) optionFn {
	return func(o *operatorapi.PersistencePostgreSql) {
		if o.ServiceRef == nil {
			o.ServiceRef = &operatorapi.PostgreSqlServiceOptions{}
		}
		o.ServiceRef.DatabaseName = dbName
	}
}

func setServiceNamespace(svcNamespace string) optionFn {
	return func(o *operatorapi.PersistencePostgreSql) {
		if o.ServiceRef == nil {
			o.ServiceRef = &operatorapi.PostgreSqlServiceOptions{}
		}
		o.ServiceRef.Namespace = svcNamespace
	}
}

func setDBPort(portNumber int) optionFn {
	return func(o *operatorapi.PersistencePostgreSql) {
		if o.ServiceRef == nil {
			o.ServiceRef = &operatorapi.PostgreSqlServiceOptions{}
		}
		o.ServiceRef.Port = &portNumber
	}
}
