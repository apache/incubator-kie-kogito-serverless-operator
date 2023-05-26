// Copyright 2023 Red Hat, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workflowdef

import (
	"context"
	"testing"

	"github.com/serverlessworkflow/sdk-go/v2/model"
	"github.com/stretchr/testify/assert"

	"github.com/kiegroup/kogito-serverless-operator/test"
)

func TestKogitoServerlessWorkflowConverter(t *testing.T) {
	t.Run("verify that when KogitoServerlessWorkflow CR is nil an error is returned", func(t *testing.T) {
		// Create a KogitoServerlessWorkflow object with metadata and spec.
		ksw := test.GetKogitoServerlessWorkflow("../../config/samples/sw.kogito_v1alpha08_kogitoserverlessworkflow.yaml", t.Name())
		out, err := ToCNCFWorkflow(context.TODO(), ksw)
		assert.NoError(t, err)
		assert.True(t, out != nil)
		assert.Equal(t, "greeting", out.ID)
		//assert.Equal(t, "greeting-key", out.Key)
		assert.Equal(t, "0.0.1", out.Version)
		assert.Equal(t, "v1alpha08", out.SpecVersion)
		assert.Equal(t, "Greeting example on k8s!", out.Description)
		assert.Equal(t, model.JqExpressionLang, out.ExpressionLang)
		assert.Equal(t, "greeting", out.Name)
		assert.True(t, out.Functions != nil && len(out.Functions) == 1)
		assert.True(t, out.States != nil && len(out.States) == 4)
	})

}
