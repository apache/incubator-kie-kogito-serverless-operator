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

package common

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"

	"github.com/kiegroup/kogito-serverless-operator/container-builder/util/log"
)

func GetDockerConnection() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return cli, nil
}

func GetCustomDockerConnectionWithIP(ip string) (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation(), client.WithHost(ip))
}

func GetCustomRegistry() (DockerLocalRegistry, *client.Client, error) {
	connectionLocal, err := GetCustomDockerConnectionWithIP(REGISTRY_CONTAINER_URL_FROM_DOCKER_SOCKET)
	if err != nil {
		log.Error(err, "error during Get registry")
		return DockerLocalRegistry{}, nil, err
	}

	d := DockerLocalRegistry{Connection: connectionLocal}
	return d, connectionLocal, nil
}

func (d DockerLocalRegistry) getConnection() (*client.Client, error) {
	connectionLocal := d.Connection
	if connectionLocal == nil {
		return GetDockerConnection()
	}
	return connectionLocal, nil
}

func (d DockerLocalRegistry) StartRegistry() string {
	//wait until Podman registry shutdown in the podman tests
	for {
		time.Sleep(1 * time.Second)
		if IsPortAvailable("5000") {
			break
		}
	}

	ctx := context.Background()
	registryID := d.GetRegistryRunningID()

	if len(registryID) > 0 {
		log.Infof("Registry ID %s is already running", registryID)
		return registryID
	}

	if !d.IsRegistryImagePresent() {
		log.Info("Registry Image Pull")
		_, err := d.Connection.ImagePull(ctx, REGISTRY_IMG, types.ImagePullOptions{})
		if err != nil {
			fmt.Println(err)
			return ""
		}
	}

	time.Sleep(2 * time.Second) // needed on CI

	log.Info("Registry Container Create")
	resp, err := d.Connection.ContainerCreate(ctx, &container.Config{
		Image:        REGISTRY_IMG,
		ExposedPorts: nat.PortSet{"5000": struct{}{}},
	},
		&container.HostConfig{
			PortBindings: map[nat.Port][]nat.PortBinding{nat.Port("5000"): {{HostIP: "127.0.0.1", HostPort: "5000"}}},
		},
		nil,
		nil,
		REGISTRY_IMG)

	if err != nil {
		log.Error(err, "error during Create registry")
	}

	log.Info("Starting Registry container")
	if err := d.Connection.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Error(err, "error during Start registry")
		return ""
	}

	// give some time to start
	log.Info("Waiting 4 sec")
	time.Sleep(4 * time.Second)
	return d.GetRegistryRunningID()
}

func (d DockerLocalRegistry) StopRegistry() bool {
	registryID := d.GetRegistryRunningID()
	if len(registryID) > 0 {
		log.Info("StopRegistry Kill registry container.ID " + registryID)
		ctx := context.Background()
		_ = d.Connection.ContainerKill(ctx, registryID, "SIGKILL")
		log.Info("StopRegistry Removing container ID " + registryID)
		err := d.Connection.ContainerRemove(ctx, registryID, types.ContainerRemoveOptions{})
		if err != nil {
			log.Error(err, "error during Stop registry")
			return false
		}
	}
	return true
}

func (d DockerLocalRegistry) StopAndRemoveContainer(containerID string) bool {
	if len(containerID) > 0 {
		ctx := context.Background()
		log.Info("Docker StopAndRemoveContainer Kill registry container container.ID " + containerID)
		_ = d.Connection.ContainerKill(ctx, containerID, "SIGKILL")
		log.Info("Docker StopAndRemoveContainer Removing container ID " + containerID)
		err := d.Connection.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
		return err == nil
	}
	fmt.Println("Docker StopAndRemoveContainer Invalid ID " + containerID)
	return true
}

func (d DockerLocalRegistry) GetRegistryRunningID() string {
	containers, err := d.Connection.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		fmt.Println(err)
		return ""
	}

	for _, container := range containers {
		if container.Image == REGISTRY_IMG {
			return container.ID
		}
	}
	return ""
}

func (d DockerLocalRegistry) IsRegistryImagePresent() bool {

	imageList, err := d.Connection.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return false
	}
	for _, imagex := range imageList {
		if (len(imagex.RepoTags) > 0 && imagex.RepoTags[0] == REGISTRY_IMG) || (imagex.RepoDigests != nil && strings.HasPrefix(imagex.RepoDigests[0], REGISTRY_IMG)) {
			return true
		}
	}
	return false
}

func (d DockerLocalRegistry) IsImagePresent(name string) bool {

	imageList, err := d.Connection.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return false
	}
	for _, imagex := range imageList {
		if imagex.RepoTags[0] == name || (imagex.RepoDigests != nil && strings.HasPrefix(imagex.RepoDigests[0], name)) {
			return true
		}
	}
	return false
}

func SetupDockerSocket() (DockerLocalRegistry, string, Docker) {
	dockerSocketConn, err := GetDockerConnection()

	if err != nil {
		log.Errorf(err, "Can't get Docker socket")
		return DockerLocalRegistry{}, "", Docker{}
	}
	dockerSock := Docker{Connection: dockerSocketConn}

	if err != nil {
		log.Errorf(err, "Pull error in SetupDockerSocket %s", err)
	}
	_, err = dockerSock.PurgeContainer("", REGISTRY_IMG)
	if err != nil {
		log.Errorf(err, "Purge  error in SetupDockerSocket %s", err)
	}

	d := DockerLocalRegistry{Connection: dockerSocketConn}
	log.Info("Check if registry image is present :", d.IsRegistryImagePresent())
	if !d.IsRegistryImagePresent() {
		dockerSock.PullImage(REGISTRY_IMG_FULL_TAG)
	}
	registryID := d.GetRegistryRunningID()
	if len(registryID) == 0 {
		registryID = d.StartRegistry()
		log.Errorf(err, "Registry started")
	} else {
		log.Infof("Registry already up and running with ID %s", registryID)
	}
	return d, registryID, dockerSock

}

func DockerTearDown(dlr DockerLocalRegistry) {
	if len(dlr.GetRegistryRunningID()) > 0 {
		registryID := dlr.GetRegistryRunningID()
		dlr.StopAndRemoveContainer(registryID)
	} else {
		dlr.StopRegistry()
	}
}
