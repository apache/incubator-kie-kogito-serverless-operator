package platform

import (
	"context"
	v08 "github.com/davidesalerno/kogito-serverless-operator/api/v08"
	"github.com/ricardozanini/kogito-builder/client"
	"github.com/ricardozanini/kogito-builder/util/log"
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
