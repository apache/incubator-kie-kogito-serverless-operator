package platform

import (
	"context"
	v08 "github.com/davidesalerno/kogito-serverless-operator/api/v08"
	"github.com/pkg/errors"
	"github.com/ricardozanini/kogito-builder/api"
	"github.com/ricardozanini/kogito-builder/client"
	"github.com/ricardozanini/kogito-builder/util/log"
)

func ConfigureDefaults(ctx context.Context, c client.Client, p *v08.KogitoServerlessPlatform, verbose bool) error {
	// Reset the state to initial values
	p.ResyncStatusFullConfig()

	// update missing fields in the resource
	if p.Status.Cluster == "" {
		// determine the kind of cluster the platform is installed into
		isOpenShift, err := IsOpenShift(c)
		switch {
		case err != nil:
			return err
		case isOpenShift:
			p.Status.Cluster = v08.PlatformClusterOpenShift
		default:
			p.Status.Cluster = v08.PlatformClusterKubernetes
		}
	}

	if p.Status.BuildPlatform.BuildStrategy == "" {
		// The build output has to be shared via a volume
		p.Status.BuildPlatform.BuildStrategy = api.BuildStrategyPod
	}

	err := SetPlatformDefaults(p, verbose)
	if err != nil {
		return err
	}

	if p.Status.BuildPlatform.BuildStrategy == api.BuildStrategyPod {
		if err := CreateBuilderServiceAccount(ctx, c, p); err != nil {
			return errors.Wrap(err, "cannot ensure service account is present")
		}
	}

	err = ConfigureRegistry(ctx, c, p, verbose)
	if err != nil {
		return err
	}

	if verbose && p.Status.BuildPlatform.Timeout.Duration != 0 {
		log.Log.Infof("Maven Timeout set to %s", p.Status.BuildPlatform.Timeout.Duration)
	}

	return nil
}
