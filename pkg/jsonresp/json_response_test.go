// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package jsonresp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestError(t *testing.T) {
	tests := []struct {
		name          string
		code          int
		message       string
		wantErrString string
	}{
		{"NoMessage", http.StatusNotFound, "", "404 Not Found"},
		{"Message", http.StatusNotFound, "blah", "blah (404 Not Found)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			je := NewError(tt.code, tt.message)
			if je.Code != tt.code {
				t.Errorf("got code %v, want %v", je.Code, tt.code)
			}
			if je.Message != tt.message {
				t.Errorf("got message %v, want %v", je.Message, tt.message)
			}
			if s := je.Error(); s != tt.wantErrString {
				t.Errorf("got string %v, want %v", s, tt.wantErrString)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name        string
		error       string
		code        int
		wantMessage string
		wantCode    int
	}{
		{"NoMessage", "", http.StatusNotFound, "", http.StatusNotFound},
		{"NoMessage", "blah", http.StatusNotFound, "blah", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			WriteError(rr, tt.error, tt.code)

			if rr.Code != tt.wantCode {
				t.Errorf("got code %v, want %v", rr.Code, tt.wantCode)
			}

			var jr Response
			if err := json.NewDecoder(rr.Body).Decode(&jr); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if jr.Error == nil {
				t.Fatalf("nil error received")
			}
			if jr.Error.Message != tt.wantMessage {
				t.Errorf("got message %v, want %v", jr.Error.Message, tt.wantMessage)
			}
			if jr.Error.Code != tt.wantCode {
				t.Errorf("got code %v, want %v", jr.Error.Code, tt.wantCode)
			}
		})
	}
}

func TestWriteResponse(t *testing.T) {
	type TestStruct struct {
		Value string
	}

	tests := []struct {
		name      string
		data      interface{}
		code      int
		wantValue string
		wantCode  int
	}{
		{"Empty", TestStruct{""}, http.StatusOK, "", http.StatusOK},
		{"NotEmpty", TestStruct{"blah"}, http.StatusOK, "blah", http.StatusOK},
		{"Created", TestStruct{"blah"}, http.StatusCreated, "blah", http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			WriteResponse(rr, tt.data, tt.code)

			var ts TestStruct
			if err := ReadResponse(rr.Body, &ts); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if ts.Value != tt.wantValue {
				t.Errorf("got value '%v', want '%v'", ts.Value, tt.wantValue)
			}
			if rr.Code != tt.wantCode {
				t.Errorf("got code '%v', want '%v'", rr.Code, tt.wantCode)
			}
		})
	}
}

func getResponseBody(v interface{}) io.Reader {
	rr := httptest.NewRecorder()
	WriteResponse(rr, v, http.StatusOK)
	return rr.Body
}

func getErrorBody(error string, code int) io.Reader {
	rr := httptest.NewRecorder()
	WriteError(rr, error, code)
	return rr.Body
}

func TestReadResponse(t *testing.T) {
	type TestStruct struct {
		Value string
	}

	tests := []struct {
		name      string
		r         io.Reader
		wantErr   bool
		wantValue string
	}{
		{"Empty", bytes.NewReader(nil), true, ""},
		{"Response", getResponseBody(TestStruct{"blah"}), false, "blah"},
		{"Error", getErrorBody("blah", http.StatusNotFound), true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts TestStruct

			err := ReadResponse(tt.r, &ts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadResponse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				if ts.Value != tt.wantValue {
					t.Errorf("got value '%v', want '%v'", ts.Value, tt.wantValue)
				}
			}
		})
	}
}
