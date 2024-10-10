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

package validation

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/apache/incubator-kie-kogito-serverless-operator/internal/controller/platform"
	"github.com/apache/incubator-kie-kogito-serverless-operator/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var kindRegistryName = func(ctx context.Context) (string, error) {
	config := corev1.ConfigMap{}
	err := utils.GetClient().Get(ctx, client.ObjectKey{Namespace: "kube-public", Name: "local-registry-hosting"}, &config)
	if err == nil {
		if data, ok := config.Data["localRegistryHosting.v1"]; ok {
			result := platform.LocalRegistryHostingV1{}
			if err := yaml.Unmarshal([]byte(data), &result); err == nil {
				return result.HostFromClusterNetwork, nil
			}
		}
	}
	return "", nil
}

func checkUrlHasPrefix(input string) bool {
	input = strings.ToLower(input)
	prefixes := []string{"http://", "https://", "docker://"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(input, prefix) {
			return true
		}
	}
	return false
}

func hostAndPortFromUri(input string) (string, string, error) {
	// we need to check if the input has a prefix, if not we add http://
	// because the url.Parse function requires a scheme
	if !checkUrlHasPrefix(input) {
		input = "http://" + input
	}

	u, err := url.Parse(input)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
	}

	host := u.Hostname()
	port := u.Port()

	// check if host is ip address
	if net.ParseIP(host) == nil {
		hosts, err := resolve(host)
		if err != nil {
			return "", "", fmt.Errorf("Failed to resolve domain: %v\n", err)
		}
		ipv4, err := getipv4(hosts)
		if err != nil {
			return "", "", fmt.Errorf("Failed to get ipv4 address: %v\n", err)
		}
		host = ipv4
	}
	return host, port, nil
}

func imageStoredInKindRegistry(ctx context.Context, image string) (bool, string, error) {
	kindRegistryHostAndPort, err := kindRegistryName(ctx)
	if err != nil {
		return false, "", fmt.Errorf("Failed to get kind registry name: %v\n", err)
	}
	kindRegistryHost, kindRegistryPort, err := hostAndPortFromUri(kindRegistryHostAndPort)
	if err != nil {
		return false, "", fmt.Errorf("Failed to get kind registry host and port: %v\n", err)
	}

	imageHost, imagePort, err := hostAndPortFromUri(image)
	if err != nil {
		return false, "", fmt.Errorf("Failed to get image host and port: %v\n", err)
	}
	if imageHost == kindRegistryHost && imagePort == kindRegistryPort {
		return true, kindRegistryHostAndPort, nil
	}
	return false, "", nil
}

func getipv4(ips []net.IP) (string, error) {
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("No ipv4 address found")
}

var resolve = func(host string) ([]net.IP, error) {
	return net.LookupIP(host)
}
