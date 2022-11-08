/*
 * Copyright 2022 Red Hat, Inc. and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
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
