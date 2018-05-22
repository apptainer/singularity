// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"net/http"
	"testing"

	"github.com/globalsign/mgo/bson"
)

//func postFile(baseURL string, filePath string, imageID string) error {
func Test_postFile(t *testing.T) {

	tests := []struct {
		description string
		imageRef    string
		testFile    string
		code        int
		reqCallback func(*http.Request, *testing.T)
		expectError bool
	}{
		{
			description: "Container not found response",
			code:        404,
			reqCallback: nil,
			imageRef:    bson.NewObjectId().Hex(),
			testFile:    "test_data/test_sha256",
			expectError: true,
		},
		{
			description: "Unauthorized response",
			code:        401,
			reqCallback: nil,
			imageRef:    bson.NewObjectId().Hex(),
			testFile:    "test_data/test_sha256",
			expectError: true,
		},
		{
			description: "Valid Response",
			code:        200,
			reqCallback: nil,
			imageRef:    bson.NewObjectId().Hex(),
			testFile:    "test_data/test_sha256",
			expectError: false,
		},
	}

	// Loop over test cases
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {

			m := mockService{
				t:        t,
				code:     test.code,
				httpPath: "/v1/imagefile/" + test.imageRef,
			}

			m.Run()

			err := postFile(m.baseURI, testToken, test.testFile, test.imageRef)

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
