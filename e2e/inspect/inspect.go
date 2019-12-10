// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package inspect

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/pkg/image"
)

type ctx struct {
	env e2e.TestEnv
}

const (
	containerTesterDEF = "testdata/inspecter_container.def"
)

func (c ctx) singularityInspect(t *testing.T) {
	testDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "inspect-", "")
	defer cleanup(t)

	sifImage := filepath.Join(testDir, "image.sif")
	squashImage := filepath.Join(testDir, "image.sqs")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("-F", sifImage, containerTesterDEF),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}
			img, err := image.Init(sifImage, false)
			if err != nil {
				t.Fatalf("failed to open %s: %s", sifImage, err)
			}
			r, err := image.NewPartitionReader(img, image.RootFs, -1)
			if err != nil {
				t.Fatalf("failed to get root partition: %s", err)
			}
			f, err := os.Create(squashImage)
			if err != nil {
				t.Fatalf("failed to create %s: %s", squashImage, err)
			}
			defer f.Close()

			if _, err := io.Copy(f, r); err != nil {
				t.Fatalf("failed to copy squash image %s: %s", squashImage, err)
			}
		}),
		e2e.ExpectExit(0),
	)

	tests := []struct {
		name      string
		insType   string   // insType the type of 'inspect' flag, eg. '--deffile'
		json      []string // json is the path to a value that we will test
		expectOut string   // expectOut should be a string of expected output
	}{
		{
			name:      "label maintainer",
			insType:   "--labels",
			json:      []string{"data", "attributes", "labels", "MAINTAINER"},
			expectOut: "\"WestleyK \u003cwestley@sylabs.io\u003e\"",
		},
		{
			name:      "label_E2E",
			insType:   "--labels",
			json:      []string{"data", "attributes", "labels", "E2E"},
			expectOut: "AWSOME",
		},
		{
			name:      "label_HI",
			insType:   "--labels",
			json:      []string{"data", "attributes", "labels", "HI"},
			expectOut: "\"HELLO WORLD\"",
		},
		{
			name:      "label_e2e",
			insType:   "--labels",
			json:      []string{"data", "attributes", "labels", "e2e"},
			expectOut: "awsome",
		},
		{
			name:      "label_hi",
			insType:   "--labels",
			json:      []string{"data", "attributes", "labels", "hi"},
			expectOut: "\"hello world\"",
		},
		{
			name:      "label_org.label-schema.usage",
			insType:   "--labels",
			json:      []string{"data", "attributes", "labels", "org.label-schema.usage"},
			expectOut: "/.singularity.d/runscript.help",
		},
		{
			name:      "label_org.label-schema.usage.singularity.deffile.bootstrap",
			insType:   "--labels",
			json:      []string{"data", "attributes", "labels", "org.label-schema.usage.singularity.deffile.bootstrap"},
			expectOut: "library",
		},
		{
			name:      "label_org.label-schema.usage.singularity.deffile.from",
			insType:   "--labels",
			json:      []string{"data", "attributes", "labels", "org.label-schema.usage.singularity.deffile.from"},
			expectOut: "alpine:latest",
		},
		{
			name:      "label_org.label-schema.usage.singularity.runscript.help",
			insType:   "--labels",
			json:      []string{"data", "attributes", "labels", "org.label-schema.usage.singularity.runscript.help"},
			expectOut: "/.singularity.d/runscript.help",
		},
		{
			name:      "runscript",
			insType:   "--runscript",
			json:      []string{"data", "attributes", "runscript"},
			expectOut: "#!/bin/sh\n\ncat /.singularity.d/runscript.help\n\n\n",
		},
		{
			name:      "list apps",
			insType:   "--list-apps",
			json:      []string{"data", "attributes", "apps"},
			expectOut: "hello\nworld\n",
		},
		{
			name:      "test",
			insType:   "--test",
			json:      []string{"data", "attributes", "test"},
			expectOut: "#!/bin/sh\n\nls /\ntest -d /\ntest -d /etc\n\n\n",
		},
		{
			name:      "environment",
			insType:   "--environment",
			json:      []string{"data", "attributes", "environment"},
			expectOut: "#!/bin/sh\n#Custom environment shell code should follow\n\n\nexport test=\"testing\"\nexport e2e=\"e2e testing\"\n\n\n",
		},
	}

	for _, tt := range tests {
		// Inspect the container, and get the output
		compareOutput := func(t *testing.T, r *e2e.SingularityCmdResult) {
			// Parse the output
			v, err := jsonparser.GetString(r.Stdout, tt.json...)
			if err != nil {
				t.Fatalf("unable to get expected output from json: %v", err)
			}
			// Compare the output, with the expected output
			if v != tt.expectOut {
				t.Fatalf("unexpected failure: got: %s, expecting: %s", v, tt.expectOut)
			}
		}

		c.env.RunSingularity(
			t,
			e2e.AsSubtest("SIF/"+tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("inspect"),
			e2e.WithArgs("--json", tt.insType, sifImage),
			e2e.ExpectExit(0, compareOutput),
		)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest("Squash/"+tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("inspect"),
			e2e.WithArgs("--json", tt.insType, squashImage),
			e2e.ExpectExit(0, compareOutput),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := ctx{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"inspect command": c.singularityInspect,
	})
}
