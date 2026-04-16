package util

import (
	"testing"
)

func TestPathProtocol(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantPath     string
		wantProtocol string
	}{
		{
			name:         "plain host",
			input:        "localhost:3000",
			wantPath:     "localhost:3000",
			wantProtocol: "http",
		},
		{
			name:         "http prefix",
			input:        "http://localhost:3000",
			wantPath:     "localhost:3000",
			wantProtocol: "http",
		},
		{
			name:         "https prefix",
			input:        "https://api.example.com",
			wantPath:     "api.example.com",
			wantProtocol: "https",
		},
		{
			name:         "http with path",
			input:        "http://localhost:3000/api",
			wantPath:     "localhost:3000/api",
			wantProtocol: "http",
		},
		{
			name:         "plain domain",
			input:        "api.example.com",
			wantPath:     "api.example.com",
			wantProtocol: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotProtocol := PathProtocol(tt.input)
			if gotPath != tt.wantPath {
				t.Errorf("PathProtocol(%q) path = %q, want %q", tt.input, gotPath, tt.wantPath)
			}
			if gotProtocol != tt.wantProtocol {
				t.Errorf("PathProtocol(%q) protocol = %q, want %q", tt.input, gotProtocol, tt.wantProtocol)
			}
		})
	}
}
