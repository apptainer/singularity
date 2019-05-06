// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"bytes"
	"os/exec"
	"testing"
)

// ExecOpts define options for singularity actions
type ExecOpts struct {
	Binds     []string
	Security  []string
	Overlay   []string
	DropCaps  string
	Home      string
	Workdir   string
	Pwd       string
	App       string
	KeepPrivs bool
	Contain   bool
	NoHome    bool
	Userns    bool
}

// ImageExec can be used to run/exec/shell a Singularity image
// it return the exitCode and err of the execution
func ImageExec(t *testing.T, cmdPath string, action string, opts ExecOpts, imagePath string, command []string) (stdout string, stderr string, exitCode int, err error) {
	// action can be run/exec/shell
	argv := []string{action}
	for _, bind := range opts.Binds {
		argv = append(argv, "--bind", bind)
	}
	for _, sec := range opts.Security {
		argv = append(argv, "--security", sec)
	}
	if opts.KeepPrivs {
		argv = append(argv, "--keep-privs")
	}
	if opts.DropCaps != "" {
		argv = append(argv, "--drop-caps", opts.DropCaps)
	}
	if opts.Contain {
		argv = append(argv, "--contain")
	}
	if opts.Userns {
		argv = append(argv, "--userns")
	}
	if opts.NoHome {
		argv = append(argv, "--no-home")
	}
	if opts.Home != "" {
		argv = append(argv, "--home", opts.Home)
	}
	for _, fs := range opts.Overlay {
		argv = append(argv, "--overlay", fs)
	}
	if opts.Workdir != "" {
		argv = append(argv, "--workdir", opts.Workdir)
	}
	if opts.Pwd != "" {
		argv = append(argv, "--pwd", opts.Pwd)
	}
	if opts.App != "" {
		argv = append(argv, "--app", opts.App)
	}
	argv = append(argv, imagePath)
	argv = append(argv, command...)

	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(cmdPath, argv...)

	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	// retrieve exit code
	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			exitCode = 1
		}
	}

	stdout = outbuf.String()
	stderr = errbuf.String()

	return
}
