// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package crypt

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/sylabs/singularity/internal/pkg/test"
)

const (
	unsupportedURI = "test://justarandominvaliduri"
	invalidPem     = "pem://nothing"
)

func TestNewPlaintextKey(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name          string
		keyURI        string
		expectedError error
	}{
		{
			name:          "empty URI",
			keyURI:        "",
			expectedError: nil,
		},
		{
			name:          "unsupported URI",
			keyURI:        unsupportedURI,
			expectedError: ErrUnsupportedKeyURI,
		},
		{
			name:          "invalid pem",
			keyURI:        invalidPem,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPlaintextKey(tt.keyURI)
			// We do not always use predefined errors so when dealing with errors, we compare the text associated
			// to the error.
			if (err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error()) ||
				((err == nil || tt.expectedError == nil) && err != tt.expectedError) {
				t.Fatalf("test %s returned an unexpected error: %s vs. %s", tt.name, err, tt.expectedError)
			}
		})
	}
}

func TestEncryptKey(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name          string
		keyURI        string
		plaintext     []byte
		expectedError error
	}{
		{
			name:          "empty URI",
			keyURI:        "",
			plaintext:     []byte(""),
			expectedError: nil,
		},
		{
			name:          "unsupported URI",
			keyURI:        unsupportedURI,
			plaintext:     []byte(""),
			expectedError: ErrUnsupportedKeyURI,
		},
		{
			name:          "invalid pem",
			keyURI:        invalidPem,
			plaintext:     []byte(""),
			expectedError: errors.Wrap(fmt.Errorf("open : no such file or directory"), "loading public key for key encryption"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncryptKey(tt.keyURI, tt.plaintext)
			// We do not always use predefined errors so when dealing with errors, we compare the text associated
			// to the error.
			if (err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error()) ||
				((err == nil || tt.expectedError == nil) && err != tt.expectedError) {
				t.Fatalf("test %s returned an unexpected error: %s vs. %s", tt.name, err, tt.expectedError)
			}
		})
	}
}

func TestPlaintextKey(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// TestPlaintestKey reads a key from an image. Creating an image does not
	// fit with unit tests testing so we only test error cases here.
	const (
		noimage = ""
	)

	tests := []struct {
		name          string
		keyURI        string
		expectedError error
	}{
		{
			name:          "empty URI",
			keyURI:        "",
			expectedError: nil,
		},
		{
			name:          "unsupported URI",
			keyURI:        unsupportedURI,
			expectedError: ErrUnsupportedKeyURI,
		},
		{
			name:          "invalid pem",
			keyURI:        invalidPem,
			expectedError: errors.Wrap(fmt.Errorf("open : no such file or directory"), "loading private key for key decryption"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PlaintextKey(tt.keyURI, noimage)
			// We do not always use predefined errors so when dealing with errors, we compare the text associated
			// to the error.
			if (err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error()) ||
				((err == nil || tt.expectedError == nil) && err != tt.expectedError) {
				t.Fatalf("test %s returned an unexpected error: %s vs. %s", tt.name, err, tt.expectedError)
			}
		})
	}
}
