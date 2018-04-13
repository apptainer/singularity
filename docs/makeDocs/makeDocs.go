/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package main

import (
    "os"
    "fmt"

    "github.com/singularityware/singularity/internal/pkg/cli"
    "github.com/spf13/cobra"
    "github.com/spf13/cobra/doc"
    "golang.org/x/sys/unix"
)

func main () {
    argv := os.Args
    argc := len(argv)
    dir  := "/tmp" // place to save man pages 

    if argc > 2 {
        fmt.Printf("ERROR: Too many arguments to %s\n", argv[1])
        return
    }

    if argc > 1 {
        dir = argv[1]
    } else if gopath := os.Getenv("GOPATH"); len(gopath) > 0 {
        dir = gopath + "/src/github.com/singularityware/singularity/docs/man"
    }

    if err := unix.Access(dir, unix.W_OK); err != nil {
        fmt.Printf("ERROR: Given directory does not exist or is not writable")
        return
    }

    fmt.Printf("Creating Singularity man pages at %s\n", dir)
    fmt.Printf("If you want to use them, copy them to /usr/share/man\n")

    makeDoc( "singularity-build",           cli.BuildCmd,          dir)
    makeDoc( "singularity-capability",      cli.CapabilityCmd,     dir)
    makeDoc( "singularity-capability-add",  cli.CapabilityAddCmd,  dir)
    makeDoc( "singularity-capability-drop", cli.CapabilityDropCmd, dir)
    makeDoc( "singularity-capability-list", cli.CapabilityListCmd, dir)
    makeDoc( "singularity-exec",            cli.ExecCmd,           dir)
    makeDoc( "singularity-instance",        cli.InstanceCmd,       dir)
    makeDoc( "singularity-instance-list",   cli.InstanceListCmd,   dir)
    makeDoc( "singularity-instance-start",  cli.InstanceStartCmd,  dir)
    makeDoc( "singularity-instance-stop",   cli.InstanceStopCmd,   dir)
    makeDoc( "singularity-run",             cli.RunCmd,            dir)
    makeDoc( "singularity-shell",           cli.ShellCmd,          dir)
    makeDoc( "singularity-signing",         cli.SignCmd,           dir)
    makeDoc( "singularity-singularity",     cli.SingularityCmd,    dir)

}

/*
makeDoc will generate a man page of a given title for a given Singularity 
command.  
*/
func makeDoc(title string, cmd *cobra.Command, dir string) {

    header := &doc.GenManHeader {
        Title: title,
        Section: "1",
    }

    if err := doc.GenManTree(cmd, header, dir); err != nil {
        fmt.Printf("ERROR: Failed to create man page for %s\n", title)
    }
}

