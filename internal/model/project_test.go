package model

import (
	"testing"
)

func TestValidateProject(t *testing.T) {
	test := []struct {
		name    string
		project Project
		wantErr bool
	}{
		{name: "valid project", project: Project{Name: "Alpha"}, wantErr: false},
		{name: "empty name", project: Project{Name: ""}, wantErr: true},
	}
	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProject(tt.project)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}
