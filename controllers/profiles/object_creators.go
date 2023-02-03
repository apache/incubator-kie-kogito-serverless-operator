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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

type objectCreator func(workflow *operatorapi.KogitoServerlessWorkflow) client.Object

const (
	defaultHTTPWorkflowPort = 8080
	defaultHTTPServicePort  = 80
)

var defaultCreators = creators{
	deployment: deploymentCreator,
	service:    serviceCreator,
}

func labels(v *operatorapi.KogitoServerlessWorkflow) map[string]string {
	// Fetches and sets labels
	return map[string]string{
		"app": v.Name,
	}
}

type creators struct {
	deployment objectCreator
	service    objectCreator
}

func deploymentCreator(workflow *operatorapi.KogitoServerlessWorkflow) client.Object {
	lbl := labels(workflow)
	size := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
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
						ImagePullPolicy: corev1.PullAlways,
						Name:            workflow.Name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: defaultHTTPWorkflowPort,
							Name:          "http",
						}},
					}},
				},
			},
		},
	}
	return deployment
}

// serviceCreator is a code for creating a Service
func serviceCreator(workflow *operatorapi.KogitoServerlessWorkflow) client.Object {
	lbl := labels(workflow)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
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
	return service
}
