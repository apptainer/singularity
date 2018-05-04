/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"gopkg.in/mgo.v2/bson"
)

const (
	authToken     = "auth_token"
	imageContents = "image_contents"
)

type mockService struct {
	t            *testing.T
	responseCode int
}

func (m *mockService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set the respone body, depending on the type of operation
	if r.Method == http.MethodPost && r.RequestURI == "/v1/build" {
		// Mock new build endpoint
		var rd RequestData
		if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
			m.t.Fatalf("failed to parse request: %v", err)
		}
		w.WriteHeader(m.responseCode)
		if m.responseCode == http.StatusCreated {
			json.NewEncoder(w).Encode(ResponseData{
				ID:         bson.NewObjectId(),
				Definition: rd.Definition,
				IsDetached: rd.IsDetached,
			})
		}
	} else if r.Method == http.MethodGet && strings.HasPrefix(r.RequestURI, "/v1/build/") {
		// Mock status endpoint
		id := r.RequestURI[strings.LastIndexByte(r.RequestURI, '/')+1:]
		if !bson.IsObjectIdHex(id) {
			m.t.Fatalf("Failed to parse ID '%v'", id)
		}
		w.WriteHeader(m.responseCode)
		if m.responseCode == http.StatusOK {
			json.NewEncoder(w).Encode(ResponseData{ID: bson.ObjectIdHex(id)})
		}
	} else if r.Method == http.MethodGet && strings.HasPrefix(r.RequestURI, "/v1/image/") {
		// Mock get image endpoint
		w.WriteHeader(m.responseCode)
		if m.responseCode == http.StatusOK {
			if _, err := strings.NewReader(imageContents).WriteTo(w); err != nil {
				m.t.Fatalf("Failed to write image")
			}
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestDoBuildRequest(t *testing.T) {
	// Craft an expired context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Table of tests to run
	tests := []struct {
		description   string
		expectSuccess bool
		responseCode  int
		ctx           context.Context
		isDetached    bool
	}{
		{"SuccessAttached", true, http.StatusCreated, context.Background(), false},
		{"SuccessDetached", true, http.StatusCreated, context.Background(), true},
		{"NotFoundAttached", false, http.StatusNotFound, context.Background(), false},
		{"NotFoundDetached", false, http.StatusNotFound, context.Background(), true},
		{"ContextExpiredAttached", false, http.StatusCreated, ctx, false},
		{"ContextExpiredDetached", false, http.StatusCreated, ctx, true},
	}

	// Start a mock server
	m := mockService{t: t}
	s := httptest.NewServer(&m)
	defer s.Close()

	// Enough of a struct to test with
	rb := RemoteBuilder{
		HTTPAddr: s.Listener.Addr().String(),
	}

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			m.responseCode = test.responseCode

			// Call the handler
			rd, err := rb.doBuildRequest(test.ctx, Definition{}, test.isDetached)

			if test.expectSuccess {
				// Ensure the handler returned no error, and the response is as expected
				if err != nil {
					t.Fatalf("unexpected failure: %v", err)
				}
				if rd.IsDetached != test.isDetached {
					t.Fatalf("unexpected value for isDetached: %v/%v", rd.IsDetached, test.isDetached)
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected success")
				}
			}
		})
	}
}

func TestDoStatusRequest(t *testing.T) {
	// Craft an expired context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Table of tests to run
	tests := []struct {
		description   string
		expectSuccess bool
		responseCode  int
		ctx           context.Context
	}{
		{"Success", true, http.StatusOK, context.Background()},
		{"NotFound", false, http.StatusNotFound, context.Background()},
		{"ContextExpired", false, http.StatusOK, ctx},
	}

	// Start a mock server
	m := mockService{t: t}
	s := httptest.NewServer(&m)
	defer s.Close()

	// Enough of a struct to test with
	rb := RemoteBuilder{
		HTTPAddr: s.Listener.Addr().String(),
	}

	// ID to test with
	id := bson.NewObjectId()

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			m.responseCode = test.responseCode

			// Call the handler
			rd, err := rb.doStatusRequest(test.ctx, id)

			if test.expectSuccess {
				// Ensure the handler returned no error, and the response is as expected
				if err != nil {
					t.Fatalf("unexpected failure: %v", err)
				}
				if rd.ID != id {
					t.Errorf("mismatched ID: %v/%v", rd.ID, id)
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected success")
				}
			}
		})
	}
}

func TestDoPullRequest(t *testing.T) {
	// Craft an expired context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Table of tests to run
	tests := []struct {
		description   string
		expectSuccess bool
		responseCode  int
		ctx           context.Context
	}{
		{"Success", true, http.StatusOK, context.Background()},
		{"NotFound", false, http.StatusNotFound, context.Background()},
		{"ContextExpired", false, http.StatusOK, ctx},
	}

	// Start a mock server
	m := mockService{t: t}
	s := httptest.NewServer(&m)
	defer s.Close()

	// Enough of a struct to test with
	rb := RemoteBuilder{}

	// Craft URL to image
	path := fmt.Sprintf("/v1/image/%v", bson.NewObjectId().Hex())
	url := url.URL{Scheme: "http", Host: s.Listener.Addr().String(), Path: path}

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			m.responseCode = test.responseCode
			b := bytes.Buffer{}

			// Call the handler
			err := rb.doPullRequest(test.ctx, url.String(), &b)

			if test.expectSuccess {
				// Ensure the handler returned no error, and the image was written as expected
				if err != nil {
					t.Fatalf("unexpected failure: %v", err)
				}
				if 0 != strings.Compare(imageContents, b.String()) {
					t.Errorf("unexpected image contents '%v'/'%v'", b.String(), imageContents)
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected success")
				}
			}
		})
	}
}
