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
	invalidPemPath = "nothing"
	testPassphrase = "test"
)

func TestNewPlaintextKey(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name          string
		keyInfo       KeyInfo
		expectedError error
	}{
		{
			name:          "unknown format",
			keyInfo:       KeyInfo{Format: Unknown},
			expectedError: ErrUnsupportedKeyURI,
		},
		{
			name:          "passphrase",
			keyInfo:       KeyInfo{Format: Passphrase, Material: testPassphrase},
			expectedError: nil,
		},
		{
			name:          "invalid pem",
			keyInfo:       KeyInfo{Format: PEM, Path: invalidPemPath},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPlaintextKey(tt.keyInfo)
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
		keyInfo       KeyInfo
		plaintext     []byte
		expectedError error
	}{
		{
			name:          "unknown format",
			keyInfo:       KeyInfo{Format: Unknown},
			plaintext:     []byte(""),
			expectedError: ErrUnsupportedKeyURI,
		},
		{
			name:          "passphrase",
			keyInfo:       KeyInfo{Format: Passphrase, Material: testPassphrase},
			plaintext:     []byte(""),
			expectedError: nil,
		},
		{
			name:          "invalid pem",
			keyInfo:       KeyInfo{Format: PEM, Path: invalidPemPath},
			plaintext:     []byte(""),
			expectedError: errors.Wrap(fmt.Errorf("open nothing: no such file or directory"), "loading public key for key encryption"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncryptKey(tt.keyInfo, tt.plaintext)
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
		keyInfo       KeyInfo
		expectedError error
	}{
		{
			name:          "unknown format",
			keyInfo:       KeyInfo{Format: Unknown},
			expectedError: ErrUnsupportedKeyURI,
		},
		{
			name:          "passphrase",
			keyInfo:       KeyInfo{Format: Passphrase, Material: testPassphrase},
			expectedError: nil,
		},
		{
			name:          "invalid pem",
			keyInfo:       KeyInfo{Format: PEM, Path: invalidPemPath},
			expectedError: errors.Wrap(fmt.Errorf("open nothing: no such file or directory"), "loading private key for key decryption"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PlaintextKey(tt.keyInfo, noimage)
			// We do not always use predefined errors so when dealing with errors, we compare the text associated
			// to the error.
			if (err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error()) ||
				((err == nil || tt.expectedError == nil) && err != tt.expectedError) {
				t.Fatalf("test %s returned an unexpected error: %s vs. %s", tt.name, err, tt.expectedError)
			}
		})
	}
}
