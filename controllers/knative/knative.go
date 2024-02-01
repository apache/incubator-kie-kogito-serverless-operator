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

package knative

import (
	"context"
	"fmt"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/utils"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	clienteventingv1 "knative.dev/eventing/pkg/client/clientset/versioned/typed/eventing/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	clientservingv1 "knative.dev/serving/pkg/client/clientset/versioned/typed/serving/v1"
)

var servingClient clientservingv1.ServingV1Interface
var eventingClient clienteventingv1.EventingV1Interface

type Availability struct {
	Eventing bool
	Serving  bool
}

const (
	KSink = "K_SINK"
)

func GetKnativeServingClient(cfg *rest.Config) (clientservingv1.ServingV1Interface, error) {
	if servingClient == nil {
		if knServingClient, err := NewKnativeServingClient(cfg); err != nil {
			return nil, err
		} else {
			servingClient = knServingClient
		}
	}
	return servingClient, nil
}

func GetKnativeEventingClient(cfg *rest.Config) (clienteventingv1.EventingV1Interface, error) {
	if eventingClient == nil {
		if knEventingClient, err := NewKnativeEventingClient(cfg); err != nil {
			return nil, err
		} else {
			eventingClient = knEventingClient
		}
	}
	return eventingClient, nil
}

func NewKnativeServingClient(cfg *rest.Config) (*clientservingv1.ServingV1Client, error) {
	return clientservingv1.NewForConfig(cfg)
}

func NewKnativeEventingClient(cfg *rest.Config) (*clienteventingv1.EventingV1Client, error) {
	return clienteventingv1.NewForConfig(cfg)
}

func GetKnativeAvailability(cfg *rest.Config) (*Availability, error) {
	if cli, err := discovery.NewDiscoveryClientForConfig(cfg); err != nil {
		return nil, err
	} else {
		apiList, err := cli.ServerGroups()
		if err != nil {
			return nil, err
		}
		result := new(Availability)
		for _, group := range apiList.Groups {
			if group.Name == "serving.knative.dev" {
				result.Serving = true
			}
			if group.Name == "eventing.knative.dev" {
				result.Eventing = true
			}
		}
		return result, nil
	}
}

func GetWorkflowSink(workflow *operatorapi.SonataFlow, pl *operatorapi.SonataFlowPlatform) (*duckv1.Destination, error) {
	if workflow == nil {
		return nil, nil
	}
	if workflow.Spec.Sink != nil {
		return workflow.Spec.Sink, nil
	}
	if pl != nil && pl.Spec.Eventing != nil {
		// no sink defined in the workflow, use the platform broker
		return pl.Spec.Eventing.Broker, nil
	} else if pl.Status.ClusterPlatformRef != nil {
		// Find the platform referred by the cluster platform
		platform := &operatorapi.SonataFlowPlatform{}
		if err := utils.GetClient().Get(context.TODO(), types.NamespacedName{Namespace: pl.Status.ClusterPlatformRef.PlatformRef.Namespace, Name: pl.Status.ClusterPlatformRef.PlatformRef.Name}, platform); err != nil {
			return nil, fmt.Errorf("error reading the platform referred by the cluster platform")
		}
		if platform.Spec.Eventing != nil {
			return platform.Spec.Eventing.Broker, nil
		}
	}
	return nil, nil
}

func IsKnativeBroker(kRef *duckv1.KReference) bool {
	return kRef.APIVersion == "eventing.knative.dev/v1" && kRef.Kind == "Broker"
}
