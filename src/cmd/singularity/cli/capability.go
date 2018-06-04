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
	SingularityCmd.AddCommand(CapabilityCmd)
	CapabilityCmd.AddCommand(CapabilityAddCmd)
	CapabilityCmd.AddCommand(CapabilityDropCmd)
	CapabilityCmd.AddCommand(CapabilityListCmd)
}

// CapabilityCmd is the capability command
var CapabilityCmd = &cobra.Command{
	Run: nil,
	DisableFlagsInUseLine: true,

	Use:     docs.CapabilityUse,
	Short:   docs.CapabilityShort,
	Long:    docs.CapabilityLong,
	Example: docs.CapabilityExample,
}
