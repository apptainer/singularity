// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package jsonresp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%v (%v %v)", e.Message, e.Code, http.StatusText(e.Code))
	}
	return fmt.Sprintf("%v %v", e.Code, http.StatusText(e.Code))
}

// Error describes an error condition.
type Error struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Response is the top level container of all of our REST API responses.
type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error *Error      `json:"error,omitempty"`
}

// NewError returns an error that contains the given code and message.
func NewError(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// WriteError encodes the supplied error in a response, and writes to w.
func WriteError(w http.ResponseWriter, error string, code int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	jr := Response{
		Error: &Error{
			Code:    code,
			Message: error,
		},
	}
	if err := json.NewEncoder(w).Encode(jr); err != nil {
		return fmt.Errorf("jsonresp: failed to write error: %v", err)
	}
	return nil
}

// WriteResponse encodes the supplied data in a response, and writes to w.
func WriteResponse(w http.ResponseWriter, data interface{}, code int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	jr := Response{
		Data: data,
	}
	if err := json.NewEncoder(w).Encode(jr); err != nil {
		return fmt.Errorf("jsonresp: failed to write response: %v", err)
	}
	return nil
}

// ReadResponse reads a JSON response, and unmarshals the supplied data.
func ReadResponse(r io.Reader, v interface{}) error {
	var u struct {
		Data  json.RawMessage `json:"data"`
		Error *Error          `json:"error"`
	}
	if err := json.NewDecoder(r).Decode(&u); err != nil {
		return fmt.Errorf("jsonresp: failed to read response: %v", err)
	}
	if u.Error != nil {
		return u.Error
	}
	if v != nil {
		if err := json.Unmarshal(u.Data, v); err != nil {
			return fmt.Errorf("jsonresp: failed to unmarshal response: %v", err)
		}
	}
	return nil
}
