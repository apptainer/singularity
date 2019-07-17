// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docker

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"golang.org/x/sys/unix"
)

type ctx struct {
	env e2e.TestEnv
}

func (c *ctx) testDockerPulls(t *testing.T) {
	const tmpContainerFile = "test_container.sif"

	tmpPath, err := fs.MakeTmpDir(c.env.TestDir, "docker-", 0755)
	if err != nil {
		t.Fatalf("failed to create temporary directory in %s: %s", c.env.TestDir, err)
	}
	defer os.RemoveAll(tmpPath)

	tmpImage := filepath.Join(tmpPath, tmpContainerFile)

	tests := []struct {
		name    string
		options []string
		image   string
		uri     string
		exit    int
	}{
		{
			name:  "AlpineLatestPull",
			image: tmpImage,
			uri:   "docker://alpine:latest",
			exit:  0,
		},
		{
			name:  "Alpine3.9Pull",
			image: filepath.Join(tmpPath, "alpine.sif"),
			uri:   "docker://alpine:3.9",
			exit:  0,
		},
		{
			name:    "Alpine3.9ForcePull",
			options: []string{"--force"},
			image:   tmpImage,
			uri:     "docker://alpine:3.9",
			exit:    0,
		},
		{
			name:    "BusyboxLatestPull",
			options: []string{"--force"},
			image:   tmpImage,
			uri:     "docker://busybox:latest",
			exit:    0,
		},
		{
			name:  "BusyboxLatestPullFail",
			image: tmpImage,
			uri:   "docker://busybox:latest",
			exit:  255,
		},
		{
			name:    "Busybox1.28Pull",
			options: []string{"--force", "--dir", tmpPath},
			image:   tmpContainerFile,
			uri:     "docker://busybox:1.28",
			exit:    0,
		},
		{
			name:  "Busybox1.28PullFail",
			image: tmpImage,
			uri:   "docker://busybox:1.28",
			exit:  255,
		},
		{
			name:  "Busybox1.28PullDirFail",
			image: "/foo/sif.sif",
			uri:   "docker://busybox:1.28",
			exit:  255,
		},
	}

	for _, tt := range tests {
		e2e.RunSingularity(
			t,
			tt.name,
			e2e.WithCommand("pull"),
			e2e.WithArgs(append(tt.options, tt.image, tt.uri)...),
			e2e.PostRun(func(t *testing.T) {
				if !t.Failed() && tt.exit == 0 {
					path := tt.image
					// handle the --dir case
					if path == tmpContainerFile {
						path = filepath.Join(tmpPath, tmpContainerFile)
					}
					e2e.ImageVerify(t, c.env.CmdPath, path)
				}
			}),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// AUFS sanity tests
func (c *ctx) testDockerAUFS(t *testing.T) {
	imagePath := path.Join(c.env.TestDir, "container")
	defer os.Remove(imagePath)

	e2e.RunSingularity(
		t,
		"DockerAUFS",
		e2e.WithoutSubTest(),
		e2e.WithCommand("build"),
		e2e.WithArgs([]string{imagePath, "docker://dctrud/docker-aufs-sanity"}...),
		e2e.ExpectExit(0),
	)

	if t.Failed() {
		return
	}

	fileTests := []struct {
		name string
		argv []string
		exit int
	}{
		{
			name: "File 2",
			argv: []string{imagePath, "ls", "/test/whiteout-dir/file2", "/test/whiteout-file/file2", "/test/normal-dir/file2"},
			exit: 0,
		},
		{
			name: "File1",
			argv: []string{imagePath, "ls", "/test/whiteout-dir/file1", "/test/whiteout-file/file1"},
			exit: 1,
		},
	}

	for _, tt := range fileTests {
		e2e.RunSingularity(
			t,
			tt.name,
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// Check force permissions for user builds #977
func (c *ctx) testDockerPermissions(t *testing.T) {
	imagePath := path.Join(c.env.TestDir, "container")
	defer os.Remove(imagePath)

	e2e.RunSingularity(
		t,
		"DockerPermissions",
		e2e.WithoutSubTest(),
		e2e.WithCommand("build"),
		e2e.WithArgs([]string{imagePath, "docker://dctrud/docker-singularity-userperms"}...),
		e2e.ExpectExit(0),
	)

	if t.Failed() {
		return
	}

	fileTests := []struct {
		name string
		argv []string
		exit int
	}{
		{
			name: "TestDir",
			argv: []string{imagePath, "ls", "/testdir/"},
			exit: 0,
		},
		{
			name: "TestDirFile",
			argv: []string{imagePath, "ls", "/testdir/testfile"},
			exit: 1,
		},
	}
	for _, tt := range fileTests {
		e2e.RunSingularity(
			t,
			tt.name,
			e2e.WithCommand("exec"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit),
		)
	}
}

// Check whiteout of symbolic links #1592 #1576
func (c *ctx) testDockerWhiteoutSymlink(t *testing.T) {
	imagePath := path.Join(c.env.TestDir, "container")
	defer os.Remove(imagePath)

	e2e.RunSingularity(
		t,
		"DockerWhiteoutSymlink",
		e2e.WithoutSubTest(),
		e2e.WithCommand("build"),
		e2e.WithArgs([]string{imagePath, "docker://dctrud/docker-singularity-linkwh"}...),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				return
			}
			e2e.ImageVerify(t, c.env.CmdPath, imagePath)
		}),
		e2e.ExpectExit(0),
	)
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

	imagePath := path.Join(c.env.TestDir, "container")

	for _, tt := range tests {
		deffile := e2e.PrepareDefFile(e2e.DefFileDetails{
			Bootstrap: "docker",
			From:      tt.from,
		})

		e2e.RunSingularity(
			t,
			tt.name,
			e2e.WithPrivileges(true),
			e2e.WithCommand("build"),
			e2e.WithArgs([]string{imagePath, deffile}...),
			e2e.PreRun(func(t *testing.T) {
				if getKernelMajor(t) < tt.kernelMajorRequired {
					t.Skipf("kernel >=%v.x required", tt.kernelMajorRequired)
				}
			}),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(imagePath)
				defer os.Remove(deffile)

				if t.Failed() {
					return
				}

				e2e.ImageVerify(t, c.env.CmdPath, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

func (c *ctx) testDockerRegistry(t *testing.T) {
	e2e.PrepRegistry(t, c.env)

	tests := []struct {
		name string
		exit int
		dfd  e2e.DefFileDetails
	}{
		{
			name: "BusyBox",
			exit: 0,
			dfd: e2e.DefFileDetails{
				Bootstrap: "docker",
				From:      "localhost:5000/my-busybox",
			},
		},
		{
			name: "BusyBoxRegistry",
			exit: 0,
			dfd: e2e.DefFileDetails{
				Bootstrap: "docker",
				From:      "my-busybox",
				Registry:  "localhost:5000",
			},
		},
		{
			name: "BusyBoxNamespace",
			exit: 255,
			dfd: e2e.DefFileDetails{
				Bootstrap: "docker",
				From:      "my-busybox",
				Registry:  "localhost:5000",
				Namespace: "not-a-namespace",
			},
		},
	}

	imagePath := path.Join(c.env.TestDir, "container")

	for _, tt := range tests {
		defFile := e2e.PrepareDefFile(tt.dfd)

		e2e.RunSingularity(
			t,
			tt.name,
			e2e.WithPrivileges(true),
			e2e.WithCommand("build"),
			e2e.WithArgs([]string{imagePath, defFile}...),
			e2e.WithEnv(append(os.Environ(), "SINGULARITY_NOHTTPS=true")),
			e2e.PostRun(func(t *testing.T) {
				defer os.Remove(imagePath)
				defer os.Remove(defFile)

				if t.Failed() || tt.exit != 0 {
					return
				}

				e2e.ImageVerify(t, c.env.CmdPath, imagePath)
			}),
			e2e.ExpectExit(tt.exit),
		)
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
