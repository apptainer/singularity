// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build linux

package cli

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/src/docs"
)

func init() {
	InstanceStopCmd.Flags().SetInterspersed(false)

	// -u|--user
	InstanceStopCmd.Flags().StringVarP(&username, "user", "u", "", `if running as root, list instances from "<username>"`)
	InstanceStopCmd.Flags().SetAnnotation("user", "argtag", []string{"<username>"})
	InstanceStopCmd.Flags().SetAnnotation("user", "envkey", []string{"USER"})

	// -a|--all
	InstanceStopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "stop all user's instances")
	InstanceStopCmd.Flags().SetAnnotation("all", "envkey", []string{"ALL"})

	// -f|--force
	InstanceStopCmd.Flags().BoolVarP(&forceStop, "force", "F", false, "force kill instance")
	InstanceStopCmd.Flags().SetAnnotation("force", "envkey", []string{"FORCE"})

	// -s|--signal
	InstanceStopCmd.Flags().StringVarP(&stopSignal, "signal", "s", "", "signal sent to the instance")
	InstanceStopCmd.Flags().SetAnnotation("signal", "argtag", []string{"<signal>"})
	InstanceStopCmd.Flags().SetAnnotation("signal", "envkey", []string{"SIGNAL"})

	// -t|--timeout
	InstanceStopCmd.Flags().IntVarP(&stopTimeout, "timeout", "t", 10, "force kill non stopped instances after X seconds")
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
