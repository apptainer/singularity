// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docker

import (
	"fmt"
	"os"
	stdexec "os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test/exec"
	"golang.org/x/sys/unix"
)

type ctx struct {
	env e2e.TestEnv
}

func (c *ctx) testDockerPulls(t *testing.T) {
	tests := []struct {
		desc          string
		srcURI        string
		imageName     string
		imagePath     string
		force         bool
		expectSuccess bool
	}{
		{
			desc:          "alpine_latest_pull",
			srcURI:        "docker://alpine:latest",
			imageName:     "",
			imagePath:     "",
			force:         false,
			expectSuccess: true,
		},
		{
			desc:          "alpine_3.9_pull",
			srcURI:        "docker://alpine:3.9",
			imageName:     "alpine.sif",
			imagePath:     "",
			force:         false,
			expectSuccess: true,
		},
		{
			desc:          "alpine_3.9_pull_force",
			srcURI:        "docker://alpine:3.9",
			imageName:     "",
			imagePath:     "",
			force:         true,
			expectSuccess: true,
		},
		{
			desc:          "busybox_latest_pull",
			srcURI:        "docker://busybox:latest",
			imageName:     "",
			imagePath:     "",
			force:         true,
			expectSuccess: true,
		},
		{
			desc:          "busybox_latest_pull_fail",
			srcURI:        "docker://busybox:latest",
			imageName:     "",
			imagePath:     "",
			force:         false,
			expectSuccess: false,
		},
		{
			desc:          "busybox_1.28_pull",
			srcURI:        "docker://busybox:1.28",
			imageName:     "",
			imagePath:     "/tmp",
			force:         true,
			expectSuccess: true,
		},
		{
			desc:          "busybox_1.28_pull_fail",
			srcURI:        "docker://busybox:1.28",
			imageName:     "",
			imagePath:     "",
			force:         false,
			expectSuccess: false,
		},
		{
			desc:          "busybox_1.28_pull_dir_fail",
			srcURI:        "docker://busybox:1.28",
			imageName:     "/foo/sif.sif",
			imagePath:     "",
			force:         false,
			expectSuccess: false,
		},
	}

	tmpImagePath := "/tmp/docker_tests"
	if err := os.RemoveAll(tmpImagePath); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpImagePath, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpImagePath)

	imagePull := func(t *testing.T, imgURI, imageName, imagePath string, force bool) (string, *exec.Result) {
		argv := []string{"pull"}
		fullImagePath := ""

		if force {
			argv = append(argv, "--force")
		}

		// TODO: this next part is messy, and needs to be cleaned up...
		if imagePath != "" {
			argv = append(argv, "--dir", imagePath)
			fullImagePath = imagePath
		}
		if imageName != "" && imagePath == "" {
			fullImagePath += filepath.Join(tmpImagePath, imageName)
			argv = append(argv, filepath.Join(tmpImagePath, imageName))
		} else {
			if imagePath != "" {
				argv = append(argv, "test_container.sif")
				fullImagePath = filepath.Join(imagePath, "test_container.sif")
			} else {
				argv = append(argv, filepath.Join(tmpImagePath, "test_container.sif"))
				fullImagePath = filepath.Join(tmpImagePath, "test_container.sif")
			}
		}

		argv = append(argv, imgURI)
		cmd := exec.Command(c.env.CmdPath, argv...)
		return fullImagePath, cmd.Run(t)
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fullPath, res := imagePull(t, tt.srcURI, tt.imageName, tt.imagePath, tt.force)
			switch {
			case tt.expectSuccess && res.Error == nil:
				e2e.ImageVerify(t, c.env.CmdPath, fullPath)
			case !tt.expectSuccess && res.Error != nil:
				// PASS: expecting failure, failed
			case tt.expectSuccess && res.Error != nil:
				// FAIL: expecting success, failed
				t.Fatalf("Unexpected failure running command.\n%s", res)
			case !tt.expectSuccess && res.Error == nil:
				// FAIL: expecting failure, succeeded
				t.Fatalf("Unexpected success running command.\n%s", res)
			}
		})
	}
}

// AUFS sanity tests
func (c *ctx) testDockerAUFS(t *testing.T) {
	imagePath := path.Join(c.env.TestDir, "container")
	defer os.Remove(imagePath)

	b, err := e2e.ImageBuild(c.env.CmdPath, e2e.BuildOpts{}, imagePath, "docker://dctrud/docker-aufs-sanity")
	if err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %s", err)
	}

	fileTests := []struct {
		name          string
		execArgs      []string
		expectSuccess bool
	}{
		{
			name:          "File 2",
			execArgs:      []string{"ls", "/test/whiteout-dir/file2", "/test/whiteout-file/file2", "/test/normal-dir/file2"},
			expectSuccess: true,
		},
		{
			name:          "File1",
			execArgs:      []string{"ls", "/test/whiteout-dir/file1", "/test/whiteout-file/file1"},
			expectSuccess: false,
		},
	}

	for _, tt := range fileTests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, exitCode, err := e2e.ImageExec(t, c.env.CmdPath, "exec", e2e.ExecOpts{}, imagePath, tt.execArgs)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%s': %s", strings.Join(tt.execArgs, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%s'", strings.Join(tt.execArgs, " "))
			}
		})
	}
}

