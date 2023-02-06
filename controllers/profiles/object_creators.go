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
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/utils"
)

type objectCreator func(workflow *operatorapi.KogitoServerlessWorkflow) (client.Object, error)

type objectEnforcer func(workflow *operatorapi.KogitoServerlessWorkflow, fetched client.Object) error

const (
	defaultHTTPWorkflowPort = 8080
	defaultHTTPServicePort  = 80
	labelApp                = "app"
	jsonFileType            = ".json"
)

func immutableObject(creator objectCreator) objectEnforcer {
	return func(workflow *operatorapi.KogitoServerlessWorkflow, fetched client.Object) error {
		fetched, err := creator(workflow)
		return err
	}
}

func labels(v *operatorapi.KogitoServerlessWorkflow) map[string]string {
	// Fetches and sets labels
	return map[string]string{
		labelApp: v.Name,
	}
}

// defaultDeploymentCreator is an objectCreator for a base Kubernetes Deployments for profiles that need to deploy the workflow on a vanilla deployment.
// It serves as a basis for a basic Quarkus Java application, expected to listen on http 8080.
// TODO: add probes to check the default port or the quarkus health check!
func defaultDeploymentCreator(workflow *operatorapi.KogitoServerlessWorkflow) (client.Object, error) {
	lbl := labels(workflow)
	size := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
			Labels:    lbl,
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
	return deployment, nil
}

func defaultDeploymentMutator(workflow *operatorapi.KogitoServerlessWorkflow, fetched client.Object) error {
	lbl := labels(workflow)
	fetched.(*appsv1.Deployment).Labels = lbl
	// TODO ensure the actual arrays (container and ports)
	fetched.(*appsv1.Deployment).Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = defaultHTTPWorkflowPort
	fetched.(*appsv1.Deployment).Spec.Template.Labels = lbl
	return nil
}

// defaultServiceCreator is an objectCreator for a basic Service aiming a vanilla Kubernetes Deployment.
// It maps the default HTTP port (80) to the target Java application webserver on port 8080.
func defaultServiceCreator(workflow *operatorapi.KogitoServerlessWorkflow) (client.Object, error) {
	lbl := labels(workflow)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
			Labels:    lbl,
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
	return service, nil
}

// workflowSpecConfigMapCreator creates a new ConfigMap that holds the definition of a workflow specification.
func workflowSpecConfigMapCreator(workflow *operatorapi.KogitoServerlessWorkflow) (client.Object, error) {
	workflowDef, err := utils.GetJSONWorkflow(workflow, context.TODO())
	if err != nil {
		return nil, err
	}
	lbl := labels(workflow)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workflow.Name,
			Namespace: workflow.Namespace,
			Labels:    lbl,
		},
		Data: map[string]string{workflow.Name + jsonFileType: string(workflowDef)},
	}
	return configMap, nil
}
