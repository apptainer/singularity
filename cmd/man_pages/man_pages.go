// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os"

	"github.com/spf13/cobra/doc"
	"github.com/sylabs/singularity/cmd/internal/cli"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"golang.org/x/sys/unix"
)

func main() {
	argv := os.Args
	argc := len(argv)
	dir := "/tmp" // place to save man pages

	if argc > 2 {
		sylog.Fatalf("Too many arguments to %s\n", argv[1])
	}

	// if the user supplied a directory argument try to save man pages there
	if argc > 1 {
		dir = argv[1]
		// otherwise try to save in the $GOPATH if it exits (failing both of these
		// options, default is to save into /tmp
	} else if gopath := os.Getenv("GOPATH"); len(gopath) > 0 {
		dir = gopath + "/src/github.com/sylabs/singularity/docs/man"
	}

	if err := unix.Access(dir, unix.W_OK); err != nil {
		sylog.Fatalf("Given directory does not exist or is not writable by calling user.")
	}

	sylog.Infof("Creating Singularity man pages at %s\n", dir)

	header := &doc.GenManHeader{
		Title:   "singularity",
		Section: "1",
	}

	// works recursively on all sub-commands (thanks bauerm97)
	if err := doc.GenManTree(cli.SingularityCmd, header, dir); err != nil {
		sylog.Fatalf("Failed to create man page for singularity\n")
	}
}
