// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
	"github.com/sylabs/singularity/internal/app/singularity"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/cmdline"
)

func init() {
	cmdManager.RegisterFlagForCmd(&instanceStartPidFileFlag, instanceStartCmd)
}

// --pid-file
var instanceStartPidFile string
var instanceStartPidFileFlag = cmdline.Flag{
	ID:           "instanceStartPidFileFlag",
	Value:        &instanceStartPidFile,
	DefaultValue: "",
	Name:         "pid-file",
	Usage:        "Write instance PID to the file with the given name",
	EnvKeys:      []string{"PID_FILE"},
}

// singularity instance start
var instanceStartCmd = &cobra.Command{
	Args:                  cobra.MinimumNArgs(2),
	PreRun:                actionPreRun,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		image := args[0]
		name := args[1]

		a := append([]string{"/.singularity.d/actions/start"}, args[2:]...)
		setVM(cmd)
		if VM {
			execVM(cmd, image, a)
			return
		}
		execStarter(cmd, image, a, name)

		if instanceStartPidFile != "" {
			err := singularity.WriteInstancePidFile(name, instanceStartPidFile)
			if err != nil {
				sylog.Warningf("Failed to write pid file: %v", err)
			}
		}
	},

	Use:     docs.InstanceStartUse,
	Short:   docs.InstanceStartShort,
	Long:    docs.InstanceStartLong,
	Example: docs.InstanceStartExample,
}
