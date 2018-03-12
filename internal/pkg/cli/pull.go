/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"github.com/singularityware/singularity/pkg/libexec"
	"github.com/spf13/cobra"
)

func init() {
	singularityCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:  "pull",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		libexec.PullImage(args[0], args[1])
	},
}
