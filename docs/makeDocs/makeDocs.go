/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package main

import (
    "github.com/singularityware/singularity/internal/pkg/cli"
    "github.com/spf13/cobra"
    "github.com/spf13/cobra/doc"
    "github.com/golang/glog"
)

func main () {

    makeDoc( "singularity-build",           cli.BuildCmd          )
    makeDoc( "singularity-capability",      cli.CapabilityCmd     )
    makeDoc( "singularity-capability-add",  cli.CapabilityAddCmd  )
    makeDoc( "singularity-capability-drop", cli.CapabilityDropCmd )
    makeDoc( "singularity-capability-list", cli.CapabilityListCmd )
    makeDoc( "singularity-exec",            cli.ExecCmd           )
    makeDoc( "singularity-instance",        cli.InstanceCmd       )
    makeDoc( "singularity-instance-list",   cli.InstanceListCmd   )
    makeDoc( "singularity-instance-start",  cli.InstanceStartCmd  )
    makeDoc( "singularity-instance-stop",   cli.InstanceStopCmd   )
    makeDoc( "singularity-run",             cli.RunCmd            )
    makeDoc( "singularity-shell",           cli.ShellCmd          )
    makeDoc( "singularity-signing",         cli.SignCmd           )
    makeDoc( "singularity-singularity",     cli.SingularityCmd    )

}

/*
makeDoc will generate a man page of a given title for a given Singularity 
command.  
*/
func makeDoc(title string, cmd *cobra.Command) {

    header := &doc.GenManHeader {
        Title: title,
        Section: "1",
    }

    err := doc.GenManTree(cmd, header, "/tmp")
        if err != nil {
            glog.Error("Failed to create man page for %s\n", title)
    }
}

