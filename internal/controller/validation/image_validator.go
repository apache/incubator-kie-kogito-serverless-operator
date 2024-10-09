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
	"archive/tar"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/serverlessworkflow/sdk-go/v2/model"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type imageValidator struct{}

var workflowPathRegex = regexp.MustCompile(`^deployments/app/[^/]+\.sw\.(json|yaml)$`)

func (v *imageValidator) Validate(ctx context.Context, client client.Client, sonataflow *operatorapi.SonataFlow, req ctrl.Request) error {
	equals, err := validateImage(ctx, sonataflow)
	if err != nil {
		return err
	}
	if !equals {
		return fmt.Errorf("Workflow, defined in the image %s doesn't match deployment workflow", sonataflow.Spec.PodTemplate.Container.Image)
	}
	return nil
}

func NewImageValidator() Validator {
	return &imageValidator{}
}

func validateImage(ctx context.Context, sonataflow *operatorapi.SonataFlow) (bool, error) {
	isInKindRegistry, _, err := imageStoredInKindRegistry(ctx, sonataflow.Spec.PodTemplate.Container.Image)
	if err != nil {
		return false, err
	}

	var ref v1.Image
	if isInKindRegistry {
		ref, err = kindRegistryImage(sonataflow)
	} else {
		ref, err = remoteImage(sonataflow)
	}

	if err != nil {
		return false, err
	}

	reader, err := readWorkflowSpecLayer(ref)
	if err != nil {
		return false, err
	}

	workflowDockerImage, err := workflowSpecFromDockerImage(reader)
	if err != nil {
		return false, err
	}

	return cmp.Equal(workflowDockerImage, sonataflow.Spec.Flow, cmpopts.IgnoreUnexported(model.Transition{})), nil
}

func remoteImage(sonataflow *operatorapi.SonataFlow) (v1.Image, error) {
	imageRef, err := name.ParseReference(sonataflow.Spec.PodTemplate.Container.Image)
	if err != nil {
		return nil, err
	}

	ref, err := remote.Image(imageRef)
	if err != nil {
		return nil, err
	}
	return ref, nil
}

func kindRegistryImage(sonataflow *operatorapi.SonataFlow) (v1.Image, error) {
	transportOptions := []remote.Option{
		remote.WithTransport(&http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}),
	}

	imageRef, err := name.ParseReference(sonataflow.Spec.PodTemplate.Container.Image, name.Insecure)
	if err != nil {
		return nil, err
	}

	ref, err := remote.Image(imageRef, transportOptions...)
	if err != nil {
		return nil, err
	}
	return ref, nil
}

func readWorkflowSpecLayer(image v1.Image) (*tar.Reader, error) {
	layers, err := image.Layers()
	if err != nil {
		return nil, err
	}

	for i := len(layers) - 1; i >= 0; i-- {
		if reader, err := findWorkflowSpecLayer(layers[i]); err == nil && reader != nil {
			return reader, nil
		} else if err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("Workflow definition was not found in the Docker image")
}

func findWorkflowSpecLayer(layer v1.Layer) (*tar.Reader, error) {
	uncompressedLayer, err := layer.Uncompressed()
	if err != nil {
		return nil, fmt.Errorf("failed to get uncompressed layer: %v", err)
	}
	defer uncompressedLayer.Close()

	tarReader := tar.NewReader(uncompressedLayer)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("error reading tar: %v", err)
		}

		if header.Typeflag == '0' && workflowPathRegex.MatchString(header.Name) {
			return tarReader, nil
		}
	}

	return nil, nil
}

func workflowSpecFromDockerImage(reader io.Reader) (operatorapi.Flow, error) {
	data, err := io.ReadAll(reader)
	workflow := &operatorapi.Flow{}
	if err = yaml.Unmarshal(data, workflow); err != nil {
		return *workflow, err
	}
	return *workflow, nil
}
