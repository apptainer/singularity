/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package client

import (
	"net/http"
	"encoding/json"
	"testing"
	"net/http/httptest"
	"reflect"

)

type mockService struct {
	t           *testing.T
	code        int
	body        interface{}
	reqCallback func(*http.Request, *testing.T)
	httpAddr    string
	httpPath    string
	httpServer  *httptest.Server
	baseURI     string
}

func (m *mockService) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc(m.httpPath, m.ServeHTTP)
	m.httpServer = httptest.NewServer(mux)
	m.httpAddr = m.httpServer.Listener.Addr().String()
	m.baseURI = "http://" + m.httpAddr
}

func (m *mockService) Stop() {
	m.httpServer.Close()
}

func (m *mockService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.reqCallback != nil {
		m.reqCallback(r, m.t)
	}
	w.WriteHeader(m.code)
	err := json.NewEncoder(w).Encode(&m.body)
	if err != nil {
		m.t.Errorf("Error encoding mock response: %v", err)
	}
}

func Test_getEntity(t *testing.T) {

	tests := []struct {
		description  string
		code         int
		body         interface{}
		reqCallback  func(*http.Request, *testing.T)
		entityRef    string
		expectEntity Entity
		expectFound  bool
		expectError  bool
	}{
		{
			description:  "Entity not found response",
			code:         400,
			body:         JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback:  nil,
			entityRef:    "notthere",
			expectEntity: Entity{},
			expectFound:  false,
			expectError:  true,
		},
		{
			description:  "Unauthorized response",
			code:         401,
			body:         JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback:  nil,
			entityRef:    "notmine",
			expectEntity: Entity{},
			expectFound:  false,
			expectError:  true,
		},
		{
			description:  "Valid Response",
			code:         200,
			body:         EntityResponse{Data: Entity{Name: "test"}, Error: JSONError{}},
			reqCallback:  nil,
			entityRef:    "test",
			expectEntity: Entity{Name: "test"},
			expectFound:  true,
			expectError:  false,
		},
	}

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        test.code,
				body:        test.body,
				reqCallback: test.reqCallback,
				httpPath:    "/v1/entities/" + test.entityRef,
			}

			m.Run()

			entity, found, err := getEntity(m.baseURI, test.entityRef)

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if found != test.expectFound {
				t.Errorf("Got found %v - expected %v", found, test.expectFound)
			}
			if !reflect.DeepEqual(entity, test.expectEntity) {
				t.Errorf("Got entity %v - expected %v", entity, test.expectEntity)
			}

			m.Stop()

		})

	}
}

func Test_getCollection(t *testing.T) {

	tests := []struct {
		description      string
		code             int
		body             interface{}
		reqCallback      func(*http.Request, *testing.T)
		collectionRef    string
		expectCollection Collection
		expectFound      bool
		expectError      bool
	}{
		{
			description:      "Collection not found response",
			code:             404,
			body:             JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback:      nil,
			collectionRef:    "notthere",
			expectCollection: Collection{},
			expectFound:      false,
			expectError:      false,
		},
		{
			description:      "Unauthorized response",
			code:             401,
			body:             JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback:      nil,
			collectionRef:    "notmine",
			expectCollection: Collection{},
			expectFound:      false,
			expectError:      true,
		},
		{
			description:      "Valid Response",
			code:             200,
			body:             CollectionResponse{Data: Collection{Name: "test"}, Error: JSONError{}},
			reqCallback:      nil,
			collectionRef:    "test",
			expectCollection: Collection{Name: "test"},
			expectFound:      true,
			expectError:      false,
		},
	}

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        test.code,
				body:        test.body,
				reqCallback: test.reqCallback,
				httpPath:    "/v1/collections/" + test.collectionRef,
			}

			m.Run()

			collection, found, err := getCollection(m.baseURI, test.collectionRef)

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if found != test.expectFound {
				t.Errorf("Got found %v - expected %v", found, test.expectFound)
			}
			if !reflect.DeepEqual(collection, test.expectCollection) {
				t.Errorf("Got entity %v - expected %v", collection, test.expectCollection)
			}

			m.Stop()

		})

	}
}


func Test_getContainer(t *testing.T) {

	tests := []struct {
		description      string
		code             int
		body             interface{}
		reqCallback      func(*http.Request, *testing.T)
		containerRef    string
		expectContainer Container
		expectFound      bool
		expectError      bool
	}{
		{
			description:      "Container not found response",
			code:             404,
			body:             JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback:      nil,
			containerRef:    "notthere",
			expectContainer: Container{},
			expectFound:      false,
			expectError:      false,
		},
		{
			description:      "Unauthorized response",
			code:             401,
			body:             JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback:      nil,
			containerRef:    "notmine",
			expectContainer: Container{},
			expectFound:      false,
			expectError:      true,
		},
		{
			description:      "Valid Response",
			code:             200,
			body:             ContainerResponse{Data: Container{Name: "test"}, Error: JSONError{}},
			reqCallback:      nil,
			containerRef:    "test",
			expectContainer: Container{Name: "test"},
			expectFound:      true,
			expectError:      false,
		},
	}

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        test.code,
				body:        test.body,
				reqCallback: test.reqCallback,
				httpPath:    "/v1/containers/" + test.containerRef,
			}

			m.Run()

			container, found, err := getContainer(m.baseURI, test.containerRef)

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if found != test.expectFound {
				t.Errorf("Got found %v - expected %v", found, test.expectFound)
			}
			if !reflect.DeepEqual(container, test.expectContainer) {
				t.Errorf("Got container %v - expected %v", container, test.expectContainer)
			}

			m.Stop()

		})

	}
}

func Test_getImage(t *testing.T) {

	tests := []struct {
		description      string
		code             int
		body             interface{}
		reqCallback      func(*http.Request, *testing.T)
		imageRef    string
		expectImage Image
		expectFound      bool
		expectError      bool
	}{
		{
			description:      "Image not found response",
			code:             404,
			body:             JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback:      nil,
			imageRef:    "notthere",
			expectImage: Image{},
			expectFound:      false,
			expectError:      false,
		},
		{
			description:      "Unauthorized response",
			code:             401,
			body:             JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback:      nil,
			imageRef:    "notmine",
			expectImage: Image{},
			expectFound:      false,
			expectError:      true,
		},
		{
			description:      "Valid Response",
			code:             200,
			body:             ImageResponse{Data: Image{Hash: "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88"}, Error: JSONError{}},
			reqCallback:      nil,
			imageRef:    "test",
			expectImage: Image{Hash: "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88"},
			expectFound:      true,
			expectError:      false,
		},
	}

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        test.code,
				body:        test.body,
				reqCallback: test.reqCallback,
				httpPath:    "/v1/images/" + test.imageRef,
			}

			m.Run()

			image, found, err := getImage(m.baseURI, test.imageRef)

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if found != test.expectFound {
				t.Errorf("Got found %v - expected %v", found, test.expectFound)
			}
			if !reflect.DeepEqual(image, test.expectImage) {
				t.Errorf("Got image %v - expected %v", image, test.expectImage)
			}

			m.Stop()

		})

	}
}