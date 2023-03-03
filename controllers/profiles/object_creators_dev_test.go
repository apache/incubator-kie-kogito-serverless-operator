// Copyright 2023 Red Hat, Inc. and/or its affiliates
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

package profiles

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/test"
)

func Test_ensureWorkflowDevServiceIsExposed(t *testing.T) {
	workflow := test.GetKogitoServerlessWorkflow("../../config/samples/"+test.KogitoServerlessWorkflowSampleDevModeYamlCR, t.Name())

	//On Kubernetes we want the service exposed in Dev with NodePort
	service, _ := defaultServiceCreator(workflow)
	service.SetUID("1")
	service.SetResourceVersion("1")

	k8sCluster := test.GetKogitoServerlessPlatform("../../config/samples/" + test.KogitoServerlessPlatformYamlCR)
	k8sCluster.Spec.Cluster = operatorapi.PlatformClusterKubernetes
	k8sCluster.Namespace = "Test_ensureWorkflowDevServiceIsExposed"
	k8sCluster.Status.Phase = operatorapi.PlatformPhaseReady

	mockedApi := test.MockServiceWithInitObjects(k8sCluster)

	visitor := devProfileServiceMutateVisitor(context.TODO(), mockedApi.Client, workflow)

	mutateFn := visitor(service)

	reflectService := service.(*v1.Service)
	assert.NoError(t, mutateFn())
	assert.NotNil(t, reflectService)
	assert.NotNil(t, reflectService.Spec.Type)
	assert.NotEmpty(t, reflectService.Spec.Type)
	assert.Equal(t, reflectService.Spec.Type, v1.ServiceTypeNodePort)

	//On OpenShift we don't want the service exposed in Dev with NodePort we will use a route!
	service, _ = defaultServiceCreator(workflow)
	service.SetUID("1")
	service.SetResourceVersion("1")

	openshiftCluster := test.GetKogitoServerlessPlatform("../../config/samples/" + test.KogitoServerlessPlatformYamlCR)
	openshiftCluster.Spec.Cluster = operatorapi.PlatformClusterOpenShift
	openshiftCluster.Namespace = "Test_ensureWorkflowDevServiceIsExposed"
	openshiftCluster.Status.Phase = operatorapi.PlatformPhaseReady

	mockedApi = test.MockServiceWithInitObjects(openshiftCluster)

	visitor = devProfileServiceMutateVisitor(context.TODO(), mockedApi.Client, workflow)

	mutateFn = visitor(service)

	reflectService = service.(*v1.Service)
	assert.NoError(t, mutateFn())
	assert.NotNil(t, reflectService)
	assert.NotNil(t, reflectService.Spec.Type)
	assert.Empty(t, reflectService.Spec.Type)
}
