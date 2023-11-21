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

package install

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	sonataflowapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"

	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj"
	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj/v1alpha08"
)

// Install registers the API group and adds types to a scheme
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(workflowproj.AddToScheme(scheme))
	utilruntime.Must(v1alpha08.AddToScheme(scheme))
	utilruntime.Must(sonataflowapi.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1alpha08.SchemeGroupVersion))
}
