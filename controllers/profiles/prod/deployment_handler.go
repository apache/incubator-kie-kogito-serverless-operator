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

	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/apache/incubator-kie-kogito-serverless-operator/api"
	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/platform"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles/common"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/profiles/common/constants"
	"github.com/apache/incubator-kie-kogito-serverless-operator/utils"
)

type deploymentReconciler struct {
	*common.StateSupport
	ensurers *objectEnsurers
}

func newDeploymentReconciler(stateSupport *common.StateSupport, ensurer *objectEnsurers) *deploymentReconciler {
	return &deploymentReconciler{
		StateSupport: stateSupport,
		ensurers:     ensurer,
	}
}

func (d *deploymentReconciler) reconcile(ctx context.Context, workflow *operatorapi.SonataFlow) (reconcile.Result, []client.Object, error) {
	return d.reconcileWithBuiltImage(ctx, workflow, "")
}

func (d *deploymentReconciler) reconcileWithBuiltImage(ctx context.Context, workflow *operatorapi.SonataFlow, image string) (reconcile.Result, []client.Object, error) {
	pl, _ := platform.GetActivePlatform(ctx, d.C, workflow.Namespace)
	userPropsCM, _, err := d.ensurers.userPropsConfigMap.Ensure(ctx, workflow)
	if err != nil {
		workflow.Status.Manager().MarkFalse(api.RunningConditionType, api.ExternalResourcesNotFoundReason, "Unable to retrieve the user properties config map")
		_, err = d.PerformStatusUpdate(ctx, workflow)
		return ctrl.Result{}, nil, err
	}
	managedPropsCM, _, err := d.ensurers.managedPropsConfigMap.Ensure(ctx, workflow, pl,
		common.UserPropertiesMutateVisitor(ctx, d.StateSupport.Catalog, workflow, pl, userPropsCM.(*v1.ConfigMap)))
	if err != nil {
		workflow.Status.Manager().MarkFalse(api.RunningConditionType, api.ExternalResourcesNotFoundReason, "Unable to retrieve the managed properties config map")
		_, err = d.PerformStatusUpdate(ctx, workflow)
		return ctrl.Result{}, nil, err
	}

	deployment, deploymentOp, err :=
		d.ensurers.deployment.Ensure(
			ctx,
			workflow,
			d.getDeploymentMutateVisitors(workflow, pl, image, userPropsCM.(*v1.ConfigMap), managedPropsCM.(*v1.ConfigMap))...,
		)
	if err != nil {
		workflow.Status.Manager().MarkFalse(api.RunningConditionType, api.DeploymentUnavailableReason, "Unable to perform the deploy due to ", err)
		_, err = d.PerformStatusUpdate(ctx, workflow)
		return reconcile.Result{}, nil, err
	}

	service, _, err := d.ensurers.service.Ensure(ctx, workflow, common.ServiceMutateVisitor(workflow, pl))
	if err != nil {
		workflow.Status.Manager().MarkFalse(api.RunningConditionType, api.DeploymentUnavailableReason, "Unable to make the service available due to ", err)
		_, err = d.PerformStatusUpdate(ctx, workflow)
		return reconcile.Result{}, nil, err
	}

	objs := []client.Object{deployment, service, managedPropsCM}

	if deploymentOp == controllerutil.OperationResultCreated {
		workflow.Status.Manager().MarkFalse(api.RunningConditionType, api.WaitingForDeploymentReason, "")
		if _, err := d.PerformStatusUpdate(ctx, workflow); err != nil {
			return reconcile.Result{Requeue: false}, nil, err
		}
		return reconcile.Result{RequeueAfter: constants.RequeueAfterFollowDeployment, Requeue: true}, objs, nil
	}

	// Follow deployment status
	result, err := common.DeploymentManager(d.C).SyncDeploymentStatus(ctx, workflow)
	if err != nil {
		return reconcile.Result{Requeue: false}, nil, err
	}

	if _, err := d.PerformStatusUpdate(ctx, workflow); err != nil {
		return reconcile.Result{Requeue: false}, nil, err
	}
	return result, objs, nil
}

func (d *deploymentReconciler) getDeploymentMutateVisitors(
	workflow *operatorapi.SonataFlow,
	platform *operatorapi.SonataFlowPlatform,
	image string,
	userPropsCM *v1.ConfigMap,
	managedPropsCM *v1.ConfigMap) []common.MutateVisitor {
	if utils.IsOpenShift() {
		return []common.MutateVisitor{common.DeploymentMutateVisitor(workflow, platform),
			mountProdConfigMapsMutateVisitor(workflow, userPropsCM, managedPropsCM),
			addOpenShiftImageTriggerDeploymentMutateVisitor(workflow, image),
			common.ImageDeploymentMutateVisitor(workflow, image),
			common.RolloutDeploymentIfCMChangedMutateVisitor(workflow, userPropsCM, managedPropsCM),
		}
	}
	return []common.MutateVisitor{common.DeploymentMutateVisitor(workflow, platform),
		common.ImageDeploymentMutateVisitor(workflow, image),
		mountProdConfigMapsMutateVisitor(workflow, userPropsCM, managedPropsCM),
		common.RolloutDeploymentIfCMChangedMutateVisitor(workflow, userPropsCM, managedPropsCM)}
}
