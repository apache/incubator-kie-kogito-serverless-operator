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

package v1alpha08

import (
	"errors"
	"path"
	"regexp"
	"strings"

	cncfmodel "github.com/serverlessworkflow/sdk-go/v2/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/kiegroup/kogito-serverless-operator/api/metadata"
)

var namingRegexp = regexp.MustCompile("^[a-z0-9](-?[a-z0-9])*$")
var allowedCharsRegexp = regexp.MustCompile("[^-a-z0-9]")
var startingDash = regexp.MustCompile("^-+")

const (
	// see https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
	dash      = "-"
	charLimit = 253
)

// FromCNCFWorkflow converts the given CNCF Serverless Workflow instance in a new KogitoServerlessWorkflow Custom Resource.
func FromCNCFWorkflow(cncfWorkflow *cncfmodel.Workflow) (*KogitoServerlessWorkflow, error) {
	if cncfWorkflow == nil {
		return nil, errors.New("CNCF Workflow is nil")
	}
	workflowCR := &KogitoServerlessWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: extractName(cncfWorkflow),
			Annotations: map[string]string{
				metadata.ExpressionLang: string(cncfWorkflow.ExpressionLang),
				metadata.Version:        cncfWorkflow.Version,
				metadata.Description:    cncfWorkflow.Description,
			},
		},
	}
	workflowBytes, err := yaml.Marshal(cncfWorkflow)
	if err != nil {
		return nil, err
	}
	workflowCRFlow := &Flow{}
	if err = yaml.Unmarshal(workflowBytes, workflowCRFlow); err != nil {
		return nil, err
	}
	workflowCR.Spec.Flow = *workflowCRFlow

	return workflowCR, nil
}

// ToCNCFWorkflow converts a KogitoServerlessWorkflow object to a Workflow one in order to be able to convert it to a YAML/Json
func ToCNCFWorkflow(workflowCR *KogitoServerlessWorkflow) (*cncfmodel.Workflow, error) {
	if workflowCR == nil {
		return nil, errors.New("kogitoServerlessWorkflow is nil")
	}
	cncfWorkflow := &cncfmodel.Workflow{}

	workflowBytes, err := yaml.Marshal(workflowCR.Spec.Flow)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(workflowBytes, cncfWorkflow); err != nil {
		return nil, err
	}

	cncfWorkflow.ID = workflowCR.ObjectMeta.Name
	cncfWorkflow.Key = workflowCR.ObjectMeta.Annotations[metadata.Key]
	cncfWorkflow.Name = workflowCR.ObjectMeta.Name
	cncfWorkflow.Description = workflowCR.ObjectMeta.Annotations[metadata.Description]
	cncfWorkflow.Version = workflowCR.ObjectMeta.Annotations[metadata.Version]
	cncfWorkflow.SpecVersion = extractSchemaVersion(workflowCR.APIVersion)
	cncfWorkflow.ExpressionLang = cncfmodel.ExpressionLangType(extractExpressionLang(workflowCR.ObjectMeta.Annotations))

	return cncfWorkflow, nil
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

func extractName(workflow *cncfmodel.Workflow) string {
	if len(workflow.ID) > 0 {
		return sanitizeNaming(workflow.ID)
	}
	if len(workflow.Key) > 0 {
		return sanitizeNaming(workflow.Key)
	}
	if len(workflow.Name) > 0 {
		return sanitizeNaming(workflow.Name)
	}
	return ""
}

func sanitizeNaming(name string) string {
	if len(name) == 0 || namingRegexp.MatchString(name) {
		return name
	}
	sanitized := startingDash.ReplaceAllString(allowedCharsRegexp.ReplaceAllString(strings.TrimSpace(strings.ToLower(name)), dash), "")
	if len(sanitized) > charLimit {
		return sanitized[:charLimit]
	}
	return sanitized
}
