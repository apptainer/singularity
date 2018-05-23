// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"

	"github.com/singularityware/singularity/docs"
)

func init() {
	SingularityCmd.AddCommand(CapabilityCmd)
	CapabilityCmd.AddCommand(CapabilityAddCmd)
	CapabilityCmd.AddCommand(CapabilityDropCmd)
	CapabilityCmd.AddCommand(CapabilityListCmd)
}

var CapabilityCmd = &cobra.Command{
	Run: nil,
	DisableFlagsInUseLine: true,

	Use:     docs.CapabilityUse,
	Short:   docs.CapabilityShort,
	Long:    docs.CapabilityLong,
	Example: docs.CapabilityExample,
}
