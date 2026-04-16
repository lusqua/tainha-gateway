package proxy

import (
	"testing"
)

func TestNewReverseProxy(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{
			name:    "valid http target",
			target:  "http://localhost:3000",
			wantErr: false,
		},
		{
			name:    "valid https target",
			target:  "https://api.example.com",
			wantErr: false,
		},
		{
			name:    "valid target with path",
			target:  "http://localhost:3000/api/v1",
			wantErr: false,
		},
		{
			name:    "invalid target",
			target:  "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy, err := NewReverseProxy(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReverseProxy(%q) error = %v, wantErr %v", tt.target, err, tt.wantErr)
				return
			}
			if !tt.wantErr && proxy == nil {
				t.Errorf("NewReverseProxy(%q) returned nil proxy without error", tt.target)
			}
		})
	}
}
