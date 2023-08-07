// Copyright 2023 Red Hat, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"strings"
)

const (
	// Current version
	OperatorVersion = "1.42.1-snapshot"

	// Should not be changed
	snapshotSuffix = "snapshot"
	latestVersion  = "2.0.0-snapshot"
)

func IsSnapshot() bool {
	return strings.HasSuffix(OperatorVersion, snapshotSuffix)
}

func IsLatestVersion() bool {
	return latestVersion == OperatorVersion
}

func GetMajorMinor() string {
	v := strings.Split(OperatorVersion, ".")
	return v[0] + "." + v[1]
}
