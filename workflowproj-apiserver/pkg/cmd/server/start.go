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

package server

import (
	"fmt"
	"io"
	"net"

	"github.com/spf13/cobra"
	"k8s.io/apiserver/pkg/util/feature"
	utilflowcontrol "k8s.io/apiserver/pkg/util/flowcontrol"
	"k8s.io/client-go/kubernetes"
	"k8s.io/component-base/featuregate"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/klog/v2"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/features"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	netutils "k8s.io/utils/net"

	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apiserver"
)

// WorkflowProjServerOptions contains state for master/api server
type WorkflowProjServerOptions struct {
	SecureServing              *genericoptions.SecureServingOptionsWithLoopback
	Authentication             *genericoptions.DelegatingAuthenticationOptions
	Authorization              *genericoptions.DelegatingAuthorizationOptions
	Audit                      *genericoptions.AuditOptions
	Features                   *genericoptions.FeatureOptions
	CoreAPI                    *genericoptions.CoreAPIOptions
	FeatureGate                featuregate.FeatureGate
	ExtraAdmissionInitializers func(c *genericapiserver.RecommendedConfig) ([]admission.PluginInitializer, error)
	Admission                  *genericoptions.AdmissionOptions
	EgressSelector             *genericoptions.EgressSelectorOptions
	Traces                     *genericoptions.TracingOptions

	StdOut io.Writer
	StdErr io.Writer

	AlternateDNS []string
}

// NewWorkflowProjServerOptions returns a new WorkflowProjServerOptions
func NewWorkflowProjServerOptions(out, errOut io.Writer) *WorkflowProjServerOptions {
	// we use the recommended options as a basis, but not entirely since we work as a proxy, and we don't need persistence
	// sending empty parameters since they are required only for etcd options.
	recommendedOptions := genericoptions.NewRecommendedOptions("", nil)
	// init our options  based on the recommended options, excluding etcd.
	o := &WorkflowProjServerOptions{
		SecureServing:              recommendedOptions.SecureServing,
		Authentication:             recommendedOptions.Authentication,
		Authorization:              recommendedOptions.Authorization,
		Audit:                      recommendedOptions.Audit,
		Features:                   recommendedOptions.Features,
		CoreAPI:                    recommendedOptions.CoreAPI,
		FeatureGate:                recommendedOptions.FeatureGate,
		ExtraAdmissionInitializers: recommendedOptions.ExtraAdmissionInitializers,
		Admission:                  recommendedOptions.Admission,
		EgressSelector:             recommendedOptions.EgressSelector,
		Traces:                     recommendedOptions.Traces,

		StdOut: out,
		StdErr: errOut,
	}
	return o
}

// NewCommandStartWorkflowProjServer provides a CLI handler for 'start master' command
// with a default WorkflowProjServerOptions.
func NewCommandStartWorkflowProjServer(defaults *WorkflowProjServerOptions, stopCh <-chan struct{}) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Short: "Launch a workflowproj API server",
		Long:  "Launch a workflowproj API server",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.RunWorkflowProjServer(stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	// TODO: add a verbose flag
	fs := cmd.Flags()
	o.SecureServing.AddFlags(fs)
	o.Authentication.AddFlags(fs)
	o.Authorization.AddFlags(fs)
	o.Audit.AddFlags(fs)
	o.Features.AddFlags(fs)
	o.CoreAPI.AddFlags(fs)
	o.Admission.AddFlags(fs)
	o.EgressSelector.AddFlags(fs)
	o.Traces.AddFlags(fs)
	utilfeature.DefaultMutableFeatureGate.AddFlag(fs)

	return cmd
}

// Validate validates WorkflowProjServerOptions
func (o *WorkflowProjServerOptions) Validate(args []string) error {
	var errors []error
	errors = append(errors, o.SecureServing.Validate()...)
	errors = append(errors, o.Authentication.Validate()...)
	errors = append(errors, o.Authorization.Validate()...)
	errors = append(errors, o.Audit.Validate()...)
	errors = append(errors, o.Features.Validate()...)
	errors = append(errors, o.CoreAPI.Validate()...)
	errors = append(errors, o.Admission.Validate()...)
	errors = append(errors, o.EgressSelector.Validate()...)
	errors = append(errors, o.Traces.Validate()...)
	return utilerrors.NewAggregate(errors)
}

