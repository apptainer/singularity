// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package verify

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

const containerTesterSIF = "testdata/verify_container.sif"

func (c *ctx) runVerifyCommand() ([]byte, []byte, error) {
	argv := []string{"verify", "--json", containerTesterSIF}
	cmd := exec.Command(c.env.CmdPath, argv...)

	cmdStdout := bytes.NewBuffer(nil)
	cmdStderr := bytes.NewBuffer(nil)

	cmd.Stdout = cmdStdout
	cmd.Stderr = cmdStderr

	err := cmd.Run()
	if err != nil {
		return cmdStdout.Bytes(), cmdStdout.Bytes(), err
	}

	err = cmd.Wait()
	if err != nil {
		return cmdStdout.Bytes(), cmdStdout.Bytes(), err
	}

	return cmdStdout.Bytes(), cmdStderr.Bytes(), nil
}

func (c *ctx) singularityInspect(t *testing.T) {
	tests := []struct {
		name       string
		jsonPath   []string
		expectOut  string
		expectExit string
	}{
		// Key number
		{
			name:       "verify number signers",
			jsonPath:   []string{"Signatures"},
			expectOut:  `3`,
			expectExit: "exit status 255",
		},

		// Signer 0
		{
			name:       "verify signer 0 Name",
			jsonPath:   []string{"SignerKeys", "[0]", "Signer", "Name"},
			expectOut:  `unknown`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 0 Fingerprint",
			jsonPath:   []string{"SignerKeys", "[0]", "Signer", "Fingerprint"},
			expectOut:  `8883491F4268F173C6E5DC49EDECE4F3F38D871E`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 0 Local",
			jsonPath:   []string{"SignerKeys", "[0]", "Signer", "Local"},
			expectOut:  `false`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 0 KeyCheck",
			jsonPath:   []string{"SignerKeys", "[0]", "Signer", "KeyCheck"},
			expectOut:  `true`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 0 DataCheck",
			jsonPath:   []string{"SignerKeys", "[0]", "Signer", "DataCheck"},
			expectOut:  `false`,
			expectExit: "exit status 255",
		},

		// Signer 1
		{
			name:       "verify signer 1 Name",
			jsonPath:   []string{"SignerKeys", "[1]", "Signer", "Name"},
			expectOut:  `westleyk (examples) \u003cwestley@sylabs.io\u003e`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 1 Fingerprint",
			jsonPath:   []string{"SignerKeys", "[1]", "Signer", "Fingerprint"},
			expectOut:  `4E28E95609D65D3C5BEA9731F1E47D55A7F3A56C`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 1 Local",
			jsonPath:   []string{"SignerKeys", "[1]", "Signer", "Local"},
			expectOut:  `false`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 1 KeyCheck",
			jsonPath:   []string{"SignerKeys", "[1]", "Signer", "KeyCheck"},
			expectOut:  `true`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 1 DataCheck",
			jsonPath:   []string{"SignerKeys", "[1]", "Signer", "DataCheck"},
			expectOut:  `true`,
			expectExit: "exit status 255",
		},

		// Signer 2
		{
			name:       "verify signer 2 Name",
			jsonPath:   []string{"SignerKeys", "[2]", "Signer", "Name"},
			expectOut:  `unknown`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 2 Fingerprint",
			jsonPath:   []string{"SignerKeys", "[2]", "Signer", "Fingerprint"},
			expectOut:  `C7E7C8C3635DD06930669A2283B30190FEEF8162`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 2 Local",
			jsonPath:   []string{"SignerKeys", "[2]", "Signer", "Local"},
			expectOut:  `false`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 2 KeyCheck",
			jsonPath:   []string{"SignerKeys", "[2]", "Signer", "KeyCheck"},
			expectOut:  `false`,
			expectExit: "exit status 255",
		},
		{
			name:       "verify signer 2 DataCheck",
			jsonPath:   []string{"SignerKeys", "[2]", "Signer", "DataCheck"},
			expectOut:  `false`,
			expectExit: "exit status 255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Inspect the container, and get the output
			stdout, stderr, err := c.runVerifyCommand()
			if err.Error() != tt.expectExit && err != nil { // TODO: theres probably a better way to do this
				t.Fatalf("unexpected failure: %s %s: %s", string(stdout), string(stderr), err) // TODO: improve error message
			}

			bOut, _, _, err := jsonparser.Get(stdout, tt.jsonPath...)
			if err != nil {
				t.Fatalf("unable to get expected output from json: %v", err)
			}

			// Compare the output, with the expected output
			if string(bOut) != tt.expectOut {
				t.Fatalf("unexpected failure: got: '%s', expecting: '%s'", string(bOut), tt.expectOut)
			}
		})
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("singularityVerify", c.singularityInspect)
	}
}
