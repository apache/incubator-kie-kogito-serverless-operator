package utils

import (
	"fmt"
	v08 "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/constants"
	"github.com/pkg/errors"
)

func GetCustomConfig(platfrom v08.KogitoServerlessPlatform) (map[string]string, error) {
	customConfig := make(map[string]string)
	if platfrom.Namespace == "" {
		return nil, errors.New(fmt.Sprintf("Unable to retrieve the namespace from platform %s", platfrom.Name))
	}
	customConfig[constants.CUSTOM_NS_KEY] = platfrom.Namespace
	if platfrom.Spec.BuildPlatform.Registry.Secret == "" {
		return nil, errors.New(fmt.Sprintf("Unable to retrieve the registry credentials from platform %s", platfrom.Name))
	}
	customConfig[constants.CUSTOM_REG_CRED_KEY] = platfrom.Spec.BuildPlatform.Registry.Secret
	if platfrom.Spec.BuildPlatform.Registry.Address == "" {
		return nil, errors.New(fmt.Sprintf("Unable to retrieve the registry address from platform %s", platfrom.Name))
	}
	customConfig[constants.CUSTOM_REG_ADDRESS_KEY] = platfrom.Spec.BuildPlatform.Registry.Address
	return customConfig, nil
}
