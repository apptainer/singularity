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
	Image string
	Mode  int
	Info  loop.Info64
}

// MountArgs defines the arguments to mount.
type MountArgs struct {
	Source     string
	Target     string
	Filesystem string
	Mountflags uintptr
	Data       string
}

// ChrootArgs defines the arguments to chroot.
type ChrootArgs struct {
	Root     string
	UsePivot bool
}

// HostnameArgs defines the arguments to sethostname.
type HostnameArgs struct {
	Hostname string
}

// HasNamespaceArgs defines the arguments to compare host namespace.
// and RPC process
type HasNamespaceArgs struct {
	Pid    int
	NsType string
}

// SetFsIDArgs defines the arguments to setfsid.
type SetFsIDArgs struct {
	UID int
	GID int
}
