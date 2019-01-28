// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package lock

import (
	"os"
	"syscall"
)

// Exclusive applies an exclusive lock on path
func Exclusive(path string) (fd int, err error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return fd, err
	}
	fd = int(f.Fd())
	err = syscall.Flock(fd, syscall.LOCK_EX)
	if err != nil {
		f.Close()
		return fd, err
	}
	return fd, nil
}

// Release removes a lock on path referenced by fd
func Release(fd int) error {
	defer syscall.Close(fd)
	if err := syscall.Flock(fd, syscall.LOCK_UN); err != nil {
		return err
	}
	return nil
}
