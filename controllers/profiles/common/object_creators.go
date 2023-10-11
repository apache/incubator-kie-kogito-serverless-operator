/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package common

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	kubeutil "github.com/kiegroup/kogito-serverless-operator/utils/kubernetes"
	"github.com/kiegroup/kogito-serverless-operator/utils/openshift"
	"github.com/kiegroup/kogito-serverless-operator/workflowproj"
)

// ObjectCreator is the func that creates the initial reference object, if the object doesn't exist in the cluster, this one is created.
// Can be used as a reference to keep the object immutable
type ObjectCreator func(workflow *operatorapi.SonataFlow) (client.Object, error)

const (
	DefaultHTTPWorkflowPortInt = 8080
	DefaultContainerName       = "workflow"

	defaultHTTPServicePort = 80

	ConfigMapWorkflowPropsVolumeName = "workflow-properties"

	// Quarkus Health Check Probe configuration.
	// See: https://quarkus.io/guides/smallrye-health#running-the-health-check

	quarkusHealthPathStarted = "/q/health/started"
	quarkusHealthPathReady   = "/q/health/ready"
	quarkusHealthPathLive    = "/q/health/live"

	// Default deployment health check configuration
	// See: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/

	healthTimeoutSeconds             = 3
	healthStartedFailureThreshold    = 5
	healthStartedPeriodSeconds       = 15
	healthStartedInitialDelaySeconds = 10
)

var defaultHTTPWorkflowPortIntStr = intstr.FromInt(DefaultHTTPWorkflowPortInt)

// DefaultApplicationProperties default application properties added to every Workflow ConfigMap
var DefaultApplicationProperties = "quarkus.http.port=" + defaultHTTPWorkflowPortIntStr.String() + "\n" +
	"quarkus.http.host=0.0.0.0\n" +
	// We disable the Knative health checks to not block the dev pod to run if Knative objects are not available
	// See: https://kiegroup.github.io/kogito-docs/serverlessworkflow/latest/eventing/consume-produce-events-with-knative-eventing.html#ref-knative-eventing-add-on-source-configuration
	"org.kie.kogito.addons.knative.eventing.health-enabled=false\n" +
	"quarkus.devservices.enabled=false\n" +
	"quarkus.kogito.devservices.enabled=false\n"

// DeploymentCreator is an objectCreator for a base Kubernetes Deployments for profiles that need to deploy the workflow on a vanilla deployment.
// It serves as a basis for a basic Quarkus Java application, expected to listen on http 8080.
func DeploymentCreator(workflow *operatorapi.SonataFlow) (client.Object, error) {
	lbl := workflowproj.GetDefaultLabels(workflow)
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
						Name: DefaultContainerName,
						Ports: []corev1.ContainerPort{{
							ContainerPort: DefaultHTTPWorkflowPortInt,
							Name:          "http",
							Protocol:      corev1.ProtocolTCP,
						}},
						TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: quarkusHealthPathLive,
									Port: defaultHTTPWorkflowPortIntStr,
								},
							},
							TimeoutSeconds: healthTimeoutSeconds,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: quarkusHealthPathReady,
									Port: defaultHTTPWorkflowPortIntStr,
								},
							},
							TimeoutSeconds: healthTimeoutSeconds,
						},
						StartupProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: quarkusHealthPathStarted,
									Port: defaultHTTPWorkflowPortIntStr,
								},
							},
							InitialDelaySeconds: healthStartedInitialDelaySeconds,
							TimeoutSeconds:      healthTimeoutSeconds,
							FailureThreshold:    healthStartedFailureThreshold,
							PeriodSeconds:       healthStartedPeriodSeconds,
						},
						ImagePullPolicy: corev1.PullAlways,
						SecurityContext: kubeutil.SecurityDefaults(),
					}},
				},
			},
		},
	}
	return deployment, nil
}

// ServiceCreator is an objectCreator for a basic Service aiming a vanilla Kubernetes Deployment.
// It maps the default HTTP port (80) to the target Java application webserver on port 8080.
func ServiceCreator(workflow *operatorapi.SonataFlow) (client.Object, error) {
	lbl := workflowproj.GetDefaultLabels(workflow)

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
				TargetPort: defaultHTTPWorkflowPortIntStr,
			}},
		},
	}

	return service, nil
}

// OpenShiftRouteCreator is an ObjectCreator for a basic Route for a workflow running on OpenShift.
// It enables the exposition of the service using an OpenShift Route.
// See: https://github.com/openshift/api/blob/d170fcdc0fa638b664e4f35f2daf753cb4afe36b/route/v1/route.crd.yaml
func OpenShiftRouteCreator(workflow *operatorapi.SonataFlow) (client.Object, error) {
	route, err := openshift.RouteForWorkflow(workflow)
	return route, err
}

// WorkflowPropsConfigMapCreator creates a ConfigMap to hold the external application properties
func WorkflowPropsConfigMapCreator(workflow *operatorapi.SonataFlow) (client.Object, error) {
	return workflowproj.CreateNewAppPropsConfigMap(workflow, DefaultApplicationProperties), nil
}