// Complete fills in fields required to have valid data
func (o *WorkflowProjServerOptions) Complete() error {
	// register admission plugins

	// add admission plugins to the RecommendedPluginOrder
	//o.RecommendedOptions.Admission.RecommendedPluginOrder = append(o.RecommendedOptions.Admission.RecommendedPluginOrder, "BanFlunder")
	return nil
}

func (o *WorkflowProjServerOptions) ApplyTo(config *genericapiserver.RecommendedConfig) error {
	// copied from the RecommendedOptions ApplyTo, excluding etcd.

	if err := o.EgressSelector.ApplyTo(&config.Config); err != nil {
		return err
	}
	if err := o.Traces.ApplyTo(config.Config.EgressSelector, &config.Config); err != nil {
		return err
	}
	if err := o.SecureServing.ApplyTo(&config.Config.SecureServing, &config.Config.LoopbackClientConfig); err != nil {
		return err
	}
	if err := o.Authentication.ApplyTo(&config.Config.Authentication, config.SecureServing, config.OpenAPIConfig); err != nil {
		return err
	}
	if err := o.Authorization.ApplyTo(&config.Config.Authorization); err != nil {
		return err
	}
	if err := o.Audit.ApplyTo(&config.Config); err != nil {
		return err
	}
	if err := o.Features.ApplyTo(&config.Config); err != nil {
		return err
	}
	if err := o.CoreAPI.ApplyTo(config); err != nil {
		return err
	}
	if initializers, err := o.ExtraAdmissionInitializers(config); err != nil {
		return err
	} else if err := o.Admission.ApplyTo(&config.Config, config.SharedInformerFactory, config.ClientConfig, o.FeatureGate, initializers...); err != nil {
		return err
	}
	if feature.DefaultFeatureGate.Enabled(features.APIPriorityAndFairness) {
		if config.ClientConfig != nil {
			if config.MaxRequestsInFlight+config.MaxMutatingRequestsInFlight <= 0 {
				return fmt.Errorf("invalid configuration: MaxRequestsInFlight=%d and MaxMutatingRequestsInFlight=%d; they must add up to something positive", config.MaxRequestsInFlight, config.MaxMutatingRequestsInFlight)

			}
			config.FlowControl = utilflowcontrol.New(
				config.SharedInformerFactory,
				kubernetes.NewForConfigOrDie(config.ClientConfig).FlowcontrolV1beta3(),
				config.MaxRequestsInFlight+config.MaxMutatingRequestsInFlight,
				config.RequestTimeout/4,
			)
		} else {
			klog.Warningf("Neither kubeconfig is provided nor service-account is mounted, so APIPriorityAndFairness will be disabled")
		}
	}
	return nil
}

// Config returns config for the api server given WorkflowProjServerOptions
func (o *WorkflowProjServerOptions) Config() (*apiserver.Config, error) {
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", o.AlternateDNS, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(apiserver.Codecs)
	/* TODO: fix localreference openapi problem
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(workflowprojopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(apiserver.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "WorkflowProj"
	serverConfig.OpenAPIConfig.Info.Version = "0.1"

	if utilfeature.DefaultFeatureGate.Enabled(features.OpenAPIV3) {
		serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(workflowprojopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(apiserver.Scheme))
		serverConfig.OpenAPIV3Config.Info.Title = "WorkflowProj"
		serverConfig.OpenAPIV3Config.Info.Version = "0.1"
	}
	*/

	if err := o.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	config := &apiserver.Config{
		GenericConfig: serverConfig,
		ExtraConfig:   apiserver.ExtraConfig{},
	}
	return config, nil
}

// RunWorkflowProjServer starts a new WorkflowProjServer given WorkflowProjServerOptions
func (o *WorkflowProjServerOptions) RunWorkflowProjServer(stopCh <-chan struct{}) error {
	// Since we may use controller-runtime via kogito-serverless-operator-package, we set the controller-runtime logger
	// to output to klog to avoid failures and empty logs.
	ctrl.SetLogger(klogr.New().WithName("workflowproj-apiserver"))

	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}

	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}
