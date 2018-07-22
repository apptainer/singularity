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
