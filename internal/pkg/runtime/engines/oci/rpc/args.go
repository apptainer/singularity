// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build oci_engine

package rpc

// SymlinkArgs defines the arguments to symlink.
type SymlinkArgs struct {
	Old string
	New string
}

// TouchArgs defines the arguments to touch.
type TouchArgs struct {
	Path string
}
