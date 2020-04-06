// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/cmdline"
	"github.com/sylabs/singularity/pkg/sylog"
)

// CapConfig contains flag variables for capability commands
type CapConfig struct {
	CapUser  string
	CapGroup string
}

var capConfig = new(CapConfig)

// -u|--user
var capUserFlag = cmdline.Flag{
	ID:           "capUserFlag",
	Value:        &capConfig.CapUser,
	DefaultValue: "",
	Name:         "user",
	ShortHand:    "u",
	Usage:        "manage capabilities for a user",
	EnvKeys:      []string{"CAP_USER"},
}

// -g|--group
var capGroupFlag = cmdline.Flag{
	ID:           "capGroupFlag",
	Value:        &capConfig.CapGroup,
	DefaultValue: "",
	Name:         "group",
	ShortHand:    "g",
	Usage:        "manage capabilities for a group",
	EnvKeys:      []string{"CAP_GROUP"},
}

// CapabilityAvailCmd singularity capability avail
var CapabilityAvailCmd = &cobra.Command{
	Args:                  cobra.RangeArgs(0, 1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		caps := ""
		if len(args) > 0 {
			caps = args[0]
		}
		c := singularity.CapAvailConfig{
			Caps: caps,
			Desc: len(args) == 0,
		}
		if err := singularity.CapabilityAvail(c); err != nil {
			sylog.Fatalf("Unable to list available capabilities: %s", err)
		}
	},

	Use:     docs.CapabilityAvailUse,
	Short:   docs.CapabilityAvailShort,
	Long:    docs.CapabilityAvailLong,
	Example: docs.CapabilityAvailExample,
}

// CapabilityAddCmd singularity capability add
var CapabilityAddCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		c := singularity.CapManageConfig{
			Caps:  args[0],
			User:  capConfig.CapUser,
			Group: capConfig.CapGroup,
		}

		if err := singularity.CapabilityAdd(buildcfg.CAPABILITY_FILE, c); err != nil {
			sylog.Fatalf("Unable to add capabilities: %s", err)
		}
	},

	Use:     docs.CapabilityAddUse,
	Short:   docs.CapabilityAddShort,
	Long:    docs.CapabilityAddLong,
	Example: docs.CapabilityAddExample,
}

// CapabilityDropCmd singularity capability drop
var CapabilityDropCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		c := singularity.CapManageConfig{
			Caps:  args[0],
			User:  capConfig.CapUser,
			Group: capConfig.CapGroup,
		}

		if err := singularity.CapabilityDrop(buildcfg.CAPABILITY_FILE, c); err != nil {
			sylog.Fatalf("Unable to drop capabilities: %s", err)
		}
	},

	Use:     docs.CapabilityDropUse,
	Short:   docs.CapabilityDropShort,
	Long:    docs.CapabilityDropLong,
	Example: docs.CapabilityDropExample,
}

// CapabilityListCmd singularity capability list
var CapabilityListCmd = &cobra.Command{
	Args:                  cobra.RangeArgs(0, 1),
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		userGroup := ""
		if len(args) == 1 {
			userGroup = args[0]
		}
		c := singularity.CapListConfig{
			User:  userGroup,
			Group: userGroup,
			All:   len(args) == 0,
		}

		if err := singularity.CapabilityList(buildcfg.CAPABILITY_FILE, c); err != nil {
			sylog.Fatalf("Unable to list capabilities: %s", err)
		}
	},

	Use:     docs.CapabilityListUse,
	Short:   docs.CapabilityListShort,
	Long:    docs.CapabilityListLong,
	Example: docs.CapabilityListExample,
}

// CapabilityCmd is the capability command
var CapabilityCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Invalid command")
	},
	DisableFlagsInUseLine: true,

	Aliases:       []string{"caps"},
	Use:           docs.CapabilityUse,
	Short:         docs.CapabilityShort,
	Long:          docs.CapabilityLong,
	Example:       docs.CapabilityExample,
	SilenceErrors: true,
}

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(CapabilityCmd)

		cmdManager.RegisterSubCmd(CapabilityCmd, CapabilityAddCmd)
		cmdManager.RegisterSubCmd(CapabilityCmd, CapabilityDropCmd)
		cmdManager.RegisterSubCmd(CapabilityCmd, CapabilityListCmd)
		cmdManager.RegisterSubCmd(CapabilityCmd, CapabilityAvailCmd)

		cmdManager.RegisterFlagForCmd(&capUserFlag, CapabilityAddCmd, CapabilityDropCmd)
		cmdManager.RegisterFlagForCmd(&capGroupFlag, CapabilityAddCmd, CapabilityDropCmd)
	})
}
