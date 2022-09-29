package platform

import (
	"context"
	"github.com/kiegroup/container-builder/client"
	"github.com/kiegroup/container-builder/util/log"
	v08 "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

// Action --.
type Action interface {
	client.Injectable
	log.Injectable

	// a user friendly name for the action
	Name() string

	// returns true if the action can handle the platform
	CanHandle(platform *v08.KogitoServerlessPlatform) bool

	// executes the handling function
	Handle(ctx context.Context, platform *v08.KogitoServerlessPlatform) (*v08.KogitoServerlessPlatform, error)
}

type baseAction struct {
	client client.Client
	L      log.Logger
}

func (action *baseAction) InjectClient(client client.Client) {
	action.client = client
}

func (action *baseAction) InjectLogger(log log.Logger) {
	action.L = log
}
