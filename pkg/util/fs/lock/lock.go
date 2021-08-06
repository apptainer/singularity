// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package lock

import (
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// Exclusive applies an exclusive lock on path
func Exclusive(path string) (fd int, err error) {
	fd, err = unix.Open(path, os.O_RDONLY, 0)
	if err != nil {
		return fd, err
	}
	err = unix.Flock(fd, unix.LOCK_EX)
	if err != nil {
		unix.Close(fd)
		return fd, err
	}
	return fd, nil
}

// Release removes a lock on path referenced by fd
func Release(fd int) error {
	defer unix.Close(fd)
	return unix.Flock(fd, unix.LOCK_UN)
}

// ErrByteRangeAcquired corresponds to the error returned
// when a file byte-range is already acquired.
var ErrByteRangeAcquired = errors.New("file byte-range lock is already acquired")

// ErrLockNotSupported corresponds to the error returned
// when file locking is not supported.
var ErrLockNotSupported = errors.New("file lock is not supported")

// ByteRange defines a file byte-range lock.
type ByteRange struct {
	fd    int
	start int64
	len   int64
}

// NewByteRange returns a file byte-range lock.
func NewByteRange(fd int, start, len int64) *ByteRange {
	return &ByteRange{fd, start, len}
}

// flock places a byte-range lock.
func (r *ByteRange) flock(lockType int16) error {
	lk := &unix.Flock_t{
		Type:   lockType,
		Whence: io.SeekStart,
		Start:  r.start,
		Len:    r.len,
	}

	err := unix.FcntlFlock(uintptr(r.fd), setLk, lk)
	if err == unix.EAGAIN || err == unix.EACCES {
		return ErrByteRangeAcquired
	} else if err == unix.ENOLCK {
		return ErrLockNotSupported
	}

	return err
}

// Lock places a write lock for the corresponding byte-range.
func (r *ByteRange) Lock() error {
	return r.flock(unix.F_WRLCK)
}

// RLock places a read lock for the corresponding byte-range.
func (r *ByteRange) RLock() error {
	return r.flock(unix.F_RDLCK)
}

// Unlock removes the lock for the corresponding byte-range.
func (r *ByteRange) Unlock() error {
	return r.flock(unix.F_UNLCK)
}
