// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (aix || darwin || dragonfly || freebsd || (!android && linux) || netbsd || openbsd || solaris) && cgo && !osusergo
// +build aix darwin dragonfly freebsd !android,linux netbsd openbsd solaris
// +build cgo
// +build !osusergo

// Copyright (c) 2019-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This is a slightly patched version of Go's os/user/cgo_lookup_unix.go file.
// We need full gecos content so the only way to do so is to call C ourselves.
// User and Group types are taken from this package rather then from os/user.

// Do not lint this file in order to keep it as close as possible to the
// original, even if the original has linter issues.

//nolint
package user

import (
	"fmt"
	osuser "os/user"
	"strconv"
	"syscall"
	"unsafe"
)

/*
#cgo solaris CFLAGS: -D_POSIX_PTHREAD_SEMANTICS
#include <unistd.h>
#include <sys/types.h>
#include <pwd.h>
#include <grp.h>
#include <stdlib.h>

static int mygetpwuid_r(int uid, struct passwd *pwd,
	char *buf, size_t buflen, struct passwd **result) {
	return getpwuid_r(uid, pwd, buf, buflen, result);
}

static int mygetpwnam_r(const char *name, struct passwd *pwd,
	char *buf, size_t buflen, struct passwd **result) {
	return getpwnam_r(name, pwd, buf, buflen, result);
}

static int mygetgrgid_r(int gid, struct group *grp,
	char *buf, size_t buflen, struct group **result) {
 return getgrgid_r(gid, grp, buf, buflen, result);
}

static int mygetgrnam_r(const char *name, struct group *grp,
	char *buf, size_t buflen, struct group **result) {
 return getgrnam_r(name, grp, buf, buflen, result);
}
*/
import "C"

func current() (*User, error) {
	return lookupUnixUid(syscall.Getuid())
}

func lookupUser(username string) (*User, error) {
	var pwd C.struct_passwd
	var result *C.struct_passwd
	nameC := make([]byte, len(username)+1)
	copy(nameC, username)

	buf := alloc(userBuffer)
	defer buf.free()

	err := retryWithBuffer(buf, func() syscall.Errno {
		// mygetpwnam_r is a wrapper around getpwnam_r to avoid
		// passing a size_t to getpwnam_r, because for unknown
		// reasons passing a size_t to getpwnam_r doesn't work on
		// Solaris.
		return syscall.Errno(C.mygetpwnam_r((*C.char)(unsafe.Pointer(&nameC[0])),
			&pwd,
			(*C.char)(buf.ptr),
			C.size_t(buf.size),
			&result))
	})
	if err != nil {
		return nil, fmt.Errorf("user: lookup username %s: %v", username, err)
	}
	if result == nil {
		return nil, osuser.UnknownUserError(username)
	}
	return buildUser(&pwd), err
}

func lookupUserId(uid string) (*User, error) {
	i, e := strconv.Atoi(uid)
	if e != nil {
		return nil, e
	}
	return lookupUnixUid(i)
}

func lookupUnixUid(uid int) (*User, error) {
	var pwd C.struct_passwd
	var result *C.struct_passwd

	buf := alloc(userBuffer)
	defer buf.free()

	err := retryWithBuffer(buf, func() syscall.Errno {
		// mygetpwuid_r is a wrapper around getpwuid_r to avoid using uid_t
		// because C.uid_t(uid) for unknown reasons doesn't work on linux.
		return syscall.Errno(C.mygetpwuid_r(C.int(uid),
			&pwd,
			(*C.char)(buf.ptr),
			C.size_t(buf.size),
			&result))
	})
	if err != nil {
		return nil, fmt.Errorf("user: lookup userid %d: %v", uid, err)
	}
	if result == nil {
		return nil, osuser.UnknownUserIdError(uid)
	}
	return buildUser(&pwd), nil
}

func buildUser(pwd *C.struct_passwd) *User {
	return &User{
		Name:  C.GoString(pwd.pw_name),
		UID:   uint32(pwd.pw_uid),
		GID:   uint32(pwd.pw_gid),
		Gecos: C.GoString(pwd.pw_gecos),
		Dir:   C.GoString(pwd.pw_dir),
		Shell: C.GoString(pwd.pw_shell),
	}
}

func currentGroup() (*Group, error) {
	return lookupUnixGid(syscall.Getgid())
}

