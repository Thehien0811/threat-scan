package service

import (
	"testing"
)

func TestValidateFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		maxSize int64
		wantErr bool
	}{
		{
			name:    "nonexistent file",
			path:    "/nonexistent/file",
			maxSize: 1000,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFile(tt.path, tt.maxSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
