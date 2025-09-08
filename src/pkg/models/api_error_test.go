package models

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/sierrasoftworks/humane-errors-go"
)

func TestNewErrorResponse(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		message  string
		cause    error
		expected *ErrorResponse
	}{
		{
			name:    "simple_error_with_message",
			message: "operation failed",
			cause:   errors.New("database connection lost"),
			expected: &ErrorResponse{
				Message: "operation failed",
				Cause: &ErrorResponse{
					Message: "database connection lost",
				},
			},
		},
		{
			name:     "nil_cause",
			message:  "validation failed",
			cause:    nil,
			expected: nil,
		},
		{
			name:    "empty_message",
			message: "",
			cause:   errors.New("some error"),
			expected: &ErrorResponse{
				Message: "",
				Cause: &ErrorResponse{
					Message: "some error",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			got := NewErrorResponse(tt.message, tt.cause)
			assertErrorResponseEqual(t, got, tt.expected)
		})
	}
}

func TestFromHumaneError(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		input    humane.Error
		expected *ErrorResponse
	}{
		{
			name:     "nil_error",
			input:    nil,
			expected: nil,
		},
		{
			name:  "simple_humane_error",
			input: humane.New("authentication failed", "check your credentials"),
			expected: &ErrorResponse{
				Message: "authentication failed",
				Advice:  []string{"check your credentials"},
			},
		},
		{
			name:  "humane_error_with_multiple_advice",
			input: humane.New("server unavailable", "check network connection", "verify server status", "try again later"),
			expected: &ErrorResponse{
				Message: "server unavailable",
				Advice:  []string{"check network connection", "verify server status", "try again later"},
			},
		},
		{
			name:  "wrapped_humane_error",
			input: humane.Wrap(humane.New("database error", "check database connection"), "service unavailable", "try again later"),
			expected: &ErrorResponse{
				Message: "service unavailable",
				Advice:  []string{"try again later"},
				Cause: &ErrorResponse{
					Message: "database error",
					Advice:  []string{"check database connection"},
				},
			},
		},
		{
			name:  "wrapped_standard_error",
			input: humane.Wrap(errors.New("connection refused"), "failed to connect", "check network"),
			expected: &ErrorResponse{
				Message: "failed to connect",
				Advice:  []string{"check network"},
				Cause: &ErrorResponse{
					Message: "connection refused",
				},
			},
		},
		{
			name:  "deeply_nested_errors",
			input: humane.Wrap(humane.Wrap(errors.New("timeout"), "database unavailable"), "service error"),
			expected: &ErrorResponse{
				Message: "service error",
				Cause: &ErrorResponse{
					Message: "database unavailable",
					Cause: &ErrorResponse{
						Message: "timeout",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			got := FromHumaneError(tt.input)
			assertErrorResponseEqual(t, got, tt.expected)
		})
	}
}

func TestErrorResponse_AsHumaneError(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		input    *ErrorResponse
		expected humane.Error
	}{
		{
			name:     "nil_error_response",
			input:    nil,
			expected: nil,
		},
		{
			name: "simple_error_response",
			input: &ErrorResponse{
				Message: "authentication failed",
				Advice:  []string{"check credentials"},
			},
			expected: humane.New("authentication failed", "check credentials"),
		},
		{
			name: "nested_error_response",
			input: &ErrorResponse{
				Message: "service unavailable",
				Advice:  []string{"try again"},
				Cause: &ErrorResponse{
					Message: "database error",
					Advice:  []string{"check connection"},
				},
			},
			expected: humane.Wrap(humane.New("database error", "check connection"), "service unavailable", "try again"),
		},
		{
			name: "deeply_nested_error_response",
			input: &ErrorResponse{
				Message: "operation failed",
				Cause: &ErrorResponse{
					Message: "service error",
					Cause: &ErrorResponse{
						Message: "network timeout",
					},
				},
			},
			expected: humane.Wrap(humane.Wrap(humane.New("network timeout"), "service error"), "operation failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			got := tt.input.AsHumaneError()
			assertHumaneErrorEqual(t, got, tt.expected)
		})
	}
}

