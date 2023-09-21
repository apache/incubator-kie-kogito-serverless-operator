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

package dev

import (
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kiegroup/kogito-serverless-operator/api/metadata"
	"github.com/kiegroup/kogito-serverless-operator/controllers/profiles"
	"github.com/kiegroup/kogito-serverless-operator/controllers/profiles/common"
	"github.com/kiegroup/kogito-serverless-operator/log"
	"github.com/kiegroup/kogito-serverless-operator/utils"
)

var _ profiles.ProfileReconciler = &developmentProfile{}

type developmentProfile struct {
	common.BaseReconciler
}

func (d developmentProfile) GetProfile() metadata.ProfileType {
	return metadata.DevProfile
}

func NewProfileReconciler(client client.Client) profiles.ProfileReconciler {
	support := &common.StateSupport{
		C: client,
	}

	var ensurers *objectEnsurers
	var enrichers *statusEnrichers
	if utils.IsOpenShift() {
		ensurers = newObjectEnsurersOpenShift(support)
		enrichers = newStatusEnrichersOpenShift(support)
	} else {
		ensurers = newObjectEnsurers(support)
		enrichers = newStatusEnrichers(support)
	}

	stateMachine := common.NewReconciliationStateMachine(
		&ensureRunningWorkflowState{StateSupport: support, ensurers: ensurers},
		&followWorkflowDeploymentState{StateSupport: support, enrichers: enrichers},
		&recoverFromFailureState{StateSupport: support})

	profile := &developmentProfile{
		BaseReconciler: common.NewBaseProfileReconciler(support, stateMachine),
	}

	klog.V(log.I).InfoS("Reconciling in", "profile", profile.GetProfile())
	return profile
}

func newObjectEnsurers(support *common.StateSupport) *objectEnsurers {
	return &objectEnsurers{
		deployment:          common.NewObjectEnsurer(support.C, deploymentCreator),
		service:             common.NewObjectEnsurer(support.C, serviceCreator),
		network:             common.NewNoopObjectEnsurer(),
		definitionConfigMap: common.NewObjectEnsurer(support.C, workflowDefConfigMapCreator),
		propertiesConfigMap: common.NewObjectEnsurer(support.C, common.WorkflowPropsConfigMapCreator),
	}
}

func newObjectEnsurersOpenShift(support *common.StateSupport) *objectEnsurers {
	return &objectEnsurers{
		deployment:          common.NewObjectEnsurer(support.C, deploymentCreator),
		service:             common.NewObjectEnsurer(support.C, serviceCreator),
		network:             common.NewObjectEnsurer(support.C, common.OpenShiftRouteCreator),
		definitionConfigMap: common.NewObjectEnsurer(support.C, workflowDefConfigMapCreator),
		propertiesConfigMap: common.NewObjectEnsurer(support.C, common.WorkflowPropsConfigMapCreator),
	}
}

func newStatusEnrichers(support *common.StateSupport) *statusEnrichers {
	return &statusEnrichers{
		networkInfo: common.NewStatusEnricher(support.C, statusEnricher),
	}
}

func newStatusEnrichersOpenShift(support *common.StateSupport) *statusEnrichers {
	return &statusEnrichers{
		networkInfo: common.NewStatusEnricher(support.C, statusEnricherOpenShift),
	}
}

type objectEnsurers struct {
	deployment          common.ObjectEnsurer
	service             common.ObjectEnsurer
	network             common.ObjectEnsurer
	definitionConfigMap common.ObjectEnsurer
	propertiesConfigMap common.ObjectEnsurer
}

type statusEnrichers struct {
	networkInfo *common.StatusEnricher
	//Here we can add more enrichers if we need in future to enrich objects with more info coming from reconciliation
}
