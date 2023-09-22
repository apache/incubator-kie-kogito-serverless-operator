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

package prod

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kiegroup/kogito-serverless-operator/controllers/profiles/common"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	clientruntime "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kiegroup/kogito-serverless-operator/api"
	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/test"
)

func Test_Reconciler_ProdOps(t *testing.T) {
	workflow := test.GetBaseSonataFlowWithProdOpsProfile(t.Name())
	client := test.NewKogitoClientBuilder().
		WithRuntimeObjects(workflow).
		WithStatusSubresource(workflow, &operatorapi.SonataFlowBuild{}).Build()
	result, err := NewProfileForOpsReconciler(client).Reconcile(context.TODO(), workflow)
	assert.NoError(t, err)

	assert.NotNil(t, result.RequeueAfter)
	assert.True(t, workflow.Status.GetCondition(api.BuiltConditionType).IsFalse())
	assert.Equal(t, api.BuildSkipped, workflow.Status.GetCondition(api.BuiltConditionType).Reason)
	// We need the deployment controller to tell us that the workflow is ready
	// Since we don't have it in a mocked env, the result must be ready == false
	assert.False(t, workflow.Status.IsReady())

	// Reconcile again to run the ddeployment handler
	result, err = NewProfileForOpsReconciler(client).Reconcile(context.TODO(), workflow)
	assert.NoError(t, err)

	// Let's check for the right creation of the workflow (one CM volume, one container with a custom image)
	deployment := &appsv1.Deployment{}
	err = client.Get(context.TODO(), clientruntime.ObjectKeyFromObject(workflow), deployment)
	assert.NoError(t, err)

	assert.Len(t, deployment.Spec.Template.Spec.Volumes, 1)
	assert.Len(t, deployment.Spec.Template.Spec.Containers, 1)
	assert.Len(t, deployment.Spec.Template.Spec.Containers[0].VolumeMounts, 1)
}

func Test_reconcilerProdBuildConditions(t *testing.T) {
	workflow := test.GetBaseSonataFlow(t.Name())
	platform := test.GetBasePlatformInReadyPhase(t.Name())
	client := test.NewKogitoClientBuilder().
		WithRuntimeObjects(workflow, platform).
		WithStatusSubresource(workflow, platform, &operatorapi.SonataFlowBuild{}).Build()

	result, err := NewProfileReconciler(client).Reconcile(context.TODO(), workflow)
	assert.NoError(t, err)

	assert.NotNil(t, result.RequeueAfter)
	assert.True(t, workflow.Status.IsBuildRunningOrUnknown())
	assert.False(t, workflow.Status.IsReady())

	// still building
	result, err = NewProfileReconciler(client).Reconcile(context.TODO(), workflow)
	assert.NoError(t, err)
	assert.Equal(t, requeueWhileWaitForBuild, result.RequeueAfter)
	assert.True(t, workflow.Status.IsBuildRunningOrUnknown())
	assert.False(t, workflow.Status.IsReady())

	// let's finish this build
	build := &operatorapi.SonataFlowBuild{}
	assert.NoError(t, client.Get(context.TODO(), clientruntime.ObjectKeyFromObject(workflow), build))
	build.Status.BuildPhase = operatorapi.BuildPhaseSucceeded
	assert.NoError(t, client.Status().Update(context.TODO(), build))

	// last reconciliation cycle waiting for build
	result, err = NewProfileReconciler(client).Reconcile(context.TODO(), workflow)
	assert.NoError(t, err)
	assert.Equal(t, requeueWhileWaitForBuild, result.RequeueAfter)
	assert.False(t, workflow.Status.IsBuildRunningOrUnknown())
	assert.False(t, workflow.Status.IsReady())
	assert.Equal(t, api.WaitingForBuildReason, workflow.Status.GetTopLevelCondition().Reason)

	// now we create the objects
	result, err = NewProfileReconciler(client).Reconcile(context.TODO(), workflow)
	assert.NoError(t, err)
	assert.False(t, workflow.Status.IsBuildRunningOrUnknown())
	assert.False(t, workflow.Status.IsReady())
	assert.Equal(t, api.WaitingForDeploymentReason, workflow.Status.GetTopLevelCondition().Reason)

	// now with the objects created, it should be running
	// let's update the deployment status to available == true
	deployment := &appsv1.Deployment{}
	err = client.Get(context.TODO(), clientruntime.ObjectKeyFromObject(workflow), deployment)
	assert.NoError(t, err)
	deployment.Status.Conditions = append(deployment.Status.Conditions, appsv1.DeploymentCondition{
		Type:   appsv1.DeploymentAvailable,
		Status: corev1.ConditionTrue,
	})
	err = client.Status().Update(context.TODO(), deployment)
	assert.NoError(t, err)

	result, err = NewProfileReconciler(client).Reconcile(context.TODO(), workflow)
	assert.NoError(t, err)
	assert.False(t, workflow.Status.IsBuildRunningOrUnknown())
	assert.True(t, workflow.Status.IsReady())
}

