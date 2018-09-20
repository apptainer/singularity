// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os/exec"
)

type execOpts struct {
	binds   []string
	contain bool
	home    string
	workdir string
	pwd     string
}

func imageExec(opts execOpts, imagePath string, command []string) ([]byte, error) {
	argv := []string{"exec"}
	for _, bind := range opts.binds {
		argv = append(argv, "--bind", bind)
	}
	if opts.contain {
		argv = append(argv, "--contain")
	}
	if opts.home != "" {
		argv = append(argv, "--home", opts.home)
	}
	if opts.workdir != "" {
		argv = append(argv, "--workdir", opts.workdir)
	}
	if opts.pwd != "" {
		argv = append(argv, "--pwd", opts.pwd)
	}
	argv = append(argv, imagePath)
	argv = append(argv, command...)

	return exec.Command(cmdPath, argv...).CombinedOutput()
}
