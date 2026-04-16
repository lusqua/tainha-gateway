package mapper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/aguiar-sh/tainha/internal/config"
)

func TestMap(t *testing.T) {
	t.Run("array response with mapping", func(t *testing.T) {
		// Mock backend that returns comments for a postId
		mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			postId := r.URL.Query().Get("postId")
			comments := []map[string]interface{}{
				{"id": "1", "text": "comment for post " + postId, "postId": postId},
			}
			json.NewEncoder(w).Encode(comments)
		}))
		defer mockService.Close()

		route := config.Route{
			Mapping: []config.RouteMapping{
				{
					Path:             "/comments?postId={id}",
					Service:          mockService.URL,
					Tag:              "comments",
					RemoveKeyMapping: false,
				},
			},
		}

		input := `[{"id":"1","title":"Post 1"},{"id":"2","title":"Post 2"}]`
		result, err := Map(route, []byte(input))
		if err != nil {
			t.Fatalf("Map() error = %v", err)
		}

		var parsed []map[string]interface{}
		if err := json.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if len(parsed) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(parsed))
		}

		for _, item := range parsed {
			if _, ok := item["comments"]; !ok {
				t.Errorf("Expected 'comments' key in mapped response, got: %v", item)
			}
			// id should still be present since removeKeyMapping is false
			if _, ok := item["id"]; !ok {
				t.Error("Expected 'id' key to remain when removeKeyMapping is false")
			}
		}
	})

	t.Run("array response with removeKeyMapping", func(t *testing.T) {
		mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"name": "Mapped Data"})
		}))
		defer mockService.Close()

		route := config.Route{
			Mapping: []config.RouteMapping{
				{
					Path:             "/details/{categoryId}",
					Service:          mockService.URL,
					Tag:              "category",
					RemoveKeyMapping: true,
				},
			},
		}

		input := `[{"id":"1","categoryId":"cat-1"}]`
		result, err := Map(route, []byte(input))
		if err != nil {
			t.Fatalf("Map() error = %v", err)
		}

		var parsed []map[string]interface{}
		json.Unmarshal(result, &parsed)

		if _, ok := parsed[0]["categoryId"]; ok {
			t.Error("Expected 'categoryId' to be removed when removeKeyMapping is true")
		}
		if _, ok := parsed[0]["category"]; !ok {
			t.Error("Expected 'category' tag to be present in mapped response")
		}
	})

	t.Run("single object response", func(t *testing.T) {
		mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"name": "Company A"})
		}))
		defer mockService.Close()

		route := config.Route{
			Mapping: []config.RouteMapping{
				{
					Path:             "/companies?id={companyId}",
					Service:          mockService.URL,
					Tag:              "company",
					RemoveKeyMapping: true,
				},
			},
		}

		input := `{"id":"1","name":"John","companyId":"10"}`
		result, err := Map(route, []byte(input))
		if err != nil {
			t.Fatalf("Map() error = %v", err)
		}

		var parsed map[string]interface{}
		json.Unmarshal(result, &parsed)

		if _, ok := parsed["company"]; !ok {
			t.Error("Expected 'company' tag in single object response")
		}
		if _, ok := parsed["companyId"]; ok {
			t.Error("Expected 'companyId' to be removed")
		}
	})

	t.Run("no mappings", func(t *testing.T) {
		route := config.Route{Mapping: nil}
		input := `[{"id":"1","name":"test"}]`

		result, err := Map(route, []byte(input))
		if err != nil {
			t.Fatalf("Map() error = %v", err)
		}

		var parsed []map[string]interface{}
		json.Unmarshal(result, &parsed)

		if len(parsed) != 1 || parsed[0]["name"] != "test" {
			t.Errorf("Expected unchanged data, got %s", result)
		}
	})

	t.Run("invalid json input", func(t *testing.T) {
		route := config.Route{}
		_, err := Map(route, []byte("not json"))
		if err == nil {
			t.Error("Expected error for invalid JSON input")
		}
	})

	t.Run("mapping service unavailable", func(t *testing.T) {
		route := config.Route{
			Mapping: []config.RouteMapping{
				{
					Path:    "/data/{id}",
					Service: "http://localhost:1", // unreachable
					Tag:     "data",
				},
			},
		}

		input := `[{"id":"1","name":"test"}]`
		result, err := Map(route, []byte(input))
		if err != nil {
			t.Fatalf("Map() should not return error for failed mappings, got: %v", err)
		}

		var parsed []map[string]interface{}
		json.Unmarshal(result, &parsed)

		// Data tag should not be present since the request failed
		if _, ok := parsed[0]["data"]; ok {
			t.Error("Expected 'data' tag to be absent when mapping service is unavailable")
		}
	})

	t.Run("multiple mappings on same item", func(t *testing.T) {
		var callCount atomic.Int32
		mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := callCount.Add(1)
			json.NewEncoder(w).Encode(map[string]string{"result": fmt.Sprintf("mapped-%d", n)})
		}))
		defer mockService.Close()

		route := config.Route{
			Mapping: []config.RouteMapping{
				{
					Path:    "/users/{authorId}",
					Service: mockService.URL,
					Tag:     "author",
				},
				{
					Path:    "/categories/{categoryId}",
					Service: mockService.URL,
					Tag:     "category",
				},
			},
		}

		input := `[{"id":"1","authorId":"a1","categoryId":"c1"}]`
		result, err := Map(route, []byte(input))
		if err != nil {
			t.Fatalf("Map() error = %v", err)
		}

		var parsed []map[string]interface{}
		json.Unmarshal(result, &parsed)

		if _, ok := parsed[0]["author"]; !ok {
			t.Error("Expected 'author' tag")
		}
		if _, ok := parsed[0]["category"]; !ok {
			t.Error("Expected 'category' tag")
		}
	})
}
