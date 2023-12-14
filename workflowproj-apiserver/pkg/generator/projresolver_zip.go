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
	"archive/zip"
	"bytes"
	"context"
	"io"
	"path"
	"strings"

	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"

	"github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj"
	workflowprojapi "github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis/workflowproj"
)

var _ ProjResolver = &zipProjResolver{}

func newZipProjResolver() ProjResolver {
	return &zipProjResolver{}
}

type zipProjResolver struct {
}

func (z zipProjResolver) Resolve(ctx context.Context, r io.Reader, opts *workflowprojapi.SonataFlowProjInstantiateRequestOptions) (*workflowproj.WorkflowProject, error) {
	ns := request.NamespaceValue(ctx)
	workflowProjHandler := workflowproj.New(ns).Named(opts.Name)

	buffer := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buffer, r)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), size)
	if err != nil {
		return nil, err
	}
	klog.V(4).Infof("Processing zip file with contents %s", zipReader.File)

	for _, file := range zipReader.File {
		fileName := file.Name
		klog.V(5).Infof("Processing file '%s' within zip package", fileName)
		// skip dirs
		if strings.HasSuffix(fileName, "/") {
			continue
		}
		fileReader, err := toBufferedReader(file)
		if err != nil {
			return nil, err
		}

		if opts.MainWorkflowFile == fileName {
			klog.V(6).Infof("Adding workflow '%s'", fileName)
			workflowProjHandler.WithWorkflow(fileReader)
			continue
		} else if len(opts.MainWorkflowFile) == 0 && strings.Contains(fileName, workflowFileSubstr) {
			klog.V(6).Infof("Adding workflow '%s'", fileName)
			workflowProjHandler.WithWorkflow(fileReader)
			continue
		}

		if applicationPropertiesFileName == fileName {
			klog.V(6).Infof("Adding properties '%s'", fileName)
			workflowProjHandler.WithAppProperties(fileReader)
			continue
		}

		pathName := path.Dir(fileName)
		if pathName == "." || len(pathName) == 0 {
			klog.V(6).Infof("Adding resource '%s'", fileName)
			workflowProjHandler.AddResource(fileName, fileReader)
		} else {
			klog.V(6).Infof("Adding resource '%s' at path '%s'", fileName, pathName)
			workflowProjHandler.AddResourceAt(path.Base(fileName), pathName, fileReader)
		}
	}

	return workflowProjHandler.AsObjects()
}

func toBufferedReader(file *zip.File) (io.Reader, error) {
	fileReader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer fileReader.Close()
	b, err := io.ReadAll(fileReader)
	if err != nil {
		return nil, err
	}
	return strings.NewReader(string(b)), nil
}
