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

package apiserver

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes"
	clientrest "k8s.io/client-go/rest"

	sonataflowclient "github.com/apache/incubator-kie-kogito-serverless-operator/api/generated/clientset"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj/install"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/generator"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/registry/projinstance"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/registry/projinstantiate"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()
	// Codecs provides methods for retrieving codecs and serializers for specific
	// versions and content types.
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	install.Install(Scheme)

	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

// ExtraConfig holds custom apiserver config
type ExtraConfig struct {
	KubeAPIServerClientConfig *clientrest.Config
}

// Config defines the config for the apiserver
type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// WorkflowProjServer contains state for a Kubernetes cluster master/api server.
type WorkflowProjServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	cfg.ExtraConfig.KubeAPIServerClientConfig = cfg.GenericConfig.ClientConfig
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}
	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of WorkflowProjServer from the given config.
func (c completedConfig) New() (*WorkflowProjServer, error) {
	genericServer, err := c.GenericConfig.New("workflowproj.sonataflow.org-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := &WorkflowProjServer{
		GenericAPIServer: genericServer,
	}

	sonataFlowClient, err := sonataflowclient.NewForConfig(c.ExtraConfig.KubeAPIServerClientConfig)
	if err != nil {
		return nil, err
	}
	kubeletClient, err := kubernetes.NewForConfig(c.ExtraConfig.KubeAPIServerClientConfig)
	if err != nil {
		return nil, err
	}
	clientGenerator := &generator.Client{
		SonataFlow: sonataFlowClient.SonataflowV1alpha08(),
		Kubelet:    kubeletClient,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(workflowproj.GroupName, Scheme, metav1.ParameterCodec, Codecs)
	v1alpha08storage := map[string]rest.Storage{}
	v1alpha08storage["sonataflowprojinstances"] = projinstance.NewStorage(sonataFlowClient.SonataflowV1alpha08())
	v1alpha08storage["sonataflowprojinstances/instantiate"] = projinstantiate.NewStorage(clientGenerator, Scheme)
	apiGroupInfo.VersionedResourcesStorageMap["v1alpha08"] = v1alpha08storage

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return s, nil
}
