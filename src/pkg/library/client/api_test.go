// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/globalsign/mgo/bson"
)

const testToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.TCYt5XsITJX1CxPCT8yAV-TVkIEq_PbChOMqsLfRoPsnsgw5WEuts01mq-pQy7UJiN5mgRxD-WUcX16dUEMGlv50aqzpqh4Qktb3rk-BuQy72IFLOqV0G_zS245-kronKb78cPN25DGlcTwLtjPAYuNzVBAh4vGHSrQyHUdBBPM"

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

			entity, found, err := getEntity(m.baseURI, testToken, test.entityRef)

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

			collection, found, err := getCollection(m.baseURI, testToken, test.collectionRef)

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
		description     string
		code            int
		body            interface{}
		reqCallback     func(*http.Request, *testing.T)
		containerRef    string
		expectContainer Container
		expectFound     bool
		expectError     bool
	}{
		{
			description:     "Container not found response",
			code:            404,
			body:            JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback:     nil,
			containerRef:    "notthere",
			expectContainer: Container{},
			expectFound:     false,
			expectError:     false,
		},
		{
			description:     "Unauthorized response",
			code:            401,
			body:            JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback:     nil,
			containerRef:    "notmine",
			expectContainer: Container{},
			expectFound:     false,
			expectError:     true,
		},
		{
			description:     "Valid Response",
			code:            200,
			body:            ContainerResponse{Data: Container{Name: "test"}, Error: JSONError{}},
			reqCallback:     nil,
			containerRef:    "test",
			expectContainer: Container{Name: "test"},
			expectFound:     true,
			expectError:     false,
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

			container, found, err := getContainer(m.baseURI, testToken, test.containerRef)

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
		description string
		code        int
		body        interface{}
		reqCallback func(*http.Request, *testing.T)
		imageRef    string
		expectImage Image
		expectFound bool
		expectError bool
	}{
		{
			description: "Image not found response",
			code:        404,
			body:        JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback: nil,
			imageRef:    "notthere",
			expectImage: Image{},
			expectFound: false,
			expectError: false,
		},
		{
			description: "Unauthorized response",
			code:        401,
			body:        JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback: nil,
			imageRef:    "notmine",
			expectImage: Image{},
			expectFound: false,
			expectError: true,
		},
		{
			description: "Valid Response",
			code:        200,
			body:        ImageResponse{Data: Image{Hash: "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88"}, Error: JSONError{}},
			reqCallback: nil,
			imageRef:    "test",
			expectImage: Image{Hash: "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88"},
			expectFound: true,
			expectError: false,
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

			image, found, err := getImage(m.baseURI, testToken, test.imageRef)

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

func Test_createEntity(t *testing.T) {

	tests := []struct {
		description  string
		code         int
		body         interface{}
		reqCallback  func(*http.Request, *testing.T)
		entityRef    string
		expectEntity Entity
		expectError  bool
	}{
		{
			description:  "Valid Request",
			code:         http.StatusOK,
			body:         EntityResponse{Data: Entity{Name: "test"}, Error: JSONError{}},
			entityRef:    "test",
			expectEntity: Entity{Name: "test"},
			expectError:  false,
		},
		{
			description:  "Error response",
			code:         http.StatusInternalServerError,
			body:         Entity{},
			entityRef:    "test",
			expectEntity: Entity{},
			expectError:  true,
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
				httpPath:    "/v1/entities/",
			}

			m.Run()

			entity, err := createEntity(m.baseURI, testToken, test.entityRef)

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if !reflect.DeepEqual(entity, test.expectEntity) {
				t.Errorf("Got created entity %v - expected %v", entity, test.expectEntity)
			}

			m.Stop()

		})

	}
}

func Test_createCollection(t *testing.T) {

	tests := []struct {
		description      string
		code             int
		body             interface{}
		reqCallback      func(*http.Request, *testing.T)
		collectionRef    string
		expectCollection Collection
		expectError      bool
	}{
		{
			description:      "Valid Request",
			code:             http.StatusOK,
			body:             CollectionResponse{Data: Collection{Name: "test"}, Error: JSONError{}},
			collectionRef:    "test",
			expectCollection: Collection{Name: "test"},
			expectError:      false,
		},
		{
			description:      "Error response",
			code:             http.StatusInternalServerError,
			body:             Collection{},
			collectionRef:    "test",
			expectCollection: Collection{},
			expectError:      true,
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
				httpPath:    "/v1/collections/",
			}

			m.Run()

			collection, err := createCollection(m.baseURI, testToken, test.collectionRef, bson.NewObjectId().Hex())

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if !reflect.DeepEqual(collection, test.expectCollection) {
				t.Errorf("Got created collection %v - expected %v", collection, test.expectCollection)
			}

			m.Stop()

		})

	}
}

