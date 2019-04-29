// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package imgbuild

import (
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv

func buildFrom(t *testing.T) {
	tests := []struct {
		name       string
		dependency string
		buildSpec  string
		sandbox    bool
	}{
		{"BusyBox", "", "../examples/busybox/Singularity", false},
		{"Debootstrap", "debootstrap", "../examples/debian/Singularity", true},
		{"DockerURI", "", "docker://busybox", true},
		{"DockerDefFile", "", "../examples/docker/Singularity", true},
		{"SHubURI", "", "shub://GodloveD/busybox", true},
		{"SHubDefFile", "", "../examples/shub/Singularity", true},
		{"LibraryDefFile", "", "../examples/library/Singularity", true},
		{"Yum", "yum", "../examples/centos/Singularity", true},
		{"Zypper", "zypper", "../examples/opensuse/Singularity", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			if tt.dependency != "" {
				if _, err := exec.LookPath(tt.dependency); err != nil {
					t.Skipf("%v not found in path", tt.dependency)
				}
			}

			opts := e2e.BuildOpts{
				Sandbox: tt.sandbox,
			}

			imagePath := path.Join(testenv.TestDir, "container")
			defer os.RemoveAll(imagePath)

			if b, err := e2e.ImageBuild(testenv.CmdPath, opts, imagePath, tt.buildSpec); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
			e2e.ImageVerify(t, testenv.CmdPath, imagePath, false, testenv.RunDisabled)
		}))
	}
}

func buildMultiStage(t *testing.T) {
	imagePath1 := path.Join(testenv.TestDir, "container1")
	imagePath2 := path.Join(testenv.TestDir, "container2")
	imagePath3 := path.Join(testenv.TestDir, "container3")

	liDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      imagePath1,
	})
	defer os.Remove(liDefFile)

	labels := make(map[string]string)
	labels["FOO"] = "bar"
	liLabelDefFile := e2e.PrepareDefFile(e2e.DefFileDetails{
		Bootstrap: "localimage",
		From:      imagePath2,
		Labels:    labels,
	})
	defer os.Remove(liLabelDefFile)

	type testSpec struct {
		name      string
		imagePath string
		buildSpec string
		force     bool
		sandbox   bool
		labels    bool
	}

	tests := []struct {
		name  string
		steps []testSpec
	}{
		{"SIFToSIF", []testSpec{
			{"BusyBox", imagePath1, "../examples/busybox/Singularity", false, false, false},
			{"SIF", imagePath2, imagePath1, false, false, false},
		}},
		{"SandboxToSIF", []testSpec{
			{"BusyBoxSandbox", imagePath1, "../examples/busybox/Singularity", false, true, false},
			{"SIF", imagePath2, imagePath1, false, false, false},
		}},
		{"LocalImage", []testSpec{
			{"BusyBox", imagePath1, "../examples/busybox/Singularity", false, false, false},
			{"LocalImage", imagePath2, liDefFile, false, false, false},
			{"LocalImageLabel", imagePath3, liLabelDefFile, false, false, true},
		}},
		{"LocalImageSandbox", []testSpec{
			{"BusyBoxSandbox", imagePath2, "../examples/busybox/Singularity", true, true, false},
			{"LocalImageLabel", imagePath3, liLabelDefFile, false, false, true},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			for _, ts := range tt.steps {
				defer os.RemoveAll(ts.imagePath)
			}

			for _, ts := range tt.steps {
				t.Run(ts.name, test.WithPrivilege(func(t *testing.T) {
					opts := e2e.BuildOpts{
						Force:   ts.force,
						Sandbox: ts.sandbox,
					}

					if b, err := e2e.ImageBuild(testenv.CmdPath, opts, ts.imagePath, ts.buildSpec); err != nil {
						t.Log(string(b))
						t.Fatalf("unexpected failure: %v", err)
					}
					e2e.ImageVerify(t, testenv.CmdPath, ts.imagePath, ts.labels, testenv.RunDisabled)
				}))
			}
		}))
	}
}

func badPath(t *testing.T) {
	test.EnsurePrivilege(t)

	imagePath := path.Join(testenv.TestDir, "container")
	defer os.RemoveAll(imagePath)

	if b, err := e2e.ImageBuild(testenv.CmdPath, e2e.BuildOpts{}, imagePath, "/some/dumb/path"); err == nil {
		t.Log(string(b))
		t.Fatal("unexpected success")
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	t.Log(testenv)

	// builds from definition file and URI
	t.Run("From", buildFrom)
	// build and image from an existing image
	t.Run("multistage", buildMultiStage)
	// try to build from a non existen path
	t.Run("badPath", badPath)
}
