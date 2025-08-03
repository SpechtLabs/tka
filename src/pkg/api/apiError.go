package api

import (
	"encoding/json"

	"github.com/sierrasoftworks/humane-errors-go"
)

// ErrorResponse represents a serializable version of a humane.Error that can be marshaled to and unmarshaled from JSON
type ErrorResponse struct {
	Message    string         `json:"message"`
	Advice     []string       `json:"advice,omitempty"`
	Cause      *ErrorResponse `json:"cause,omitempty"`
	StatusCode int            `json:"-,omitempty"`
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
		if humaneErr, ok := cause.(humane.Error); ok {
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
