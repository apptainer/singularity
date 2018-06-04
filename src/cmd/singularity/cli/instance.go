// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
)

func init() {
	SingularityCmd.AddCommand(InstanceCmd)
	InstanceCmd.AddCommand(InstanceStartCmd)
	InstanceCmd.AddCommand(InstanceStopCmd)
	InstanceCmd.AddCommand(InstanceListCmd)
}

// InstanceCmd singularity instance
var InstanceCmd = &cobra.Command{
	Run: nil,
	DisableFlagsInUseLine: true,

	Use:     docs.InstanceUse,
	Short:   docs.InstanceShort,
	Long:    docs.InstanceLong,
	Example: docs.InstanceExample,
}
