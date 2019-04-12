// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sypgp

import (
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

const (
	testName    = "Test Name"
	testComment = "blah"
	testEmail   = "test@test.com"
)

var (
	testEntity *openpgp.Entity
)

type mockPKSLookup struct {
	code int
	el   openpgp.EntityList
}

func (ms *mockPKSLookup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(ms.code)
	if ms.code == http.StatusOK {
		w.Header().Set("Content-Type", "application/pgp-keys")

		wr, err := armor.Encode(w, openpgp.PublicKeyType, nil)
		if err != nil {
			log.Fatalf("failed to get encoder: %v", err)
		}
		defer wr.Close()

		for _, e := range ms.el {
			if err = e.Serialize(wr); err != nil {
				log.Fatalf("failed to serialize entity: %v", err)
			}
		}
	}
}

func TestSearchPubkey(t *testing.T) {
	ms := &mockPKSLookup{}
	srv := httptest.NewServer(ms)
	defer srv.Close()

	tests := []struct {
		name      string
		code      int
		el        openpgp.EntityList
		search    string
		uri       string
		authToken string
		wantErr   bool
	}{
		{"Success", http.StatusOK, openpgp.EntityList{testEntity}, "search", srv.URL, "", false},
		{"SuccessToken", http.StatusOK, openpgp.EntityList{testEntity}, "search", srv.URL, "token", false},
		{"BadURL", http.StatusOK, openpgp.EntityList{testEntity}, "search", ":", "", true},
		{"TerribleURL", http.StatusOK, openpgp.EntityList{testEntity}, "search", "terrible:", "", true},
		{"NotFound", http.StatusNotFound, nil, "search", srv.URL, "", true},
		{"Unauthorized", http.StatusUnauthorized, nil, "search", srv.URL, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms.code = tt.code
			ms.el = tt.el

			if err := SearchPubkey(tt.search, tt.uri, tt.authToken); (err != nil) != tt.wantErr {
				t.Fatalf("got err %v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetchPubkey(t *testing.T) {
	ms := &mockPKSLookup{}
	srv := httptest.NewServer(ms)
	defer srv.Close()

	fp := hex.EncodeToString(testEntity.PrimaryKey.Fingerprint[:])

	tests := []struct {
		name        string
		code        int
		el          openpgp.EntityList
		fingerprint string
		uri         string
		authToken   string
		wantErr     bool
	}{
		{"Success", http.StatusOK, openpgp.EntityList{testEntity}, fp, srv.URL, "", false},
		{"SuccessToken", http.StatusOK, openpgp.EntityList{testEntity}, fp, srv.URL, "token", false},
		{"NoKeys", http.StatusOK, openpgp.EntityList{}, fp, srv.URL, "token", true},
		{"TwoKeys", http.StatusOK, openpgp.EntityList{testEntity, testEntity}, fp, srv.URL, "token", true},
		{"BadURL", http.StatusOK, openpgp.EntityList{testEntity}, fp, ":", "", true},
		{"TerribleURL", http.StatusOK, openpgp.EntityList{testEntity}, fp, "terrible:", "", true},
		{"NotFound", http.StatusNotFound, nil, fp, srv.URL, "", true},
		{"Unauthorized", http.StatusUnauthorized, nil, fp, srv.URL, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms.code = tt.code
			ms.el = tt.el

			el, err := FetchPubkey(tt.fingerprint, tt.uri, tt.authToken, false)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
				return
			}

			if !tt.wantErr {
				if len(el) != 1 {
					t.Fatalf("unexpected number of entities returned: %v", len(el))
				}
				for i := range tt.el {
					if fp := el[i].PrimaryKey.Fingerprint; fp != tt.el[i].PrimaryKey.Fingerprint {
						t.Errorf("fingerprint mismatch: %v / %v", fp, tt.el[i].PrimaryKey.Fingerprint)
					}
				}
			}
		})
	}
}

type mockPKSAdd struct {
	t       *testing.T
	keyText string
	code    int
}

func (m *mockPKSAdd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.code != http.StatusOK {
		w.WriteHeader(m.code)
		return
	}

	if got, want := r.Header.Get("Content-Type"), "application/x-www-form-urlencoded"; got != want {
		m.t.Errorf("got content type %v, want %v", got, want)
	}

	if err := r.ParseForm(); err != nil {
		m.t.Fatalf("failed to parse form: %v", err)
	}
	if got, want := r.Form.Get("keytext"), m.keyText; got != want {
		m.t.Errorf("got key text %v, want %v", got, want)
	}
}

func TestPushPubkey(t *testing.T) {
	keyText, err := serializeEntity(testEntity, openpgp.PublicKeyType)
	if err != nil {
		t.Fatalf("failed to serialize entity: %v", err)
	}

	ms := &mockPKSAdd{
		t:       t,
		keyText: keyText,
	}
	srv := httptest.NewServer(ms)
	defer srv.Close()

	tests := []struct {
		name      string
		uri       string
		authToken string
		code      int
		wantErr   bool
	}{
		{"Success", srv.URL, "", http.StatusOK, false},
		{"SuccessToken", srv.URL, "token", http.StatusOK, false},
		{"BadURL", ":", "", http.StatusOK, true},
		{"TerribleURL", "terrible:", "", http.StatusOK, true},
		{"Unauthorized", srv.URL, "", http.StatusUnauthorized, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms.code = tt.code

			if err := PushPubkey(testEntity, tt.uri, tt.authToken); (err != nil) != tt.wantErr {
				t.Fatalf("got err %v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnsureDirPrivate(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-ensure-dir-private")
	if err != nil {
		t.Fatalf("Cannot create temporary directory")
	}
	defer os.RemoveAll(tmpdir)

	cases := []struct {
		name        string
		preFunc     func(string) error
		entry       string
		expectError bool
	}{
		{
			name:        "non-existent directory",
			entry:       "d1",
			expectError: false,
		},
		{
			name:        "directory exists",
			entry:       "d2",
			expectError: false,
			preFunc: func(d string) error {
				if err := os.MkdirAll(d, 0777); err != nil {
					return err
				}
				return os.Chmod(d, 0777)
			},
		},
	}

	for _, tc := range cases {
		testEntry := filepath.Join(tmpdir, tc.entry)

		if tc.preFunc != nil {
			if err := tc.preFunc(testEntry); err != nil {
				t.Errorf("Unexpected failure when calling prep function: %+v", err)
				continue
			}
		}

		err := ensureDirPrivate(testEntry)
		switch {
		case !tc.expectError && err != nil:
			t.Errorf("Unexpected failure when calling ensureDirPrivate(%q): %+v", testEntry, err)

		case tc.expectError && err == nil:
			t.Errorf("Expecting failure when calling ensureDirPrivate(%q), but got none", testEntry)

		case tc.expectError && err != nil:
			t.Logf("Expecting failure when calling ensureDirPrivate(%q), got: %+v", testEntry, err)
			continue

		case !tc.expectError && err == nil:
			// everything ok

			fi, err := os.Stat(testEntry)
			if err != nil {
				t.Errorf("Error while examining test directory %q: %+v", testEntry, err)
			}

			if !fi.IsDir() {
				t.Errorf("Expecting a directory after calling ensureDirPrivate(%q), found something else", testEntry)
			}

			if actual, expected := fi.Mode() & ^os.ModeDir, os.FileMode(0700); actual != expected {
				t.Errorf("Expecting mode %o, got %o after calling ensureDirPrivate(%q)", expected, actual, testEntry)
			}
		}
	}
}

func TestEnsureFilePrivate(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-ensure-file-private")
	if err != nil {
		t.Fatalf("Cannot create temporary directory")
	}
	defer os.RemoveAll(tmpdir)

	cases := []struct {
		name        string
		preFunc     func(string) error
		entry       string
		expectError bool
	}{
		{
			name:        "non-existent file",
			entry:       "f1",
			expectError: false,
		},
		{
			name:        "file exists",
			entry:       "f2",
			expectError: false,
			preFunc: func(fn string) error {
				fh, err := os.Create(fn)
				if err != nil {
					return err
				}
				fh.Close()
				return os.Chmod(fn, 0666)
			},
		},
	}

	for _, tc := range cases {
		testEntry := filepath.Join(tmpdir, tc.entry)

		if tc.preFunc != nil {
			if err := tc.preFunc(testEntry); err != nil {
				t.Errorf("Unexpected failure when calling prep function: %+v", err)
				continue
			}
		}

		err := ensureFilePrivate(testEntry)
		switch {
		case !tc.expectError && err != nil:
			t.Errorf("Unexpected failure when calling ensureFilePrivate(%q): %+v", testEntry, err)

		case tc.expectError && err == nil:
			t.Errorf("Expecting failure when calling ensureFilePrivate(%q), but got none", testEntry)

		case tc.expectError && err != nil:
			t.Logf("Expecting failure when calling ensureFilePrivate(%q), got: %+v", testEntry, err)
			continue

		case !tc.expectError && err == nil:
			// everything ok

			fi, err := os.Stat(testEntry)
			if err != nil {
				t.Errorf("Error while examining test directory %q: %+v", testEntry, err)
			}

			if fi.IsDir() {
				t.Errorf("Expecting a non-directory after calling ensureFilePrivate(%q), found something else", testEntry)
			}

			if actual, expected := fi.Mode(), os.FileMode(0600); actual != expected {
				t.Errorf("Expecting mode %o, got %o after calling ensureFilePrivate(%q)", expected, actual, testEntry)
			}
		}
	}
}

func TestMain(m *testing.M) {
	useragent.InitValue("singularity", "3.0.0-alpha.1-303-gaed8d30-dirty")

	e, err := openpgp.NewEntity(testName, testComment, testEmail, nil)
	if err != nil {
		log.Fatalf("failed to create entity: %v", err)
	}
	testEntity = e

	os.Exit(m.Run())
}
