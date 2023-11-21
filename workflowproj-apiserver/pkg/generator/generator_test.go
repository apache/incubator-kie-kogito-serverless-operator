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

package generator

import (
	"context"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	fakekubelet "k8s.io/client-go/kubernetes/fake"

	fakesonataflow "github.com/apache/incubator-kie-kogito-serverless-operator/api/generated/clientset/fake"

	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj"
)

func TestSonataFlowGenerator_Instantiate(t *testing.T) {
	workflowProjZip := mustGetFile("testdata/workflow-service.zip")
	fakeClient := newFakeClientGenerator()
	generator := NewGenerator(fakeClient)
	ctx := request.WithNamespace(context.TODO(), t.Name())

	projInstance, err := generator.Instantiate(ctx,
		workflowProjZip,
		newHeaderOctetStream(),
		&workflowproj.SonataFlowProjInstantiateRequestOptions{
			ObjectMeta: metav1.ObjectMeta{Name: "workflow-service"},
		})
	assert.NoError(t, err)
	assert.NotNil(t, projInstance)
	assert.Equal(t, "workflow-service", projInstance.Status.ObjectsRef.Workflow.Name)
	assert.Equal(t, "workflow-service-props", projInstance.Status.ObjectsRef.Properties.Name)
	assert.Equal(t, 1, len(projInstance.Status.ObjectsRef.Resources))

	sonataFlow, err := fakeClient.SonataFlow.SonataFlows(t.Name()).Get(ctx, projInstance.Status.ObjectsRef.Workflow.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Len(t, sonataFlow.Spec.Flow.States, 1)

	properties, err := fakeClient.Kubelet.CoreV1().ConfigMaps(t.Name()).Get(ctx, projInstance.Status.ObjectsRef.Properties.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotEmpty(t, properties.Data)

	for _, resourceRef := range projInstance.Status.ObjectsRef.Resources {
		resource, err := fakeClient.Kubelet.CoreV1().ConfigMaps(t.Name()).Get(ctx, resourceRef.Name, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.NotEmpty(t, resource.Data)
	}
}

func TestSonataFlowGenerator_Instantiate_OnlyWorkflow(t *testing.T) {
	workflowProjZip := mustGetFile("testdata/workflow-hello.zip")
	fakeClient := newFakeClientGenerator()
	generator := NewGenerator(fakeClient)
	ctx := request.WithNamespace(context.TODO(), t.Name())

	projInstance, err := generator.Instantiate(ctx,
		workflowProjZip,
		newHeaderOctetStream(),
		&workflowproj.SonataFlowProjInstantiateRequestOptions{
			ObjectMeta: metav1.ObjectMeta{Name: "workflow-hello"},
		})
	assert.NoError(t, err)
	assert.NotNil(t, projInstance)
	assert.Equal(t, "workflow-hello", projInstance.Status.ObjectsRef.Workflow.Name)
	assert.Empty(t, projInstance.Status.ObjectsRef.Properties.Name)
	assert.Len(t, projInstance.Status.ObjectsRef.Resources, 0)

	sonataFlow, err := fakeClient.SonataFlow.SonataFlows(t.Name()).Get(ctx, projInstance.Status.ObjectsRef.Workflow.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Len(t, sonataFlow.Spec.Flow.States, 2)
}

func mustGetFile(filepath string) io.Reader {
	file, err := os.OpenFile(filepath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return file
}

func newFakeClientGenerator(objects ...runtime.Object) *Client {
	fakeSonataFlowClient := fakesonataflow.NewSimpleClientset(objects...)
	fakeKubelet := fakekubelet.NewSimpleClientset(objects...)
	return &Client{
		SonataFlow: fakeSonataFlowClient.SonataflowV1alpha08(),
		Kubelet:    fakeKubelet,
	}
}

func newHeaderOctetStream() http.Header {
	h := make(http.Header)
	h.Add("Content-Type", string(contentTypeOctetStream))
	return h
}
