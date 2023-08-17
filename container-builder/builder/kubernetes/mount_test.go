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

package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kiegroup/kogito-serverless-operator/container-builder/api"
	"github.com/kiegroup/kogito-serverless-operator/container-builder/util/test"
)

func Test_addResourcesToBuilderContextVolume_specificPath(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cm-data",
			Namespace: t.Name(),
		},
		Data: map[string]string{
			"specfile.json": "{}",
		},
	}
	task := api.PublishTask{
		ContextDir: "/build/context",
	}
	build := &api.ContainerBuild{
		ObjectReference: api.ObjectReference{
			Name:      "build",
			Namespace: t.Name(),
		},
		Status: api.ContainerBuildStatus{
			ResourceVolumes: []api.ContainerBuildResourceVolume{
				{
					ReferenceName:  "cm-data",
					ReferenceType:  api.ResourceReferenceTypeConfigMap,
					DestinationDir: "specs",
				},
			},
		},
	}
	volumes := make([]corev1.Volume, 0)
	volumeMounts := make([]corev1.VolumeMount, 0)
	client := test.NewFakeClient(cm)

	err := addResourcesToBuilderContextVolume(context.TODO(), client, task, build, &volumes, &volumeMounts)
	assert.NoError(t, err)

	assert.Len(t, volumes, 1)
	assert.Len(t, volumeMounts, 1)
	assert.Contains(t, volumeMounts[0].MountPath, task.ContextDir)
	assert.Len(t, volumes[0].Projected.Sources, 1)
}

// TODO: add test case to verify current behavior (one created CM with a resource file)
// TODO: add test case to verify current behavior + additional CM resources
