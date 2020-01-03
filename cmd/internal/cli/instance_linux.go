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

func init() {
	addCmdInit(func(cmdManager *cmdline.CommandManager) {
		cmdManager.RegisterCmd(instanceCmd)
		cmdManager.RegisterSubCmd(instanceCmd, instanceStartCmd)
		cmdManager.RegisterSubCmd(instanceCmd, instanceStopCmd)
		cmdManager.RegisterSubCmd(instanceCmd, instanceListCmd)
	})
}

// singularity instance
var instanceCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("invalid command")
	},
	DisableFlagsInUseLine: true,

	Use:           docs.InstanceUse,
	Short:         docs.InstanceShort,
	Long:          docs.InstanceLong,
	Example:       docs.InstanceExample,
	SilenceErrors: true,
}
