// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpcng/singularity/e2e/internal/e2e"
	"github.com/hpcng/singularity/e2e/internal/testhelper"
	"github.com/hpcng/singularity/internal/pkg/test/tool/require"
	"github.com/hpcng/singularity/internal/pkg/util/fs"
)

var buildDefinition = `Bootstrap: localimage
From: %[1]s

%%setup
	touch $SINGULARITY_ROOTFS%[2]s
%%post
	%[3]s
%%test
	%[3]s
`

type ctx struct {
	env e2e.TestEnv
}

func (c ctx) testNvidia(t *testing.T) {
	require.Nvidia(t)
	// Use Ubuntu 20.04 as this is a recent distro officially supported by Nvidia CUDA.
	// We can't use our test image as it's alpine based and we need a compatible glibc.
	imageURL := "docker://ubuntu:20.04"
	imageFile, err := fs.MakeTmpFile("", "test-nvidia-", 0o755)
	if err != nil {
		t.Fatalf("Could not create test file: %v", err)
	}
	imageFile.Close()
	imagePath := imageFile.Name()
	defer os.Remove(imagePath)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs("--force", imagePath, imageURL),
		e2e.ExpectExit(0),
	)

	// Basic test that we can run the bound in `nvidia-smi` which *should* be on the PATH
	tests := []struct {
		name    string
		profile e2e.Profile
		args    []string
	}{
		{
			name:    "User",
			profile: e2e.UserProfile,
			args:    []string{"--nv", imagePath, "nvidia-smi"},
		},
		{
			name:    "UserContain",
			profile: e2e.UserProfile,
			args:    []string{"--contain", "--nv", imagePath, "nvidia-smi"},
		},
		{
			name:    "UserNamespace",
			profile: e2e.UserNamespaceProfile,
			args:    []string{"--nv", imagePath, "nvidia-smi"},
		},
		{
			name:    "Fakeroot",
			profile: e2e.FakerootProfile,
			args:    []string{"--nv", imagePath, "nvidia-smi"},
		},
		{
			name:    "Root",
			profile: e2e.RootProfile,
			args:    []string{"--nv", imagePath, "nvidia-smi"},
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(0),
		)
	}
}

func (c ctx) testRocm(t *testing.T) {
	require.Rocm(t)
	// Use Ubuntu 20.04 as this is the most recent distro officially supported by ROCm.
	// We can't use our test image as it's alpine based and we need a compatible glibc.
	imageURL := "docker://ubuntu:20.04"
	imageFile, err := fs.MakeTmpFile("", "test-rocm-", 0o755)
	if err != nil {
		t.Fatalf("Could not create test file: %v", err)
	}
	imageFile.Close()
	imagePath := imageFile.Name()
	defer os.Remove(imagePath)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("pull"),
		e2e.WithArgs("--force", imagePath, imageURL),
		e2e.ExpectExit(0),
	)

	// Basic test that we can run the bound in `rocminfo` which *should* be on the PATH
	tests := []struct {
		name    string
		profile e2e.Profile
		args    []string
	}{
		{
			name:    "User",
			profile: e2e.UserProfile,
			args:    []string{"--rocm", imagePath, "rocminfo"},
		},
		{
			name:    "UserContain",
			profile: e2e.UserProfile,
			args:    []string{"--contain", "--rocm", imagePath, "rocminfo"},
		},
		{
			name:    "UserNamespace",
			profile: e2e.UserNamespaceProfile,
			args:    []string{"--rocm", imagePath, "rocminfo"},
		},
		{
			name:    "Fakeroot",
			profile: e2e.FakerootProfile,
			args:    []string{"--rocm", imagePath, "rocminfo"},
		},
		{
			name:    "Root",
			profile: e2e.RootProfile,
			args:    []string{"--rocm", imagePath, "rocminfo"},
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.args...),
			e2e.ExpectExit(0),
		)
	}
}

