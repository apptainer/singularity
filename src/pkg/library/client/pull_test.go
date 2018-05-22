// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

type mockRawService struct {
	t           *testing.T
	code        int
	testFile    string
	reqCallback func(*http.Request, *testing.T)
	httpAddr    string
	httpPath    string
	httpServer  *httptest.Server
	baseURI     string
}

func (m *mockRawService) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc(m.httpPath, m.ServeHTTP)
	m.httpServer = httptest.NewServer(mux)
	m.httpAddr = m.httpServer.Listener.Addr().String()
	m.baseURI = "http://" + m.httpAddr
}

func (m *mockRawService) Stop() {
	m.httpServer.Close()
}

func (m *mockRawService) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if m.reqCallback != nil {
		m.reqCallback(r, m.t)
	}
	w.WriteHeader(m.code)
	inFile, err := os.Open(m.testFile)
	if err != nil {
		m.t.Errorf("error opening file %v:", err)
	}
	defer inFile.Close()

	_, err = io.Copy(w, bufio.NewReader(inFile))
	if err != nil {
		sylog.Fatalf("Test HTTP server unable to output file: %v", err)
	}

}

func Test_DownloadImage(t *testing.T) {

	f, err := ioutil.TempFile(".", "test")
	if err != nil {
		t.Fatalf("Error creating a temporary file for testing")
	}
	tempFile := f.Name()
	f.Close()
	os.Remove(tempFile)

	tests := []struct {
		name         string
		libraryRef   string
		outFile      string
		force        bool
		code         int
		testFile     string
		tokenFile    string
		checkContent bool
		expectError  bool
	}{
		{"Bad filename", "entity/collection/image:tag", "notadir/test.sif", false, http.StatusBadRequest, "test_data/test_sha256", "test_data/test_token", false, true},
		{"Bad library ref", "entity/collection/im,age:tag", tempFile, false, http.StatusBadRequest, "test_data/test_sha256", "test_data/test_token", false, true},
		{"Server error", "entity/collection/image:tag", tempFile, false, http.StatusInternalServerError, "test_data/test_sha256", "test_data/test_token", false, true},
		{"Good Download", "entity/collection/image:tag", tempFile, false, http.StatusOK, "test_data/test_sha256", "test_data/test_token", true, false},
		{"Should not overwrite", "entity/collection/image:tag", tempFile, false, http.StatusOK, "test_data/test_sha256", "test_data/test_token", true, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			m := mockRawService{
				t:        t,
				code:     test.code,
				testFile: test.testFile,
				httpPath: "/v1/imagefile/" + test.libraryRef,
			}

			m.Run()
			defer m.Stop()

			err := DownloadImage(test.outFile, test.libraryRef, m.baseURI, test.force, test.tokenFile)

			if err != nil && !test.expectError {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectError {
				t.Errorf("Unexpected success. Expected error.")
			}

			if test.checkContent {
				fileContent, err := ioutil.ReadFile(test.outFile)
				if err != nil {
					t.Errorf("Error reading test output file: %v", err)
				}
				testContent, err := ioutil.ReadFile(test.testFile)
				if err != nil {
					t.Errorf("Error reading test file: %v", err)
				}
				if !bytes.Equal(fileContent, testContent) {
					t.Errorf("File contains '%v' - expected '%v'", fileContent, testContent)
				}
			}

		})
	}
}
