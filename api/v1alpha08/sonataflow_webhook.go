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
// limitations under the License

package v1alpha08

import (
	"fmt"

	cncfvalidator "github.com/serverlessworkflow/sdk-go/v2/validator"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kiegroup/kogito-serverless-operator/api/metadata"
	"github.com/kiegroup/kogito-serverless-operator/log"
)

func (r *SonataFlow) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// Uncomment to enable the mutating webhook.
// Uncomment also the mutating webhook in the config/default/webhookcainjection_patch.yaml file.
// //+kubebuilder:webhook:path=/mutate-sonataflow-org-v1alpha08-sonataflow,mutating=true,failurePolicy=fail,sideEffects=None,groups=sonataflow.org,resources=sonataflows,verbs=create;update,versions=v1alpha08,name=msonataflow.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &SonataFlow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (s *SonataFlow) Default() {
	klog.V(log.I).InfoS("Applying default values for ", "name", s.Name)
	// Add defaults
	if len(s.ObjectMeta.Annotations[metadata.Version]) == 0 {
		s.ObjectMeta.Annotations[metadata.Key] = metadata.SpecVersion
	}
	klog.V(log.I).InfoS("BBBBBBB Applying default values for ", "specVersion", s.ObjectMeta.Annotations[metadata.Key])
}

// change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-sonataflow-org-v1alpha08-sonataflow,mutating=false,failurePolicy=fail,sideEffects=None,groups=sonataflow.org,resources=sonataflows,verbs=create;update;delete,versions=v1alpha08,name=vsonataflow.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &SonataFlow{}

var requiredMetadataFields = [2]string{metadata.Description, metadata.Version}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *SonataFlow) ValidateCreate() (admission.Warnings, error) {
	klog.V(log.I).InfoS("validate create", "name", r.Name)
	return nil, validate(r)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *SonataFlow) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	klog.V(log.I).InfoS("validate update", "name", r.Name)
	return nil, validate(r)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *SonataFlow) ValidateDelete() (admission.Warnings, error) {
	klog.V(log.I).InfoS("validate delete", "name", r.Name)
	// TODO
	return nil, nil
}

func validate(r *SonataFlow) error {
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

	klog.V(log.I).InfoS("Validating workflow", "flow", r.Spec.Flow)

	validator := cncfvalidator.GetValidator()
	fmt.Printf("validator: %+v", r.Spec.Flow)
	if err := validator.Struct(r.Spec.Flow); err != nil {
		return err
	}

	return nil
}
