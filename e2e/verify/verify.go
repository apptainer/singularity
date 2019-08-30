// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package verify

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env            e2e.TestEnv
	corruptedImage string
	successImage   string
}

type verifyOutput struct {
	name        string
	fingerprint string
	local       bool
	keyCheck    bool
	dataCheck   bool
}

const successURL = "library://sylabs/tests/verify_success:1.0.1"
const corruptedURL = "library://sylabs/tests/verify_corrupted:1.0.1"

func getNameJSON(keyNum int) []string {
	return []string{"SignerKeys", fmt.Sprintf("[%d]", keyNum), "Signer", "Name"}
}

func getFingerprintJSON(keyNum int) []string {
	return []string{"SignerKeys", fmt.Sprintf("[%d]", keyNum), "Signer", "Fingerprint"}
}

func getLocalJSON(keyNum int) []string {
	return []string{"SignerKeys", fmt.Sprintf("[%d]", keyNum), "Signer", "KeyLocal"}
}

func getKeyCheckJSON(keyNum int) []string {
	return []string{"SignerKeys", fmt.Sprintf("[%d]", keyNum), "Signer", "KeyCheck"}
}

func getDataCheckJSON(keyNum int) []string {
	return []string{"SignerKeys", fmt.Sprintf("[%d]", keyNum), "Signer", "DataCheck"}
}

func (c *ctx) singularityVerifyKeyNum(t *testing.T) {
	keyNumPath := []string{"Signatures"}

	tests := []struct {
		name         string
		expectNumOut int64  // Is the expected number of Signatures
		imageURL     string // Is the URL to the container
		imagePath    string // Is the path to the container
		expectExit   int
	}{
		{
			name:         "verify number signers fail",
			expectNumOut: 3,
			imageURL:     corruptedURL,
			imagePath:    c.corruptedImage,
			expectExit:   255,
		},
		{
			name:         "verify number signers success",
			expectNumOut: 1,
			imageURL:     successURL,
			imagePath:    c.successImage,
			expectExit:   0,
		},
	}

	for _, tt := range tests {
		if !e2e.FileExists(t, tt.imagePath) {
			t.Fatalf("image file (%s) does not exist", tt.imagePath)
		}

		verifyOutput := func(t *testing.T, r *e2e.SingularityCmdResult) {
			// Get the Signatures and compare it
			eNum, err := jsonparser.GetInt(r.Stdout, keyNumPath...)
			if err != nil {
				err = errors.Wrap(err, "getting key number from JSON")
				t.Fatalf("unable to get expected output from json: %+v", err)
			}
			if eNum != tt.expectNumOut {
				t.Fatalf("unexpected failure: got: '%d', expecting: '%d'", eNum, tt.expectNumOut)
			}
		}

		// Inspect the container, and get the output
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("verify"),
			e2e.WithArgs("--json", tt.imagePath),
			e2e.ExpectExit(tt.expectExit, verifyOutput),
		)
	}
}

