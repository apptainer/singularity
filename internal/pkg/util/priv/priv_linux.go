// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package priv

import (
	"os"
	"runtime"
	"syscall"
)

// Escalate escalates thread privileges.
func Escalate() error {
	runtime.LockOSThread()
	uid := os.Getuid()
	return syscall.Setresuid(uid, 0, uid)
}

// Drop drops thread privileges.
func Drop() error {
	defer runtime.UnlockOSThread()
	uid := os.Getuid()
	return syscall.Setresuid(uid, uid, 0)
}
