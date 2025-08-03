package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
	"github.com/spf13/viper"
)

func doRequestAndDecode[T any](method, uri string, body io.Reader, expectedStatus ...int) (*T, int, humane.Error) {
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
	server := viper.GetString("server")
	url := fmt.Sprintf("%s%s", server, uri)

	// Create the request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, humane.Wrap(err, "failed to create request")
	}

	// Do the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, humane.Wrap(err, "failed to perform request", "ensure the server is reachable")
	}
	defer func() { _ = resp.Body.Close() }()

	// Grab the response
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, humane.Wrap(err, "failed to read response body")
	}

	// based on the HTTP response, handle the API error
	if !okStatus[resp.StatusCode] {
		return nil, resp.StatusCode, handleAPIError(resp, respBytes)
	}

	// attempt parsing the response body
	var result T
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, resp.StatusCode, humane.Wrap(err, "failed to decode response body")
	}

	return &result, resp.StatusCode, nil
}

func handleAPIError(resp *http.Response, body []byte) humane.Error {
	var errBody tailscale.ErrorResponse
	if err := json.Unmarshal(body, &errBody); err == nil {
		return humane.Wrap(errBody.AsHumaneError(), fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	var fallback map[string]any
	if err := json.Unmarshal(body, &fallback); err != nil {
		return humane.Wrap(err, "failed to parse error response")
	}

	msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
	if m, ok := fallback["error"].(string); ok {
		msg = m
	}
	cause := ""
	if c, ok := fallback["internal_error"].(string); ok {
		cause = c
	}

	return humane.Wrap(humane.New(cause), msg)
}
