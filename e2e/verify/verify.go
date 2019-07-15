// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package verify

import (
	"testing"

	"github.com/buger/jsonparser"
	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

func getNameJSON(keyNum string) []string {
	return []string{"SignerKeys", keyNum, "Signer", "Name"}
}

func getFingerprintJSON(keyNum string) []string {
	return []string{"SignerKeys", keyNum, "Signer", "Fingerprint"}
}

func getLocalJSON(keyNum string) []string {
	return []string{"SignerKeys", keyNum, "Signer", "Local"}
}

func getKeyCheckJSON(keyNum string) []string {
	return []string{"SignerKeys", keyNum, "Signer", "KeyCheck"}
}

func getDataCheckJSON(keyNum string) []string {
	return []string{"SignerKeys", keyNum, "Signer", "DataCheck"}
}

func (c *ctx) singularityVerifyKeyNum(t *testing.T) {
	keyNumPath := []string{"Signatures"}

	tests := []struct {
		name         string
		expectNumOut int64  // Is the expected number of Signatures
		cpath        string // Is the path to the container
		expectExit   int
	}{
		{
			name:         "verify number signers",
			expectNumOut: 3,
			cpath:        "testdata/verify_container_corrupted.sif",
			expectExit:   255,
		},
		{
			name:         "verify number signers success container",
			expectNumOut: 1,
			cpath:        "testdata/verify_container_success.sif",
			expectExit:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyOutput := func(t *testing.T, r *e2e.SingularityCmdResult) {
				// Get the Signatures and compare it
				eNum, err := jsonparser.GetInt(r.Stdout, keyNumPath...)
				if err != nil {
					t.Fatalf("unable to get expected output from json: %s", err)
				}
				if eNum != tt.expectNumOut {
					t.Fatalf("unexpected failure: got: '%s', expecting: '%s'", eNum, tt.expectNumOut)
				}
			}

			// Inspect the container, and get the output
			e2e.RunSingularity(
				t,
				tt.name,
				e2e.WithPrivileges(false),
				e2e.WithCommand("verify"),
				e2e.WithArgs("--json", tt.cpath),
				e2e.ExpectExit(tt.expectExit, verifyOutput),
			)
		})
	}
}

func (c *ctx) singularityVerifySigner(t *testing.T) {
	tests := []struct {
		name                 string
		jsonPath             []string
		keyNum               string // Is the number of which key to test. Must be in '[]' bracket
		cpath                string // Is the path to the container
		expectNameOut        string // The expected out for Name
		expectFingerprintOut string // The expected out for Fingerprint
		expectLocalOut       bool   // The expected out for Local
		expectKeyCheckOut    bool   // The expected out for KeyCheck
		expectDataCheckOut   bool   // The expected out for DataCheck
		expectExit           int
	}{
		// Signer 0
		{
			name:                 "verify signer 0",
			keyNum:               "[0]",
			expectNameOut:        "unknown",
			expectFingerprintOut: "8883491F4268F173C6E5DC49EDECE4F3F38D871E",
			expectLocalOut:       false,
			expectKeyCheckOut:    true,
			expectDataCheckOut:   false,
			cpath:                "testdata/verify_container_corrupted.sif",
			expectExit:           255,
		},

		// Signer 1
		{
			name:                 "verify signer 1",
			keyNum:               "[1]",
			expectNameOut:        "westleyk (examples) \u003cwestley@sylabs.io\u003e",
			expectFingerprintOut: "4E28E95609D65D3C5BEA9731F1E47D55A7F3A56C",
			expectLocalOut:       false,
			expectKeyCheckOut:    true,
			expectDataCheckOut:   true,
			cpath:                "testdata/verify_container_corrupted.sif",
			expectExit:           255,
		},

		// Signer 2
		{
			name:                 "verify signer 2",
			keyNum:               "[2]",
			expectNameOut:        "unknown",
			expectFingerprintOut: "C7E7C8C3635DD06930669A2283B30190FEEF8162",
			expectLocalOut:       false,
			expectKeyCheckOut:    false,
			expectDataCheckOut:   false,
			cpath:                "testdata/verify_container_corrupted.sif",
			expectExit:           255,
		},

		// Verify 'verify_container_success.sif'
		{
			name:                 "verify success container",
			keyNum:               "[0]",
			expectNameOut:        "WestleyK (Testing key; used for signing test containers) \u003cwestley@sylabs.io\u003e",
			expectFingerprintOut: "7605BC2716168DF057D6C600ACEEC62C8BD91BEE",
			expectLocalOut:       false,
			expectKeyCheckOut:    true,
			expectDataCheckOut:   true,
			cpath:                "testdata/verify_container_success.sif",
			expectExit:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifyOutput := func(t *testing.T, r *e2e.SingularityCmdResult) {
				eName, err := jsonparser.GetString(r.Stdout, getNameJSON(tt.keyNum)...)
				if err != nil {
					t.Fatalf("unable to get expected output from json: %s", err)
				}
				if eName != tt.expectNameOut {
					t.Fatalf("unexpected failure: got: '%s', expecting: '%s'", eName, tt.expectNameOut)
				}

				// Get the Fingerprint and compare it
				eFingerprint, err := jsonparser.GetString(r.Stdout, getFingerprintJSON(tt.keyNum)...)
				if err != nil {
					t.Fatalf("unable to get expected output from json: %s", err)
				}
				if eFingerprint != tt.expectFingerprintOut {
					t.Fatalf("unexpected failure: got: '%s', expecting: '%s'", eFingerprint, tt.expectFingerprintOut)
				}

				// Get the Local and compare it
				eLocal, err := jsonparser.GetBoolean(r.Stdout, getLocalJSON(tt.keyNum)...)
				if err != nil {
					t.Fatalf("unable to get expected output from json: %s", err)
				}
				if eLocal != tt.expectLocalOut {
					t.Fatalf("unexpected failure: got: '%v', expecting: '%v'", eLocal, tt.expectLocalOut)
				}

				// Get the KeyCheck and compare it
				eKeyCheck, err := jsonparser.GetBoolean(r.Stdout, getKeyCheckJSON(tt.keyNum)...)
				if err != nil {
					t.Fatalf("unable to get expected output from json: %s", err)
				}
				if eKeyCheck != tt.expectKeyCheckOut {
					t.Fatalf("unexpected failure: got: '%v', expecting: '%v'", eKeyCheck, tt.expectKeyCheckOut)
				}

				// Get the DataCheck and compare it
				eDataCheck, err := jsonparser.GetBoolean(r.Stdout, getDataCheckJSON(tt.keyNum)...)
				if err != nil {
					t.Fatalf("unable to get expected output from json: %s", err)
				}
				if eDataCheck != tt.expectDataCheckOut {
					t.Fatalf("unexpected failure: got: '%v', expecting: '%v'", eDataCheck, tt.expectDataCheckOut)
				}
			}

			// Inspect the container, and get the output
			e2e.RunSingularity(
				t,
				tt.name,
				e2e.WithPrivileges(false),
				e2e.WithCommand("verify"),
				e2e.WithArgs("--json", tt.cpath),
				e2e.ExpectExit(tt.expectExit, verifyOutput),
			)
		})
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("singularityVerifyKeyNum", c.singularityVerifyKeyNum)
		t.Run("singularityVerifySigner", c.singularityVerifySigner)
	}
}
