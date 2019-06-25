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
	"github.com/sylabs/singularity/internal/pkg/test"
)

type testingEnv struct {
	// base env for running tests
	CmdPath string `split_words:"true"`
}

var testenv testingEnv

const containerTesterSIF = "testdata/inspecter_container.sif"

const expectedLabelsJSON = `
{
	"attributes": {
		"apps": "",
		"labels": {
			"E2E": "AWSOME",
			"HI": "\"HELLO WORLD\"",
			"MAINTAINER": "\"WestleyK \u003cwestley@sylabs.io\u003e\"",
			"e2e": "awsome",
			"hi": "\"hello world\"",
			"org.label-schema.build-date": "Friday_14_June_2019_16:49:57_PDT",
			"org.label-schema.schema-version": "1.0",
			"org.label-schema.usage": "/.singularity.d/runscript.help",
			"org.label-schema.usage.singularity.deffile.bootstrap": "library",
			"org.label-schema.usage.singularity.deffile.from": "alpine:latest",
			"org.label-schema.usage.singularity.runscript.help": "/.singularity.d/runscript.help",
			"org.label-schema.usage.singularity.version": "3.2.1-660.g4c8a84050"
		}
	},
	"type": "container"
}
`

const expectedRunscriptJSON = `
{
	"attributes": {
		"apps": "",
		"runscript": "#!/bin/sh\n\ncat /.singularity.d/runscript.help\n\n\n"
	},
	"type": "container"
}
`

const expectedListAppsJSON = `
{
	"attributes": {
		"apps": "hello\nworld\n"
	},
	"type": "container"
}
`

const expectedTestJSON = `
{
	"attributes": {
		"apps": "",
		"test": "#!/bin/sh\n\nls /\ntest -d /\ntest -d /etc\n\n\n"
	},
	"type": "container"
}
`

const expectedEnvironmentJSON = `
{
	"attributes": {
		"apps": "",
		"environment": {
			"90-environment.sh": "#!/bin/sh\n#Custom environment shell code should follow\n\n\nexport test=\"testing\"\nexport e2e=\"e2e testing\"\n\n\n"
		}
	},
	"type": "container"
}
`

func runInspectCommand(inspectType string) ([]byte, error) {
	argv := []string{"inspect", "--json", inspectType, containerTesterSIF}
	cmd := exec.Command(testenv.CmdPath, argv...)

	return cmd.CombinedOutput()
}

func singularityInspect(t *testing.T) {
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
			expectOut: `"WestleyK <westley@sylabs.io>"`,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "E2E"},
			expectOut: `AWSOME`,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "HI"},
			expectOut: `"HELLO WORLD"`,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "e2e"},
			expectOut: `awsome`,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "hi"},
			expectOut: `"hello world"`,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "org.label-schema.usage"},
			expectOut: `/.singularity.d/runscript.help`,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "org.label-schema.usage.singularity.deffile.bootstrap"},
			expectOut: `library`,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "org.label-schema.usage.singularity.deffile.from"},
			expectOut: `alpine:latest`,
		},
		{
			name:      "label",
			insType:   "--labels",
			json:      []string{"attributes", "labels", "org.label-schema.usage.singularity.runscript.help"},
			expectOut: `/.singularity.d/runscript.help`,
		},
		{
			name:    "runscript",
			insType: "--runscript",
			json:    []string{"attributes", "runscript"},
			expectOut: `#!/bin/sh

cat /.singularity.d/runscript.help


`,
		},
		{
			name:    "list apps",
			insType: "--list-apps",
			json:    []string{"attributes", "apps"},
			expectOut: `hello
world
`,
		},
		{
			name:    "test",
			insType: "--test",
			json:    []string{"attributes", "test"},
			expectOut: `#!/bin/sh

ls /
test -d /
test -d /etc


`,
		},
		{
			name:    "environment",
			insType: "--environment",
			json:    []string{"attributes", "environment", "90-environment.sh"},
			expectOut: `#!/bin/sh
#Custom environment shell code should follow


export test="testing"
export e2e="e2e testing"


`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			out, err := runInspectCommand(tt.insType)
			if err != nil {
				t.Fatalf("unexpected failure: %s: %s", string(out), err)
			}

			// Check the E2E label in test.sif, does it match our expected output
			v, err := jsonparser.GetString(out, tt.json...)
			if err != nil {
				t.Fatalf("unable to get expected output from json: %v", err)
			}
			// Get the expected output, and compare them
			//e, err := jsonparser.GetString([]byte(tt.expectOut), tt.json...)
			//if err != nil {
			//	t.Fatalf("unable to get expected output from json: %v", err)
			//}
			if v != tt.expectOut {
				t.Fatalf("unexpected failure: got: %s, expecting: %s", v, tt.expectOut)
			}

		}))
	}

}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	t.Run("singularityInspect", singularityInspect)
}
