// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
)

func init() {
	// SingularityCmd.AddCommand(instanceDotStopCmd)
	InstanceStopCmd.Flags().SetInterspersed(false)
}

// InstanceStopCmd singularity instance stop
var InstanceStopCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stopping instance")
	},

	Use:     docs.InstanceStopUse,
	Short:   docs.InstanceStopShort,
	Long:    docs.InstanceStopLong,
	Example: docs.InstanceStopExample,
}

/*
var instanceDotStopCmd = &cobra.Command{
	Use:    "instance.stop",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stopping instance")
	},
}
*/
