/*
Copyright 2022.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/kiegroup/kogito-serverless-operator/platform"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"github.com/go-logr/logr"
	"github.com/kiegroup/container-builder/api"
	clientr "github.com/kiegroup/container-builder/client"
	api08 "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/builder"
	"github.com/kiegroup/kogito-serverless-operator/constants"
	"github.com/kiegroup/kogito-serverless-operator/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// KogitoServerlessBuildReconciler reconciles a KogitoServerlessBuild object
type KogitoServerlessBuildReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	commonBuildConf corev1.ConfigMap
}

// +kubebuilder:rbac:groups=sw.kogito.kie.org,resources=kogitoserverlessbuilds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sw.kogito.kie.org,resources=kogitoserverlessbuilds/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the KogitoServerlessBuild object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *KogitoServerlessBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	build := &api08.KogitoServerlessBuild{}
	err := r.Client.Get(ctx, req.NamespacedName, build)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get KogitoServerlessWorkflow")
		return ctrl.Result{}, err
	}

	phase := build.Status.BuildPhase
	if r.commonBuildConf.Data == nil {
		r.commonBuildConf, err = utils.GetBuilderCommonConfigMap(r.Client)
	}

	if err != nil || len(r.commonBuildConf.Data[r.commonBuildConf.Data[constants.DEFAULT_BUILDER_RESOURCE_NAME_KEY]]) == 0 {
		return ctrl.Result{}, errors.NewNotFound(schema.GroupResource{
			Resource: "ConfigMap",
		}, "builder-config")
	}
	// Fetch the Platform build with the information we need for the build
	pl, err := platform.GetActivePlatform(ctx, r.Client, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup
			// logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, fmt.Sprintf("Error retrieving the active platfor. Workflow %s build cannot be performed!", build.Spec.WorkflowId))
		return reconcile.Result{RequeueAfter: 60 * time.Second}, err
	}

	customConfig, err := utils.GetCustomConfig(*pl)
	if err != nil {
		log.Error(err, fmt.Sprintf("Error retrieving the custom configuration from the platform %s in namespace %s. Workflow %s build cannot be performed", pl.Name, pl.Namespace, build.Spec.WorkflowId))
		return reconcile.Result{RequeueAfter: 60 * time.Second}, err
	}

	builder := builder.NewBuilderWithConfig(ctx, r.commonBuildConf, customConfig)

	if phase == api.BuildPhaseNone {
		workflow, err := r.retrieveWorkflowFromCR(build.Spec.WorkflowId, ctx, req)
		if err == nil {
			buildStatus, err := builder.ScheduleNewBuildWithContainerFile(build.Spec.WorkflowId, workflow)
			if err == nil {
				manageStatusUpdate(ctx, buildStatus, build, r, log)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}
		}
	} else if phase != api.BuildPhaseSucceeded && phase != api.BuildPhaseError && phase != api.BuildPhaseFailed {
		cli, _ := clientr.NewClient(true)
		buildStatus, err := builder.ReconcileBuild(&build.Status.Builder, cli)
		if err == nil {
			manageStatusUpdate(ctx, buildStatus, build, r, log)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *KogitoServerlessBuildReconciler) retrieveWorkflowFromCR(workflowId string, ctx context.Context, req ctrl.Request) ([]byte, error) {
	instance := &api08.KogitoServerlessWorkflow{}
	error := r.Client.Get(ctx, types.NamespacedName{Name: workflowId, Namespace: req.Namespace}, instance)
	workflowBytes, error := utils.GetWorkflowFromCR(instance, ctx)
	return workflowBytes, error
}

func manageStatusUpdate(ctx context.Context, build *api.Build, instance *api08.KogitoServerlessBuild, r *KogitoServerlessBuildReconciler, log logr.Logger) {
	if build.Status.Phase != instance.Status.BuildPhase {
		instance.Status.Builder = *build
		instance.Status.BuildPhase = build.Status.Phase
		err := r.Status().Update(ctx, instance)
		if err == nil {
			r.Recorder.Event(instance, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated buildphase to  %s", instance.Status.BuildPhase))
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *KogitoServerlessBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api08.KogitoServerlessBuild{}).
		Complete(r)
}
