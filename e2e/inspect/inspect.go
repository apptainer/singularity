// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package inspect

import (
	"os/exec"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

const containerTesterSIF = "testdata/inspecter_container.sif"

func (c *ctx) runInspectCommand(inspectType string) ([]byte, error) {
	argv := []string{"inspect", "--json", inspectType, containerTesterSIF}
	cmd := exec.Command(c.env.CmdPath, argv...)

	return cmd.CombinedOutput()
}

func (c *ctx) singularityInspect(t *testing.T) {
	tests := []struct {
		name      string
		insType   string   // insType the type of 'inspect' flag, eg. '--deffile'
		json      []string // json is the path to a value that we will test
		expectOut string   // expectOut should be a string of expected output
	}{
		{
			name:      "label maintainer",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "MAINTAINER"},
			expectOut: "\"WestleyK <westley@sylabs.io>\"",
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "E2E"},
			expectOut: "AWSOME",
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "HI"},
			expectOut: "\"HELLO WORLD\"",
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "e2e"},
			expectOut: "awsome",
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "hi"},
			expectOut: "\"hello world\"",
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "org.label-schema.usage"},
			expectOut: "/.singularity.d/runscript.help",
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "org.label-schema.usage.singularity.deffile.bootstrap"},
			expectOut: "library",
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "org.label-schema.usage.singularity.deffile.from"},
			expectOut: "alpine:latest",
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "org.label-schema.usage.singularity.runscript.help"},
			expectOut: "/.singularity.d/runscript.help",
		},
		{
			name:      "runscript",
			insType:   "--runscript",
			json:      []string{"attributes", "runscript"},
			expectOut: "#!/bin/sh\n\ncat /.singularity.d/runscript.help\n\n\n",
		},
		{
			name:      "list apps",
			insType:   "--list-apps",
			json:      []string{"attributes", "apps"},
			expectOut: "hello\nworld\n",
		},
		{
			name:      "test",
			insType:   "--test",
			json:      []string{"attributes", "test"},
			expectOut: "#!/bin/sh\n\nls /\ntest -d /\ntest -d /etc\n\n\n",
		},
		{
			name:      "environment",
			insType:   "--environment",
			json:      []string{"attributes", "environment", "90-environment.sh"},
			expectOut: "#!/bin/sh\n#Custom environment shell code should follow\n\n\nexport test=\"testing\"\nexport e2e=\"e2e testing\"\n\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Inspect the container, and get the output
			out, err := c.runInspectCommand(tt.insType)
			if err != nil {
				t.Fatalf("unexpected failure: %s: %s", string(out), err)
			}

			// Parse the output
			v, err := jsonparser.GetString(out, tt.json...)
			if err != nil {
				t.Fatalf("unable to get expected output from json: %v", err)
			}
			// Compare the output, with the expected output
			if v != tt.expectOut {
				t.Fatalf("unexpected failure: got: %s, expecting: %s", v, tt.expectOut)
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
		t.Run("singularityInspect", c.singularityInspect)
	}
}
