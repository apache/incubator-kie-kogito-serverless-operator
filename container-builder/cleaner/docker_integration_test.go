//go:build integration_docker

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

package cleaner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/kiegroup/kogito-serverless-operator/container-builder/common"
	"github.com/kiegroup/kogito-serverless-operator/container-builder/util/log"
)

func TestDockerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DockerTestSuite))
}

func (suite *DockerTestSuite) TestImagesOperationsOnDockerRegistryForTest() {
	registryContainer, err := common.GetRegistryContainer()
	assert.NotNil(suite.T(), registryContainer)
	assert.Nil(suite.T(), err)
	repos, err := registryContainer.GetRepositories()
	initialSize := len(repos)
	assert.Nil(suite.T(), err)

	pullErr := suite.Docker.PullImage(common.TEST_IMG + ":" + common.LATEST_TAG)
	if pullErr != nil {
		log.Infof("Pull Error:%s", pullErr)
	}
	assert.Nil(suite.T(), pullErr, "Pull image failed")
	time.Sleep(2 * time.Second) // Needed on CI
	assert.True(suite.T(), suite.LocalRegistry.IsImagePresent(common.TEST_IMG), "Test image not found in the registry after the pull")
	tagErr := suite.Docker.TagImage(common.TEST_IMG, common.TEST_IMG_LOCAL_TAG)
	if tagErr != nil {
		log.Infof("Tag Error:%s", tagErr)
	}

	assert.Nil(suite.T(), tagErr, "Tag image failed")
	time.Sleep(2 * time.Second) // Needed on CI
	pushErr := suite.Docker.PushImage(common.TEST_IMG_LOCAL_TAG, common.REGISTRY_CONTAINER_URL_FROM_DOCKER_SOCKET, "", "")
	if pushErr != nil {
		log.Infof("Push Error:%s", pushErr)
	}

	assert.Nil(suite.T(), pushErr, "Push image in the Docker container failed")
	//give the time to update the registry status
	time.Sleep(2 * time.Second)
	repos, err = registryContainer.GetRepositories()
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), repos)
	assert.True(suite.T(), len(repos) == initialSize+1)

	digest, erroDIgest := registryContainer.Connection.ManifestDigest(common.TEST_IMG, common.LATEST_TAG)
	assert.Nil(suite.T(), erroDIgest)
	assert.NotNil(suite.T(), digest)
	assert.NotNil(suite.T(), registryContainer.DeleteImage(common.TEST_IMG, common.LATEST_TAG), "Delete Image not allowed")
}