func TestErrorResponse_JSON_RoundTrip(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		input    *ErrorResponse
		expected string
	}{
		{
			name: "simple_error",
			input: &ErrorResponse{
				Message: "test error",
				Advice:  []string{"test advice"},
			},
			expected: `{"message":"test error","advice":["test advice"]}`,
		},
		{
			name: "error_with_cause",
			input: &ErrorResponse{
				Message: "outer error",
				Cause: &ErrorResponse{
					Message: "inner error",
				},
			},
			expected: `{"message":"outer error","cause":{"message":"inner error"}}`,
		},
		{
			name: "error_without_advice",
			input: &ErrorResponse{
				Message: "simple error",
			},
			expected: `{"message":"simple error"}`,
		},
		{
			name: "error_with_status_code_excluded",
			input: &ErrorResponse{
				Message:    "server error",
				StatusCode: 500,
			},
			expected: `{"message":"server error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			// Test marshaling
			gotJSON, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			if string(gotJSON) != tt.expected {
				t.Errorf("json.Marshal() = %s, want %s", string(gotJSON), tt.expected)
			}

			// Test unmarshaling
			var got ErrorResponse
			if err := json.Unmarshal(gotJSON, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			// StatusCode should not be unmarshaled (it's excluded from JSON)
			expectedForUnmarshal := *tt.input
			expectedForUnmarshal.StatusCode = 0

			assertErrorResponseEqual(t, &got, &expectedForUnmarshal)
		})
	}
}

func TestErrorResponse_JSON_Unmarshal_InvalidJSON(t *testing.T) {
	t.Helper()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "invalid_json",
			input:       `{"message": invalid}`,
			expectError: true,
		},
		{
			name:        "valid_json",
			input:       `{"message": "test"}`,
			expectError: false,
		},
		{
			name:        "empty_json",
			input:       `{}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			var got ErrorResponse
			err := json.Unmarshal([]byte(tt.input), &got)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestErrorResponse_Conversion_RoundTrip(t *testing.T) {
	t.Helper()

	tests := []struct {
		name  string
		input humane.Error
	}{
		{
			name:  "simple_error",
			input: humane.New("test error", "test advice"),
		},
		{
			name:  "wrapped_error",
			input: humane.Wrap(errors.New("inner error"), "outer error", "some advice"),
		},
		{
			name:  "deeply_nested_error",
			input: humane.Wrap(humane.Wrap(errors.New("deep error"), "middle error"), "outer error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			// Convert to ErrorResponse and back
			errorResp := FromHumaneError(tt.input)
			got := errorResp.AsHumaneError()

			assertHumaneErrorEqual(t, got, tt.input)
		})
	}
}

// Helper functions for assertions

func assertErrorResponseEqual(t *testing.T, got, expected *ErrorResponse) {
	t.Helper()

	if expected == nil {
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
		return
	}

	if got == nil {
		t.Errorf("expected %+v, got nil", expected)
		return
	}

	if got.Message != expected.Message {
		t.Errorf("Message = %q, want %q", got.Message, expected.Message)
	}

	if !reflect.DeepEqual(got.Advice, expected.Advice) {
		t.Errorf("Advice = %v, want %v", got.Advice, expected.Advice)
	}

	if got.StatusCode != expected.StatusCode {
		t.Errorf("StatusCode = %d, want %d", got.StatusCode, expected.StatusCode)
	}

	// Recursively check cause
	assertErrorResponseEqual(t, got.Cause, expected.Cause)
}

func assertHumaneErrorEqual(t *testing.T, got, expected humane.Error) {
	t.Helper()

	if expected == nil {
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
		return
	}

	if got == nil {
		t.Errorf("expected %v, got nil", expected)
		return
	}

	if got.Error() != expected.Error() {
		t.Errorf("Error() = %q, want %q", got.Error(), expected.Error())
	}

	if !reflect.DeepEqual(got.Advice(), expected.Advice()) {
		t.Errorf("Advice() = %v, want %v", got.Advice(), expected.Advice())
	}

	// Compare causes recursively
	gotCause := got.Cause()
	expectedCause := expected.Cause()

	if (gotCause == nil) != (expectedCause == nil) {
		t.Errorf("Cause mismatch: got %v, want %v", gotCause, expectedCause)
		return
	}

	if gotCause != nil && expectedCause != nil {
		if gotCause.Error() != expectedCause.Error() {
			t.Errorf("Cause().Error() = %q, want %q", gotCause.Error(), expectedCause.Error())
		}
	}
}
