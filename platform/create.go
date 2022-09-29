package platform

import (
	"context"
	v08 "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

// NewCreateAction returns a action that creates resources needed by the platform.
func NewCreateAction() Action {
	return &createAction{}
}

type createAction struct {
	baseAction
}

func (action *createAction) Name() string {
	return "create"
}

func (action *createAction) CanHandle(platform *v08.KogitoServerlessPlatform) bool {
	return platform.Status.Phase == v08.PlatformPhaseCreating
}

func (action *createAction) Handle(ctx context.Context, platform *v08.KogitoServerlessPlatform) (*v08.KogitoServerlessPlatform, error) {
	//TODO: Perform the actions needed for the Platform creation
	platform.Status.Phase = v08.PlatformPhaseReady

	return platform, nil
}
