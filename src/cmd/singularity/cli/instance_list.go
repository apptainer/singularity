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

var uid string

func init() {
	InstanceListCmd.Flags().SetInterspersed(false)

	// SingularityCmd.AddCommand(instanceDotListCmd)
	InstanceListCmd.Flags().StringVarP(&uid, "user", "u", "", `If running as root, list instances from "username">`)
}

// InstanceListCmd singularity instance list
var InstanceListCmd = &cobra.Command{
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listing instances")
	},
	DisableFlagsInUseLine: true,

	Use:     docs.InstanceListUse,
	Short:   docs.InstanceListShort,
	Long:    docs.InstanceListLong,
	Example: docs.InstanceListExample,
}

/*
var instanceDotListCmd = &cobra.Command{
	Use:  "instance.list [list options...] [patterns]",
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("listing instances")
	},
	Hidden:                true,
	DisableFlagsInUseLine: true,
}
*/
