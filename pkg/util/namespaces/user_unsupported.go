// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build !linux

package namespaces

import "os"

// IsInsideUserNamespace checks if a process is already running in a
// user namespace and also returns if the process has permissions to use
// setgroups in this user namespace.
func IsInsideUserNamespace(pid int) (bool, bool) {
	return false, false
}

// HostUID attempts to find the original host UID if the current
// process is running inside a user namespace, if it doesn't it
// simply returns the current UID
func HostUID() (int, error) {
	return os.Getuid(), nil
}
