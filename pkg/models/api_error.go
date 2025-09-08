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

func NewErrorResponse(message string, cause error) *ErrorResponse {
	return FromHumaneError(humane.Wrap(cause, message))
}

// FromHumaneError converts a humane.Error to an ErrorResponse for serialization
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

// MarshalJSON implements the json.Marshaler interface
func (e *ErrorResponse) MarshalJSON() ([]byte, error) {
	// Create a temporary type to avoid infinite recursion
	type Alias ErrorResponse
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (e *ErrorResponse) UnmarshalJSON(data []byte) error {
	// Create a temporary type to avoid infinite recursion
	type Alias ErrorResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	return nil
}
