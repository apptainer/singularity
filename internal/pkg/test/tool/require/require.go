// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package require

import (
	"os/exec"
	"testing"

	"github.com/containerd/cgroups"
	"github.com/sylabs/singularity/internal/pkg/security/seccomp"
	"github.com/sylabs/singularity/pkg/util/fs/proc"
)

// Filesystem checks that the current test could use the
// corresponding filesystem, if the filesystem is not
// listed in /proc/filesystems, the current test is skipped
// with a message.
func Filesystem(t *testing.T, fs string) {
	has, err := proc.HasFilesystem(fs)
	if err != nil {
		t.Fatalf("error while checking filesystem presence: %s", err)
	}
	if !has {
		t.Skipf("%s filesystem seems not supported", fs)
	}
}

// Cgroups checks that cgroups is enabled, if not the
// current test is skipped with a message.
func Cgroups(t *testing.T) {
	_, err := cgroups.V1()
	if err != nil {
		t.Skipf("cgroups disabled")
	}
}

// CgroupsFreezer checks that cgroup freezer subsystem is
// available, if not the current test is skipped with a
// message
func CgroupsFreezer(t *testing.T) {
	subSys, err := cgroups.V1()
	if err != nil {
		t.Skipf("cgroups disabled")
	}
	for _, s := range subSys {
		if s.Name() == "freezer" {
			return
		}
	}
	t.Skipf("no cgroups freezer subsystem available")
}

// Command checks if the provided command is found
// in one the path defined in the PATH environment variable,
// if not found the current test is skipped with a message.
func Command(t *testing.T, command string) {
	_, err := exec.LookPath(command)
	if err != nil {
		t.Skipf("%s command not found in $PATH", command)
	}
}

// Seccomp checks that seccomp is enabled, if not the
// current test is skipped with a message.
func Seccomp(t *testing.T) {
	if !seccomp.Enabled() {
		t.Skipf("seccomp disabled, Singularity was compiled without the seccomp library")
	}
}
