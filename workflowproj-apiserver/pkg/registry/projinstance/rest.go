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

package projinstance

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"

	sonataflowclient "github.com/apache/incubator-kie-kogito-serverless-operator/api/generated/clientset/typed/sonataflow/v1alpha08"

	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj/v1alpha08"
)

func NewStorage(client sonataflowclient.SonataFlowProjsGetter) *REST {
	return &REST{
		SonataFlowProjClient: client,
	}
}

var _ rest.Getter = &REST{}
var _ rest.Lister = &REST{}
var _ rest.Storage = &REST{}
var _ rest.StorageMetadata = &REST{}
var _ rest.Scoper = &REST{}
var _ rest.SingularNameProvider = &REST{}

type REST struct {
	SonataFlowProjClient sonataflowclient.SonataFlowProjsGetter
}

func (r *REST) GetSingularName() string {
	return "sonataflowprojinstance"
}

func (r *REST) NamespaceScoped() bool {
	return true
}

func (r *REST) ProducesMIMETypes(verb string) []string {
	return nil
}

func (r *REST) ProducesObject(verb string) interface{} {
	return v1alpha08.SonataFlowProjInstance{}
}

func (r *REST) New() runtime.Object {
	return &workflowproj.SonataFlowProjInstance{}
}

func (r *REST) Destroy() {

}

func (r *REST) NewList() runtime.Object {
	return &workflowproj.SonataFlowProjInstanceList{}
}

func (r *REST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	klog.Info("Called sonataflowprojinstance list", "options", options)
	opt := metav1.ListOptions{
		LabelSelector:        options.LabelSelector.String(),
		FieldSelector:        options.FieldSelector.String(),
		Watch:                options.Watch,
		AllowWatchBookmarks:  options.AllowWatchBookmarks,
		ResourceVersion:      options.ResourceVersion,
		ResourceVersionMatch: options.ResourceVersionMatch,
		TimeoutSeconds:       options.TimeoutSeconds,
		Limit:                options.Limit,
		Continue:             options.Continue,
		SendInitialEvents:    options.SendInitialEvents,
	}
	projs, err := r.SonataFlowProjClient.SonataFlowProjs(request.NamespaceValue(ctx)).List(ctx, opt)
	if err != nil {
		return nil, err
	}
	instances := &workflowproj.SonataFlowProjInstanceList{}
	for _, item := range projs.Items {
		// TODO: fill the rest
		proj := workflowproj.SonataFlowProjInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      item.Name,
				Namespace: item.Namespace,
			},
		}
		rest.FillObjectMetaSystemFields(&proj)
		instances.Items = append(instances.Items, proj)
	}
	return instances, nil
}

func (r *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(workflowproj.Resource("sonataflowprojinstances")).
		ConvertToTable(ctx, object, tableOptions)
}

func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	klog.Info("Called sonataflowprojinstance getter", "options", options)
	sonataFlowProj, err := r.SonataFlowProjClient.SonataFlowProjs(request.NamespaceValue(ctx)).Get(ctx, name, *options)
	if err != nil {
		return nil, err
	}
	projInstance := &workflowproj.SonataFlowProjInstance{
		// TODO: fill the rest
		ObjectMeta: metav1.ObjectMeta{
			Name:      sonataFlowProj.Name,
			Namespace: sonataFlowProj.Namespace,
		}}
	rest.FillObjectMetaSystemFields(projInstance)
	return projInstance, nil
}
