// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/websocket"
)

const (
	authToken      = "auth_token"
	stdoutContents = "some_output"
	imageContents  = "image_contents"
	buildPath      = "/v1/build"
	wsPath         = "/v1/build-ws/"
	imagePath      = "/v1/image"
)

type mockService struct {
	t                  *testing.T
	buildResponseCode  int
	wsResponseCode     int
	wsCloseCode        int
	statusResponseCode int
	imageResponseCode  int
	httpAddr           string
}

var upgrader = websocket.Upgrader{}

func newResponse(m *mockService, id bson.ObjectId, d Definition) ResponseData {
	wsURL := url.URL{
		Scheme: "ws",
		Host:   m.httpAddr,
		Path:   fmt.Sprintf("%s%s", wsPath, id.Hex()),
	}
	imageURL := url.URL{
		Scheme: "http",
		Host:   m.httpAddr,
		Path:   fmt.Sprintf("%s/%s", imagePath, id.Hex()),
	}

	return ResponseData{
		ID:         id,
		Definition: d,
		WSURL:      wsURL.String(),
		ImageURL:   imageURL.String(),
	}
}

func (m *mockService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set the respone body, depending on the type of operation
	if r.Method == http.MethodPost && r.RequestURI == buildPath {
		// Mock new build endpoint
		var rd RequestData
		if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
			m.t.Fatalf("failed to parse request: %v", err)
		}
		w.WriteHeader(m.buildResponseCode)
		if m.buildResponseCode == http.StatusCreated {
			id := bson.NewObjectId()
			json.NewEncoder(w).Encode(newResponse(m, id, rd.Definition))
		}
	} else if r.Method == http.MethodGet && strings.HasPrefix(r.RequestURI, buildPath) {
		// Mock status endpoint
		id := r.RequestURI[strings.LastIndexByte(r.RequestURI, '/')+1:]
		if !bson.IsObjectIdHex(id) {
			m.t.Fatalf("failed to parse ID '%v'", id)
		}
		w.WriteHeader(m.statusResponseCode)
		if m.statusResponseCode == http.StatusOK {
			json.NewEncoder(w).Encode(newResponse(m, bson.ObjectIdHex(id), Definition{}))
		}
	} else if r.Method == http.MethodGet && strings.HasPrefix(r.RequestURI, imagePath) {
		// Mock get image endpoint
		w.WriteHeader(m.imageResponseCode)
		if m.imageResponseCode == http.StatusOK {
			if _, err := strings.NewReader(imageContents).WriteTo(w); err != nil {
				m.t.Fatalf("failed to write image")
			}
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *mockService) ServeWebsocket(w http.ResponseWriter, r *http.Request) {
	if m.wsResponseCode != http.StatusOK {
		w.WriteHeader(m.wsResponseCode)
	} else {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			m.t.Fatalf("failed to upgrade websocket: %v", err)
		}
		defer ws.Close()

		// Write some output and then cleanly close the connection
		ws.WriteMessage(websocket.TextMessage, []byte(stdoutContents))
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(m.wsCloseCode, ""))
	}
}

func TestBuild(t *testing.T) {
	// Craft an expired context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Table of tests to run
	tests := []struct {
		description        string
		expectSuccess      bool
		imagePath          string
		buildResponseCode  int
		wsResponseCode     int
		wsCloseCode        int
		statusResponseCode int
		imageResponseCode  int
		ctx                context.Context
		isDetached         bool
	}{
		{"SuccessAttached", true, "test.img", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"SuccessDetached", true, "test.img", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), true},
		{"BadImagePath", false, "/tmp/bad/", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), true},
		{"AddBuildFailure", false, "test.img", http.StatusUnauthorized, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), true},
		{"WebsocketFailure", false, "test.img", http.StatusCreated, http.StatusUnauthorized, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"WebsocketAbnormalClosure", false, "test.img", http.StatusCreated, http.StatusOK, websocket.CloseAbnormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"GetStatusFailureAttached", false, "test.img", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusUnauthorized, http.StatusOK, context.Background(), false},
		{"GetStatusFailureDetached", false, "test.img", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusUnauthorized, http.StatusOK, context.Background(), true},
		{"GetImageFailureAttached", false, "test.img", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusUnauthorized, context.Background(), false},
		{"GetImageFailureDetached", false, "test.img", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusUnauthorized, context.Background(), true},
		{"ContextExpired", false, "test.img", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, ctx, true},
	}

	// Start a mock server
	m := mockService{t: t}
	mux := http.NewServeMux()
	mux.HandleFunc("/", m.ServeHTTP)
	mux.HandleFunc(wsPath, m.ServeWebsocket)
	s := httptest.NewServer(mux)
	defer s.Close()

	// Mock server address is fixed for all tests
	m.httpAddr = s.Listener.Addr().String()

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			rb := NewRemoteBuilder(test.imagePath, Definition{}, test.isDetached, s.Listener.Addr().String(), authToken)

			// Set the response codes for each stage of the build
			m.buildResponseCode = test.buildResponseCode
			m.wsResponseCode = test.wsResponseCode
			m.wsCloseCode = test.wsCloseCode
			m.statusResponseCode = test.statusResponseCode
			m.imageResponseCode = test.imageResponseCode

			// Do it!
			err := rb.Build(test.ctx)

			if test.expectSuccess {
				// Ensure the handler returned no error, and the response is as expected
				if err != nil {
					t.Fatalf("unexpected failure: %v", err)
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
	}{
		{"SuccessAttached", true, http.StatusCreated, context.Background()},
		{"NotFoundAttached", false, http.StatusNotFound, context.Background()},
		{"ContextExpiredAttached", false, http.StatusCreated, ctx},
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
			m.buildResponseCode = test.responseCode

			// Call the handler
			rd, err := rb.doBuildRequest(test.ctx)

			if test.expectSuccess {
				// Ensure the handler returned no error, and the response is as expected
				if err != nil {
					t.Fatalf("unexpected failure: %v", err)
				}
				if !rd.ID.Valid() {
					t.Fatalf("invalid ID")
				}
				if rd.WSURL == "" {
					t.Errorf("empty websocket URL")
				}
				if rd.ImageURL == "" {
					t.Errorf("empty image URL")
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
			m.statusResponseCode = test.responseCode

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
				if rd.WSURL == "" {
					t.Errorf("empty websocket URL")
				}
				if rd.ImageURL == "" {
					t.Errorf("empty image URL")
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
