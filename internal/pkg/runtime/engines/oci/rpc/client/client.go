// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build oci_engine

package client

import (
	"os"

	ociargs "github.com/sylabs/singularity/internal/pkg/runtime/engines/oci/rpc"
	args "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc"
	client "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/client"
)

// RPC holds the state necessary for remote procedure calls.
type RPC struct {
	client.RPC
}

// MkdirAll calls the mkdir RPC using the supplied arguments.
func (t *RPC) MkdirAll(path string, perm os.FileMode) (int, error) {
	arguments := &args.MkdirArgs{
		Path: path,
		Perm: perm,
	}
	var reply int
	err := t.Client.Call(t.Name+".MkdirAll", arguments, &reply)
	return reply, err
}

// Symlink calls the mkdir RPC using the supplied arguments.
func (t *RPC) Symlink(old string, new string) (int, error) {
	arguments := &ociargs.SymlinkArgs{
		Old: old,
		New: new,
	}
	var reply int
	err := t.Client.Call(t.Name+".Symlink", arguments, &reply)
	return reply, err
}

// Touch calls the touch RPC using the supplied arguments.
func (t *RPC) Touch(path string) (int, error) {
	arguments := &ociargs.TouchArgs{
		Path: path,
	}
	var reply int
	err := t.Client.Call(t.Name+".Touch", arguments, &reply)
	return reply, err
}
