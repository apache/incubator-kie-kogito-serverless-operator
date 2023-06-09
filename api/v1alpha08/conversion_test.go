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
