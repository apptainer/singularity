// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docker

import (
	"fmt"
	"os"
	"path"
	"strings"
	//	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"
	//"unix"

	"github.com/kelseyhightower/envconfig"
	//	"github.com/sylabs/singularity/e2e/testutils"
	"github.com/sylabs/singularity/e2e/actions"
	"github.com/sylabs/singularity/e2e/imgbuild"
	"github.com/sylabs/singularity/internal/pkg/test"
	"golang.org/x/sys/unix"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv
var testDir string

func testDockerPulls(t *testing.T) {
	tests := []struct {
		desc          string
		srcURI        string
		imageName     string
		imagePath     string
		force         bool
		sandBox       bool
		expectSuccess bool
	}{
		// TODO: go thrught and fix this up
		{
			desc:          "alpine latest pull",
			srcURI:        "docker://alpine:latest",
			imageName:     "",
			imagePath:     "",
			force:         false,
			sandBox:       false,
			expectSuccess: true,
		},
		{
			desc:          "alpine 3.9 pull",
			srcURI:        "docker://alpine:3.9",
			imageName:     "",
			imagePath:     "",
			force:         true,
			sandBox:       false,
			expectSuccess: true,
		},
		{
			desc:          "busybox pull",
			srcURI:        "docker://busybox:latest",
			imageName:     "",
			imagePath:     "",
			force:         true,
			sandBox:       false,
			expectSuccess: true,
		},
		{
			desc:          "busybox pull",
			srcURI:        "docker://busybox:1.28",
			imageName:     "",
			imagePath:     "",
			force:         true,
			sandBox:       false,
			expectSuccess: true,
		},
		{
			desc:          "busybox pull fail",
			srcURI:        "docker://busybox:1.28",
			imageName:     "",
			imagePath:     "",
			force:         false,
			sandBox:       false,
			expectSuccess: false,
		},
		{
			desc:          "busybox pull",
			srcURI:        "docker://busybox:1.28",
			imageName:     "",
			imagePath:     "",
			force:         true,
			sandBox:       false,
			expectSuccess: true,
		},
	}

	tmpImagePath := "/tmp/docker_tests"
	t.Run("Makeing_tmp_dir", test.WithoutPrivilege(func(t *testing.T) {
		if err := os.RemoveAll(tmpImagePath); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(tmpImagePath, os.ModePerm); err != nil {
			t.Fatal(err)
		}
	}))

	imagePull := func(imgURI, imageName, imagePath string, force, sandBox bool) ([]byte, error) {
		argv := []string{"pull"}

		if force {
			argv = append(argv, "--force")
		}

		if sandBox {
			argv = append(argv, "--sandbox")
		}

		if imagePath != "" {
			argv = append(argv, "--dir", imagePath)
			//} else {
			//argv = append(argv, "--dir", imagePath)
		}

		if imageName != "" {
			argv = append(argv, filepath.Join(tmpImagePath, imageName))
		} else {
			argv = append(argv, filepath.Join(tmpImagePath, "test_container.sif"))
		}

		argv = append(argv, imgURI)

		fmt.Println("COMMAND: ", argv)

		cmd := exec.Command(testenv.CmdPath, argv...)

		b, err := cmd.CombinedOutput()

		return b, err
	}

	for _, tt := range tests {
		t.Run(tt.desc, test.WithoutPrivilege(func(t *testing.T) {
			b, err := imagePull(tt.srcURI, tt.imageName, tt.imagePath, tt.force, tt.sandBox)
			switch {
			case tt.expectSuccess && err == nil:
				// PASS: expecting success, succeeded

				if tt.imageName == "" && tt.imagePath == "" {
					imgbuild.ImageVerify(t, testenv.CmdPath, filepath.Join(tmpImagePath, "test_container.sif"), false, testenv.RunDisabled)
				} else {
					imgbuild.ImageVerify(t, testenv.CmdPath, tt.imagePath, false, testenv.RunDisabled)
				}
				//imageVerify(t, tt.imagePath, false)

			case !tt.expectSuccess && err != nil:
				// PASS: expecting failure, failed

			case tt.expectSuccess && err != nil:
				// FAIL: expecting success, failed

				t.Log(string(b))
				t.Fatalf("unexpected failure: %s", err)

			case !tt.expectSuccess && err == nil:
				// FAIL: expecting failure, succeeded

				t.Log(string(b))
				t.Fatalf("unexpected success: command should have failed")
			}
		}))
	}
	t.Run("Removing_tmp_dir", func(t *testing.T) {
		os.RemoveAll(tmpImagePath)
	})
}

// AUFS sanity tests
func testDockerAUFS(t *testing.T) {
	test.EnsurePrivilege(t)

	imagePath := path.Join(testDir, "container")
	defer os.Remove(imagePath)

	b, err := imgbuild.ImageBuild(testenv.CmdPath, imgbuild.Opts{}, imagePath, "docker://dctrud/docker-aufs-sanity")
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
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := actions.ImageExec(t, testenv.CmdPath, "exec", actions.Opts{}, imagePath, tt.execArgs)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%s': %s", strings.Join(tt.execArgs, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%s'", strings.Join(tt.execArgs, " "))
			}
		}))
	}
}

