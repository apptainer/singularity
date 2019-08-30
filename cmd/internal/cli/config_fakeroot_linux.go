// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
)

// -a|--add
var fakerootConfigAdd bool
var fakerootConfigAddFlag = cmdline.Flag{
	ID:           "fakerootConfigAddFlag",
	Value:        &fakerootConfigAdd,
	DefaultValue: false,
	Name:         "add",
	ShortHand:    "a",
	Usage:        "add a fakeroot mapping entry for a user allowing him to use the fakeroot feature",
}

// -r|--remove
var fakerootConfigRemove bool
var fakerootConfigRemoveFlag = cmdline.Flag{
	ID:           "fakerootConfigRemoveFlag",
	Value:        &fakerootConfigRemove,
	DefaultValue: false,
	Name:         "remove",
	ShortHand:    "r",
	Usage:        "remove the user fakeroot mapping entry preventing him to use the fakeroot feature",
}

// -e|--enable
var fakerootConfigEnable bool
var fakerootConfigEnableFlag = cmdline.Flag{
	ID:           "fakerootConfigEnableFlag",
	Value:        &fakerootConfigEnable,
	DefaultValue: false,
	Name:         "enable",
	ShortHand:    "e",
	Usage:        "enable a user fakeroot mapping entry allowing him to use the fakeroot feature (the user mapping must be present)",
}

// -d|--disable
var fakerootConfigDisable bool
var fakerootConfigDisableFlag = cmdline.Flag{
	ID:           "fakerootConfigDisableFlag",
	Value:        &fakerootConfigDisable,
	DefaultValue: false,
	Name:         "disable",
	ShortHand:    "d",
	Usage:        "disable a user fakeroot mapping entry preventing him to use the fakeroot feature (the user mapping must be present)",
}

// configFakerootCmd singularity config fakeroot
var configFakerootCmd = &cobra.Command{
	Args:                  cobra.ExactArgs(1),
	DisableFlagsInUseLine: true,
	PreRun:                EnsureRootPriv,
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		var op singularity.FakerootConfigOp

		if fakerootConfigAdd {
			op = singularity.FakerootAddUser
		} else if fakerootConfigRemove {
			op = singularity.FakerootRemoveUser
		} else if fakerootConfigEnable {
			op = singularity.FakerootEnableUser
		} else if fakerootConfigDisable {
			op = singularity.FakerootDisableUser
		} else {
			return fmt.Errorf("you must specify an option (eg: --add/--remove)")
		}

		if err := singularity.FakerootConfig(username, op); err != nil {
			sylog.Fatalf("%s", err)
		}

		return nil
	},

	Use:     docs.ConfigFakerootUse,
	Short:   docs.ConfigFakerootShort,
	Long:    docs.ConfigFakerootLong,
	Example: docs.ConfigFakerootExample,
}

func init() {
	cmdManager.RegisterFlagForCmd(&fakerootConfigAddFlag, configFakerootCmd)
	cmdManager.RegisterFlagForCmd(&fakerootConfigRemoveFlag, configFakerootCmd)
	cmdManager.RegisterFlagForCmd(&fakerootConfigEnableFlag, configFakerootCmd)
	cmdManager.RegisterFlagForCmd(&fakerootConfigDisableFlag, configFakerootCmd)
}
