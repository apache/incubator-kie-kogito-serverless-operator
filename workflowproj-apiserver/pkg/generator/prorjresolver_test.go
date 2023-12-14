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

package generator

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
)

func TestNewProjResolver(t *testing.T) {
	resolver, err := NewProjResolver(newHeaderOctetStream())
	assert.NoError(t, err)
	assert.IsType(t, &zipProjResolver{}, resolver)
}

func TestNewProjResolver_UnsupportedType(t *testing.T) {
	h := make(http.Header)
	h.Add("Content-Type", "image/bmp")
	resolver, err := NewProjResolver(h)
	assert.True(t, errors.IsBadRequest(err))
	assert.Nil(t, resolver)
}

func TestNewProjResolver_Empty(t *testing.T) {
	h := make(http.Header)
	resolver, err := NewProjResolver(h)
	assert.True(t, errors.IsBadRequest(err))
	assert.Nil(t, resolver)
}
