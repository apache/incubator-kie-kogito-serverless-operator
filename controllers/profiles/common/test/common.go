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

package common_test

import (
	"context"

	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/discovery"
)

const (
	DefaultNamespace  = "default-namespace"
	namespace1        = "namespace1"
	myService1        = "my-service1"
	MyService1Address = "http://10.110.90.1:80"
	myService2        = "my-service2"
	MyService2Address = "http://10.110.90.2:80"
	myService3        = "my-service3"
	MyService3Address = "http://10.110.90.3:80"

	myKnService1        = "my-kn-service1"
	MyKnService1Address = "http://my-kn-service1.namespace1.svc.cluster.local"

	myKnService2        = "my-kn-service2"
	MyKnService2Address = "http://my-kn-service2.namespace1.svc.cluster.local"

	myKnService3        = "my-kn-service3"
	MyKnService3Address = "http://my-kn-service3.default-namespace.svc.cluster.local"

	myKnBroker1        = "my-kn-broker1"
	MyKnBroker1Address = "http://broker-ingress.knative-eventing.svc.cluster.local/namespace1/my-kn-broker1"

	myKnBroker2        = "my-kn-broker2"
	MyKnBroker2Address = "http://broker-ingress.knative-eventing.svc.cluster.local/default-namespace/my-kn-broker2"
)

type MockCatalogService struct {
}

func (c *MockCatalogService) Query(ctx context.Context, uri discovery.ResourceUri, outputFormat string) (string, error) {
	if uri.Scheme == discovery.KubernetesScheme && uri.Namespace == namespace1 && uri.Name == myService1 {
		return MyService1Address, nil
	}
	if uri.Scheme == discovery.KubernetesScheme && uri.Name == myService2 && uri.Namespace == DefaultNamespace {
		return MyService2Address, nil
	}
	if uri.Scheme == discovery.KubernetesScheme && uri.Name == myService3 && uri.Namespace == DefaultNamespace && uri.GetPort() == "http-port" {
		return MyService3Address, nil
	}
	if uri.Scheme == discovery.KnativeScheme && uri.Name == myKnService1 && uri.Namespace == namespace1 {
		return MyKnService1Address, nil
	}
	if uri.Scheme == discovery.KnativeScheme && uri.Name == myKnService2 && uri.Namespace == namespace1 {
		return MyKnService2Address, nil
	}
	if uri.Scheme == discovery.KnativeScheme && uri.Name == myKnService3 && uri.Namespace == DefaultNamespace {
		return MyKnService3Address, nil
	}
	if uri.Scheme == discovery.KnativeScheme && uri.Name == myKnBroker1 && uri.Namespace == namespace1 {
		return MyKnBroker1Address, nil
	}
	if uri.Scheme == discovery.KnativeScheme && uri.Name == myKnBroker2 && uri.Namespace == DefaultNamespace {
		return MyKnBroker2Address, nil
	}

	return "", nil
}
