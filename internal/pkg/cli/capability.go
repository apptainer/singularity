/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	singularityCmd.AddCommand(capabilityCmd)
	capabilityCmd.AddCommand(capabilityAddCmd)
	capabilityCmd.AddCommand(capabilityDropCmd)
	capabilityCmd.AddCommand(capabilityListCmd)
}

var capabilityCmd = &cobra.Command{
	Use: "capability <subcommand>",
	Run: nil,
	DisableFlagsInUseLine: true,
}
