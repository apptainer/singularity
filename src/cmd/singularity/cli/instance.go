/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"github.com/singularityware/singularity/docs"
	"github.com/spf13/cobra"
)

var instanceUse string = `instance <subcommand>`

var instanceShort string = `Manage containers running in the background`

var instanceLong string = `
Instances allow you to run containers as background processes. This can be 
useful for running services such as web servers or databases.`

var instanceExample string = `
All group commands have their own help output:

$ singularity help instance.start
$ singularity instance.start --help`

func init() {

	manHelp := func(c *cobra.Command, args []string) {
		docs.DispManPg("singularity-instance")
	}

	SingularityCmd.AddCommand(InstanceCmd)
	InstanceCmd.SetHelpFunc(manHelp)
	InstanceCmd.AddCommand(InstanceStartCmd)
	InstanceCmd.AddCommand(InstanceStopCmd)
	InstanceCmd.AddCommand(InstanceListCmd)
}

var InstanceCmd = &cobra.Command{
	Run: nil,
	DisableFlagsInUseLine: true,

	Use:     instanceUse,
	Short:   instanceShort,
	Long:    instanceLong,
	Example: instanceExample,
}
