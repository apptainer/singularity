// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"testing"
)

const (
	testSearchOutput = `Found 1 users for 'test'
	library://test-user

Found 1 collections for 'test'
	library://test-user/test-collection

Found 1 containers for 'test'
	library://test-user/test-collection/test-container
		Tags: latest test-tag

`

	testSearchOutputEmpty = `No users found for 'test'

No collections found for 'test'

No containers found for 'test'

`
)

func Test_SearchLibrary(t *testing.T) {
	m := mockService{
		t:        t,
		code:     http.StatusOK,
		body:     JSONResponse{Data: testSearch, Error: JSONError{}},
		httpPath: "/v1/search",
	}

	m.Run()
	defer m.Stop()

	err := SearchLibrary("a", m.baseURI, "")
	if err == nil {
		t.Errorf("Search of 1 character shouldn't be submitted")
	}
	err = SearchLibrary("ab", m.baseURI, "")
	if err == nil {
		t.Errorf("Search of 2 characters shouldn't be submitted")
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = SearchLibrary("test", m.baseURI, "")

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	w.Close()
	os.Stdout = old
	out := <-outC

	if err != nil {
		t.Errorf("Search of test should succeed")
	}
	log.SetOutput(os.Stderr)

	if out != testSearchOutput {
		t.Errorf("Output of search not as expected")
		t.Errorf("=== EXPECTED ===")
		t.Errorf(testSearchOutput)
		t.Errorf("=== ACTUAL ===")
		t.Errorf(out)
	}
}

func Test_SearchLibraryEmpty(t *testing.T) {
	m := mockService{
		t:        t,
		code:     http.StatusOK,
		body:     JSONResponse{Data: SearchResults{}, Error: JSONError{}},
		httpPath: "/v1/search",
	}

	m.Run()
	defer m.Stop()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := SearchLibrary("test", m.baseURI, "")

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	w.Close()
	os.Stdout = old
	out := <-outC

	if err != nil {
		t.Errorf("Search of test should succeed")
	}
	log.SetOutput(os.Stderr)

	if out != testSearchOutputEmpty {
		t.Errorf("Output of search not as expected")
		t.Errorf("=== EXPECTED ===")
		t.Errorf(testSearchOutputEmpty)
		t.Errorf("=== ACTUAL ===")
		t.Errorf(out)
	}
}
