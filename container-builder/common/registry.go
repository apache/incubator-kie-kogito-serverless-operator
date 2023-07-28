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
	"net"
	"net/http"
	"os"
	"time"

	"k8s.io/klog/v2"

	"github.com/docker/docker/client"
	registryContainer "github.com/heroku/docker-registry-client/registry"
	"github.com/opencontainers/go-digest"

	"github.com/kiegroup/kogito-serverless-operator/container-builder/util/log"
)

const REGISTRY_IMG = "registry"
const REGISTRY_IMG_FULL = "docker.io/library/registry"
const REGISTRY_IMG_FULL_TAG = "docker.io/library/registry:latest"
const REGISTRY_CONTAINER_URL_FROM_DOCKER_SOCKET = "tcp://localhost:5000"
const REGISTRY_CONTAINER_URL = "http://localhost:5000"
const TEST_IMG = "busybox"
const TEST_REGISTRY_REPO = "localhost:5000/"
const TEST_REPO = "docker.io/library/"
const LATEST_TAG = "latest"
const TEST_IMG_TAG = "busybox:latest"
const TEST_IMG_SECOND = "alpine"
const TEST_IMG_SECOND_TAG = "alpine:latest"
const TEST_IMG_SECOND_LOCAL_TAG = "localhost:5000/alpine:latest"
const TEST_IMG_LOCAL_TAG = "localhost:5000/busybox:latest"

type Registry interface {
	StartRegistry()
	StopRegistry()
}

type DockerLocalRegistry struct {
	Connection *client.Client
}

type RegistryContainer struct {
	Connection registryContainer.Registry
	URL        string
	Client     *http.Client
}

func (r RegistryContainer) GetRepositories() ([]string, error) {
	return r.Connection.Repositories()
}

func (r RegistryContainer) GetRepositoriesTags(repo string) ([]string, error) {
	return r.Connection.Tags(repo)
}

func (r RegistryContainer) DeleteManifest(repo string, tag string) error {
	digest, error := r.Connection.ManifestDigest(repo, tag)
	if error != nil {
		return error
	}
	return r.Connection.DeleteManifest(repo, digest)
}

func (r RegistryContainer) DeleteImageByDigest(repository string, digest digest.Digest) error {
	url := r.url("/v2/%s/manifests/%s", repository, digest)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := r.Connection.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	return nil
}

func (r RegistryContainer) DeleteImage(repository string, tag string) error {
	url := r.url("/v2/%s/manifests/%s", repository, tag)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := r.Connection.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		klog.V(log.E).ErrorS(err, "error during DeleteImage")
		return err
	}
	return nil
}

func (r *RegistryContainer) url(pathTemplate string, args ...interface{}) string {
	pathSuffix := fmt.Sprintf(pathTemplate, args...)
	url := fmt.Sprintf("%s%s", r.Connection.URL, pathSuffix)
	return url
}

func GetRegistryContainer() (RegistryContainer, error) {
	registryContainerConnection, err := GetRegistryConnection(REGISTRY_CONTAINER_URL, "", "")
	if err != nil {
		klog.V(log.E).ErrorS(err, "Can't connect to the RegistryContainer")
		return RegistryContainer{}, err
	}
	return RegistryContainer{Connection: *registryContainerConnection}, nil
}

func IsPortAvailable(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't listen on port %q: %s", port, err)
		return false
	}
	ln.Close()
	return true
}

func GetRegistryConnection(url string, username string, password string) (*registryContainer.Registry, error) {
	registryConn, err := registryContainer.New(url, username, password)
	if err != nil {
		klog.V(log.E).ErrorS(err, "First Attempt to connect with RegistryContainer")
	}
	// we try ten times if the machine is slow and the registry needs time to start
	if err != nil {
		klog.V(log.I).InfoS("Waiting for a correct ping with RegistryContainer")

		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			if registryConn == nil {
				registryConn, _ = registryContainer.New(url, username, password)
			}
			if registryConn != nil {
				if err := registryConn.Ping(); err != nil {
					continue
				}
			}
		}
	}
	return registryConn, err
}
