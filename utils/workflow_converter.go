// Copyright 2022 Red Hat, Inc. and/or its affiliates
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

package utils

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/go-logr/logr"
	"github.com/serverlessworkflow/sdk-go/v2/model"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kiegroup/kogito-serverless-operator/api/metadata"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

var logger logr.Logger

// ToCNCFWorkflow converts a KogitoServerlessWorkflow object to a model.Workflow one in order to be able to convert it to a YAML/Json
func ToCNCFWorkflow(ctx context.Context, serverlessWorkflow *operatorapi.KogitoServerlessWorkflow) (*model.Workflow, error) {
	if serverlessWorkflow != nil {
		logger = ctrllog.FromContext(ctx)
		newBaseWorkflow := &model.BaseWorkflow{ID: serverlessWorkflow.ObjectMeta.Name,
			Key:            serverlessWorkflow.ObjectMeta.Annotations[metadata.Key],
			Name:           serverlessWorkflow.ObjectMeta.Name,
			Description:    serverlessWorkflow.ObjectMeta.Annotations[metadata.Description],
			Version:        serverlessWorkflow.ObjectMeta.Annotations[metadata.Version],
			SpecVersion:    extractSchemaVersion(serverlessWorkflow.APIVersion),
			ExpressionLang: model.ExpressionLangType(extractExpressionLang(serverlessWorkflow.ObjectMeta.Annotations)),
			KeepActive:     serverlessWorkflow.Spec.BaseWorkflow.KeepActive,
			AutoRetries:    serverlessWorkflow.Spec.BaseWorkflow.AutoRetries,
			Start:          serverlessWorkflow.Spec.BaseWorkflow.Start,
		}
		logger.V(DebugV).Info("Created new Base Workflow with name", "name", newBaseWorkflow.Name)
		newWorkflow := &model.Workflow{
			BaseWorkflow: *newBaseWorkflow,
			Functions:    serverlessWorkflow.Spec.Functions,
			States:       serverlessWorkflow.Spec.States}
		return newWorkflow, nil
	}
	return nil, errors.New("kogitoServerlessWorkflow is nil")
}

func extractExpressionLang(annotations map[string]string) string {
	expressionLang := annotations[metadata.ExpressionLang]
	if expressionLang != "" {
		return expressionLang
	}
	return metadata.DefaultExpressionLang
}

// Function to extract from the apiVersion the ServerlessWorkflow schema version
// For example given sw.kogito.kie.org/operatorapi we would like to extract v0.8
func extractSchemaVersion(version string) string {
	schemaVersion := path.Base(version)
	strings.Replace(schemaVersion, "v0", "v0.", 1)
	return schemaVersion
}
