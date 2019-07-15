// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
)

var (
	// uid of original user running test.
	origUID = os.Getuid()
	// gid of original group running test.
	origGID = os.Getgid()
)

// Privileged wraps the supplied test function with calls to ensure
// the test is run with elevated privileges.
func Privileged(f func(*testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		runtime.LockOSThread()

		if err := syscall.Setresuid(0, 0, origUID); err != nil {
			t.Fatalf("privileges escalation failed: %s", err)
		}
		if err := syscall.Setresgid(0, 0, origGID); err != nil {
			t.Fatalf("privileges escalation failed: %s", err)
		}
		// NEED FIX: it shouldn't be set/restored globally, only
		// when executing singularity command with privileges.
		os.Setenv(cache.DirEnv, cacheDirPriv)

		defer func() {
			if err := syscall.Setresgid(origGID, origGID, 0); err != nil {
				t.Fatalf("privileges drop failed: %s", err)
			}
			if err := syscall.Setresuid(origUID, origUID, 0); err != nil {
				t.Fatalf("privileges drop failed: %s", err)
			}
			// NEED FIX: see above comment
			os.Setenv(cache.DirEnv, cacheDirUnpriv)
			runtime.UnlockOSThread()
		}()

		f(t)
	}
}
