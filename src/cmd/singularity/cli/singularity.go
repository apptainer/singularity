/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cli

import (
	"os"
    "text/template"

	"github.com/spf13/cobra"

	// "github.com/singularityware/singularity/docs"
)

// Global variables for singularity CLI
var (
	debug   bool
	silent  bool
	verbose bool
	quiet   bool
)

func init() {

    /*
	manHelp := func(c *cobra.Command, args []string) {
		docs.DispManPg("singularity")
	}

	SingularityCmd.SetHelpFunc(manHelp)
    */
	SingularityCmd.Flags().SetInterspersed(false)
	SingularityCmd.PersistentFlags().SetInterspersed(false)
    templateFuncs := template.FuncMap{
        "TraverseParentsUses": TraverseParentsUses,
    }
    cobra.AddTemplateFuncs(templateFuncs)

    SingularityCmd.SetHelpTemplate(
`Usage:
  {{.UseLine}}{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsagesWrapped 80 | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}

Global Options:
{{.InheritedFlags.FlagUsagesWrapped 80 | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}
Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasExample}}

Examples:{{.Example}}{{end}}

For additional help or support, please visit:

    https://docs.sylabs.io/
`)

    SingularityCmd.SetUsageTemplate(
        `Usage:
  {{TraverseParentsUses . | trimTrailingWhitespaces}}{{if .HasAvailableSubCommands}} <command>

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}`)


	SingularityCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Print debugging information")
	SingularityCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Only print errors")
	SingularityCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all normal output")
	SingularityCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Increase verbosity +1")

}

// singularity is the base command when called without any subcommands
var SingularityCmd = &cobra.Command{
	TraverseChildren:      true,
	DisableFlagsInUseLine: true,
	Run: nil,

	Use: "singularity [global options...]",

	Short: `a Linux container platform optimized for High Performance Computing 
(HPC) and Enterprise Performance Computing (EPC)`,

	Long: `Singularity containers provide an application virtualization layer 
enabling mobility of compute via both application and environment portability. 
With Singularity one is capable of building a root file system and running that 
root file system on any other Linux system where Singularity is installed.`,

	Example: `
$ singularity help
    Will print a generalized usage summary and available commands.

$ singularity help <command>
    Additional help for any Singularity subcommand can be seen by appending the subcommand name to the above command.`,
}

/*
Execute adds all child commands to the root command and sets flags
appropriately.  This is called by main.main(). It only needs to happen once to
the root command (singularity).
*/
func ExecuteSingularity() {
	if err := SingularityCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func TraverseParentsUses(cmd *cobra.Command) string {
	if cmd.HasParent() {
		return TraverseParentsUses(cmd.Parent()) + cmd.Use + " "
	}

	return cmd.Use + " "
}
