// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package fs

import "golang.org/x/sys/unix"

// IsWritable returns true of the directory that is passed in is writable by the
// the current user.
func IsWritable(dir string) bool {
	return unix.Access(dir, unix.W_OK) == nil
}
