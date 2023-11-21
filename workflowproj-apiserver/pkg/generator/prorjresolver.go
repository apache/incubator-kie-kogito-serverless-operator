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
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj"
	workflowprojapi "github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj"
)

type ContentType string

// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types
const (
	contentTypeZip         ContentType = "application/zip"
	contentTypeOctetStream ContentType = "application/octet-stream"
)

var supportedHandlers = map[ContentType]ProjResolverBuilder{
	// zip is the default handler
	contentTypeOctetStream: newZipProjResolver,
	contentTypeZip:         newZipProjResolver,
}

var supportedContentTypes = []ContentType{contentTypeOctetStream, contentTypeZip}

type ProjResolverBuilder func() ProjResolver

// GetSupportedContentTypes a list of supported content types by the resolvers.
func GetSupportedContentTypes() []ContentType {
	return supportedContentTypes
}

// NewProjResolver creates a new ProjResolver based on the content type of the given HTTP header.
// See GetSupportedContentTypes for a list of supported content types
func NewProjResolver(h http.Header) (ProjResolver, error) {
	contentType := h.Get("Content-Type")
	if len(contentType) == 0 {
		return nil, errors.NewBadRequest("Request has an empty Content-Type")
	}
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	if _, ok := supportedHandlers[ContentType(media)]; !ok {
		return nil, errors.NewBadRequest(fmt.Sprintf("Content-Type %s not supported. Supported formats are %s", media, GetSupportedContentTypes()))
	}
	return supportedHandlers[ContentType(media)](), nil
}

type ProjResolver interface {
	Resolve(ctx context.Context, reader io.Reader, opts *workflowprojapi.SonataFlowProjInstantiateRequestOptions) (*workflowproj.WorkflowProject, error)
}
