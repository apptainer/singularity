// Copyright (c) 2019-2021 Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

var (
	// uid of original user running test.
	origUID = os.Getuid()
	// gid of original group running test.
	origGID = os.Getgid()
)

// OrigUID returns the UID of the user running the test suite.
func OrigUID() int {
	return origUID
}

// OrigGID returns the GID of the user running the test suite.
func OrigGID() int {
	return origGID
}

// Privileged wraps the supplied test function with calls to ensure
// the test is run with elevated privileges.
func Privileged(f func(*testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		runtime.LockOSThread()

		if err := unix.Setresuid(0, 0, origUID); err != nil {
			err = errors.Wrap(err, "changing user ID to 0")
			t.Fatalf("privileges escalation failed: %+v", err)
		}
		if err := unix.Setresgid(0, 0, origGID); err != nil {
			err = errors.Wrap(err, "changing group ID to 0")
			t.Fatalf("privileges escalation failed: %+v", err)
		}

		defer func() {
			if err := unix.Setresgid(origGID, origGID, 0); err != nil {
				err = errors.Wrapf(err, "changing group ID to %d", origUID)
				t.Fatalf("privileges drop failed: %+v", err)
			}
			if err := unix.Setresuid(origUID, origUID, 0); err != nil {
				err = errors.Wrapf(err, "changing group ID to %d", origGID)
				t.Fatalf("privileges drop failed: %+v", err)
			}
			runtime.UnlockOSThread()
		}()

		f(t)
	}
}
