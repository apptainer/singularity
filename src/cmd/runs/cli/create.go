// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/singularityware/singularity/src/docs"
	"github.com/singularityware/singularity/src/pkg/sylog"

	"github.com/spf13/cobra"
)

func init() {
	CreateCmd.Flags().SetInterspersed(false)

	cwd, err := os.Getwd()
	if err != nil {
		sylog.Fatalf("%v", err)
	}

	CreateCmd.Flags().StringVarP(&bundlePath, "bundle", "b", cwd, "path to singularity image file (SIF), default to current directory")
	ExecRunCmd.AddCommand(CreateCmd)
}

// CreateCmd singularity instance list
var CreateCmd = &cobra.Command{
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

	},
	DisableFlagsInUseLine: true,

	Use:   docs.RunsCreateUse,
	Short: docs.RunsCreateShort,
	Long:  docs.RunsCreateLong,
}
