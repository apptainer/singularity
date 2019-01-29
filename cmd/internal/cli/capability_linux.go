// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
)

// contains flag variables for capability commands
var (
	CapUser    string
	CapGroup   string
	CapDesc    bool
	CapListAll bool
)

func init() {
	SingularityCmd.AddCommand(CapabilityCmd)
	CapabilityCmd.AddCommand(CapabilityAddCmd)
	CapabilityCmd.AddCommand(CapabilityDropCmd)
	CapabilityCmd.AddCommand(CapabilityListCmd)
}

// CapabilityCmd is the capability command
var CapabilityCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.CapabilityUse,
	Short:         docs.CapabilityShort,
	Long:          docs.CapabilityLong,
	Example:       docs.CapabilityExample,
	SilenceErrors: true,
}
