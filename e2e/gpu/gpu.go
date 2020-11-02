// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

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

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	return testhelper.Tests{
		"nvidia":     c.testNvidia,
		"rocm":       c.testRocm,
		"issue 5002": c.issue5002, // https://github.com/sylabs/singularity/issues/5002
	}
}