// Check force permissions for user builds #977
func testDockerPermissions(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	imagePath := path.Join(testDir, "container")
	defer os.Remove(imagePath)

	b, err := imgbuild.ImageBuild(testenv.CmdPath, imgbuild.Opts{}, imagePath, "docker://dctrud/docker-singularity-userperms")
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
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := actions.ImageExec(t, testenv.CmdPath, "exec", actions.Opts{}, imagePath, tt.execArgs)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%s': %s", strings.Join(tt.execArgs, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.execArgs, " "))
			}
		}))
	}
}

// Check whiteout of symbolic links #1592 #1576
func testDockerWhiteoutSymlink(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	imagePath := path.Join(testDir, "container")
	defer os.Remove(imagePath)

	b, err := imgbuild.ImageBuild(testenv.CmdPath, imgbuild.Opts{}, imagePath, "docker://dctrud/docker-singularity-linkwh")
	if err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %s", err)
	}
	imgbuild.ImageVerify(t, testenv.CmdPath, imagePath, false, testenv.RunDisabled)
}

func testDockerDefFile(t *testing.T) {
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
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			if getKernelMajor(t) < tt.kernelMajorRequired {
				t.Skipf("kernel >=%v.x required", tt.kernelMajorRequired)
			}

			imagePath := path.Join(testDir, "container")
			defer os.Remove(imagePath)

			deffile := imgbuild.PrepareDefFile(imgbuild.DefFileDetail{
				Bootstrap: "docker",
				From:      tt.from,
			})
			defer os.Remove(deffile)

			if b, err := imgbuild.ImageBuild(testenv.CmdPath, imgbuild.Opts{}, imagePath, deffile); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %s", err)
			}
			imgbuild.ImageVerify(t, testenv.CmdPath, imagePath, false, testenv.RunDisabled)
		}))
	}
}

func prepRegistry(t *testing.T) {
	commands := [][]string{
		{"run", "-d", "-p", "5000:5000", "--restart=always", "--name", "registry", "registry:2"},
		{"pull", "busybox"},
		{"tag", "busybox", "localhost:5000/my-busybox"},
		{"push", "localhost:5000/my-busybox"},
	}

	for _, command := range commands {
		cmd := exec.Command("docker", command...)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Logf(string(b))
			t.Fatalf("command failed: %v", strings.Join(command, " "))
		}
	}
}

func killRegistry(t *testing.T) {
	commands := [][]string{
		{"kill", "registry"},
		{"rm", "registry"},
	}

	for _, command := range commands {
		cmd := exec.Command("docker", command...)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Logf(string(b))
			t.Fatalf("command failed: %v", strings.Join(command, " "))
		}
	}
}

func testDockerRegistry(t *testing.T) {
	test.EnsurePrivilege(t)

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}

	prepRegistry(t)
	defer killRegistry(t)

	tests := []struct {
		name          string
		expectSuccess bool
		dfd           imgbuild.DefFileDetail
	}{
		{"BusyBox", true, imgbuild.DefFileDetail{
			Bootstrap: "docker",
			From:      "localhost:5000/my-busybox",
		}},
		{"BusyBoxRegistry", true, imgbuild.DefFileDetail{
			Bootstrap: "docker",
			From:      "my-busybox",
			Registry:  "localhost:5000",
		}},
		{"BusyBoxNamespace", false, imgbuild.DefFileDetail{
			Bootstrap: "docker",
			From:      "my-busybox",
			Registry:  "localhost:5000",
			Namespace: "not-a-namespace",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			opts := imgbuild.Opts{
				Env: append(os.Environ(), "SINGULARITY_NOHTTPS=true"),
			}
			//opts := buildOpts{
			//	env: append(os.Environ(), "SINGULARITY_NOHTTPS=true"),
			//}
			imagePath := path.Join(testDir, "container")
			defer os.Remove(imagePath)

			defFile := imgbuild.PrepareDefFile(tt.dfd)
			//defFile := prepareDefFile(tt.dfd)

			b, err := imgbuild.ImageBuild(testenv.CmdPath, opts, imagePath, defFile)
			//b, err := imageBuild(opts, imagePath, defFile)
			if tt.expectSuccess {
				if err != nil {
					t.Log(string(b))
					t.Fatalf("unexpected failure: %v", err)
				}
				imgbuild.ImageVerify(t, testenv.CmdPath, imagePath, false, testenv.RunDisabled)
				//imageVerify(t, imagePath, false)
			} else if !tt.expectSuccess && (err == nil) {
				t.Log(string(b))
				t.Fatalf("unexpected success")
			}
		}))
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("dockerPulls", testDockerPulls)
	t.Run("testDockerAUFS", testDockerAUFS)
	t.Run("testDockerPermissions", testDockerPermissions)
	t.Run("testDockerWhiteoutSymlink", testDockerWhiteoutSymlink)
	t.Run("testDockerDefFile", testDockerDefFile)
	t.Run("testDockerRegistry", testDockerRegistry)
}
