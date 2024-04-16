package metadata

import (
	"testing"
)

func TestGetProfile(t *testing.T) {
	type args struct {
		annotation map[string]string
	}
	tests := []struct {
		name string
		args args
		want ProfileType
	}{
		{"Empty Annotations", args{annotation: nil}, PreviewProfile},
		{"Non-existent Profile", args{annotation: map[string]string{Profile: "IDontExist"}}, PreviewProfile},
		{"Regular Annotation", args{annotation: map[string]string{Profile: PreviewProfile.String()}}, PreviewProfile},
		{"Deprecated Annotation", args{annotation: map[string]string{Profile: ProdProfile.String()}}, PreviewProfile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetProfileOrDefault(tt.args.annotation); got != tt.want {
				t.Errorf("GetProfileOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
