// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package rpc

import "github.com/singularityware/singularity/src/pkg/util/loop"

// MkdirArgs defines the arguments to mkdir
type MkdirArgs struct {
	Path string
}

// LoopArgs defines the arguments to create a loop device
type LoopArgs struct {
	Image      string
	MaxDevices uint
	Info       loop.Info64
}

// MountArgs defines the arguments to mount
type MountArgs struct {
	Source     string
	Target     string
	Filesystem string
	Mountflags uintptr
	Data       string
}

// ChrootArgs defines the arguments to chroot
type ChrootArgs struct {
	Root string
}
