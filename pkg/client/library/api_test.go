// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/globalsign/mgo/bson"
	"github.com/sylabs/singularity/internal/pkg/test"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

const testToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.TCYt5XsITJX1CxPCT8yAV-TVkIEq_PbChOMqsLfRoPsnsgw5WEuts01mq-pQy7UJiN5mgRxD-WUcX16dUEMGlv50aqzpqh4Qktb3rk-BuQy72IFLOqV0G_zS245-kronKb78cPN25DGlcTwLtjPAYuNzVBAh4vGHSrQyHUdBBPM"

var (
	testEntity = Entity{
		ID:          bson.NewObjectId(),
		Name:        "test-user",
		Description: "A test user",
	}

	testCollection = Collection{
		ID:          bson.NewObjectId(),
		Name:        "test-collection",
		Description: "A test collection",
		Entity:      testEntity.ID,
		EntityName:  testEntity.Name,
	}

	testContainer = Container{
		ID:             bson.NewObjectId(),
		Name:           "test-container",
		Description:    "A test container",
		Entity:         testEntity.ID,
		EntityName:     testEntity.Name,
		Collection:     testEntity.ID,
		CollectionName: testCollection.Name,
		ImageTags: map[string]bson.ObjectId{
			"test-tag": bson.NewObjectId(),
			"latest":   bson.NewObjectId()},
	}

	testImage = Image{
		ID:             bson.NewObjectId(),
		Hash:           "sha256.e50a30881ace3d5944f5661d222db7bee5296be9e4dc7c1fcb7604bcae926e88",
		Entity:         testEntity.ID,
		EntityName:     testEntity.Name,
		Collection:     testEntity.ID,
		CollectionName: testCollection.Name,
		Container:      testContainer.ID,
		ContainerName:  testContainer.Name,
	}

	testSearch = SearchResults{
		Entities:    []Entity{testEntity},
		Collections: []Collection{testCollection},
		Containers:  []Container{testContainer},
	}
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

func TestMain(m *testing.M) {
	useragent.InitValue("singularity", "3.0.0-alpha.1-303-gaed8d30-dirty")

	os.Exit(m.Run())
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
			description:  "NotFound",
			code:         http.StatusNotFound,
			body:         JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback:  nil,
			entityRef:    "notthere",
			expectEntity: Entity{},
			expectFound:  false,
			expectError:  false,
		},
		{
			description:  "Unauthorized",
			code:         http.StatusUnauthorized,
			body:         JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback:  nil,
			entityRef:    "notmine",
			expectEntity: Entity{},
			expectFound:  false,
			expectError:  true,
		},
		{
			description:  "ValidResponse",
			code:         http.StatusOK,
			body:         EntityResponse{Data: testEntity, Error: JSONError{}},
			reqCallback:  nil,
			entityRef:    "test-user",
			expectEntity: testEntity,
			expectFound:  true,
			expectError:  false,
		},
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				body:        tt.body,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/entities/" + tt.entityRef,
			}

			m.Run()

			entity, found, err := getEntity(m.baseURI, testToken, tt.entityRef)

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if found != tt.expectFound {
				t.Errorf("Got found %v - expected %v", found, tt.expectFound)
			}
			if !reflect.DeepEqual(entity, tt.expectEntity) {
				t.Errorf("Got entity %v - expected %v", entity, tt.expectEntity)
			}

			m.Stop()

		}))

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
			description:      "NotFound",
			code:             http.StatusNotFound,
			body:             JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback:      nil,
			collectionRef:    "notthere",
			expectCollection: Collection{},
			expectFound:      false,
			expectError:      false,
		},
		{
			description:      "Unauthorized",
			code:             http.StatusUnauthorized,
			body:             JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback:      nil,
			collectionRef:    "notmine",
			expectCollection: Collection{},
			expectFound:      false,
			expectError:      true,
		},
		{
			description:      "ValidResponse",
			code:             http.StatusOK,
			body:             CollectionResponse{Data: testCollection, Error: JSONError{}},
			reqCallback:      nil,
			collectionRef:    "test-entity/test-collection",
			expectCollection: testCollection,
			expectFound:      true,
			expectError:      false,
		},
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				body:        tt.body,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/collections/" + tt.collectionRef,
			}

			m.Run()

			collection, found, err := getCollection(m.baseURI, testToken, tt.collectionRef)

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if found != tt.expectFound {
				t.Errorf("Got found %v - expected %v", found, tt.expectFound)
			}
			if !reflect.DeepEqual(collection, tt.expectCollection) {
				t.Errorf("Got entity %v - expected %v", collection, tt.expectCollection)
			}

			m.Stop()

		}))

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
			description:     "NotFound",
			code:            http.StatusNotFound,
			body:            JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback:     nil,
			containerRef:    "notthere",
			expectContainer: Container{},
			expectFound:     false,
			expectError:     false,
		},
		{
			description:     "Unauthorized",
			code:            http.StatusUnauthorized,
			body:            JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback:     nil,
			containerRef:    "notmine",
			expectContainer: Container{},
			expectFound:     false,
			expectError:     true,
		},
		{
			description:     "ValidResponse",
			code:            http.StatusOK,
			body:            ContainerResponse{Data: testContainer, Error: JSONError{}},
			reqCallback:     nil,
			containerRef:    "test-entity/test-collection/test-container",
			expectContainer: testContainer,
			expectFound:     true,
			expectError:     false,
		},
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				body:        tt.body,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/containers/" + tt.containerRef,
			}

			m.Run()

			container, found, err := getContainer(m.baseURI, testToken, tt.containerRef)

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if found != tt.expectFound {
				t.Errorf("Got found %v - expected %v", found, tt.expectFound)
			}
			if !reflect.DeepEqual(container, tt.expectContainer) {
				t.Errorf("Got container %v - expected %v", container, tt.expectContainer)
			}

			m.Stop()

		}))

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
			description: "NotFound",
			code:        http.StatusNotFound,
			body:        JSONResponse{Error: JSONError{Code: http.StatusNotFound, Status: http.StatusText(http.StatusNotFound)}},
			reqCallback: nil,
			imageRef:    "notthere",
			expectImage: Image{},
			expectFound: false,
			expectError: false,
		},
		{
			description: "Unauthorized",
			code:        http.StatusUnauthorized,
			body:        JSONResponse{Error: JSONError{Code: http.StatusUnauthorized, Status: http.StatusText(http.StatusUnauthorized)}},
			reqCallback: nil,
			imageRef:    "notmine",
			expectImage: Image{},
			expectFound: false,
			expectError: true,
		},
		{
			description: "ValidResponse",
			code:        http.StatusOK,
			body:        ImageResponse{Data: testImage, Error: JSONError{}},
			reqCallback: nil,
			imageRef:    "test",
			expectImage: testImage,
			expectFound: true,
			expectError: false,
		},
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				body:        tt.body,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/images/" + tt.imageRef,
			}

			m.Run()

			image, found, err := getImage(m.baseURI, testToken, tt.imageRef)

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if found != tt.expectFound {
				t.Errorf("Got found %v - expected %v", found, tt.expectFound)
			}
			if !reflect.DeepEqual(image, tt.expectImage) {
				t.Errorf("Got image %v - expected %v", image, tt.expectImage)
			}

			m.Stop()

		}))

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
			body:         EntityResponse{Data: testEntity, Error: JSONError{}},
			entityRef:    "test",
			expectEntity: testEntity,
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
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				body:        tt.body,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/entities/",
			}

			m.Run()

			entity, err := createEntity(m.baseURI, testToken, tt.entityRef)

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if !reflect.DeepEqual(entity, tt.expectEntity) {
				t.Errorf("Got created entity %v - expected %v", entity, tt.expectEntity)
			}

			m.Stop()

		}))

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
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				body:        tt.body,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/collections/",
			}

			m.Run()

			collection, err := createCollection(m.baseURI, testToken, tt.collectionRef, bson.NewObjectId().Hex())

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if !reflect.DeepEqual(collection, tt.expectCollection) {
				t.Errorf("Got created collection %v - expected %v", collection, tt.expectCollection)
			}

			m.Stop()

		}))

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
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				body:        tt.body,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/containers/",
			}

			m.Run()

			container, err := createContainer(m.baseURI, testToken, tt.containerRef, bson.NewObjectId().Hex())

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if !reflect.DeepEqual(container, tt.expectContainer) {
				t.Errorf("Got created collection %v - expected %v", container, tt.expectContainer)
			}

			m.Stop()

		}))

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
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				body:        tt.body,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/images/",
			}

			m.Run()

			image, err := createImage(m.baseURI, testToken, tt.imageRef, bson.NewObjectId().Hex(), "No Description")

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if !reflect.DeepEqual(image, tt.expectImage) {
				t.Errorf("Got created collection %v - expected %v", image, tt.expectImage)
			}

			m.Stop()

		}))

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
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/tags/" + tt.containerRef,
			}

			m.Run()

			err := setTags(m.baseURI, testToken, tt.containerRef, tt.imageRef, tt.tags)

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}

			m.Stop()

		}))

	}
}

func Test_search(t *testing.T) {
	tests := []struct {
		description   string
		code          int
		body          interface{}
		reqCallback   func(*http.Request, *testing.T)
		value         string
		expectResults SearchResults
		expectError   bool
	}{
		{
			description:   "ValidRequest",
			value:         "test",
			code:          http.StatusOK,
			body:          JSONResponse{Data: testSearch, Error: JSONError{}},
			expectResults: testSearch,
			expectError:   false,
		},
		{
			description: "InternalServerError",
			value:       "test",
			code:        http.StatusInternalServerError,
			expectError: true,
		},
		{
			description: "BadRequest",
			value:       "test",
			code:        http.StatusBadRequest,
			expectError: true,
		},
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, test.WithoutPrivilege(func(t *testing.T) {

			m := mockService{
				t:           t,
				code:        tt.code,
				body:        tt.body,
				reqCallback: tt.reqCallback,
				httpPath:    "/v1/search",
			}

			m.Run()

			results, err := search(m.baseURI, testToken, tt.value)

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}
			if !reflect.DeepEqual(results, tt.expectResults) {
				t.Errorf("Got created collection %v - expected %v", results, tt.expectResults)
			}

			m.Stop()

		}))

	}
}
