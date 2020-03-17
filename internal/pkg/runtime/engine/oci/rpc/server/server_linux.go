// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package server

import (
	"os"
	"syscall"

	"github.com/sylabs/singularity/internal/pkg/util/fs"

	ociargs "github.com/sylabs/singularity/internal/pkg/runtime/engine/oci/rpc"
	args "github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc"
	server "github.com/sylabs/singularity/internal/pkg/runtime/engine/singularity/rpc/server"
	"github.com/sylabs/singularity/internal/pkg/util/mainthread"
)

// Methods is a receiver type.
type Methods struct {
	*server.Methods
}

// MkdirAll performs a mkdir with the specified arguments.
func (t *Methods) MkdirAll(arguments *args.MkdirArgs, reply *int) (err error) {
	mainthread.Execute(func() {
		oldmask := syscall.Umask(0)
		err = os.MkdirAll(arguments.Path, arguments.Perm)
		syscall.Umask(oldmask)
	})
	return err
}

// Touch performs a touch with the specified arguments.
func (t *Methods) Touch(arguments *ociargs.TouchArgs, reply *int) (err error) {
	return fs.Touch(arguments.Path)
}
