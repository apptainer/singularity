// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	buildargs "github.com/sylabs/singularity/internal/pkg/runtime/engines/imgbuild/rpc"
	client "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/client"
)

// RPC holds the state necessary for remote procedure calls.
type RPC struct {
	client.RPC
}

// Copy calls the copy RPC using the supplied arguments.
func (t *RPC) Copy(source string, dest string) (int, error) {
	arguments := &buildargs.CopyArgs{
		Source: source,
		Dest:   dest,
	}
	var reply int
	err := t.Client.Call(t.Name+".Copy", arguments, &reply)
	return reply, err
}

// RunScript calls the RunScript RPC methods using the supplied arguments.
func (t *RPC) RunScript(script string, args, envs []string) (int, error) {
	arguments := &buildargs.RunScriptArgs{
		Script: script,
		Args:   args,
		Envs:   envs,
	}
	var reply int
	err := t.Client.Call(t.Name+".RunScript", arguments, &reply)
	return reply, err
}
