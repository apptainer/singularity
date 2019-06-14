// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This test sets singularity image specific environment variables and
// verifies that they are properly set.

package singularityenv

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
)

type testingEnv struct {
	// base env for running tests
	CmdPath string `split_words:"true"`
}

var testenv testingEnv

const expectedLabelsJson = `{
	"attributes": {
		"apps": "",
		"labels": {
			"E2E": "AWSOME",
			"HI": "\"HELLO WORLD\"",
			"e2e": "awsome",
			"hi": "\"hello world\"",
			"org.label-schema.build-date": "Friday_14_June_2019_11:59:57_PDT",
			"org.label-schema.schema-version": "1.0",
			"org.label-schema.usage": "/.singularity.d/runscript.help",
			"org.label-schema.usage.singularity.deffile.bootstrap": "library",
			"org.label-schema.usage.singularity.deffile.from": "alpine:latest",
			"org.label-schema.usage.singularity.runscript.help": "/.singularity.d/runscript.help",
			"org.label-schema.usage.singularity.version": "3.2.1-660.g4c8a84050"
		}
	},
	"type": "container"
}`

func runInspectCommand(inspectType string) ([]byte, error) {
	argv := []string{"inspect", "--json", inspectType, "testdata/test.sif"}
	cmd := exec.Command("singularity", argv...)

	return cmd.CombinedOutput()
}

func singularityInspect(t *testing.T) {
	tests := []struct {
		name      string
		insType   string
		json      []string
		expectOut string
	}{
		{
			name:      "label E2E",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "E2E"},
			expectOut: expectedLabelsJson,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "HI"},
			expectOut: expectedLabelsJson,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "e2e"},
			expectOut: expectedLabelsJson,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "hi"},
			expectOut: expectedLabelsJson,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			out, err := runInspectCommand(tt.insType)
			if err != nil {
				t.Fatalf("Unexpected failure: %s: %s", string(out), err)
			}

			// Check the E2E label in test.sif, does it match our expected output
			v, err := jsonparser.GetString(out, tt.json...)
			if err != nil {
				fmt.Println("ERROR: ", err)
			}
			// Get the expected output, and compair them
			e, err := jsonparser.GetString([]byte(tt.expectOut), tt.json...)
			if err != nil {
				t.Fatalf("Unable to get expected output from json: %v", err)
			}
			if v != e {
				t.Fatalf("Unexpected faulure: got: %s, expecting: %s", v, e)
			}

		}))
	}

}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	// try to build from a non existen path
	t.Run("singularityEnv", singularityInspect)
}
