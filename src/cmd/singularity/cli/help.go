// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"

	"github.com/singularityware/singularity/src/docs"
	"github.com/spf13/cobra"
)

func init() {
	HelpCmd.Flags().SetInterspersed(false)
	SingularityCmd.SetHelpCommand(HelpCmd)

	SingularityCmd.AddCommand(HelpCmd)
}

// HelpCmd singularity help
var HelpCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	PreRun:                sylabsToken,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Root().Help()
			return
		}

		c, _, e := cmd.Root().Find(args)
		if _, err := os.Stat(args[0]); err == nil {
			// Help prints (if set) the sourced %help section on the definition file
			a := []string{"/bin/cat", "/.singularity.d/runscript.help"}
			execStarter(cmd, args[0], a, "")
		} else if c == nil || e != nil {
			c.Printf("Unknown help topic %#q\n", args)
			c.Root().Usage()
		} else {
			c.InitDefaultHelpFlag() // make possible 'help' flag to be shown
			c.Help()
		}
	},

	Use:     docs.HelpUse,
	Short:   docs.HelpShort,
	Long:    docs.HelpLong,
	Example: docs.HelpExample,
}
