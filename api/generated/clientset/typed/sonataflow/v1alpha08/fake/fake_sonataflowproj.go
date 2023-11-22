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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha08 "github.com/apache/incubator-kie-kogito-serverless-operator/api/sonataflow/v1alpha08"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSonataFlowProjs implements SonataFlowProjInterface
type FakeSonataFlowProjs struct {
	Fake *FakeSonataflowV1alpha08
	ns   string
}

var sonataflowprojsResource = v1alpha08.SchemeGroupVersion.WithResource("sonataflowprojs")

var sonataflowprojsKind = v1alpha08.SchemeGroupVersion.WithKind("SonataFlowProj")

// Get takes name of the sonataFlowProj, and returns the corresponding sonataFlowProj object, and an error if there is any.
func (c *FakeSonataFlowProjs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha08.SonataFlowProj, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(sonataflowprojsResource, c.ns, name), &v1alpha08.SonataFlowProj{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha08.SonataFlowProj), err
}

// List takes label and field selectors, and returns the list of SonataFlowProjs that match those selectors.
func (c *FakeSonataFlowProjs) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha08.SonataFlowProjList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(sonataflowprojsResource, sonataflowprojsKind, c.ns, opts), &v1alpha08.SonataFlowProjList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha08.SonataFlowProjList{ListMeta: obj.(*v1alpha08.SonataFlowProjList).ListMeta}
	for _, item := range obj.(*v1alpha08.SonataFlowProjList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested sonataFlowProjs.
func (c *FakeSonataFlowProjs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(sonataflowprojsResource, c.ns, opts))

}

// Create takes the representation of a sonataFlowProj and creates it.  Returns the server's representation of the sonataFlowProj, and an error, if there is any.
func (c *FakeSonataFlowProjs) Create(ctx context.Context, sonataFlowProj *v1alpha08.SonataFlowProj, opts v1.CreateOptions) (result *v1alpha08.SonataFlowProj, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(sonataflowprojsResource, c.ns, sonataFlowProj), &v1alpha08.SonataFlowProj{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha08.SonataFlowProj), err
}

// Update takes the representation of a sonataFlowProj and updates it. Returns the server's representation of the sonataFlowProj, and an error, if there is any.
func (c *FakeSonataFlowProjs) Update(ctx context.Context, sonataFlowProj *v1alpha08.SonataFlowProj, opts v1.UpdateOptions) (result *v1alpha08.SonataFlowProj, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(sonataflowprojsResource, c.ns, sonataFlowProj), &v1alpha08.SonataFlowProj{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha08.SonataFlowProj), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeSonataFlowProjs) UpdateStatus(ctx context.Context, sonataFlowProj *v1alpha08.SonataFlowProj, opts v1.UpdateOptions) (*v1alpha08.SonataFlowProj, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(sonataflowprojsResource, "status", c.ns, sonataFlowProj), &v1alpha08.SonataFlowProj{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha08.SonataFlowProj), err
}

// Delete takes name of the sonataFlowProj and deletes it. Returns an error if one occurs.
func (c *FakeSonataFlowProjs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(sonataflowprojsResource, c.ns, name, opts), &v1alpha08.SonataFlowProj{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSonataFlowProjs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(sonataflowprojsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha08.SonataFlowProjList{})
	return err
}

// Patch applies the patch and returns the patched sonataFlowProj.
func (c *FakeSonataFlowProjs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha08.SonataFlowProj, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(sonataflowprojsResource, c.ns, name, pt, data, subresources...), &v1alpha08.SonataFlowProj{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha08.SonataFlowProj), err
}