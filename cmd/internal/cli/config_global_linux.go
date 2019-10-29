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

// -s|--set
var globalConfigSet bool
var globalConfigSetFlag = cmdline.Flag{
	ID:           "globalConfigSetFlag",
	Value:        &globalConfigSet,
	DefaultValue: false,
	Name:         "set",
	ShortHand:    "s",
	Usage:        "set value of the configuration directive (for multi-value directives, it will add it)",
}

// -u|--unset
var globalConfigUnset bool
var globalConfigUnsetFlag = cmdline.Flag{
	ID:           "globalConfigUnsetFlag",
	Value:        &globalConfigUnset,
	DefaultValue: false,
	Name:         "unset",
	ShortHand:    "u",
	Usage:        "unset value of the configuration directive (for multi-value directives, it will remove matching values)",
}

// -g|--get
var globalConfigGet bool
var globalConfigGetFlag = cmdline.Flag{
	ID:           "globalConfigGetFlag",
	Value:        &globalConfigGet,
	DefaultValue: false,
	Name:         "get",
	ShortHand:    "g",
	Usage:        "get value of the configuration directive",
}

// -r|--reset
var globalConfigReset bool
var globalConfigResetFlag = cmdline.Flag{
	ID:           "globalConfigResetFlag",
	Value:        &globalConfigReset,
	DefaultValue: false,
	Name:         "reset",
	ShortHand:    "r",
	Usage:        "reset the configuration directive value to its default value",
}

// -d|--dry-run
var globalConfigDryRun bool
var globalConfigDryRunFlag = cmdline.Flag{
	ID:           "globalConfigDryRunFlag",
	Value:        &globalConfigDryRun,
	DefaultValue: false,
	Name:         "dry-run",
	ShortHand:    "d",
	Usage:        "dump resulting configuration on stdout but doesn't write it to singularity.conf",
}

// configGlobalCmd singularity config global
var configGlobalCmd = &cobra.Command{
	Args:                  cobra.RangeArgs(1, 2),
	DisableFlagsInUseLine: true,
	PreRun:                EnsureRootPriv,
	RunE: func(cmd *cobra.Command, args []string) error {
		var op singularity.GlobalConfigOp

		if globalConfigSet {
			op = singularity.GlobalConfigSet
		} else if globalConfigUnset {
			op = singularity.GlobalConfigUnset
		} else if globalConfigReset {
			op = singularity.GlobalConfigReset
		} else if globalConfigGet {
			op = singularity.GlobalConfigGet
		} else {
			return fmt.Errorf("you must specify an option (eg: --set/--unset)")
		}

		if err := singularity.GlobalConfig(args, globalConfigDryRun, op); err != nil {
			sylog.Fatalf("%s", err)
		}

		return nil
	},

	Use:     docs.ConfigGlobalUse,
	Short:   docs.ConfigGlobalShort,
	Long:    docs.ConfigGlobalLong,
	Example: docs.ConfigGlobalExample,
}

func init() {
	cmdManager.RegisterFlagForCmd(&globalConfigSetFlag, configGlobalCmd)
	cmdManager.RegisterFlagForCmd(&globalConfigUnsetFlag, configGlobalCmd)
	cmdManager.RegisterFlagForCmd(&globalConfigGetFlag, configGlobalCmd)
	cmdManager.RegisterFlagForCmd(&globalConfigResetFlag, configGlobalCmd)
	cmdManager.RegisterFlagForCmd(&globalConfigDryRunFlag, configGlobalCmd)
}
