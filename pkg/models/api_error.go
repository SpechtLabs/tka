// Package models defines data structures and types used throughout the TKA service.
// This package contains API request/response models, error types, and other
// shared data structures that are used by both the HTTP layer and business logic.
package models

import (
	"encoding/json"
	"errors"

	"github.com/sierrasoftworks/humane-errors-go"
)

// ErrorResponse represents a serializable version of a humane.Error that can be marshaled to and unmarshaled from JSON
// @Description Structured error response with contextual advice
type ErrorResponse struct {
	// Primary error message
	// example: Failed to authenticate user
	Message string `json:"message"`

	// List of suggestions to help resolve the error
	// example: ["Check your Tailscale connection", "Verify you have the required capabilities"]
	Advice []string `json:"advice,omitempty"`

	// Nested error that caused this error (not included in Swagger documentation)
	Cause *ErrorResponse `json:"cause,omitempty" swaggerignore:"true"`

	// HTTP status code (not included in JSON response)
	StatusCode int `json:"-"`
}

// NewErrorResponse creates a new ErrorResponse by wrapping an optional cause error.
// If multiple causes are provided, they are wrapped in order such that
// the first cause is caused by the second, and so on.
func NewErrorResponse(message string, cause ...error) *ErrorResponse {
	// Filter out nils so we never try to wrap them
	nonNilCauses := make([]error, 0, len(cause))
	for _, c := range cause {
		if c != nil {
			nonNilCauses = append(nonNilCauses, c)
		}
	}

	// If no real causes left, just return the message alone
	if len(nonNilCauses) == 0 {
		return FromHumaneError(humane.New(message))
	}

	// Build from the last cause (deepest), preserving advice if it's a humane error
	var herr humane.Error
	lastCause := nonNilCauses[len(nonNilCauses)-1]
	if he, ok := lastCause.(humane.Error); ok {
		herr = he
	} else {
		herr = humane.New(lastCause.Error())
	}

	// Wrap each earlier one around it, preserving advice
	for i := len(nonNilCauses) - 2; i >= 0; i-- {
		c := nonNilCauses[i]
		if he, ok := c.(humane.Error); ok {
			herr = humane.Wrap(herr, he.Error(), he.Advice()...)
		} else {
			herr = humane.Wrap(herr, c.Error())
		}
	}

	// Finally, wrap with the external message
	return FromHumaneError(humane.Wrap(herr, message))
}

// FromHumaneError converts a humane.Error to an ErrorResponse for JSON serialization.
// This is the primary way to convert business logic errors into HTTP API responses.
func FromHumaneError(err humane.Error) *ErrorResponse {
	if err == nil {
		return nil
	}

	// Create the response with the current error's details
	resp := &ErrorResponse{
		Message: err.Error(),
		Advice:  err.Advice(),
	}

	// Handle the cause chain recursively
	cause := err.Cause()
	if cause != nil {
		// If the cause is a humane error, convert it recursively
		var humaneErr humane.Error
		if errors.As(cause, &humaneErr) {
			resp.Cause = FromHumaneError(humaneErr)
		} else {
			// If it's a regular error, create a simple error response
			resp.Cause = &ErrorResponse{
				Message: cause.Error(),
			}
		}
	}

	return resp
}

// AsHumaneError converts the ErrorResponse back to a humane.Error
func (e *ErrorResponse) AsHumaneError() humane.Error {
	if e == nil {
		return nil
	}

	// Create the humane error for the current level
	var err humane.Error

	// Handle the cause chain
	if e.Cause != nil {
		// First convert the cause recursively
		causeErr := e.Cause.AsHumaneError()

		// Then wrap the cause with the current message
		err = humane.Wrap(causeErr, e.Message, e.Advice...)
	} else {
		// Base case: create a new humane error with no cause
		err = humane.New(e.Message, e.Advice...)
	}

	return err
}

// MarshalJSON implements the json.Marshaler interface.
// Alias is used to avoid infinite recursion during marshaling.
func (e *ErrorResponse) MarshalJSON() ([]byte, error) {
	// Alias is a type alias to avoid infinite recursion during JSON marshaling.
	type Alias ErrorResponse
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Alias is used to avoid infinite recursion during unmarshaling.
func (e *ErrorResponse) UnmarshalJSON(data []byte) error {
	// Alias is a type alias to avoid infinite recursion during JSON unmarshaling.
	type Alias ErrorResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	return json.Unmarshal(data, &aux)
}
