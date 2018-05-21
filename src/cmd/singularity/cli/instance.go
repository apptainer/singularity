// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	singularityCmd.AddCommand(instanceCmd)
	instanceCmd.AddCommand(instanceStartCmd)
	instanceCmd.AddCommand(instanceStopCmd)
	instanceCmd.AddCommand(instanceListCmd)
}

var instanceCmd = &cobra.Command{
	Use: "instance",
	Run: nil,
	DisableFlagsInUseLine: true,
}
