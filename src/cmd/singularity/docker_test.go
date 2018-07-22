// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/singularityware/singularity/src/pkg/test"
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
			imagePath := path.Join(testDir, "container")
			defer os.Remove(imagePath)

			b, err := imageBuild(buildOpts{}, imagePath, tt.imagePath)
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

	imagePath := path.Join(testDir, "container")
	defer os.Remove(imagePath)

	b, err := imageBuild(buildOpts{}, imagePath, "docker://dctrud/docker-aufs-sanity")
	if err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}

	fileTests := []struct {
		name          string
		command       []string
		expectSuccess bool
	}{
		{"File2", []string{"ls", "/test/whiteout-dir/file2", "/test/whiteout-file/file2", "/test/normal-dir/file2"}, true},
		{"File1", []string{"ls", "/test/whiteout-dir/file1", "/test/whiteout-file/file1"}, false},
		{"Glob", []string{"ls", "/test/*/.wh*"}, false},
	}
	for _, ft := range fileTests {
		t.Run(ft.name, test.WithoutPrivilege(func(t *testing.T) {
			b, err := imageExec(execOpts{}, imagePath, ft.command)
			if ft.expectSuccess && (err != nil) {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			} else if !ft.expectSuccess && (err == nil) {
				t.Log(string(b))
				t.Fatalf("unexpected success")
			}
		}))
	}
}

// Check force permissions for user builds #977
func TestDockerPermissions(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	imagePath := path.Join(testDir, "container")
	defer os.Remove(imagePath)

	b, err := imageBuild(buildOpts{}, imagePath, "docker://dctrud/docker-singularity-userperms")
	if err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}

	fileTests := []struct {
		name          string
		command       []string
		expectSuccess bool
	}{
		{"TestDir", []string{"ls", "/testdir/"}, true},
		{"TestDirFile", []string{"ls", "/testdir/testfile"}, false},
	}
	for _, ft := range fileTests {
		t.Run(ft.name, test.WithoutPrivilege(func(t *testing.T) {
			b, err := imageExec(execOpts{}, imagePath, ft.command)
			if ft.expectSuccess && (err != nil) {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			} else if !ft.expectSuccess && (err == nil) {
				t.Log(string(b))
				t.Fatalf("unexpected success")
			}
		}))
	}
}

// Check whiteout of symbolic links #1592 #1576
func TestDockerWhiteoutSymlink(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	imagePath := path.Join(testDir, "container")
	defer os.Remove(imagePath)

	b, err := imageBuild(buildOpts{}, imagePath, "docker://dctrud/docker-singularity-linkwh")
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

			if b, err := imageBuild(buildOpts{}, imagePath, deffile); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
			imageVerify(t, imagePath, false)
		}))
	}
}
