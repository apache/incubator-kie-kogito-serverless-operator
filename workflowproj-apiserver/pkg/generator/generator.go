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

package generator

import (
	"context"
	"io"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"

	"github.com/apache/incubator-kie-kogito-serverless-operator/api/sonataflow/v1alpha08"
	workflowprojapi "github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj"
)

const (
	applicationPropertiesFileName = "application.properties"
	workflowFileSubstr            = ".sw."
)

func NewGenerator(client *Client) *SonataFlowGenerator {
	return &SonataFlowGenerator{SonataFlowClient: client}
}

type SonataFlowGenerator struct {
	SonataFlowClient *Client
}

func (g *SonataFlowGenerator) Instantiate(ctx context.Context, r io.Reader, h http.Header, opts *workflowprojapi.SonataFlowProjInstantiateRequestOptions) (*workflowprojapi.SonataFlowProjInstance, error) {
	projResolver, err := NewProjResolver(h)
	if err != nil {
		return nil, err
	}
	workflowProject, err := projResolver.Resolve(ctx, r, opts)
	if err != nil {
		return nil, err
	}

	if err = g.createOrUpdateSonataFlow(ctx, (*v1alpha08.SonataFlow)(workflowProject.Workflow)); err != nil {
		return nil, err
	}
	if workflowProject.Properties != nil {
		if err = g.createOrUpdateConfigMap(ctx, workflowProject.Properties); err != nil {
			return nil, err
		}
	}
	for _, resource := range workflowProject.Resources {
		if err = g.createOrUpdateConfigMap(ctx, resource); err != nil {
			return nil, err
		}
	}

	projInstance := &workflowprojapi.SonataFlowProjInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
		},
		Status: workflowprojapi.SonataFlowProjInstanceStatus{
			ObjectsRef: workflowprojapi.ProjObjectsRef{
				Workflow:  corev1.LocalObjectReference{Name: workflowProject.Workflow.Name},
				Resources: resourcesConfigMapToLocalRef(workflowProject.Resources),
			},
		},
	}
	if workflowProject.Properties != nil {
		projInstance.Status.ObjectsRef.Properties = corev1.LocalObjectReference{Name: workflowProject.Properties.Name}
	}
	rest.FillObjectMetaSystemFields(projInstance)

	return projInstance, nil
}

func (g *SonataFlowGenerator) createOrUpdateConfigMap(ctx context.Context, newCM *corev1.ConfigMap) error {
	ns := request.NamespaceValue(ctx)
	cm, err := g.SonataFlowClient.Kubelet.CoreV1().ConfigMaps(ns).Get(ctx, newCM.Name, metav1.GetOptions{})
	if err == nil {
		cm.Data = newCM.Data
		cm.TypeMeta = newCM.TypeMeta
		if _, err = g.SonataFlowClient.Kubelet.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
			return err
		}
	} else if errors.IsNotFound(err) {
		klog.V(4).Infof("Creating configMap %s in the namespace %s", newCM.Name, ns)
		if _, err = g.SonataFlowClient.Kubelet.CoreV1().ConfigMaps(ns).Create(ctx, newCM, metav1.CreateOptions{}); err != nil {
			return err
		}
	}
	return err
}

func (g *SonataFlowGenerator) createOrUpdateSonataFlow(ctx context.Context, newSonataFlow *v1alpha08.SonataFlow) error {
	ns := request.NamespaceValue(ctx)
	sonataFlow, err := g.SonataFlowClient.SonataFlow.SonataFlows(ns).Get(ctx, newSonataFlow.Name, metav1.GetOptions{})
	if err == nil {
		sonataFlow.Spec = newSonataFlow.Spec
		sonataFlow.TypeMeta = newSonataFlow.TypeMeta
		if _, err = g.SonataFlowClient.SonataFlow.SonataFlows(ns).Update(ctx, sonataFlow, metav1.UpdateOptions{}); err != nil {
			return err
		}
	} else if errors.IsNotFound(err) {
		klog.V(4).Infof("Creating workflow %s in the namespace %s", newSonataFlow.Name, ns)
		if _, err = g.SonataFlowClient.SonataFlow.SonataFlows(ns).Create(ctx, newSonataFlow, metav1.CreateOptions{}); err != nil {
			return err
		}
	}
	return err
}

func resourcesConfigMapToLocalRef(cms []*corev1.ConfigMap) []corev1.LocalObjectReference {
	var refs []corev1.LocalObjectReference
	for _, cm := range cms {
		refs = append(refs, corev1.LocalObjectReference{Name: cm.Name})
	}
	return refs
}
