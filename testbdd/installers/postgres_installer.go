// Copyright 2024 Apache Software Foundation (ASF)
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

package installers

import (
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/incubator-kie-kogito-serverless-operator/bddframework/pkg/config"
	"github.com/apache/incubator-kie-kogito-serverless-operator/bddframework/pkg/framework"
	"github.com/apache/incubator-kie-kogito-serverless-operator/bddframework/pkg/installers"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/workflowdef"
	srvframework "github.com/apache/incubator-kie-kogito-serverless-operator/testbdd/framework"
)

const defaultPostgresImage = "docker.io/postgres:13.6"

var (
	// postgresYamlClusterInstaller installs Postgres operator cluster wide using YAMLs
	postgresYamlClusterInstaller = installers.YamlNamespacedServiceInstaller{
		InstallNamespacedYaml:           installPostgresUsingYaml,
		WaitForNamespacedServiceRunning: waitForPostgresDeploymentUsingYamlRunning,
		GetAllNamespaceYamlCrs:          getPostgresCrsInNamespace,
		UninstallNamespaceYaml:          uninstallPostgresUsingYaml,
		NamespacedYamlServiceName:       postgresServiceName,
		CleanupNamespaceYamlCrs:         cleanupPostgresCrsInNamespace,
	}

	// postgresOlmClusterWideInstaller installs Postgres cluster wide using OLM with community catalog
	postgresOlmClusterWideInstaller = installers.OlmClusterWideServiceInstaller{
		SubscriptionName:                    postgresDeploymentSubscriptionName,
		Channel:                             postgresDeploymentSubscriptionChannel,
		Catalog:                             framework.GetCommunityCatalog,
		InstallationTimeoutInMinutes:        5,
		GetAllClusterWideOlmCrsInNamespace:  getPostgresCrsInNamespace,
		CleanupClusterWideOlmCrsInNamespace: cleanupPostgresCrsInNamespace,
	}

	// PostgresNamespace is the Postgres namespace for yaml cluster-wide deployment
	PostgresNamespace   = "postgres-deployment-system"
	postgresServiceName = "Postgres deployment"

	postgresDeploymentSubscriptionName    = "postgres"
	postgresDeploymentSubscriptionChannel = "alpha"
)

// GetPostgresInstaller returns Postgres installer
func GetPostgresInstaller() (installers.ServiceInstaller, error) {
	// If user doesn't pass Postgres operator image then use community OLM catalog to install operator
	if len(config.GetOperatorImageTag()) == 0 {
		framework.GetMainLogger().Info("Installing Postgres operator using community catalog.")
		return &postgresOlmClusterWideInstaller, nil
	}

	if config.IsOperatorInstalledByYaml() || config.IsOperatorProfiling() {
		framework.GetMainLogger().Info("Installing Postgres operator using YAML.")
		return &postgresYamlClusterInstaller, nil
	}

	return nil, errors.New("no Postgres operator installer available for provided configuration")
}

func installPostgresUsingYaml(namespace string) error {
	framework.GetMainLogger().Info("Installing Postgres database from /testdata/persistence/01-postgres.yaml")

	yamlContent, err := framework.ReadFromURI(config.GetPostgresYamlURI())
	if err != nil {
		framework.GetMainLogger().Error(err, "Error while reading the postgres YAML file")
		return err
	}
	/*
		No need to do the regex repalcement cos we want to use one DB for testing
		regexp, err := regexp.Compile(getDefaultPostgresImageTag())
		if err != nil {
			return err
		}
		yamlContent = regexp.ReplaceAllString(yamlContent, config.GetOperatorImageTag())
	*/

	tempFilePath, err := framework.CreateTemporaryFile("postgres*.yaml", yamlContent)
	if err != nil {
		framework.GetMainLogger().Error(err, "Error while storing adjusted YAML content to temporary file")
		return err
	}

	_, err = framework.CreateCommand("oc", "create", "-f", tempFilePath, "-n", namespace).Execute()
	if err != nil {
		framework.GetMainLogger().Error(err, "Error while installing Postgres from YAML file")
		return err
	}

	return nil
}

func waitForPostgresDeploymentUsingYamlRunning(namespace string) error {
	return srvframework.WaitForPostgresDeploymentRunning(namespace)
}

func uninstallPostgresUsingYaml(namespace string) error {
	framework.GetMainLogger().Info("Uninstalling Postgres")

	output, err := framework.CreateCommand("oc", "delete", "-f", config.GetPostgresYamlURI(), "--timeout=30s", "--ignore-not-found=true", "-n", namespace).Execute()
	if err != nil {
		framework.GetMainLogger().Error(err, fmt.Sprintf("Deleting Postgres deployment failed, output: %s", output))
		return err
	}

	return nil
}

func getPostgresCrsInNamespace(namespace string) ([]client.Object, error) {
	var crs []client.Object

	// TODO: Enhacement - make this return actuall CRs

	return crs, nil
}

func cleanupPostgresCrsInNamespace(namespace string) bool {
	crs, err := getPostgresCrsInNamespace(namespace)
	if err != nil {
		framework.GetLogger(namespace).Error(err, "Error getting Postgres CRs.")
		return false
	}

	for _, cr := range crs {
		if err := framework.DeleteObject(cr); err != nil {
			framework.GetLogger(namespace).Error(err, "Error deleting Postgres CR.", "CR name", cr.GetName())
			return false
		}
	}
	return true
}

func getDefaultPostgresImageTag() string {
	return workflowdef.GetDefaultImageTag(defaultPostgresImage)
}
