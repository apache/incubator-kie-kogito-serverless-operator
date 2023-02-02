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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

const (
	defaultHTTPWorkflowPort = 8080
	defaultHTTPServicePort  = 80
)

type defaultServiceHandler struct {
	workflow *operatorapi.KogitoServerlessWorkflow
	scheme   *runtime.Scheme
	client   client.Client
}

func (d defaultServiceHandler) ensureServiceObject() (*reconcile.Result, error) {
	service := d.createServiceObject()
	// See if service already exists and create if it doesn't
	found := &corev1.Service{}
	err := d.client.Get(context.TODO(), types.NamespacedName{
		Name:      service.Name,
		Namespace: d.workflow.Namespace,
	}, found)
	if err != nil && errors.IsNotFound(err) {

		// Create the service
		err = d.client.Create(context.TODO(), service)

		if err != nil {
			// Service creation failed
			return &reconcile.Result{}, err
		} else {
			// Service creation was successful
			return nil, nil
		}
	} else if err != nil {
		// Error that isn't due to the service not existing
		return &reconcile.Result{}, err
	}

	return nil, nil
}

// createServiceObject is a code for creating a Service
func (d defaultServiceHandler) createServiceObject() *corev1.Service {
	lbl := labels(d.workflow)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.workflow.Name,
			Namespace: d.workflow.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: lbl,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       defaultHTTPServicePort,
				TargetPort: intstr.FromInt(defaultHTTPWorkflowPort),
			}},
		},
	}

	controllerutil.SetControllerReference(d.workflow, service, d.scheme)
	return service
}