func Test_deployWorkflowReconciliationHandler_handleObjects(t *testing.T) {
	workflow := test.GetBaseSonataFlow(t.Name())
	platform := test.GetBasePlatformInReadyPhase(t.Name())
	build := test.GetLocalSucceedSonataFlowBuild(workflow.Name, workflow.Namespace)
	client := test.NewKogitoClientBuilder().
		WithRuntimeObjects(workflow, platform, build).
		WithStatusSubresource(workflow, platform, build).
		Build()
	handler := &deployWithBuildWorkflowState{
		StateSupport: fakeReconcilerSupport(client),
		ensurers:     newObjectEnsurers(&common.StateSupport{C: client}),
	}
	result, objects, err := handler.Do(context.TODO(), workflow)
	assert.Greater(t, result.RequeueAfter, int64(0))
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, objects, 3)

	deployment := &appsv1.Deployment{}
	err = client.Get(context.TODO(), clientruntime.ObjectKeyFromObject(workflow), deployment)
	assert.NoError(t, err)
	assert.NotEmpty(t, deployment.Spec.Template.Spec.Containers[0].Image)

	err = client.Get(context.TODO(), clientruntime.ObjectKeyFromObject(workflow), workflow)
	assert.NoError(t, err)
	assert.False(t, workflow.Status.IsReady())
	assert.Equal(t, api.WaitingForDeploymentReason, workflow.Status.GetTopLevelCondition().Reason)

	// let's mess with the deployment
	/* TODO the state should be able to enforce: https://issues.redhat.com/browse/KOGITO-8524
	deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = 9090
	err = client.Update(context.TODO(), deployment)
	assert.NoError(t, err)
	result, objects, err = handler.Do(context.TODO(), workflow)
	assert.True(t, result.Requeue)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, objects, 2)
	// the reconciliation state should guarantee our port
	deployment = &appsv1.Deployment{}
	err = client.Get(context.TODO(), clientruntime.ObjectKeyFromObject(workflow), deployment)
	assert.NoError(t, err)
	assert.Equal(t, int32(8080), deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	*/

}

func Test_GenerationAnnotationCheck(t *testing.T) {
	// we load a workflow with metadata.generation to 0
	workflow := test.GetBaseSonataFlow(t.Name())
	platform := test.GetBasePlatformInReadyPhase(t.Name())
	client := test.NewKogitoClientBuilder().
		WithRuntimeObjects(workflow, platform).
		WithStatusSubresource(workflow, platform, &operatorapi.SonataFlowBuild{}).Build()

	handler := &deployWithBuildWorkflowState{
		StateSupport: fakeReconcilerSupport(client),
		ensurers:     newObjectEnsurers(&common.StateSupport{C: client}),
	}
	result, objects, err := handler.Do(context.TODO(), workflow)
	assert.Greater(t, result.RequeueAfter, int64(time.Second))
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, objects, 3)

	// then we load a workflow with metadata.generation set to 1
	workflowChanged := &operatorapi.SonataFlow{}
	err = client.Get(context.TODO(), clientruntime.ObjectKeyFromObject(workflow), workflowChanged)
	assert.NoError(t, err)
	//we set the generation to 1
	workflowChanged.Generation = int64(1)
	err = client.Update(context.TODO(), workflowChanged)
	assert.NoError(t, err)
	// reconcile
	handler = &deployWithBuildWorkflowState{
		StateSupport: fakeReconcilerSupport(client),
		ensurers:     newObjectEnsurers(&common.StateSupport{C: client}),
	}
	result, objects, err = handler.Do(context.TODO(), workflowChanged)
	assert.NoError(t, err)
	// no requeue, no objects since the workflow has changed
	assert.Equal(t, time.Duration(0), result.RequeueAfter)
	assert.False(t, result.Requeue)
	assert.Len(t, objects, 0)
}

func fakeReconcilerSupport(client clientruntime.Client) *common.StateSupport {
	return &common.StateSupport{
		C: client,
	}
}
