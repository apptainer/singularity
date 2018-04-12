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

    makeDoc("singularity-build", cli.BuildCmd)
    //  makeDoc("singularity-capability", cli.capability)
    //  makeDoc("singularity-capability-add", cli.capability_add)
    //  makeDoc("singularity-capability-drop", cli.capability_drop)
    //  makeDoc("singularity-capability-list", cli.capability_list)
    makeDoc("singularity-exec", cli.ExecCmd)
    //  makeDoc("singularity-instance", cli.instance)
    //  makeDoc("singularity-instance-list", cli.insance_list)
    //  makeDoc("singularity-instance-start", cli.instance_start)
    //  makeDoc("singularity-instance-stop", cli.instance_stop)
    //  makeDoc("singularity-signing", cli.singing)
    //  makeDoc("singularity-singularity", cli.singularity)

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

