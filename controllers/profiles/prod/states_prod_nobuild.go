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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/controllers/profiles/common"
)

type followDeployWorkflowState struct {
	*common.StateSupport
	ensurers *objectEnsurers
}

func (f *followDeployWorkflowState) CanReconcile(workflow *operatorapi.SonataFlow) bool {
	// we always reconcile since in this flow we don't mind building anything, just reconcile the deployment state
	return true
}

func (f *followDeployWorkflowState) Do(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, []client.Object, error) {
	// didn't change, business as usual
	return newDeploymentHandler(f.StateSupport, f.ensurers).handle(ctx, workflow)
}
