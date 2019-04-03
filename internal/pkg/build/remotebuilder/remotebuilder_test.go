// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package remotebuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/websocket"
	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/singularity/internal/pkg/test"
	types "github.com/sylabs/singularity/pkg/build/legacy"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
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

func TestMain(m *testing.M) {
	useragent.InitValue("singularity", "3.0.0-alpha.1-303-gaed8d30-dirty")

	os.Exit(m.Run())
}

func newResponse(m *mockService, id bson.ObjectId, d types.Definition, libraryRef string) types.ResponseData {
	wsURL := url.URL{
		Scheme: "ws",
		Host:   m.httpAddr,
		Path:   fmt.Sprintf("%s%s", wsPath, id.Hex()),
	}
	libraryURL := url.URL{
		Scheme: "http",
		Host:   m.httpAddr,
	}
	if libraryRef == "" {
		libraryRef = "library://user/collection/image"
	}

	return types.ResponseData{
		ID:         id,
		Definition: d,
		WSURL:      wsURL.String(),
		LibraryURL: libraryURL.String(),
		LibraryRef: libraryRef,
		IsComplete: true,
		ImageSize:  1,
	}
}

func (m *mockService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set the response body, depending on the type of operation
	if r.Method == http.MethodPost && r.RequestURI == buildPath {
		// Mock new build endpoint
		var rd types.RequestData
		if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
			m.t.Fatalf("failed to parse request: %v", err)
		}
		if m.buildResponseCode == http.StatusCreated {
			id := bson.NewObjectId()
			jsonresp.WriteResponse(w, newResponse(m, id, rd.Definition, rd.LibraryRef), m.buildResponseCode)
		} else {
			jsonresp.WriteError(w, "", m.buildResponseCode)
		}
	} else if r.Method == http.MethodGet && strings.HasPrefix(r.RequestURI, buildPath) {
		// Mock status endpoint
		id := r.RequestURI[strings.LastIndexByte(r.RequestURI, '/')+1:]
		if !bson.IsObjectIdHex(id) {
			m.t.Fatalf("failed to parse ID '%v'", id)
		}
		if m.statusResponseCode == http.StatusOK {
			jsonresp.WriteResponse(w, newResponse(m, bson.ObjectIdHex(id), types.Definition{}, ""), m.statusResponseCode)
		} else {
			jsonresp.WriteError(w, "", m.statusResponseCode)
		}
	} else if r.Method == http.MethodGet && strings.HasPrefix(r.RequestURI, imagePath) {
		// Mock get image endpoint
		if m.imageResponseCode == http.StatusOK {
			if _, err := strings.NewReader(imageContents).WriteTo(w); err != nil {
				m.t.Fatalf("failed to write image")
			}
		} else {
			jsonresp.WriteError(w, "", m.imageResponseCode)
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
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Craft an expired context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Create a temporary file for testing
	f, err := ioutil.TempFile("/tmp", "TestBuild")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	// Start a mock server
	m := mockService{t: t}
	mux := http.NewServeMux()
	mux.HandleFunc("/", m.ServeHTTP)
	mux.HandleFunc(wsPath, m.ServeWebsocket)
	s := httptest.NewServer(mux)
	defer s.Close()

	// Mock server address is fixed for all tests
	m.httpAddr = s.Listener.Addr().String()

	// Table of tests to run
	// nolint:maligned
	tests := []struct {
		description        string
		expectSuccess      bool
		imagePath          string
		libraryURL         string
		buildResponseCode  int
		wsResponseCode     int
		wsCloseCode        int
		statusResponseCode int
		imageResponseCode  int
		ctx                context.Context
		isDetached         bool
	}{
		{"SuccessAttached", true, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"SuccessDetached", true, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), true},
		{"SuccessLibraryRef", true, "library://user/collection/image", "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"SuccessLibraryRefURL", true, "library://user/collection/image", m.httpAddr, http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"BadImagePath", false, "/tmp/bad/", "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"BadLibraryRef", false, "library://bad", "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"AddBuildFailure", false, f.Name(), "", http.StatusUnauthorized, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"WebsocketFailure", false, f.Name(), "", http.StatusCreated, http.StatusUnauthorized, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"WebsocketAbnormalClosure", false, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseAbnormalClosure, http.StatusOK, http.StatusOK, context.Background(), false},
		{"GetStatusFailure", false, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusUnauthorized, http.StatusOK, context.Background(), false},
		{"GetImageFailure", false, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusUnauthorized, context.Background(), false},
		{"ContextExpired", false, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, ctx, false},
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {
			rb, err := New(tt.imagePath, "", types.Definition{}, tt.isDetached, false, s.URL, authToken)
			if err != nil {
				t.Fatalf("failed to get new remote builder: %v", err)
			}
			rb.Force = true

			// Set the response codes for each stage of the build
			m.buildResponseCode = tt.buildResponseCode
			m.wsResponseCode = tt.wsResponseCode
			m.wsCloseCode = tt.wsCloseCode
			m.statusResponseCode = tt.statusResponseCode
			m.imageResponseCode = tt.imageResponseCode

			// Do it!
			err = rb.Build(tt.ctx)

			if tt.expectSuccess {
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
		}))
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
		libraryRef    string
		responseCode  int
		ctx           context.Context
	}{
		{"SuccessAttached", true, "", http.StatusCreated, context.Background()},
		{"SuccessLibraryRef", true, "library://user/collection/image", http.StatusCreated, context.Background()},
		{"BadLibraryRef", false, "library://bad", http.StatusCreated, context.Background()},
		{"NotFoundAttached", false, "", http.StatusNotFound, context.Background()},
		{"ContextExpiredAttached", false, "", http.StatusCreated, ctx},
	}

	// Start a mock server
	m := mockService{t: t}
	s := httptest.NewServer(&m)
	defer s.Close()

	// Enough of a struct to test with
	url, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	rb := RemoteBuilder{
		BuilderURL: url,
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {
			m.buildResponseCode = tt.responseCode

			// Call the handler
			rd, err := rb.doBuildRequest(tt.ctx, types.Definition{}, tt.libraryRef)

			if tt.expectSuccess {
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
				if rd.LibraryRef == "" {
					t.Errorf("empty Library ref")
				}
				if rd.LibraryURL == "" {
					t.Errorf("empty Library URL")
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected success")
				}
			}
		}))
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
	url, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	rb := RemoteBuilder{
		BuilderURL: url,
	}

	// ID to test with
	id := bson.NewObjectId()

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {
			m.statusResponseCode = tt.responseCode

			// Call the handler
			rd, err := rb.doStatusRequest(tt.ctx, id)

			if tt.expectSuccess {
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
				if rd.LibraryRef == "" {
					t.Errorf("empty Library ref")
				}
				if rd.LibraryURL == "" {
					t.Errorf("empty Library URL")
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected success")
				}
			}
		}))
	}
}
