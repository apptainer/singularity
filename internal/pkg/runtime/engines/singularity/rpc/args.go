// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package rpc

import (
	"os"

	"github.com/sylabs/singularity/pkg/util/loop"
)

// MkdirArgs defines the arguments to mkdir.
type MkdirArgs struct {
	Path string
	Perm os.FileMode
}

// LoopArgs defines the arguments to create a loop device.
type LoopArgs struct {
	Image      string
	Mode       int
	Info       loop.Info64
	MaxDevices int
	Shared     bool
}

// MountArgs defines the arguments to mount.
type MountArgs struct {
	Source     string
	Target     string
	Filesystem string
	Mountflags uintptr
	Data       string
}

// CryptArgs defines the arguments to mount.
type CryptArgs struct {
	Offset  uint64
	Loopdev string
	Cipher  []byte
}

// ChrootArgs defines the arguments to chroot.
type ChrootArgs struct {
	Root   string
	Method string
}

// HostnameArgs defines the arguments to sethostname.
type HostnameArgs struct {
	Hostname string
}

// SetFsIDArgs defines the arguments to setfsid.
type SetFsIDArgs struct {
	UID int
	GID int
}

// ChdirArgs defines the arguments to chdir.
type ChdirArgs struct {
	Dir string
}
