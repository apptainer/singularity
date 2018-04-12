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
	SingularityCmd.AddCommand(CapabilityCmd)
	CapabilityCmd.AddCommand(CapabilityAddCmd)
	CapabilityCmd.AddCommand(CapabilityDropCmd)
	CapabilityCmd.AddCommand(CapabilityListCmd)
}

var CapabilityCmd = &cobra.Command{
	Use: "capability <subcommand>",
	Run: nil,
	DisableFlagsInUseLine: true,
}
