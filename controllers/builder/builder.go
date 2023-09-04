// Copyright 2023 Red Hat, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/controllers/platform"
	"github.com/kiegroup/kogito-serverless-operator/log"
)

type buildManagerContext struct {
	ctx          context.Context
	client       client.Client
	platform     *operatorapi.SonataFlowPlatform
	commonConfig *v1.ConfigMap
}

type BuildManager interface {
	Schedule(build *operatorapi.SonataFlowBuild) error
	Reconcile(build *operatorapi.SonataFlowBuild) error
}

func NewBuildManager(ctx context.Context, client client.Client, cliConfig *rest.Config, targetName, targetNamespace string) (BuildManager, error) {
	p, err := platform.GetActivePlatform(ctx, client, targetNamespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, err
		}
		klog.V(log.E).ErrorS(err, "Error retrieving the active platform. Workflow build cannot be performed!", "workflow", targetName)
		return nil, err
	}
	cmName := SonataPrefix + "-" + targetName + "-builder"
	cmConfig, err := GetNamespaceConfigMap(client, cmName, targetNamespace)
	if err != nil {
		klog.V(log.E).ErrorS(err, fmt.Sprintf("Failed to get %s configMap for Workflow Builder. Make sure that %s is present in the namespace.", cmName, cmName))
		return nil, err
	}
	managerContext := buildManagerContext{
		ctx:          ctx,
		client:       client,
		platform:     p,
		commonConfig: cmConfig,
	}
	switch p.Status.Cluster {
	case operatorapi.PlatformClusterOpenShift:
		return newOpenShiftBuilderManager(managerContext, cliConfig)
	case operatorapi.PlatformClusterKubernetes:
		return newContainerBuilderManager(managerContext, cliConfig), nil
	default:
		klog.V(log.I).InfoS("Impossible to check the Cluster type in the SonataFlowPlatform")
		return newContainerBuilderManager(managerContext, cliConfig), nil
	}
}

// fetchWorkflowForBuild fetches the k8s API for the workflow from the given build
func (b *buildManagerContext) fetchWorkflowForBuild(build *operatorapi.SonataFlowBuild) (workflow *operatorapi.SonataFlow, err error) {
	workflow = &operatorapi.SonataFlow{}
	if err = b.client.Get(b.ctx, client.ObjectKeyFromObject(build), workflow); err != nil {
		return nil, err
	}
	return
}