func (c *ctx) singularityVerifySigner(t *testing.T) {
	tests := []struct {
		expectOutput []verifyOutput
		name         string
		imagePath    string
		imageURL     string
		expectExit   int
		verifyLocal  bool
	}{
		// corrupted verify
		{
			name:        "corrupted signatures",
			verifyLocal: false,
			imageURL:    corruptedURL,
			imagePath:   c.corruptedImage,
			expectExit:  255,
			expectOutput: []verifyOutput{
				{
					name:        "unknown",
					fingerprint: "8883491F4268F173C6E5DC49EDECE4F3F38D871E",
					local:       false,
					keyCheck:    true,
					dataCheck:   false,
				},
				{
					name:        "WestleyK (Testing key; used for signing test containers) \u003cwestley@sylabs.io\u003e",
					fingerprint: "7605BC2716168DF057D6C600ACEEC62C8BD91BEE",
					local:       false,
					keyCheck:    true,
					dataCheck:   true,
				},
				{
					name:        "unknown",
					fingerprint: "F69C21F759C8EA06FD32CCF4536523CE1E109AF3",
					local:       false,
					keyCheck:    false,
					dataCheck:   false,
				},
			},
		},

		// corrupted verify with --local
		{
			name:        "corrupted signatures local",
			imageURL:    corruptedURL,
			imagePath:   c.corruptedImage,
			verifyLocal: true,
			expectExit:  255,
			expectOutput: []verifyOutput{
				{
					name:        "unknown",
					fingerprint: "8883491F4268F173C6E5DC49EDECE4F3F38D871E",
					local:       false,
					keyCheck:    true,
					dataCheck:   false,
				},
				{
					name:        "unknown",
					fingerprint: "7605BC2716168DF057D6C600ACEEC62C8BD91BEE",
					local:       false,
					keyCheck:    true,
					dataCheck:   true,
				},
				{
					name:        "unknown",
					fingerprint: "F69C21F759C8EA06FD32CCF4536523CE1E109AF3",
					local:       false,
					keyCheck:    false,
					dataCheck:   false,
				},
			},
		},

		// Verify 'verify_container_success.sif'
		{
			name:        "verify success",
			verifyLocal: false,
			imageURL:    successURL,
			imagePath:   c.successImage,
			expectExit:  0,
			expectOutput: []verifyOutput{
				{
					name:        "WestleyK (Testing key; used for signing test containers) \u003cwestley@sylabs.io\u003e",
					fingerprint: "7605BC2716168DF057D6C600ACEEC62C8BD91BEE",
					local:       false,
					keyCheck:    true,
					dataCheck:   true,
				},
			},
		},

		// Verify 'verify_container_success.sif' with --local
		{
			name:        "verify non local fail",
			imageURL:    successURL,
			imagePath:   c.successImage,
			verifyLocal: true,
			expectExit:  255,
			expectOutput: []verifyOutput{
				{
					name:        "unknown",
					fingerprint: "7605BC2716168DF057D6C600ACEEC62C8BD91BEE",
					local:       false,
					keyCheck:    true,
					dataCheck:   true,
				},
			},
		},
	}

	for _, tt := range tests {
		verifyOutput := func(t *testing.T, r *e2e.SingularityCmdResult) {
			for keyNum, vo := range tt.expectOutput {
				eName, err := jsonparser.GetString(r.Stdout, getNameJSON(keyNum)...)
				if err != nil {
					err = errors.Wrap(err, "getting string from JSON")
					t.Fatalf("unable to get expected output from json: %+v", err)
				}
				if eName != vo.name {
					t.Fatalf("unexpected failure: got: '%s', expecting: '%s'", eName, vo.name)
				}

				// Get the Fingerprint and compare it
				eFingerprint, err := jsonparser.GetString(r.Stdout, getFingerprintJSON(keyNum)...)
				if err != nil {
					err = errors.Wrap(err, "getting string from JSON")
					t.Fatalf("unable to get expected output from json: %+v", err)
				}
				if eFingerprint != vo.fingerprint {
					t.Fatalf("unexpected failure: got: '%s', expecting: '%s'", eFingerprint, vo.fingerprint)
				}

				// Get the Local and compare it
				eLocal, err := jsonparser.GetBoolean(r.Stdout, getLocalJSON(keyNum)...)
				if err != nil {
					err = errors.Wrap(err, "getting boolean from JSON")
					t.Fatalf("unable to get expected output from json: %+v", err)
				}
				if eLocal != vo.local {
					t.Fatalf("unexpected failure: got: '%v', expecting: '%v'", eLocal, vo.local)
				}

				// Get the KeyCheck and compare it
				eKeyCheck, err := jsonparser.GetBoolean(r.Stdout, getKeyCheckJSON(keyNum)...)
				if err != nil {
					err = errors.Wrap(err, "getting boolean from JSON")
					t.Fatalf("unable to get expected output from json: %+v", err)
				}
				if eKeyCheck != vo.keyCheck {
					t.Fatalf("unexpected failure: got: '%v', expecting: '%v'", eKeyCheck, vo.keyCheck)
				}

				// Get the DataCheck and compare it
				eDataCheck, err := jsonparser.GetBoolean(r.Stdout, getDataCheckJSON(keyNum)...)
				if err != nil {
					err = errors.Wrap(err, "getting boolean from JSON")
					t.Fatalf("unable to get expected output from json: %+v", err)
				}
				if eDataCheck != vo.dataCheck {
					t.Fatalf("unexpected failure: got: '%v', expecting: '%v'", eDataCheck, vo.dataCheck)
				}
			}
		}

		if !e2e.FileExists(t, tt.imagePath) {
			t.Fatalf("image file (%s) does not exist", tt.imagePath)
		}

		args := []string{"--json"}
		if tt.verifyLocal {
			args = append(args, "--local")
		}
		args = append(args, tt.imagePath)

		// Inspect the container, and get the output
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("verify"),
			e2e.WithArgs(args...),
			e2e.ExpectExit(tt.expectExit, verifyOutput),
		)
	}
}

func (c *ctx) checkGroupidOption(t *testing.T) {
	cmdArgs := []string{"--groupid", "0", c.successImage}
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("verify"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.RegexMatch, "^Container is signed by"),
		),
	)
}

func (c *ctx) checkIDOption(t *testing.T) {
	cmdArgs := []string{"--id", "0", c.successImage}
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("verify"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.RegexMatch, "^Container is signed by"),
		),
	)
}

func (c *ctx) checkURLOption(t *testing.T) {
	if !e2e.FileExists(t, c.successImage) {
		t.Fatalf("image file (%s) does not exist", c.successImage)
	}

	cmdArgs := []string{"--url", "https://keys.sylabs.io", c.successImage}
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("verify"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.RegexMatch, "^Container is signed by"),
		),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env:            env,
		corruptedImage: filepath.Join(env.TestDir, "verify_corrupted.sif"),
		successImage:   filepath.Join(env.TestDir, "verify_success.sif"),
	}

	return func(t *testing.T) {
		// We pull the two images required for the tests once
		e2e.PullImage(t, c.env, successURL, c.successImage)
		e2e.PullImage(t, c.env, corruptedURL, c.corruptedImage)

		t.Run("singularityVerifyKeyNum", c.singularityVerifyKeyNum)
		t.Run("singularityVerifySigner", c.singularityVerifySigner)
		t.Run("singularityVerifyGroupIdOption", c.checkGroupidOption)
		t.Run("singularityVerifyIDOption", c.checkIDOption)
		t.Run("singularityVerifyURLOption", c.checkURLOption)
	}
}
