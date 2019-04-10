// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/pkg/cmdline"
)

// -u|--user
var instanceStopUserFlag = cmdline.Flag{
	ID:           "instanceStopUserFlag",
	Value:        &username,
	DefaultValue: "",
	Name:         "user",
	ShortHand:    "u",
	Usage:        "If running as root, stop instances belonging to user",
	Tag:          "<username>",
	EnvKeys:      []string{"USER"},
}

// -a|--all
var instanceStopAllFlag = cmdline.Flag{
	ID:           "instanceStopAllFlag",
	Value:        &stopAll,
	DefaultValue: false,
	Name:         "all",
	ShortHand:    "a",
	Usage:        "stop all user's instances",
	EnvKeys:      []string{"ALL"},
}

// -f|--force
var instanceStopForceFlag = cmdline.Flag{
	ID:           "instanceStopForceFlag",
	Value:        &forceStop,
	DefaultValue: false,
	Name:         "force",
	ShortHand:    "F",
	Usage:        "force kill instance",
	EnvKeys:      []string{"FORCE"},
}

// -s|--signal
var instanceStopSignalFlag = cmdline.Flag{
	ID:           "instanceStopSignalFlag",
	Value:        &stopSignal,
	DefaultValue: "",
	Name:         "signal",
	ShortHand:    "s",
	Usage:        "signal sent to the instance",
	Tag:          "<signal>",
	EnvKeys:      []string{"SIGNAL"},
}

// -t|--timeout
var instanceStopTimeoutFlag = cmdline.Flag{
	ID:           "instanceStopTimeoutFlag",
	Value:        &stopTimeout,
	DefaultValue: 10,
	Name:         "timeout",
	ShortHand:    "t",
	Usage:        "force kill non stopped instances after X seconds",
}

func init() {
	cmdManager.RegisterCmdFlag(&instanceStopUserFlag, InstanceStopCmd)
	cmdManager.RegisterCmdFlag(&instanceStopAllFlag, InstanceStopCmd)
	cmdManager.RegisterCmdFlag(&instanceStopForceFlag, InstanceStopCmd)
	cmdManager.RegisterCmdFlag(&instanceStopSignalFlag, InstanceStopCmd)
	cmdManager.RegisterCmdFlag(&instanceStopTimeoutFlag, InstanceStopCmd)
}

// InstanceStopCmd singularity instance stop
var InstanceStopCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && !stopAll {
			stopInstance(args[0])
			return nil
		} else if stopAll {
			stopInstance("*")
			return nil
		} else {
			return errors.New("Invalid command")
		}
	},

	Use:     docs.InstanceStopUse,
	Short:   docs.InstanceStopShort,
	Long:    docs.InstanceStopLong,
	Example: docs.InstanceStopExample,
}
