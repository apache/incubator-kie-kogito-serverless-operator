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

package v1alpha08

import (
	"reflect"
	"testing"

	cncfmodel "github.com/serverlessworkflow/sdk-go/v2/model"
)

func TestFromCNCFWorkflow(t *testing.T) {
	type args struct {
		cncfWorkflow *cncfmodel.Workflow
	}
	tests := []struct {
		name    string
		args    args
		want    *KogitoServerlessWorkflow
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromCNCFWorkflow(tt.args.cncfWorkflow)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromCNCFWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromCNCFWorkflow() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToCNCFWorkflow(t *testing.T) {
	type args struct {
		workflowCR *KogitoServerlessWorkflow
	}
	tests := []struct {
		name    string
		args    args
		want    *cncfmodel.Workflow
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToCNCFWorkflow(tt.args.workflowCR)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToCNCFWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToCNCFWorkflow() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sanitizeNaming(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Success", args{name: "camel-flow"}, "camel-flow"},
		{"Starting Dash", args{name: "-camel-flow"}, "camel-flow"},
		{"All caps", args{name: "CAMEL FLOW"}, "camel-flow"},
		{"Many Dashes", args{name: "--------camel-flow"}, "camel-flow"},
		{"Weird Chars", args{name: "$%#$%$#&#$%#$%#$cm"}, "cm"},
		{"Many Chars", args{name: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque posuere nec sapien ac ultricies. Mauris id quam justo. Donec pellentesque facilisis odio eu gravida. Aliquam nisl felis, tincidunt at dignissim id, malesuada eget erat. Duis tempus sapien."}, "lorem-ipsum-dolor-sit-amet--consectetur-adipiscing-elit--quisque-posuere-nec-sapien-ac-ultricies--mauris-id-quam-justo--donec-pellentesque-facilisis-odio-eu-gravida--aliquam-nisl-felis--tincidunt-at-dignissim-id--malesuada-eget-erat--duis-tempus-sapien-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeNaming(tt.args.name); got != tt.want {
				t.Errorf("sanitizeNaming() = %v, want %v", got, tt.want)
			}
		})
	}
}
