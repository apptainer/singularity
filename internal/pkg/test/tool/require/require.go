// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package require

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"

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

// Arch checks the test machine has the specified architecture.
// If not, the test is skipped with a message.
func Arch(t *testing.T, arch string) {
	if arch != "" && runtime.GOARCH != arch {
		t.Skipf("test requires architecture %s", arch)
	}

}

// ArchIn checks the test machine is one of the specified archs.
// If not, the test is skipped with a message.
func ArchIn(t *testing.T, archs []string) {
	if len(archs) > 0 {
		b := runtime.GOARCH
		for _, a := range archs {
			if b == a {
				return
			}
		}
		t.Skipf("test requires architecture %s", strings.Join(archs, "|"))
	}
}

// Fusermount3 checks for a version 3 of fusermount, as
// Singularity requires 3 for fd based mounts.
func Fusermount3(t *testing.T) {
	cmd := exec.Command("fusermount", "-V")
	output, err := cmd.CombinedOutput()
	msg := string(output)
	if err != nil {
		t.Skipf("could not run fusermount: %v", err)
	}
	if strings.Contains(msg, "version: 3") {
		return
	}
	t.Skipf("need fusermount 3, found: %s", msg)
}
