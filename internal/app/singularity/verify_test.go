// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package singularity

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/sif/pkg/integrity"
	"github.com/sylabs/sif/pkg/sif"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

const (
	testFingerPrint    = "12045C8C0B1004D058DE4BEDA20C27EE7FF7BA84"
	invalidFingerPrint = "0000000000000000000000000000000000000000"
)

// getTestEntity returns a fixed test PGP entity.
func getTestEntity(t *testing.T) *openpgp.Entity {
	t.Helper()

	f, err := os.Open(filepath.Join("testdata", "keys", "private.asc"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	el, err := openpgp.ReadArmoredKeyRing(f)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(el), 1; got != want {
		t.Fatalf("got %v entities, want %v", got, want)
	}
	return el[0]
}

type mockHKP struct {
	e *openpgp.Entity
}

func (m mockHKP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/pgp-keys")

	wr, err := armor.Encode(w, openpgp.PublicKeyType, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer wr.Close()

	if err := m.e.Serialize(wr); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func Test_newVerifier(t *testing.T) {
	opts := []client.Option{client.OptBearerToken("token")}

	tests := []struct {
		name         string
		opts         []VerifyOpt
		wantErr      error
		wantVerifier verifier
	}{
		{
			name:         "Defaults",
			wantVerifier: verifier{},
		},
		{
			name:         "OptVerifyUseKeyServerOpts",
			opts:         []VerifyOpt{OptVerifyUseKeyServer(opts...)},
			wantVerifier: verifier{opts: opts},
		},
		{
			name:         "OptVerifyGroup",
			opts:         []VerifyOpt{OptVerifyGroup(1)},
			wantVerifier: verifier{groupIDs: []uint32{1}},
		},
		{
			name:         "OptVerifyObject",
			opts:         []VerifyOpt{OptVerifyObject(1)},
			wantVerifier: verifier{objectIDs: []uint32{1}},
		},
		{
			name:         "OptVerifyAll",
			opts:         []VerifyOpt{OptVerifyAll()},
			wantVerifier: verifier{all: true},
		},
		{
			name:         "OptVerifyLegacy",
			opts:         []VerifyOpt{OptVerifyLegacy()},
			wantVerifier: verifier{legacy: true},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			v, err := newVerifier(tt.opts)

			if got, want := err, tt.wantErr; !errors.Is(got, want) {
				t.Errorf("got error %v, want %v", got, want)
			}

			if got, want := v, tt.wantVerifier; !reflect.DeepEqual(got, want) {
				t.Errorf("got verifier %v, want %v", got, want)
			}
		})
	}
}

func Test_verifier_getOpts(t *testing.T) {
	emptyImage, err := sif.LoadContainer(filepath.Join("testdata", "images", "empty.sif"), true)
	if err != nil {
		t.Fatal(err)
	}
	defer emptyImage.UnloadContainer()

	oneGroupImage, err := sif.LoadContainer(filepath.Join("testdata", "images", "one-group.sif"), true)
	if err != nil {
		t.Fatal(err)
	}
	defer oneGroupImage.UnloadContainer()

	cb := func(*sif.FileImage, integrity.VerifyResult) bool { return false }

	tests := []struct {
		name     string
		v        verifier
		f        *sif.FileImage
		wantErr  error
		wantOpts int
	}{
		{
			name: "TLSRequired",
			f:    &emptyImage,
			v: verifier{
				opts: []client.Option{
					client.OptBaseURL("hkp://pool.sks-keyservers.net"),
					client.OptBearerToken("blah"),
				},
			},
			wantErr: client.ErrTLSRequired,
		},
		{
			name:    "NotFound",
			f:       &emptyImage,
			v:       verifier{legacy: true},
			wantErr: sif.ErrNotFound,
		},
		{
			name:     "Defaults",
			f:        &oneGroupImage,
			wantOpts: 1,
		},
		{
			name: "ClientConfig",
			v: verifier{
				opts: []client.Option{
					client.OptBearerToken("token"),
				},
			},
			f:        &oneGroupImage,
			wantOpts: 1,
		},
		{
			name:     "Group1",
			v:        verifier{groupIDs: []uint32{1}},
			f:        &oneGroupImage,
			wantOpts: 2,
		},
		{
			name:     "Object1",
			v:        verifier{objectIDs: []uint32{1}},
			f:        &oneGroupImage,
			wantOpts: 2,
		},
		{
			name:     "All",
			v:        verifier{all: true},
			f:        &oneGroupImage,
			wantOpts: 1,
		},
		{
			name:     "Legacy",
			v:        verifier{legacy: true},
			f:        &oneGroupImage,
			wantOpts: 3,
		},
		{
			name:     "LegacyGroup1",
			v:        verifier{legacy: true, groupIDs: []uint32{1}},
			f:        &oneGroupImage,
			wantOpts: 3,
		},
		{
			name:     "LegacyObject1",
			v:        verifier{legacy: true, objectIDs: []uint32{1}},
			f:        &oneGroupImage,
			wantOpts: 3,
		},
		{
			name:     "LegacyAll",
			v:        verifier{legacy: true, all: true},
			f:        &oneGroupImage,
			wantOpts: 2,
		},
		{
			name:     "Callcack",
			v:        verifier{cb: cb},
			f:        &oneGroupImage,
			wantOpts: 2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			opts, err := tt.v.getOpts(context.Background(), tt.f)

			if got, want := err, tt.wantErr; !errors.Is(got, want) {
				t.Errorf("got error %v, want %v", got, want)
			}

			if got, want := len(opts), tt.wantOpts; got != want {
				t.Errorf("got %v options, want %v", got, want)
			}
		})
	}
}

func TestVerify(t *testing.T) {
	// Start up a mock HKP server.
	e := getTestEntity(t)
	s := httptest.NewServer(mockHKP{e: e})
	defer s.Close()

	// Create an option that points to the mock HKP server.
	keyServerOpt := OptVerifyUseKeyServer(client.OptBaseURL(s.URL))

	tests := []struct {
		name         string
		path         string
		opts         []VerifyOpt
		wantVerified [][]uint32
		wantEntity   *openpgp.Entity
		wantErr      error
	}{
		{
			name:    "SignatureNotFound",
			path:    filepath.Join("testdata", "images", "one-group.sif"),
			opts:    []VerifyOpt{keyServerOpt},
			wantErr: &integrity.SignatureNotFoundError{},
		},
		{
			name:    "SignatureNotFoundNonLegacy",
			path:    filepath.Join("testdata", "images", "one-group-signed.sif"),
			opts:    []VerifyOpt{keyServerOpt, OptVerifyLegacy()},
			wantErr: &integrity.SignatureNotFoundError{},
		},
		{
			name:    "SignatureNotFoundLegacy",
			path:    filepath.Join("testdata", "images", "one-group-signed-legacy.sif"),
			opts:    []VerifyOpt{keyServerOpt},
			wantErr: &integrity.SignatureNotFoundError{},
		},
		{
			name:    "SignatureNotFoundLegacyAll",
			path:    filepath.Join("testdata", "images", "one-group-signed-legacy-all.sif"),
			opts:    []VerifyOpt{keyServerOpt},
			wantErr: &integrity.SignatureNotFoundError{},
		},
		{
			name:    "SignatureNotFoundLegacyGroup",
			path:    filepath.Join("testdata", "images", "one-group-signed-legacy-group.sif"),
			opts:    []VerifyOpt{keyServerOpt},
			wantErr: &integrity.SignatureNotFoundError{},
		},
		{
			name:         "Defaults",
			path:         filepath.Join("testdata", "images", "one-group-signed.sif"),
			opts:         []VerifyOpt{keyServerOpt},
			wantVerified: [][]uint32{{1, 2}},
			wantEntity:   e,
		},
		{
			name:         "OptVerifyGroup",
			path:         filepath.Join("testdata", "images", "one-group-signed.sif"),
			opts:         []VerifyOpt{keyServerOpt, OptVerifyGroup(1)},
			wantVerified: [][]uint32{{1, 2}},
			wantEntity:   e,
		},
		{
			name:         "OptVerifyObject",
			path:         filepath.Join("testdata", "images", "one-group-signed.sif"),
			opts:         []VerifyOpt{keyServerOpt, OptVerifyObject(1)},
			wantVerified: [][]uint32{{1}},
			wantEntity:   e,
		},
		{
			name:         "LegacyDefaults",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy.sif"),
			opts:         []VerifyOpt{keyServerOpt, OptVerifyLegacy()},
			wantVerified: [][]uint32{{2}},
			wantEntity:   e,
		},
		{
			name:         "LegacyOptVerifyObject",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy-all.sif"),
			opts:         []VerifyOpt{keyServerOpt, OptVerifyLegacy(), OptVerifyObject(1)},
			wantVerified: [][]uint32{{1}},
			wantEntity:   e,
		},
		{
			name:         "LegacyOptVerifyAll",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy-all.sif"),
			opts:         []VerifyOpt{keyServerOpt, OptVerifyLegacy(), OptVerifyAll()},
			wantVerified: [][]uint32{{1}, {2}},
			wantEntity:   e,
		},
		{
			name:         "LegacyOptVerifyGroup",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy-group.sif"),
			opts:         []VerifyOpt{keyServerOpt, OptVerifyLegacy(), OptVerifyGroup(1)},
			wantVerified: [][]uint32{{1, 2}},
			wantEntity:   e,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cb := func(f *sif.FileImage, r integrity.VerifyResult) bool {
				if len(tt.wantVerified) == 0 {
					t.Fatalf("wantVerified consumed")
				}
				if got, want := r.Verified(), tt.wantVerified[0]; !reflect.DeepEqual(got, want) {
					t.Errorf("got verified %v, want %v", got, want)
				}
				tt.wantVerified = tt.wantVerified[1:]

				if got, want := r.Entity().PrimaryKey, tt.wantEntity.PrimaryKey; !reflect.DeepEqual(got, want) {
					t.Errorf("got entity public key %+v, want %+v", got, want)
				}

				if got, want := r.Error(), tt.wantErr; !errors.Is(got, want) {
					t.Errorf("got error %v, want %v", got, want)
				}

				return false
			}
			tt.opts = append(tt.opts, OptVerifyCallback(cb))

			err := Verify(context.Background(), tt.path, tt.opts...)

			if got, want := err, tt.wantErr; !errors.Is(got, want) {
				t.Errorf("got error %v, want %v", got, want)
			}
		})
	}
}

func TestVerifyFingerPrint(t *testing.T) {
	// Start up a mock HKP server.
	e := getTestEntity(t)
	s := httptest.NewServer(mockHKP{e: e})
	defer s.Close()

	// Create an option that points to the mock HKP server.
	keyServerOpt := OptVerifyUseKeyServer(client.OptBaseURL(s.URL))

	tests := []struct {
		name         string
		path         string
		fingerprints []string
		opts         []VerifyOpt
		wantVerified [][]uint32
		wantEntity   *openpgp.Entity
		wantErr      error
	}{
		{
			name:         "SignatureNotFound",
			path:         filepath.Join("testdata", "images", "one-group.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt},
			wantErr:      &integrity.SignatureNotFoundError{},
		},
		{
			name:         "SignatureNotFoundNonLegacy",
			path:         filepath.Join("testdata", "images", "one-group-signed.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt, OptVerifyLegacy()},
			wantErr:      &integrity.SignatureNotFoundError{},
		},
		{
			name:         "SignatureNotFoundLegacy",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt},
			wantErr:      &integrity.SignatureNotFoundError{},
		},
		{
			name:         "SignatureNotFoundLegacyAll",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy-all.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt},
			wantErr:      &integrity.SignatureNotFoundError{},
		},
		{
			name:         "SignatureNotFoundLegacyGroup",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy-group.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt},
			wantErr:      &integrity.SignatureNotFoundError{},
		},
		{
			name:         "Defaults",
			path:         filepath.Join("testdata", "images", "one-group-signed.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt},
			wantVerified: [][]uint32{{1, 2}},
			wantEntity:   e,
		},
		{
			name:         "OptVerifyGroup",
			path:         filepath.Join("testdata", "images", "one-group-signed.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt, OptVerifyGroup(1)},
			wantVerified: [][]uint32{{1, 2}},
			wantEntity:   e,
		},
		{
			name:         "OptVerifyObject",
			path:         filepath.Join("testdata", "images", "one-group-signed.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt, OptVerifyObject(1)},
			wantVerified: [][]uint32{{1}},
			wantEntity:   e,
		},
		{
			name:         "LegacyDefaults",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt, OptVerifyLegacy()},
			wantVerified: [][]uint32{{2}},
			wantEntity:   e,
		},
		{
			name:         "LegacyOptVerifyObject",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy-all.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt, OptVerifyLegacy(), OptVerifyObject(1)},
			wantVerified: [][]uint32{{1}},
			wantEntity:   e,
		},
		{
			name:         "LegacyOptVerifyAll",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy-all.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt, OptVerifyLegacy(), OptVerifyAll()},
			wantVerified: [][]uint32{{1}, {2}},
			wantEntity:   e,
		},
		{
			name:         "LegacyOptVerifyGroup",
			path:         filepath.Join("testdata", "images", "one-group-signed-legacy-group.sif"),
			fingerprints: []string{testFingerPrint},
			opts:         []VerifyOpt{keyServerOpt, OptVerifyLegacy(), OptVerifyGroup(1)},
			wantVerified: [][]uint32{{1, 2}},
			wantEntity:   e,
		},
		{
			name:         "SingleFingerprintWrong",
			path:         filepath.Join("testdata", "images", "one-group-signed.sif"),
			fingerprints: []string{invalidFingerPrint},
			opts:         []VerifyOpt{keyServerOpt},
			wantVerified: [][]uint32{{1, 2}},
			wantEntity:   e,
			wantErr:      errNotSignedByRequired,
		},
		{
			name:         "MultipleFingerprintOneWrong",
			path:         filepath.Join("testdata", "images", "one-group-signed.sif"),
			fingerprints: []string{testFingerPrint, invalidFingerPrint},
			opts:         []VerifyOpt{keyServerOpt},
			wantVerified: [][]uint32{{1, 2}},
			wantEntity:   e,
			wantErr:      errNotSignedByRequired,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cb := func(f *sif.FileImage, r integrity.VerifyResult) bool {
				if len(tt.wantVerified) == 0 {
					t.Fatalf("wantVerified consumed")
				}
				if got, want := r.Verified(), tt.wantVerified[0]; !reflect.DeepEqual(got, want) {
					t.Errorf("got verified %v, want %v", got, want)
				}
				tt.wantVerified = tt.wantVerified[1:]

				if got, want := r.Entity().PrimaryKey, tt.wantEntity.PrimaryKey; !reflect.DeepEqual(got, want) {
					t.Errorf("got entity public key %+v, want %+v", got, want)
				}

				return false
			}
			tt.opts = append(tt.opts, OptVerifyCallback(cb))
			err := VerifyFingerprints(context.Background(), tt.path, tt.fingerprints, tt.opts...)
			if got, want := err, tt.wantErr; !errors.Is(got, want) {
				t.Errorf("got error %v, want %v", got, want)
			}
		})
	}
}
