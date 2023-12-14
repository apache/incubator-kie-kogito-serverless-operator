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

package projinstantiate

import (
	"context"
	"io"
	"net/http"

	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog/v2"

	sonataflowv1alpha08 "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"

	"github.com/apache/incubator-kie-kogito-serverless-operator/api"

	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/generator"

	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj/v1alpha08"

	workflowprojapi "github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj"
)

func NewStorage(client *generator.Client, scheme *runtime.Scheme) *InstantiateREST {
	return &InstantiateREST{
		SonataFlowClient: client,
		Generator:        generator.NewGenerator(client),
		Scheme:           scheme,
	}
}

type InstantiateREST struct {
	Generator        *generator.SonataFlowGenerator
	SonataFlowClient *generator.Client
	Scheme           *runtime.Scheme
}

var _ rest.Connecter = &InstantiateREST{}
var _ rest.Storage = &InstantiateREST{}
var _ rest.StorageMetadata = &InstantiateREST{}

func (w *InstantiateREST) ProducesMIMETypes(verb string) []string {
	return nil
}

func (w *InstantiateREST) ProducesObject(verb string) interface{} {
	// documentation only
	return v1alpha08.SonataFlowProjInstance{}
}

func (w *InstantiateREST) New() runtime.Object {
	return &workflowprojapi.SonataFlowProjInstantiateRequestOptions{}
}

func (w *InstantiateREST) Destroy() {}

func (w *InstantiateREST) Connect(ctx context.Context, id string, options runtime.Object, r rest.Responder) (http.Handler, error) {
	return &sonataFlowInstantiateHandler{
		r:         w,
		responder: r,
		ctx:       ctx,
		name:      id,
		options:   options.(*workflowprojapi.SonataFlowProjInstantiateRequestOptions),
	}, nil
}

func (w *InstantiateREST) NewConnectOptions() (runtime.Object, bool, string) {
	return &workflowprojapi.SonataFlowProjInstantiateRequestOptions{}, false, ""
}

func (w *InstantiateREST) ConnectMethods() []string {
	return []string{"POST"}
}

// sonataFlowInstantiateHandler responds to upload requests
type sonataFlowInstantiateHandler struct {
	r *InstantiateREST

	responder rest.Responder
	ctx       context.Context
	name      string
	options   *workflowprojapi.SonataFlowProjInstantiateRequestOptions
}

var _ http.Handler = &sonataFlowInstantiateHandler{}

func (h *sonataFlowInstantiateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	build, err := h.handle(r.Body, r.Header)
	if err != nil {
		h.responder.Error(err)
		return
	}
	h.responder.Object(http.StatusCreated, build)
}

// TODO: handle instantiate requests
func (h *sonataFlowInstantiateHandler) handle(r io.Reader, header http.Header) (runtime.Object, error) {
	klog.V(8).Infof("Instantiate HTTP Server Handle Call, resource name '%s'", h.name)
	h.options.Name = h.name
	objectMeta, err := meta.Accessor(h.options)
	if err != nil {
		return nil, err
	}
	rest.FillObjectMetaSystemFields(objectMeta)

	ns := request.NamespaceValue(h.ctx)

	// find the proj ref
	sonataFlowProj, err := h.r.SonataFlowClient.SonataFlow.SonataFlowProjs(ns).Get(h.ctx, h.options.Name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// instantiate the new project
	sonataFlowProjInstance, err := h.r.Generator.Instantiate(h.ctx, r, header, h.options)
	if err != nil {
		return nil, err
	}

	// update the proj ref status
	sonataFlowProj.Status.ObjectsRef.Properties = sonataFlowProjInstance.Status.ObjectsRef.Properties
	sonataFlowProj.Status.ObjectsRef.Workflow = sonataFlowProjInstance.Status.ObjectsRef.Workflow
	sonataFlowProj.Status.ObjectsRef.Resources = sonataFlowProjInstance.Status.ObjectsRef.Resources

	klog.V(1).Infof("SonataFlowProj type is %s", sonataFlowProj.TypeMeta.String())

	if err = api.SetTypeToObject((*sonataflowv1alpha08.SonataFlowProj)(sonataFlowProj), h.r.Scheme); err != nil {
		return nil, err
	}
	if _, err = h.r.SonataFlowClient.SonataFlow.SonataFlowProjs(ns).UpdateStatus(h.ctx, sonataFlowProj, v1.UpdateOptions{}); err != nil {
		return nil, err
	}

	return sonataFlowProjInstance, nil
}
