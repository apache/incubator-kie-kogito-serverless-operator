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

package steps

import (
	"github.com/cucumber/godog"

	"github.com/apache/incubator-kie-kogito-serverless-operator/bddframework/pkg/framework"
	kogitoInstallers "github.com/apache/incubator-kie-kogito-serverless-operator/bddframework/pkg/installers"
	"github.com/apache/incubator-kie-kogito-serverless-operator/testbdd/installers"
)

func registerPostgresSteps(ctx *godog.ScenarioContext, data *Data) {
	ctx.Step(`^Postgres is deployed$`, data.postgresIsDeployed)
	// Unused currently
	//ctx.Step(`^Postgres deployment has (\d+) (?:pod|pods) running"$`, data.sonataFlowOperatorHasPodsRunning)
}

func (data *Data) postgresIsDeployed() (err error) {
	var installer kogitoInstallers.ServiceInstaller
	installer, err = installers.GetPostgresInstaller()

	if err != nil {
		return err
	}
	return installer.Install(data.Namespace)
}

func (data *Data) postgresDeploymentHasPodsRunning(numberOfPods int, name, phase string) error {
	return framework.WaitForPodsWithLabel(data.Namespace, "app.kubernetes.io/name", "postgres", numberOfPods, 1)
}
