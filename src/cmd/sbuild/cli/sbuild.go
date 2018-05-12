/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package cli

import (
	"context"
	"os"
	"strings"

	// "github.com/singularityware/singularity/docs"
	"github.com/singularityware/singularity/src/pkg/build"
	"github.com/spf13/cobra"
)

var sbuildUse string = `sbuild <builder type> <definition json> <image path>`

var sbuildShort string = ``

var sbuildLong string = ``

var sbuildExample string = ``

func init() {
	sbuildCmd.Flags().SetInterspersed(false)
}

// ExecuteSbuild executes the image build wrapper
func ExecuteSbuild() {
	if err := sbuildCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var sbuildCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Args: cobra.ExactArgs(3),

	Run: func(cmd *cobra.Command, args []string) {
		var b build.Builder
		var err error

		if args[0] == "sif" {
			b, err = build.NewSifBuilderJSON(strings.NewReader(args[1]), args[2])
			if err != nil {
				return
			}
		} else {
			return
		}

		b.Build(context.TODO())
	},

	Use:     sbuildUse,
	Short:   sbuildShort,
	Long:    sbuildLong,
	Example: sbuildExample,
}