// Check force permissions for user builds #977
func (c *ctx) testDockerPermissions(t *testing.T) {
	imagePath := path.Join(c.env.TestDir, "container")
	defer os.Remove(imagePath)

	b, err := e2e.ImageBuild(c.env.CmdPath, e2e.BuildOpts{}, imagePath, "docker://dctrud/docker-singularity-userperms")
	if err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %s", err)
	}

	fileTests := []struct {
		name          string
		execArgs      []string
		expectSuccess bool
	}{
		{
			name:          "TestDir",
			execArgs:      []string{"ls", "/testdir/"},
			expectSuccess: true,
		},
		{
			name:          "TestDirFile",
			execArgs:      []string{"ls", "/testdir/testfile"},
			expectSuccess: false,
		},
	}
	for _, tt := range fileTests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, exitCode, err := e2e.ImageExec(t, c.env.CmdPath, "exec", e2e.ExecOpts{}, imagePath, tt.execArgs)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%s': %s", strings.Join(tt.execArgs, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.execArgs, " "))
			}
		})
	}
}

// Check whiteout of symbolic links #1592 #1576
func (c *ctx) testDockerWhiteoutSymlink(t *testing.T) {
	imagePath := path.Join(c.env.TestDir, "container")
	defer os.Remove(imagePath)

	b, err := e2e.ImageBuild(c.env.CmdPath, e2e.BuildOpts{}, imagePath, "docker://dctrud/docker-singularity-linkwh")
	if err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %s", err)
	}
	e2e.ImageVerify(t, c.env.CmdPath, imagePath)
}

func (c *ctx) testDockerDefFile(t *testing.T) {
	getKernelMajor := func(t *testing.T) (major int) {
		var buf unix.Utsname
		if err := unix.Uname(&buf); err != nil {
			t.Fatalf("uname failed: %s", err)
		}
		n, err := fmt.Sscanf(string(buf.Release[:]), "%d.", &major)
		if n != 1 || err != nil {
			t.Fatalf("Sscanf failed: %v %s", n, err)
		}
		return
	}

	tests := []struct {
		name                string
		kernelMajorRequired int
		from                string
	}{
		{
			name:                "Arch",
			kernelMajorRequired: 3,
			from:                "dock0/arch:latest",
		},
		{
			name:                "BusyBox",
			kernelMajorRequired: 0,
			from:                "busybox:latest",
		},
		{
			name:                "CentOS",
			kernelMajorRequired: 0,
			from:                "centos:latest",
		},
		{
			name:                "Ubuntu",
			kernelMajorRequired: 0,
			from:                "ubuntu:16.04",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, e2e.Privileged(func(t *testing.T) {
			if getKernelMajor(t) < tt.kernelMajorRequired {
				t.Skipf("kernel >=%v.x required", tt.kernelMajorRequired)
			}

			imagePath := path.Join(c.env.TestDir, "container")
			defer os.Remove(imagePath)

			deffile := e2e.PrepareDefFile(e2e.DefFileDetails{
				Bootstrap: "docker",
				From:      tt.from,
			})
			defer os.Remove(deffile)

			if b, err := e2e.ImageBuild(c.env.CmdPath, e2e.BuildOpts{}, imagePath, deffile); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %s", err)
			}
			e2e.ImageVerify(t, c.env.CmdPath, imagePath)
		}))
	}
}

func (c *ctx) testDockerRegistry(t *testing.T) {
	e2e.PrepRegistry(t, c.env)

	if _, err := stdexec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}

	tests := []struct {
		name          string
		expectSuccess bool
		dfd           e2e.DefFileDetails
	}{
		{"BusyBox", true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      fmt.Sprintf("%s/my-busybox", c.env.TestRegistry),
		}},
		{"BusyBoxRegistry", true, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "my-busybox",
			Registry:  c.env.TestRegistry,
		}},
		{"BusyBoxNamespace", false, e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      "my-busybox",
			Registry:  c.env.TestRegistry,
			Namespace: "not-a-namespace",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, e2e.Privileged(func(t *testing.T) {
			opts := e2e.BuildOpts{
				Env: append(os.Environ(), "SINGULARITY_NOHTTPS=true"),
			}
			//opts := buildOpts{
			//	env: append(os.Environ(), "SINGULARITY_NOHTTPS=true"),
			//}
			imagePath := path.Join(c.env.TestDir, "container")
			defer os.Remove(imagePath)

			defFile := e2e.PrepareDefFile(tt.dfd)
			//defFile := prepareDefFile(tt.dfd)

			b, err := e2e.ImageBuild(c.env.CmdPath, opts, imagePath, defFile)
			//b, err := imageBuild(opts, imagePath, defFile)
			if tt.expectSuccess {
				if err != nil {
					t.Log(string(b))
					t.Fatalf("unexpected failure: %v", err)
				}
				e2e.ImageVerify(t, c.env.CmdPath, imagePath)
				//imageVerify(t, imagePath, false)
			} else if !tt.expectSuccess && (err == nil) {
				t.Log(string(b))
				t.Fatalf("unexpected success")
			}
		}))
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("dockerPulls", c.testDockerPulls)
		t.Run("testDockerAUFS", c.testDockerAUFS)
		t.Run("testDockerPermissions", c.testDockerPermissions)
		t.Run("testDockerWhiteoutSymlink", c.testDockerWhiteoutSymlink)
		t.Run("testDockerDefFile", c.testDockerDefFile)
		t.Run("testDockerRegistry", c.testDockerRegistry)
	}
}
