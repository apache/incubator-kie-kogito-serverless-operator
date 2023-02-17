// Copyright 2023 Red Hat, Inc. and/or its affiliates
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

package utils

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	javaPropertyValidKeyRegExp = "^[a-zA-Z$_][a-zA-Z0-9$_\\.\\-]*$"
)

var _ JavaProperties = &javaProperties{}
var validKeyRegExp = regexp.MustCompile(javaPropertyValidKeyRegExp)

// JavaProperties is the entry point to create and manage a new Java Properties reference.
// The order you call the methods overrides the property key.
// For example, NewJavaProperties("").WithKeyValue("key", "value").WithKeyValue("key", "new") it will produce a property file with key=new.
//
// See: https://en.wikipedia.org/wiki/.properties
type JavaProperties interface {
	WithKeyValueInt(key string, value int) JavaProperties
	WithKeyValue(key, value string) JavaProperties
	With(properties string) JavaProperties
	Get(key string) string
	Map() map[string]string
	String() string
}

func NewJavaProperties(initialProperties string) JavaProperties {
	return &javaProperties{
		properties: fromStringToJavaProperty(initialProperties),
	}
}

func fromStringToJavaProperty(properties string) map[string]string {
	splitProps := strings.Split(properties, "\n")
	propMap := make(map[string]string, len(splitProps))

	for _, kv := range splitProps {
		if len(kv) == 0 {
			continue
		}
		prop := strings.Split(kv, "=")
		if len(prop) == 0 {
			propMap[kv] = ""
		}

		if match := validKeyRegExp.MatchString(prop[0]); !match {
			continue
		}

		if len(prop) == 1 {
			propMap[strings.TrimSpace(prop[0])] = ""
		} else {
			propMap[strings.TrimSpace(prop[0])] = strings.TrimSpace(prop[1])
		}
	}
	return propMap
}

type javaProperties struct {
	properties     map[string]string
	validKeyRegExp *regexp.Regexp
}

func (j javaProperties) Get(key string) string {
	return j.properties[key]
}

func (j javaProperties) Map() map[string]string {
	return j.properties
}

func (j javaProperties) With(properties string) JavaProperties {
	replaceWith := fromStringToJavaProperty(properties)
	for k, v := range replaceWith {
		j.properties[k] = v
	}
	return j
}

func (j javaProperties) WithKeyValueInt(key string, value int) JavaProperties {
	j.properties[key] = strconv.Itoa(value)
	return j
}

func (j javaProperties) WithKeyValue(key, value string) JavaProperties {
	j.properties[key] = value
	return j
}

func (j javaProperties) String() string {
	var sb = strings.Builder{}
	for k, v := range j.properties {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
		sb.WriteString("\n")
	}
	return sb.String()
}
