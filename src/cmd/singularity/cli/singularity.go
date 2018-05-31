// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"os"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/singularityware/singularity/src/docs"
)

// Global variables for singularity CLI
var (
	debug   bool
	silent  bool
	verbose bool
	quiet   bool
)

func init() {
	SingularityCmd.Flags().SetInterspersed(false)
	SingularityCmd.PersistentFlags().SetInterspersed(false)

	templateFuncs := template.FuncMap{
		"TraverseParentsUses": TraverseParentsUses,
	}
	cobra.AddTemplateFuncs(templateFuncs)

	SingularityCmd.SetHelpTemplate(docs.HelpTemplate)
	SingularityCmd.SetUsageTemplate(docs.UseTemplate)

	SingularityCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Print debugging information")
	SingularityCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Only print errors")
	SingularityCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all normal output")
	SingularityCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Increase verbosity +1")

}

// SingularityCmd is the base command when called without any subcommands
var SingularityCmd = &cobra.Command{
	TraverseChildren:      true,
	DisableFlagsInUseLine: true,
	Run: nil,

	Use:     docs.SingularityUse,
	Short:   docs.SingularityShort,
	Long:    docs.SingularityLong,
	Example: docs.SingularityExample,
}

// ExecuteSingularity adds all child commands to the root command and sets
// flags appropriately. This is called by main.main(). It only needs to happen
// once to the root command (singularity).
func ExecuteSingularity() {
	if err := SingularityCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// TraverseParentsUses walks the parent commands and outputs a properly formatted use string
func TraverseParentsUses(cmd *cobra.Command) string {
	if cmd.HasParent() {
		return TraverseParentsUses(cmd.Parent()) + cmd.Use + " "
	}

	return cmd.Use + " "
}
