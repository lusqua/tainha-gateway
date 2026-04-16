package mapper

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aguiar-sh/tainha/internal/config"
	"github.com/aguiar-sh/tainha/internal/util"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        256,
		MaxIdleConnsPerHost: 64,
		MaxConnsPerHost:     256,
		IdleConnTimeout:     90 * time.Second,
	},
}

func Map(route config.Route, response []byte) ([]byte, error) {
	var responseData interface{}
	if err := json.Unmarshal(response, &responseData); err != nil {
		slog.Error("error parsing JSON", "error", err)
		return nil, fmt.Errorf("failed to parse response body: %w", err)
	}

	var dataToProcess []map[string]interface{}
	switch v := responseData.(type) {
	case []interface{}:
		dataToProcess = make([]map[string]interface{}, len(v))
		for i, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				dataToProcess[i] = m
			} else {
				return nil, fmt.Errorf("invalid item in array at index %d", i)
			}
		}
	case map[string]interface{}:
		dataToProcess = []map[string]interface{}{v}
	default:
		return nil, fmt.Errorf("unsupported response type: %T", responseData)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(dataToProcess)*len(route.Mapping))

	// One mutex per item to avoid race conditions on concurrent map writes
	itemMu := make([]sync.Mutex, len(dataToProcess))

	for i := range dataToProcess {
		for _, mapping := range route.Mapping {
			wg.Add(1)
			go func(idx int, item map[string]interface{}, mapping config.RouteMapping) {
				defer wg.Done()
				pathParams := util.ExtractPathParams(mapping.Path)

				for _, param := range pathParams {
					itemMu[idx].Lock()
					value, exists := item[param]
					itemMu[idx].Unlock()

					if !exists {
						continue
					}

					valueStr := fmt.Sprintf("%v", value)
					path, protocol := util.PathProtocol(mapping.Service)

					fullPath := fmt.Sprintf("%s://%s", protocol, path)

					mappedURL := fmt.Sprintf("%s%s%s", fullPath, strings.ReplaceAll(mapping.Path, "{"+param+"}", ""), valueStr)

					slog.Debug("mapping request", "url", mappedURL)

					resp, err := httpClient.Get(mappedURL)
					if err != nil {
						errChan <- fmt.Errorf("error making request to %s: %v", mappedURL, err)
						return
					}
					defer resp.Body.Close()

					body, err := io.ReadAll(resp.Body)
					if err != nil {
						errChan <- fmt.Errorf("error reading response from %s: %v", mappedURL, err)
						return
					}

					var mappedData interface{}
					if err := json.Unmarshal(body, &mappedData); err != nil {
						errChan <- fmt.Errorf("error parsing JSON from %s: %v", mappedURL, err)
						return
					}

					itemMu[idx].Lock()
					item[mapping.Tag] = mappedData
					if mapping.RemoveKeyMapping {
						delete(item, param)
					}
					itemMu[idx].Unlock()
				}
			}(i, dataToProcess[i], mapping)
		}
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			slog.Warn("mapping error", "error", err)
		}
	}

	finalResponse, err := json.Marshal(responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal final response: %w", err)
	}

	return finalResponse, nil
}