func (c ctx) testBuildNvidia(t *testing.T) {
	require.Nvidia(t)

	// ignore the error as it's already done in the require call above
	nvsmi, _ := exec.LookPath("nvidia-smi")

	// Use Ubuntu 20.04 as this is the most recent distro officially supported by Nvidia CUDA.
	// We can't use our test image as it's alpine based and we need a compatible glibc.
	imageURL := "docker://ubuntu:20.04"

	tmpdir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "build-nvidia-image", "build with nvidia")
	defer cleanup(t)

	sourceImage := filepath.Join(tmpdir, "source")
	sandboxImage := filepath.Join(tmpdir, "sandbox")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--force", "--sandbox", sourceImage, imageURL),
		e2e.ExpectExit(0),
	)

	// Basic test that we can run the bound in `rocminfo` which *should* be on the PATH
	tests := []struct {
		name      string
		profile   e2e.Profile
		setNvFlag bool
		exit      int
	}{
		{
			name:      "Build with nv and run nvidia-smi (root)",
			profile:   e2e.RootProfile,
			setNvFlag: true,
			exit:      0,
		},
		{
			name:      "Build with nv and run nvidia-smi (fakeroot)",
			profile:   e2e.FakerootProfile,
			setNvFlag: true,
			exit:      0,
		},
		{
			name:      "Build without nv and run nvidia-smi (root)",
			profile:   e2e.RootProfile,
			setNvFlag: false,
			exit:      255,
		},
		{
			name:      "Build without nv and run nvidia-smi (fakeroot)",
			profile:   e2e.FakerootProfile,
			setNvFlag: false,
			exit:      255,
		},
	}

	rawDef := fmt.Sprintf(buildDefinition, sourceImage, nvsmi, "nvidia-smi")

	for _, tt := range tests {
		defFile := e2e.RawDefFile(t, tmpdir, strings.NewReader(rawDef))

		args := []string{}
		if tt.setNvFlag {
			args = append(args, "--nv")
		}
		args = append(args, "-F", "--sandbox", sandboxImage, defFile)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

func (c ctx) testBuildRocm(t *testing.T) {
	require.Rocm(t)

	// ignore the error as it's already done in the require call above
	rocmInfo, _ := exec.LookPath("rocminfo")

	// Use Ubuntu 20.04 as this is the most recent distro officially supported by ROCm.
	// We can't use our test image as it's alpine based and we need a compatible glibc.
	imageURL := "docker://ubuntu:20.04"

	tmpdir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "build-rocm-image", "build with rocm")
	defer cleanup(t)

	sourceImage := filepath.Join(tmpdir, "source")
	sandboxImage := filepath.Join(tmpdir, "sandbox")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--force", "--sandbox", sourceImage, imageURL),
		e2e.ExpectExit(0),
	)

	// Basic test that we can run the bound in `rocminfo` which *should* be on the PATH
	tests := []struct {
		name        string
		profile     e2e.Profile
		setRocmFlag bool
		exit        int
	}{
		{
			name:        "Build with rocm and run rocminfo (root)",
			profile:     e2e.RootProfile,
			setRocmFlag: true,
			exit:        0,
		},
		{
			name:        "Build with rocm and run rocminfo (fakeroot)",
			profile:     e2e.FakerootProfile,
			setRocmFlag: true,
			exit:        0,
		},
		{
			name:        "Build without rocm and run rocminfo (root)",
			profile:     e2e.RootProfile,
			setRocmFlag: false,
			exit:        255,
		},
		{
			name:        "Build without rocm and run rocminfo (fakeroot)",
			profile:     e2e.FakerootProfile,
			setRocmFlag: false,
			exit:        255,
		},
	}

	rawDef := fmt.Sprintf(buildDefinition, sourceImage, rocmInfo, "rocminfo")

	for _, tt := range tests {
		defFile := e2e.RawDefFile(t, tmpdir, strings.NewReader(rawDef))

		args := []string{}
		if tt.setRocmFlag {
			args = append(args, "--rocm")
		}
		args = append(args, "--force", "--sandbox", sandboxImage, defFile)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(tt.profile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	return testhelper.Tests{
		"nvidia":       c.testNvidia,
		"rocm":         c.testRocm,
		"build nvidia": c.testBuildNvidia,
		"build rocm":   c.testBuildRocm,
		"issue 5002":   c.issue5002, // https://github.com/hpcng/singularity/issues/5002
	}
}