func Test_createContainer(t *testing.T) {

	tests := []struct {
		description     string
		code            int
		body            interface{}
		reqCallback     func(*http.Request, *testing.T)
		containerRef    string
		expectContainer Container
		expectError     bool
	}{
		{
			description:     "Valid Request",
			code:            http.StatusOK,
			body:            ContainerResponse{Data: Container{Name: "test"}, Error: JSONError{}},
			containerRef:    "test",
			expectContainer: Container{Name: "test"},
			expectError:     false,
		},
		{
			description:     "Error response",
			code:            http.StatusInternalServerError,
			body:            Container{},
			containerRef:    "test",
			expectContainer: Container{},
			expectError:     true,
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
				httpPath:    "/v1/containers/",
			}

			m.Run()

			container, err := createContainer(m.baseURI, testToken, test.containerRef, bson.NewObjectId().Hex())

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if !reflect.DeepEqual(container, test.expectContainer) {
				t.Errorf("Got created collection %v - expected %v", container, test.expectContainer)
			}

			m.Stop()

		})

	}
}

func Test_createImage(t *testing.T) {

	tests := []struct {
		description string
		code        int
		body        interface{}
		reqCallback func(*http.Request, *testing.T)
		imageRef    string
		expectImage Image
		expectError bool
	}{
		{
			description: "Valid Request",
			code:        http.StatusOK,
			body:        ImageResponse{Data: Image{Hash: "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88"}, Error: JSONError{}},
			imageRef:    "test",
			expectImage: Image{Hash: "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88"},
			expectError: false,
		},
		{
			description: "Error response",
			code:        http.StatusInternalServerError,
			body:        Image{},
			imageRef:    "test",
			expectImage: Image{},
			expectError: true,
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
				httpPath:    "/v1/images/",
			}

			m.Run()

			image, err := createImage(m.baseURI, testToken, test.imageRef, bson.NewObjectId().Hex())

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if !reflect.DeepEqual(image, test.expectImage) {
				t.Errorf("Got created collection %v - expected %v", image, test.expectImage)
			}

			m.Stop()

		})

	}
}

func Test_setTags(t *testing.T) {

	tests := []struct {
		description  string
		code         int
		reqCallback  func(*http.Request, *testing.T)
		containerRef string
		imageRef     string
		tags         []string
		expectError  bool
	}{
		{
			description:  "Valid Request",
			code:         http.StatusOK,
			containerRef: "test",
			imageRef:     bson.NewObjectId().Hex(),
			tags:         []string{"tag1", "tag2", "tag3"},
			expectError:  false,
		},
		{
			description:  "Error response",
			code:         http.StatusInternalServerError,
			containerRef: "test",
			imageRef:     bson.NewObjectId().Hex(),
			tags:         []string{"tag1", "tag2", "tag3"},
			expectError:  true,
		},
	}

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        test.code,
				reqCallback: test.reqCallback,
				httpPath:    "/v1/tags/" + test.containerRef,
			}

			m.Run()

			err := setTags(m.baseURI, testToken, test.containerRef, test.imageRef, test.tags)

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}

			m.Stop()

		})

	}
}
