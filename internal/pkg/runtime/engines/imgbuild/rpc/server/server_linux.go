// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package server

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/sylabs/singularity/internal/pkg/build/files"

	buildargs "github.com/sylabs/singularity/internal/pkg/runtime/engines/imgbuild/rpc"
	server "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/rpc/server"
)

// Methods is a receiver type.
type Methods struct {
	*server.Methods
}

// Copy performs a file copy with the specified arguments.
func (t *Methods) Copy(arguments *buildargs.CopyArgs, reply *int) (err error) {
	return files.Copy(arguments.Source, arguments.Dest)
}

// RunScript executes a section script.
func (t *Methods) RunScript(arguments *buildargs.RunScriptArgs, reply *int) (err error) {
	var b bytes.Buffer
	if _, err := b.WriteString(arguments.Script); err != nil {
		return fmt.Errorf("failed to write script on stdin: %v", err)
	}

	cmd := exec.Command("/bin/sh", arguments.Args...)
	cmd.Env = arguments.Envs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = &b

	return cmd.Run()
}
