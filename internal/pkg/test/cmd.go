// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package test

import (
	"testing"

	"gotest.tools/icmd"
)

type Cmd struct {
	path string
}

type Result struct {
	*icmd.Result
}

func NewCmd(path string) *Cmd {
	c := &Cmd{
		path: path,
	}

	return c
}

func (c *Cmd) Run(t *testing.T, args ...string) *Result {
	t.Logf("Running cmd %s with args %s", c.path, args)

	result := icmd.RunCommand(c.path, args...)

	return &Result{result}
}
