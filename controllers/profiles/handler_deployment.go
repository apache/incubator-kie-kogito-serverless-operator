/*
 * Copyright 2023 Red Hat, Inc. and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package profiles

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

type deploymentHandler interface {
	EnsureDeployment(ctx context.Context, image string) (*reconcile.Result, error)
}

func labels(v *operatorapi.KogitoServerlessWorkflow) map[string]string {
	// Fetches and sets labels
	return map[string]string{
		"app": v.Name,
	}
}

type defaultDeploymentHandler struct {
	workflow *operatorapi.KogitoServerlessWorkflow
	scheme   *runtime.Scheme
	client   client.Client
}

func (d defaultDeploymentHandler) EnsureDeployment(ctx context.Context, image string) (*reconcile.Result, error) {
	dep := d.createDeployment(image)
	// See if deployment already exists and create if it doesn't
	found := &appsv1.Deployment{}
	err := d.client.Get(ctx, types.NamespacedName{
		Name:      dep.Name,
		Namespace: d.workflow.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		// Create the deployment
		err = d.client.Create(context.TODO(), dep)

		if err != nil {
			// Deployment failed
			return &reconcile.Result{}, err
		} else {
			// Deployment was successful
			return nil, nil
		}
	} else if err != nil {
		// Error that isn't due to the deployment not existing
		return &reconcile.Result{}, err
	} else {
		// If the deployment exists already there is an update to do
		updateErr := d.client.Update(context.TODO(), dep)
		if err != nil {
			// Error that isn't due to the deployment not existing
			return &reconcile.Result{}, updateErr
		} else {
			// Deployment was successful
			return nil, nil
		}
	}
}

// createDeployment is a code for Creating Deployment
func (d defaultDeploymentHandler) createDeployment(image string) *appsv1.Deployment {
	lbl := labels(d.workflow)
	size := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.workflow.Name,
			Namespace: d.workflow.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: lbl,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbl,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           image,
						ImagePullPolicy: corev1.PullAlways,
						Name:            d.workflow.Name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: defaultHTTPWorkflowPort,
							Name:          "http",
						}},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(d.workflow, deployment, d.scheme)
	return deployment
}
