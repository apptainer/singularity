// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This file is been deprecated and will disappear on with version 3.3
// of singularity. The functionality has been moved to e2e/docker/docker.go

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/test"
	"golang.org/x/sys/unix"
)

func TestDocker(t *testing.T) {
	tests := []struct {
		name          string
		imagePath     string
		expectSuccess bool
	}{
		{"BusyBox", "docker://busybox", true},
		{"DoesNotExist", "docker://something_that_doesnt_exist_ever", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			// We create a clean image cache
			imgCacheDir := test.SetCacheDir(t, "")
			defer test.CleanCacheDir(t, imgCacheDir)
			imgCache, err := cache.HdlInit(imgCacheDir)
			if imgCache == nil || err != nil {
				t.Fatal("failed to create an image cache handle")
			}

			imagePath := path.Join(testDir, "container")
			defer os.Remove(imagePath)

			b, err := imageBuild(imgCache, buildOpts{}, imagePath, tt.imagePath)
			if tt.expectSuccess {
				if err != nil {
					t.Log(string(b))
					t.Fatalf("unexpected failure: %v", err)
				}
				imageVerify(t, imagePath, false)
			} else if !tt.expectSuccess && err == nil {
				t.Log(string(b))
				t.Fatal("unexpected success")
			}
		}))
	}
}

// AUFS sanity tests
func TestDockerAUFS(t *testing.T) {
	test.EnsurePrivilege(t)

	// We create a clean image cache
	imgCacheDir := test.SetCacheDir(t, "")
	defer test.CleanCacheDir(t, imgCacheDir)
	imgCache, err := cache.HdlInit(imgCacheDir)
	if imgCache == nil || err != nil {
		t.Fatal("failed to create an image cache handle")
	}

	imagePath := path.Join(testDir, "container")
	defer os.Remove(imagePath)

	b, err := imageBuild(imgCache, buildOpts{}, imagePath, "docker://dctrud/docker-aufs-sanity")
	if err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}

	fileTests := []struct {
		name          string
		execArgs      []string
		expectSuccess bool
	}{
		{"File2", []string{"ls", "/test/whiteout-dir/file2", "/test/whiteout-file/file2", "/test/normal-dir/file2"}, true},
		{"File1", []string{"ls", "/test/whiteout-dir/file1", "/test/whiteout-file/file1"}, false},
		{"Glob", []string{"ls", "/test/*/.wh*"}, false},
	}
	for _, tt := range fileTests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := imageExec(t, "exec", opts{}, imagePath, tt.execArgs)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.execArgs, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.execArgs, " "))
			}
		}))
	}
}

// Check force permissions for user builds #977
func TestDockerPermissions(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Create a clean image cache
	imgCacheDir := test.SetCacheDir(t, "")
	defer test.CleanCacheDir(t, imgCacheDir)
	imgCache, err := cache.HdlInit(imgCacheDir)
	if imgCache == nil || err != nil {
		t.Fatal("failed to create an image cache handle")
	}

	imagePath := path.Join(testDir, "container")
	defer os.Remove(imagePath)

	b, err := imageBuild(imgCache, buildOpts{}, imagePath, "docker://dctrud/docker-singularity-userperms")
	if err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}

	fileTests := []struct {
		name          string
		execArgs      []string
		expectSuccess bool
	}{
		{"TestDir", []string{"ls", "/testdir/"}, true},
		{"TestDirFile", []string{"ls", "/testdir/testfile"}, false},
	}
	for _, tt := range fileTests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := imageExec(t, "exec", opts{}, imagePath, tt.execArgs)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.execArgs, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.execArgs, " "))
			}
		}))
	}
}

// Check whiteout of symbolic links #1592 #1576
func TestDockerWhiteoutSymlink(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Create a clean image cache
	imgCacheDir := test.SetCacheDir(t, "")
	defer test.CleanCacheDir(t, imgCacheDir)
	imgCache, err := cache.HdlInit(imgCacheDir)
	if imgCache == nil || err != nil {
		t.Fatal("failed to create an image cache handle")
	}

	imagePath := path.Join(testDir, "container")
	defer os.Remove(imagePath)

	b, err := imageBuild(imgCache, buildOpts{}, imagePath, "docker://dctrud/docker-singularity-linkwh")
	if err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	imageVerify(t, imagePath, false)
}

func getKernelMajor(t *testing.T) (major int) {
	var buf unix.Utsname
	if err := unix.Uname(&buf); err != nil {
		t.Fatalf("uname failed: %v", err)
	}
	n, err := fmt.Sscanf(string(buf.Release[:]), "%d.", &major)
	if n != 1 || err != nil {
		t.Fatalf("Sscanf failed: %v %v", n, err)
	}
	return
}

func TestDockerDefFile(t *testing.T) {
	tests := []struct {
		name                string
		kernelMajorRequired int
		from                string
	}{
		{"Arch", 3, "dock0/arch:latest"},
		{"BusyBox", 0, "busybox:latest"},
		{"CentOS", 0, "centos:latest"},
		{"Ubuntu", 0, "ubuntu:16.04"},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			imgCacheDir := test.SetCacheDir(t, "")
			defer test.CleanCacheDir(t, imgCacheDir)
			imgCache, err := cache.HdlInit(imgCacheDir)
			if imgCache == nil || err != nil {
				t.Fatal("failed to create an image cache handle")
			}

			if getKernelMajor(t) < tt.kernelMajorRequired {
				t.Skipf("kernel >=%v.x required", tt.kernelMajorRequired)
			}

			imagePath := path.Join(testDir, "container")
			defer os.Remove(imagePath)

			deffile := prepareDefFile(DefFileDetail{
				Bootstrap: "docker",
				From:      tt.from,
			})
			defer os.Remove(deffile)

			if b, err := imageBuild(imgCache, buildOpts{}, imagePath, deffile); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
			imageVerify(t, imagePath, false)
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

func TestDockerRegistry(t *testing.T) {
	test.EnsurePrivilege(t)

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}

	prepRegistry(t)
	defer killRegistry(t)

	tests := []struct {
		name          string
		expectSuccess bool
		dfd           DefFileDetail
	}{
		{"BusyBox", true, DefFileDetail{
			Bootstrap: "docker",
			From:      "localhost:5000/my-busybox",
		}},
		{"BusyBoxRegistry", true, DefFileDetail{
			Bootstrap: "docker",
			From:      "my-busybox",
			Registry:  "localhost:5000",
		}},
		{"BusyBoxNamespace", false, DefFileDetail{
			Bootstrap: "docker",
			From:      "my-busybox",
			Registry:  "localhost:5000",
			Namespace: "not-a-namespace",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithPrivilege(func(t *testing.T) {
			// We create a clean image cache
			imgCacheDir := test.SetCacheDir(t, "")
			defer test.CleanCacheDir(t, imgCacheDir)
			imgCache, err := cache.HdlInit(imgCacheDir)
			if imgCache == nil || err != nil {
				t.Fatal("failed to create an image cache handle")
			}

			opts := buildOpts{
				env: append(os.Environ(), "SINGULARITY_NOHTTPS=true"),
			}
			imagePath := path.Join(testDir, "container")
			defer os.Remove(imagePath)

			defFile := prepareDefFile(tt.dfd)

			b, err := imageBuild(imgCache, opts, imagePath, defFile)
			if tt.expectSuccess {
				if err != nil {
					t.Log(string(b))
					t.Fatalf("unexpected failure: %v", err)
				}
				imageVerify(t, imagePath, false)
			} else if !tt.expectSuccess && (err == nil) {
				t.Log(string(b))
				t.Fatalf("unexpected success")
			}
		}))
	}
}
