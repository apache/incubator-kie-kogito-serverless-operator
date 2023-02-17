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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_JavaProperties_Replace(t *testing.T) {
	props := NewJavaProperties("key=value").WithKeyValue("key", "new")
	assert.Equal(t, "new", props.Get("key"))

	props = NewJavaProperties("key-value=value").With("key-value=new")
	assert.Equal(t, "new", props.Get("key-value"))

	props = NewJavaProperties("key-value=value\nkey2_key2=value").With("key=new\nkey2_key2=new")
	assert.Equal(t, "new", props.Get("key"))
	assert.Equal(t, "new", props.Get("key2_key2"))
}

func Test_JavaProperties_MessyData(t *testing.T) {
	props := NewJavaProperties("\n").String()
	assert.Equal(t, "", props)

	props = NewJavaProperties("=").String()
	assert.Equal(t, "", props)

	props = NewJavaProperties("! This is a comment").String()
	assert.Equal(t, "", props)

	props = NewJavaProperties("# This is a comment").String()
	assert.Equal(t, "", props)
}

func Test_JavaProperties_Multiline(t *testing.T) {
	props := NewJavaProperties("# this is a comment\nkey.my.property=value")
	assert.Equal(t, "value", props.Get("key.my.property"))
}
