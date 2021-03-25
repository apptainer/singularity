// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package inspect

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/internal/pkg/test/tool/exec"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/inspect"
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
	sandboxImage := filepath.Join(testDir, "sandbox")

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

	require.Command(t, "unsquashfs")

	// First try with -user-xattrs since unsquashfs 4.4 gives an error code if
	// it can't set system xattrs while rootless.
	cmd := exec.Command("unsquashfs", "-user-xattrs", "-d", sandboxImage, squashImage)
	if res := cmd.Run(t); res.Error != nil {
		// If we failed, then try without -user-xattrs for older unsquashfs
		// versions that don't have that flag.
		cmd := exec.Command("unsquashfs", "-d", sandboxImage, squashImage)
		if res := cmd.Run(t); res.Error != nil {
			t.Fatalf("Unexpected error while running command.\n%s", res)
		}
	}

	compareLabel := func(label, out string, appName string) func(*testing.T, *inspect.Metadata) {
		return func(t *testing.T, meta *inspect.Metadata) {
			v := meta.Attributes.Labels[label]
			if appName != "" && meta.Attributes.Apps[appName] != nil {
				v = meta.Attributes.Apps[appName].Labels[label]
			}
			if v != out {
				t.Errorf("unexpected %s label value, got %s instead of %s", label, out, v)
			}
		}
	}

	tests := []struct {
		name      string
		insType   string                              // insType the type of 'inspect' flag, eg. '--deffile'
		appName   string                              // appName append --app <name> to the inspect command
		compareFn func(*testing.T, *inspect.Metadata) // json is the path to a value that we will test
	}{
		{
			name:      "label maintainer",
			insType:   "--labels",
			compareFn: compareLabel("MAINTAINER", "\"WestleyK <westley@sylabs.io>\"", ""),
		},
		{
			name:      "label_E2E",
			insType:   "--labels",
			compareFn: compareLabel("E2E", "AWSOME", ""),
		},
		{
			name:      "label_HI",
			insType:   "--labels",
			compareFn: compareLabel("HI", "\"HELLO WORLD\"", ""),
		},
		{
			name:      "label_e2e",
			insType:   "--labels",
			compareFn: compareLabel("e2e", "awsome", ""),
		},
		{
			name:      "label_hi",
			insType:   "--labels",
			compareFn: compareLabel("hi", "\"hello world\"", ""),
		},
		{
			name:      "build_label_first",
			insType:   "--labels",
			compareFn: compareLabel("first.build.labels", "first", ""),
		},
		{
			name:      "build_label_second",
			insType:   "--labels",
			compareFn: compareLabel("second.build.labels", "second", ""),
		},
		{
			name:      "label_org.label-schema.usage",
			insType:   "--labels",
			compareFn: compareLabel("org.label-schema.usage", "/.singularity.d/runscript.help", ""),
		},
		{
			name:      "label_org.label-schema.usage.singularity.deffile.bootstrap",
			insType:   "--labels",
			compareFn: compareLabel("org.label-schema.usage.singularity.deffile.bootstrap", "library", ""),
		},
		{
			name:      "label_org.label-schema.usage.singularity.deffile.from",
			insType:   "--labels",
			compareFn: compareLabel("org.label-schema.usage.singularity.deffile.from", "alpine:3.11.5", ""),
		},
		{
			name:      "label_org.label-schema.usage.singularity.runscript.help",
			insType:   "--labels",
			compareFn: compareLabel("org.label-schema.usage.singularity.runscript.help", "/.singularity.d/runscript.help", ""),
		},
		{
			name:    "runscript",
			insType: "--runscript",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				out := "#!/bin/sh\n\ncat /.singularity.d/runscript.help"
				v := meta.Attributes.Runscript
				if v != out {
					t.Errorf("unexpected runscript output, got %s instead of %s", v, out)
				}
			},
		},
		{
			name:    "startscript",
			insType: "--startscript",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				out := "#!/bin/sh\n\nexec \"$@\""
				v := meta.Attributes.Startscript
				if v != out {
					t.Errorf("unexpected startscript output, got %s instead of %s", v, out)
				}
			},
		},
		{
			name:    "list apps",
			insType: "--list-apps",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				out := []string{"hello", "world"}
				apps := make([]string, 0, len(meta.Attributes.Apps))
				for app := range meta.Attributes.Apps {
					apps = append(apps, app)
				}
				sort.Strings(apps)
				if !reflect.DeepEqual(apps, out) {
					t.Errorf("unexpected apps returned, got %v instead of %v", apps, out)
				}
			},
		},
		{
			name:    "test",
			insType: "--test",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				out := "#!/bin/sh\n\nls /\ntest -d /\ntest -d /etc"
				v := meta.Attributes.Test
				if v != out {
					t.Errorf("unexpected testscript output, got %s instead of %s", v, out)
				}
			},
		},
		{
			name:    "helpfile",
			insType: "--helpfile",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				helpFile := "/.singularity.d/runscript.help"
				out := "This is a e2e test container used for testing the 'inspect'\ncommand. This container \"inspector_container.sif\" should be placed\nin the \"e2e/testdata\" directory of Singularity."
				v := meta.Attributes.Helpfile
				if v != out {
					t.Errorf("unexpected %s output, got %s instead of %s", helpFile, v, out)
				}
			},
		},
		{
			name:    "environment",
			insType: "--environment",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				envFile := "/.singularity.d/env/90-environment.sh"
				out := "#!/bin/sh\n# Custom environment shell code should follow\n\n\nexport test=\"testing\"\nexport e2e=\"e2e testing\""
				v := meta.Attributes.Environment[envFile]
				if v != out {
					t.Errorf("unexpected environment for %s, got %s instead of %s", envFile, v, out)
				}

				envFile = "/.singularity.d/env/91-environment.sh"
				out = "export hello=\"world\""
				v = meta.Attributes.Environment[envFile]
				if v != out {
					t.Errorf("unexpected environment for %s, got %s instead of %s", envFile, v, out)
				}
			},
		},
		{
			name:      "label app hello",
			insType:   "--labels",
			appName:   "hello",
			compareFn: compareLabel("HELLOTHISIS", "hello", "hello"),
		},
		{
			name:    "help app hello",
			insType: "--helpfile",
			appName: "hello",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				out := "This is the help for hello!"
				if a, ok := meta.Attributes.Apps["hello"]; ok {
					v := a.Helpfile
					if v != out {
						t.Errorf("unexpected testscript output, got %s instead of %s", v, out)
					}
				} else {
					t.Errorf("hello app not found")
				}
			},
		},
		{
			name:    "env app hello",
			insType: "--environment",
			appName: "hello",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				envFile := "/scif/apps/hello/scif/env/90-environment.sh"
				out := "HELLOTHISIS=hello\nexport HELLOTHISIS"
				if a, ok := meta.Attributes.Apps["hello"]; ok {
					v := a.Environment[envFile]
					if v != out {
						t.Errorf("unexpected environment for %s, got %s instead of %s", envFile, v, out)
					}
				} else {
					t.Errorf("hello app not found")
				}
			},
		},
		{
			name:    "runscript app hello",
			insType: "--runscript",
			appName: "hello",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				out := "#!/bin/sh\n\necho \"hello\""
				if a, ok := meta.Attributes.Apps["hello"]; ok {
					v := a.Runscript
					if v != out {
						t.Errorf("unexpected runscript output, got %s instead of %s", v, out)
					}
				} else {
					t.Errorf("hello app not found")
				}
			},
		},
		{
			name:    "test app hello",
			insType: "--test",
			appName: "hello",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				out := "#!/bin/sh\n\necho \"THIS IS A HELLO TEST\""
				if a, ok := meta.Attributes.Apps["hello"]; ok {
					v := a.Test
					if v != out {
						t.Errorf("unexpected testscript output, got %s instead of %s", v, out)
					}
				} else {
					t.Errorf("hello app not found")
				}
			},
		},
		{
			name:    "runscript app world",
			insType: "--runscript",
			appName: "world",
			compareFn: func(t *testing.T, meta *inspect.Metadata) {
				out := "#!/bin/sh\n\necho \"world\""
				if a, ok := meta.Attributes.Apps["world"]; ok {
					v := a.Runscript
					if v != out {
						t.Errorf("unexpected runscript output, got %s instead of %s", v, out)
					}
				} else {
					t.Errorf("world app not found")
				}
			},
		},
	}

	for _, tt := range tests {
		// Inspect the container, and get the output
		compareOutput := func(t *testing.T, r *e2e.SingularityCmdResult) {
			meta := new(inspect.Metadata)
			if err := json.Unmarshal(r.Stdout, meta); err != nil {
				t.Errorf("unable to parse json output: %s", err)
			}
			tt.compareFn(t, meta)
		}

		args := []string{"--json", tt.insType}

		if tt.appName != "" {
			args = append(args, "--app", tt.appName)
		}

		c.env.RunSingularity(
			t,
			e2e.AsSubtest("SIF/"+tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("inspect"),
			e2e.WithArgs(append(args, sifImage)...),
			e2e.ExpectExit(0, compareOutput),
		)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest("Squash/"+tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("inspect"),
			e2e.WithArgs(append(args, squashImage)...),
			e2e.ExpectExit(0, compareOutput),
		)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest("Sandbox/"+tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("inspect"),
			e2e.WithArgs(append(args, sandboxImage)...),
			e2e.ExpectExit(0, compareOutput),
		)
	}

	// test --all
	compareAll := func(t *testing.T, r *e2e.SingularityCmdResult) {
		meta := new(inspect.Metadata)
		if err := json.Unmarshal(r.Stdout, meta); err != nil {
			t.Errorf("unable to parse json output: %s", err)
		}
		for _, tt := range tests {
			tt.compareFn(t, meta)
		}
	}

	c.env.RunSingularity(
		t,
		e2e.AsSubtest("SIF/all"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("inspect"),
		e2e.WithArgs("--all", sifImage),
		e2e.ExpectExit(0, compareAll),
	)

	c.env.RunSingularity(
		t,
		e2e.AsSubtest("Squash/all"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("inspect"),
		e2e.WithArgs("--all", squashImage),
		e2e.ExpectExit(0, compareAll),
	)

	c.env.RunSingularity(
		t,
		e2e.AsSubtest("Sandbox/all"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("inspect"),
		e2e.WithArgs("--all", sandboxImage),
		e2e.ExpectExit(0, compareAll),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	return testhelper.Tests{
		"inspect command": c.singularityInspect,
	}
}