func lookupGroup(groupname string) (*Group, error) {
	var grp C.struct_group
	var result *C.struct_group

	buf := alloc(groupBuffer)
	defer buf.free()
	cname := make([]byte, len(groupname)+1)
	copy(cname, groupname)

	err := retryWithBuffer(buf, func() syscall.Errno {
		return syscall.Errno(C.mygetgrnam_r((*C.char)(unsafe.Pointer(&cname[0])),
			&grp,
			(*C.char)(buf.ptr),
			C.size_t(buf.size),
			&result))
	})
	if err != nil {
		return nil, fmt.Errorf("user: lookup groupname %s: %v", groupname, err)
	}
	if result == nil {
		return nil, osuser.UnknownGroupError(groupname)
	}
	return buildGroup(&grp), nil
}

func lookupGroupId(gid string) (*Group, error) {
	i, e := strconv.Atoi(gid)
	if e != nil {
		return nil, e
	}
	return lookupUnixGid(i)
}

func lookupUnixGid(gid int) (*Group, error) {
	var grp C.struct_group
	var result *C.struct_group

	buf := alloc(groupBuffer)
	defer buf.free()

	err := retryWithBuffer(buf, func() syscall.Errno {
		// mygetgrgid_r is a wrapper around getgrgid_r to avoid using gid_t
		// because C.gid_t(gid) for unknown reasons doesn't work on linux.
		return syscall.Errno(C.mygetgrgid_r(C.int(gid),
			&grp,
			(*C.char)(buf.ptr),
			C.size_t(buf.size),
			&result))
	})
	if err != nil {
		return nil, fmt.Errorf("user: lookup groupid %d: %v", gid, err)
	}
	if result == nil {
		return nil, osuser.UnknownGroupIdError(strconv.Itoa(gid))
	}
	return buildGroup(&grp), nil
}

func buildGroup(grp *C.struct_group) *Group {
	g := &Group{
		GID:  uint32(grp.gr_gid),
		Name: C.GoString(grp.gr_name),
	}
	return g
}

type bufferKind C.int

const (
	userBuffer  = bufferKind(C._SC_GETPW_R_SIZE_MAX)
	groupBuffer = bufferKind(C._SC_GETGR_R_SIZE_MAX)
)

func (k bufferKind) initialSize() C.size_t {
	sz := C.sysconf(C.int(k))
	if sz == -1 {
		// DragonFly and FreeBSD do not have _SC_GETPW_R_SIZE_MAX.
		// Additionally, not all Linux systems have it, either. For
		// example, the musl libc returns -1.
		return 1024
	}
	if !isSizeReasonable(int64(sz)) {
		// Truncate.  If this truly isn't enough, retryWithBuffer will error on the first run.
		return maxBufferSize
	}
	return C.size_t(sz)
}

type memBuffer struct {
	ptr  unsafe.Pointer
	size C.size_t
}

func alloc(kind bufferKind) *memBuffer {
	sz := kind.initialSize()
	return &memBuffer{
		ptr:  C.malloc(sz),
		size: sz,
	}
}

func (mb *memBuffer) resize(newSize C.size_t) {
	mb.ptr = C.realloc(mb.ptr, newSize)
	mb.size = newSize
}

func (mb *memBuffer) free() {
	C.free(mb.ptr)
}

// retryWithBuffer repeatedly calls f(), increasing the size of the
// buffer each time, until f succeeds, fails with a non-ERANGE error,
// or the buffer exceeds a reasonable limit.
func retryWithBuffer(buf *memBuffer, f func() syscall.Errno) error {
	for {
		errno := f()
		if errno == 0 {
			return nil
		} else if errno != syscall.ERANGE {
			return errno
		}
		newSize := buf.size * 2
		if !isSizeReasonable(int64(newSize)) {
			return fmt.Errorf("internal buffer exceeds %d bytes", maxBufferSize)
		}
		buf.resize(newSize)
	}
}

const maxBufferSize = 1 << 20

func isSizeReasonable(sz int64) bool {
	return sz > 0 && sz <= maxBufferSize
}

// Because we can't use cgo in tests:
func structPasswdForNegativeTest() C.struct_passwd {
	sp := C.struct_passwd{}
	sp.pw_uid = 1<<32 - 2
	sp.pw_gid = 1<<32 - 3
	return sp
}
