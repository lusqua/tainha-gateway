package util

import (
	"reflect"
	"testing"
)

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "no params",
			path: "/posts",
			want: []string{},
		},
		{
			name: "single param",
			path: "/users/{userId}",
			want: []string{"userId"},
		},
		{
			name: "multiple params",
			path: "/companies/{companyId}/users/{userId}",
			want: []string{"companyId", "userId"},
		},
		{
			name: "query string param",
			path: "/comments?postId={id}",
			want: []string{"id"},
		},
		{
			name: "empty path",
			path: "",
			want: []string{},
		},
		{
			name: "param at root",
			path: "/{id}",
			want: []string{"id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPathParams(tt.path)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractPathParams(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
