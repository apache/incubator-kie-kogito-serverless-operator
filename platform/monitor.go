package platform

import (
	"context"
	v08 "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/constants"
)

// NewMonitorAction returns an action that monitors the build platform after it's fully initialized.
func NewMonitorAction() Action {
	return &monitorAction{}
}

type monitorAction struct {
	baseAction
}

func (action *monitorAction) Name() string {
	return "monitor"
}

func (action *monitorAction) CanHandle(platform *v08.KogitoServerlessPlatform) bool {
	return platform.Status.Phase == v08.PlatformPhaseReady
}

func (action *monitorAction) Handle(ctx context.Context, platform *v08.KogitoServerlessPlatform) (*v08.KogitoServerlessPlatform, error) {
	// Just track the version of the operator in the platform resource
	if platform.Status.Version != constants.VERSION {
		platform.Status.Version = constants.VERSION
		action.L.Info("Platform version updated", "version", platform.Status.Version)
	}

	// Refresh applied configuration
	if err := ConfigureDefaults(ctx, action.client, platform, false); err != nil {
		return nil, err
	}

	return platform, nil
}
