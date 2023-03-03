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

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/test"
)

func Test_verifyThatWeAreOnOpenShift(t *testing.T) {
	workflow := test.GetKogitoServerlessWorkflow("../../config/samples/"+test.KogitoServerlessWorkflowSampleDevModeYamlCR, t.Name())

	openshiftCluster := test.GetKogitoServerlessPlatform("../../config/samples/" + test.KogitoServerlessPlatformYamlCR)
	openshiftCluster.Spec.Cluster = operatorapi.PlatformClusterOpenShift
	openshiftCluster.Namespace = "Test_verifyThatWeAreOnOpenShift"
	openshiftCluster.Status.Phase = operatorapi.PlatformPhaseReady

	mockedApi := test.MockServiceWithInitObjects(openshiftCluster)

	ok, _ := onlyForOpenShiftClusters(context.TODO(), mockedApi.Client, workflow)
	assert.True(t, ok)

	k8sCluster := test.GetKogitoServerlessPlatform("../../config/samples/" + test.KogitoServerlessPlatformYamlCR)
	k8sCluster.Spec.Cluster = operatorapi.PlatformClusterKubernetes
	k8sCluster.Namespace = "Test_verifyThatWeAreOnOpenShift"
	k8sCluster.Status.Phase = operatorapi.PlatformPhaseReady

	mockedApi = test.MockServiceWithInitObjects(k8sCluster)

	ko, _ := onlyForOpenShiftClusters(context.TODO(), mockedApi.Client, workflow)
	assert.False(t, ko)

}

func Test_verifyThatWeAreOnKubernetes(t *testing.T) {

	workflow := test.GetKogitoServerlessWorkflow("../../config/samples/"+test.KogitoServerlessWorkflowSampleDevModeYamlCR, t.Name())

	k8sCluster := test.GetKogitoServerlessPlatform("../../config/samples/" + test.KogitoServerlessPlatformYamlCR)
	k8sCluster.Spec.Cluster = operatorapi.PlatformClusterKubernetes
	k8sCluster.Namespace = "Test_verifyThatWeAreOnKubernetes"
	k8sCluster.Status.Phase = operatorapi.PlatformPhaseReady

	mockedApi := test.MockServiceWithInitObjects(k8sCluster)

	ok, _ := onlyForKubernetesClusters(context.TODO(), mockedApi.Client, workflow)
	assert.True(t, ok)

	openshiftCluster := test.GetKogitoServerlessPlatform("../../config/samples/" + test.KogitoServerlessPlatformYamlCR)
	openshiftCluster.Spec.Cluster = operatorapi.PlatformClusterOpenShift
	openshiftCluster.Namespace = "Test_verifyThatWeAreOnKubernetes"
	openshiftCluster.Status.Phase = operatorapi.PlatformPhaseReady
	mockedApi = test.MockServiceWithInitObjects(openshiftCluster)

	ko, _ := onlyForKubernetesClusters(context.TODO(), mockedApi.Client, workflow)
	assert.False(t, ko)
}
