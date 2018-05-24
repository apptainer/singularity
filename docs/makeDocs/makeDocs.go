/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package main

import (
	"fmt"
	"os"

	"github.com/singularityware/singularity/src/cmd/singularity/cli"
	// "github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"golang.org/x/sys/unix"
)

func main() {
	argv := os.Args
	argc := len(argv)
	dir := "/tmp" // place to save man pages

	if argc > 2 {
		fmt.Printf("ERROR: Too many arguments to %s\n", argv[1])
		return
	}

	// if the user supplied a directory argument try to save man pages there
	if argc > 1 {
		dir = argv[1]
		// otherwise try to save in the $GOPATH if it exits (failing both of these
		// options, default is to save into /tmp
	} else if gopath := os.Getenv("GOPATH"); len(gopath) > 0 {
		dir = gopath + "/src/github.com/singularityware/singularity/docs/man"
	}

	if err := unix.Access(dir, unix.W_OK); err != nil {
		fmt.Printf("ERROR: Given directory does not exist or is not writable by calling user.")
		return
	}

	fmt.Printf("Creating Singularity man pages at %s\n", dir)

	header := &doc.GenManHeader{
		Title:   "singularity",
		Section: "1",
	}

	// works recursively on all sub-commands (thanks bauerm97)
	if err := doc.GenManTree(cli.SingularityCmd, header, dir); err != nil {
		fmt.Printf("ERROR: Failed to create man page for singularity\n")
	}
}
