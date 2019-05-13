// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sypgp

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
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
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

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
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

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

func TestPrintEntity(t *testing.T) {
	getPublicKey := func(data string) *packet.PublicKey {
		pkt, err := packet.Read(readerFromHex(data))
		if err != nil {
			panic(err)
		}

		pk, ok := pkt.(*packet.PublicKey)
		if !ok {
			panic("expecting packet.PublicKey, got something else")
		}

		return pk
	}

	cases := []struct {
		name     string
		index    int
		entity   *openpgp.Entity
		expected string
	}{
		{
			name:  "zero value",
			index: 0,
			entity: &openpgp.Entity{
				PrimaryKey: &packet.PublicKey{},
				Identities: map[string]*openpgp.Identity{
					"": {
						UserId: &packet.UserId{},
					},
				},
			},
			expected: "0) U:  () <>\n   C: 0001-01-01 00:00:00 +0000 UTC\n   F: 0000000000000000000000000000000000000000\n   L: 0\n",
		},
		{
			name:  "RSA key",
			index: 1,
			entity: &openpgp.Entity{
				PrimaryKey: getPublicKey(rsaPkDataHex),
				Identities: map[string]*openpgp.Identity{
					"name": {
						UserId: &packet.UserId{
							Name:    "name 1",
							Comment: "comment 1",
							Email:   "email.1@example.org",
						},
					},
				},
			},
			expected: "1) U: name 1 (comment 1) <email.1@example.org>\n   C: 2011-01-23 16:49:20 +0000 UTC\n   F: 5FB74B1D03B1E3CB31BC2F8AA34D7E18C20C31BB\n   L: 1024\n",
		},
		{
			name:  "DSA key",
			index: 2,
			entity: &openpgp.Entity{
				PrimaryKey: getPublicKey(dsaPkDataHex),
				Identities: map[string]*openpgp.Identity{
					"name": {
						UserId: &packet.UserId{
							Name:    "name 2",
							Comment: "comment 2",
							Email:   "email.2@example.org",
						},
					},
				},
			},
			expected: "2) U: name 2 (comment 2) <email.2@example.org>\n   C: 2011-01-28 21:05:13 +0000 UTC\n   F: EECE4C094DB002103714C63C8E8FBE54062F19ED\n   L: 1024\n",
		},
		{
			name:  "ECDSA key",
			index: 3,
			entity: &openpgp.Entity{
				PrimaryKey: getPublicKey(ecdsaPkDataHex),
				Identities: map[string]*openpgp.Identity{
					"name": {
						UserId: &packet.UserId{
							Name:    "name 3",
							Comment: "comment 3",
							Email:   "email.3@example.org",
						},
					},
				},
			},
			expected: "3) U: name 3 (comment 3) <email.3@example.org>\n   C: 2012-10-07 17:57:40 +0000 UTC\n   F: 9892270B38B8980B05C8D56D43FE956C542CA00B\n   L: 0\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b bytes.Buffer

			printEntity(&b, tc.index, tc.entity)

			if actual := b.String(); actual != tc.expected {
				t.Errorf("Unexpected output from printEntity: expecting %q, got %q",
					tc.expected,
					actual)
			}
		})
	}
}

func TestPrintEntities(t *testing.T) {
	getPublicKey := func(data string) *packet.PublicKey {
		pkt, err := packet.Read(readerFromHex(data))
		if err != nil {
			panic(err)
		}

		pk, ok := pkt.(*packet.PublicKey)
		if !ok {
			panic("expecting packet.PublicKey, got something else")
		}

		return pk
	}

	entities := []*openpgp.Entity{
		{
			PrimaryKey: getPublicKey(rsaPkDataHex),
			Identities: map[string]*openpgp.Identity{
				"name": {
					UserId: &packet.UserId{
						Name:    "name 1",
						Comment: "comment 1",
						Email:   "email.1@example.org",
					},
				},
			},
		},
		{
			PrimaryKey: getPublicKey(dsaPkDataHex),
			Identities: map[string]*openpgp.Identity{
				"name": {
					UserId: &packet.UserId{
						Name:    "name 2",
						Comment: "comment 2",
						Email:   "email.2@example.org",
					},
				},
			},
		},
		{
			PrimaryKey: getPublicKey(ecdsaPkDataHex),
			Identities: map[string]*openpgp.Identity{
				"name": {
					UserId: &packet.UserId{
						Name:    "name 3",
						Comment: "comment 3",
						Email:   "email.3@example.org",
					},
				},
			},
		},
	}

	expected := "0) U: name 1 (comment 1) <email.1@example.org>\n   C: 2011-01-23 16:49:20 +0000 UTC\n   F: 5FB74B1D03B1E3CB31BC2F8AA34D7E18C20C31BB\n   L: 1024\n" +
		"   --------\n" +
		"1) U: name 2 (comment 2) <email.2@example.org>\n   C: 2011-01-28 21:05:13 +0000 UTC\n   F: EECE4C094DB002103714C63C8E8FBE54062F19ED\n   L: 1024\n" +
		"   --------\n" +
		"2) U: name 3 (comment 3) <email.3@example.org>\n   C: 2012-10-07 17:57:40 +0000 UTC\n   F: 9892270B38B8980B05C8D56D43FE956C542CA00B\n   L: 0\n" +
		"   --------\n"

	var b bytes.Buffer

	printEntities(&b, entities)

	if actual := b.String(); actual != expected {
		t.Errorf("Unexpected output from printEntities: expecting %q, got %q",
			expected,
			actual)
	}
}

func TestMain(m *testing.M) {
	// Set TZ to UTC so that the code converting a time.Time value
	// to a string produces consistent output.
	if err := os.Setenv("TZ", "UTC"); err != nil {
		log.Fatalf("Cannot set timezone: %v", err)
	}

	useragent.InitValue("singularity", "3.0.0-alpha.1-303-gaed8d30-dirty")

	e, err := openpgp.NewEntity(testName, testComment, testEmail, nil)
	if err != nil {
		log.Fatalf("failed to create entity: %v", err)
	}
	testEntity = e

	os.Exit(m.Run())
}
