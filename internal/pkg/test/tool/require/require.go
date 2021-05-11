// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package require

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/hpcng/singularity/internal/pkg/security/seccomp"
	"github.com/hpcng/singularity/pkg/util/fs/proc"
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

// MkfsExt3 checks that mkfs.ext3 is available and
// support -d option to create writable overlay layout.
func MkfsExt3(t *testing.T) {
	mkfs, err := exec.LookPath("mkfs.ext3")
	if err != nil {
		t.Skipf("mkfs.ext3 not found in $PATH")
	}

	buf := new(bytes.Buffer)
	cmd := exec.Command(mkfs, "--help")
	cmd.Stderr = buf
	_ = cmd.Run()

	if !strings.Contains(buf.String(), "[-d ") {
		t.Skipf("mkfs.ext3 is too old and doesn't support -d")
	}
}
