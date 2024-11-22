/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package validation

import (
	"context"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var validators = []Validator{NewImageValidator()}

type Validator interface {
	Validate(ctx context.Context, client client.Client, sonataflow *operatorapi.SonataFlow, req ctrl.Request) error
}

// Validate the SonataFlow object, right now it's only check if workflow is in deployment has image declared as that image
// is the same as the image in the SonataFlow object
func Validate(ctx context.Context, client client.Client, sonataflow *operatorapi.SonataFlow, req ctrl.Request) error {
	if sonataflow.HasContainerSpecImage() {
		for _, validator := range validators {
			if err := validator.Validate(ctx, client, sonataflow, req); err != nil {
				return err
			}
		}
	}
	return nil
}