package kwork

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FlexID handles IDs that might come as numbers or strings from the API.
type FlexID int64

func (f *FlexID) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s == "" {
			*f = 0
			return nil
		}
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*f = FlexID(val)
		return nil
	}
	var val int64
	if err := json.Unmarshal(b, &val); err != nil {
		return err
	}
	*f = FlexID(val)
	return nil
}

// Actor represents the currently authenticated user.
type Actor struct {
	ID       FlexID `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Category represents a Kwork category.
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// APIResponse represents a generic wrapper for Kwork API responses.
type APIResponse struct {
	Success bool            `json:"success"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"response"` // sometimes map, sometimes array
}

// ErrorResponse represents a structured error returned by the API logic.
type ErrorResponse struct {
	Code    int
	Message string
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("api error %d: %s", e.Code, e.Message)
}
