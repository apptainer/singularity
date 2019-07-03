// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package rpc

// CopyArgs defines the arguments to copy.
type CopyArgs struct {
	Source string
	Dest   string
}

// RunScriptArgs defines the arguments to run script.
type RunScriptArgs struct {
	Script string
	Args   []string
	Envs   []string
}
