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

package v1alpha08

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	validator2 "github.com/serverlessworkflow/sdk-go/v2/validator"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kiegroup/kogito-serverless-operator/api/metadata"
)

// log is for logging in this package.
var logger = logf.Log.WithName("kws-webhook")

func (r *KogitoServerlessWorkflow) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// Uncomment to enable the mutating webhook.

//+kubebuilder:webhook:path=/mutate-sw-kogito-kie-org-v1alpha08-kogitoserverlessworkflow,mutating=true,failurePolicy=fail,sideEffects=None,groups=sw.kogito.kie.org,resources=kogitoserverlessworkflows,verbs=create;update,versions=v1alpha08,name=mkogitoserverlessworkflow.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &KogitoServerlessWorkflow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *KogitoServerlessWorkflow) Default() {
	logger.Info("Applying default values for ", "name", r.Name)
	// Add defaults
	if len(r.Spec.Flow.SpecVersion) == 0 {
		r.Spec.Flow.SpecVersion = metadata.SpecVersion
	}

	r.Spec.Flow.ID = r.ObjectMeta.Name
	r.Spec.Flow.Key = r.ObjectMeta.Annotations[metadata.Key]
	r.Spec.Flow.Name = r.ObjectMeta.Name
}

// change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-sw-kogito-kie-org-v1alpha08-kogitoserverlessworkflow,mutating=false,failurePolicy=fail,sideEffects=None,groups=sw.kogito.kie.org,resources=kogitoserverlessworkflows,verbs=create;update,versions=v1alpha08,name=vkogitoserverlessworkflow.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &KogitoServerlessWorkflow{}

var requiredMetadataFields = [2]string{metadata.Description, metadata.Version}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *KogitoServerlessWorkflow) ValidateCreate() (admission.Warnings, error) {
	logger.Info("validate create", "name", r.Name)
	return nil, validate(r)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *KogitoServerlessWorkflow) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	logger.Info("validate update", "name", r.Name)
	return nil, validate(r)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *KogitoServerlessWorkflow) ValidateDelete() (admission.Warnings, error) {
	logger.Info("validate delete", "name", r.Name)
	// TODO
	return nil, nil
}

func validate(r *KogitoServerlessWorkflow) error {
	// validate the required metadata
	response := "Field metadata.annotation.%s.%s not set."
	var missingAnnotations []string
	for _, field := range requiredMetadataFields {
		if len(r.Annotations[field]) == 0 {
			missingAnnotations = append(missingAnnotations, fmt.Sprintf(response, metadata.Domain, field))
		}
	}
	if len(missingAnnotations) > 0 {
		return fmt.Errorf("%+v", missingAnnotations)
	}

	logger.V(1).Info("Validating workflow", "flow", r.Spec.Flow)

	validator := validator2.GetValidator()
	if err := validator.Struct(r.Spec.Flow); err != nil {
		return err
	}

	return nil
}
