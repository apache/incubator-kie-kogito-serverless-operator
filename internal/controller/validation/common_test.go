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
	"net"
	"testing"
)

func TestCheckUrlHasPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"docker://my-image", true},

		{"example.com", false},
		{"127.0.0.1", false},
		{"my-image", false},

		{"HTTP://example.com", true},
		{"HTTPS://example.com", true},
		{"DOCKER://my-image", true},
	}

	for _, test := range tests {
		result := checkUrlHasPrefix(test.input)
		if result != test.expected {
			t.Errorf("checkUrlHasPrefix(%q) = %v; expected %v", test.input, result, test.expected)
		}
	}
}

func TestHostAndPortFromUri(t *testing.T) {
	tests := []struct {
		input          string
		expectedHost   string
		expectedPort   string
		expectingError bool
	}{
		{"http://192.168.1.1:8080", "192.168.1.1", "8080", false},
		{"https://192.168.1.1:8080", "192.168.1.1", "8080", false},
		{"docker://192.168.1.1:8080", "192.168.1.1", "8080", false},
		{"http://localhost:5000", "127.0.0.1", "5000", false},
		{"https://localhost:5000", "127.0.0.1", "5000", false},
		{"docker://localhost:5000", "127.0.0.1", "5000", false},
		{"localhost:5000", "127.0.0.1", "5000", false},
		{"ftp://example.com", "", "", true},
		{"invalid_url", "", "", true},
	}

	for _, test := range tests {
		host, port, err := hostAndPortFromUri(test.input)

		if (err != nil) != test.expectingError {
			t.Errorf("hostAndPortFromUri(%q) error = %v, expected error = %v", test.input, err, test.expectingError)
		}
		if host != test.expectedHost || port != test.expectedPort {
			t.Errorf("hostAndPortFromUri(%q) = (%q, %q), expected (%q, %q)", test.input, host, port, test.expectedHost, test.expectedPort)
		}
	}
}

func TestImageStoredInKindRegistry(t *testing.T) {
	originalKindRegistryNameFunc := kindRegistryName
	originalResolveFunc := resolve
	defer func() {
		kindRegistryName = originalKindRegistryNameFunc
		resolve = originalResolveFunc
	}()

	tests := []struct {
		image          string
		expectedResult bool
		expectedHost   string
		resolvedIp     string
		expectingError bool
	}{
		{"kind-registry:5000/my-image", true, "172.18.0.4:5000", "172.18.0.4", false},
		{"172.18.0.4:5000/my-image", true, "172.18.0.4:5000", "0.0.0.0", false},
		{"docker.io/my-image", false, "", "1.1.1.1", false},
		{"172.18.0.4:6000/my-image", false, "", "0.0.0.0", false},
	}

	for _, test := range tests {
		kindRegistryName = func(ctx context.Context) (string, error) {
			return "172.18.0.4:5000", nil
		}
		resolve = func(host string) ([]net.IP, error) {
			return []net.IP{net.ParseIP(test.resolvedIp)}, nil
		}

		result, hostAndPort, err := imageStoredInKindRegistry(context.Background(), test.image)

		if (err != nil) != test.expectingError {
			t.Errorf("ImageStoredInKindRegistry(%q) error = %v, expected error = %v", test.image, err, test.expectingError)
		}

		if result != test.expectedResult || hostAndPort != test.expectedHost {
			t.Errorf("ImageStoredInKindRegistry(%q) = (%v, %q), expected (%v, %q)", test.image, result, hostAndPort, test.expectedResult, test.expectedHost)
		}
	}
}
