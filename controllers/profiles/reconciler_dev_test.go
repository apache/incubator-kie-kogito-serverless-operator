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
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/test"
)

func Test_newDevProfile(t *testing.T) {
	logger := ctrllog.FromContext(context.TODO())
	workflow := test.GetKogitoServerlessWorkflow("../../config/samples/"+test.KogitoServerlessWorkflowSampleYamlCR, t.Name())
	client := test.NewKogitoClientBuilder().WithRuntimeObjects(workflow).Build()
	devReconciler := newDevProfileReconciler(client, &logger, workflow)

	result, err := devReconciler.Reconcile(context.TODO())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// check if the objects have been created
	deployment := test.MustGetDeployment(t, client, workflow)
	assert.Equal(t, defaultKogitoServerlessWorkflowDevImage, deployment.Spec.Template.Spec.Containers[0].Image)

	cm := test.MustGetConfigMap(t, client, workflow)
	assert.NotEmpty(t, cm.Data[workflow.Name+jsonFileType])

	service := test.MustGetService(t, client, workflow)
	assert.Equal(t, int32(defaultHTTPWorkflowPort), service.Spec.Ports[0].TargetPort.IntVal)

	workflow.Status.Condition = operatorapi.RunningConditionType
	err = client.Status().Update(context.TODO(), workflow)
	assert.NoError(t, err)

	// Mess with the object
	service.Spec.Ports[0].TargetPort = intstr.FromInt(9090)
	err = client.Update(context.TODO(), service)
	assert.NoError(t, err)

	// reconcile again
	result, err = devReconciler.Reconcile(context.TODO())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// check if the reconciliation ensures the object correctly
	service = test.MustGetService(t, client, workflow)
	assert.Equal(t, int32(defaultHTTPWorkflowPort), service.Spec.Ports[0].TargetPort.IntVal)

	// now with the deployment
	deployment = test.MustGetDeployment(t, client, workflow)
	deployment.Spec.Template.Spec.Containers[0].Image = "default"
	err = client.Update(context.TODO(), deployment)
	assert.NoError(t, err)

	// reconcile
	workflow.Status.Condition = operatorapi.RunningConditionType
	err = client.Update(context.TODO(), workflow)
	assert.NoError(t, err)
	result, err = devReconciler.Reconcile(context.TODO())
	assert.NoError(t, err)
	assert.NotNil(t, result)

	deployment = test.MustGetDeployment(t, client, workflow)
	assert.Equal(t, defaultKogitoServerlessWorkflowDevImage, deployment.Spec.Template.Spec.Containers[0].Image)

}
