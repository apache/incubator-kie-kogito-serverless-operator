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

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	clientruntime "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/test"
)

func Test_deployWorkflowReconciliationHandler_handleObjects(t *testing.T) {
	logger := logr.Discard()
	workflow := test.GetKogitoServerlessWorkflow("../../config/samples/"+test.KogitoServerlessWorkflowSampleYamlCR, t.Name())
	platform := test.GetKogitoServerlessPlatformInReadyPhase("../../config/samples/"+test.KogitoServerlessPlatformWithCacheYamlCR, t.Name())
	client := test.NewKogitoClientBuilder().WithRuntimeObjects(workflow, platform).Build()
	handler := &deployWorkflowReconciliationHandler{
		reconcilerSupport: fakeReconcilerSupport(client),
		ensurers: &productionEnsurers{
			deployment: newObjectEnsurer(client.Scheme(), client, logger, defaultCreators.deployment),
			service:    newObjectEnsurer(client.Scheme(), client, logger, defaultCreators.service),
		},
	}
	result, objects, err := handler.Do(context.TODO(), workflow)
	assert.True(t, result.Requeue)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, objects, 2)

	deployment := &v1.Deployment{}
	err = client.Get(context.TODO(), clientruntime.ObjectKeyFromObject(workflow), deployment)
	assert.NoError(t, err)
	assert.NotEmpty(t, deployment.Spec.Template.Spec.Containers[0].Image)

	err = client.Get(context.TODO(), clientruntime.ObjectKeyFromObject(workflow), workflow)
	assert.Equal(t, v1alpha08.RunningConditionType, workflow.Status.Condition)
}
