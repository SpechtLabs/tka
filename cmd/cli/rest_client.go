package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/service/api"
)

// httpClient is a custom HTTP client with timeout for CLI requests.
// Using a shared client allows connection reuse.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func doRequestAndDecode[T any](ctx context.Context, method, uri string, body io.Reader, expectedStatus ...int) (*T, int, humane.Error) {
	// Allow 200 OK by default if no status codes are passed in
	okStatus := map[int]bool{}
	if len(expectedStatus) == 0 {
		okStatus[http.StatusOK] = true
	} else {
		for _, code := range expectedStatus {
			okStatus[code] = true
		}
	}

	// Assemble the request URL
	serverAddr := getServerAddr()
	url := fmt.Sprintf("%s%s%s", serverAddr, api.ApiRouteV1Alpha1, uri)

	// Create the request with context
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, humane.Wrap(err, "failed to create request", "this indicates a bug in the CLI; please report it")
	}

	// Do the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, humane.Wrap(err, "failed to perform request", "ensure tailscale is running and the TKA server is reachable")
	}
	defer func() { _ = resp.Body.Close() }()

	// Grab the response
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, humane.Wrap(err, "failed to read response body", "the server may have closed the connection unexpectedly")
	}

	// based on the HTTP response, handle the API error
	if !okStatus[resp.StatusCode] {
		return nil, resp.StatusCode, handleAPIError(resp, respBytes)
	}

	// attempt parsing the response body
	var result T
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, resp.StatusCode, humane.Wrap(err, "failed to decode response body", "the server returned an unexpected response format")
	}

	return &result, resp.StatusCode, nil
}

func handleAPIError(resp *http.Response, body []byte) humane.Error {
	var errBody models.ErrorResponse
	if err := json.Unmarshal(body, &errBody); err == nil {
		return humane.Wrap(errBody.AsHumaneError(), fmt.Sprintf("HTTP %d", resp.StatusCode), "check the error details for more information")
	}

	var fallback map[string]any
	if err := json.Unmarshal(body, &fallback); err == nil {
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if m, ok := fallback["error"].(string); ok {
			msg = m
		}
		cause := ""
		if c, ok := fallback["internal_error"].(string); ok {
			cause = c
		}

		return humane.Wrap(humane.New(cause, "check server logs for more details"), msg, "the server returned an error")
	}

	return humane.New(string(body), "the server returned an unexpected error format")

}
