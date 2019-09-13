// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package lock

import "golang.org/x/sys/unix"

const (
	setLk = unix.F_SETLK64
)
