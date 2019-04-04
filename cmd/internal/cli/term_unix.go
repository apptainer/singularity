// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package cli

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

func getColumns() int {
	if columns := os.Getenv("COLUMNS"); columns != "" {
		if n, err := strconv.ParseInt(columns, 10, 0); err == nil {
			return int(n)
		}
	}

	fd := int(os.Stdout.Fd())
	if ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ); err == nil {
		return int(ws.Col)
	}

	return 80
}
