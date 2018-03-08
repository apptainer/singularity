/*
Copyright (c) 2018, Sylabs, Inc. All rights reserved.

This software is licensed under a 3-clause BSD license.  Please
consult LICENSE file distributed with the sources of this project regarding
your rights to use or distribute this software.
*/
package cli

import (
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Global variables for singularity CLI
var (
	debug   bool
	silent  bool
	verbose bool
	quiet   bool
)

// singularity is the base command when called without any subcommands
var singularityCmd = &cobra.Command{
	Use: "singularity [global options...]",
	DisableFlagsInUseLine: true,
	Run: nil,
}

/*
Execute adds all child commands to the root command and sets flags
appropriately.  This is called by main.main(). It only needs to happen once to
the root command (singularity).
*/
func Execute() {
	if err := singularityCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func TraverseParentsUses(cmd *cobra.Command) string {
	if cmd.HasParent() {
		return TraverseParentsUses(cmd.Parent()) + cmd.Use + " "
	}

	return cmd.Use + " "
}

func PrintFlagUsages(flagSet *pflag.FlagSet) string {
	return strings.Replace(flagSet.FlagUsages(), ", ", "|", 1)
}

func init() {
	templateFuncs := template.FuncMap{
		"PrintFlagUsages":     PrintFlagUsages,
		"TraverseParentsUses": TraverseParentsUses,
	}

	singularityCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Print debugging information")
	singularityCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "Only print errors")
	singularityCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all normal output")
	singularityCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Increase verbosity +1")

	cobra.AddTemplateFuncs(templateFuncs)

	singularityCmd.SetHelpTemplate(
		`{{.UsageString}}{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}
Global Options:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasExample}}
Examples:{{.Example}}
{{end}}
For additional help, please visit our public documentation pages which are
found at:

    https://sylabs.io/
`)

	singularityCmd.SetUsageTemplate(
		`Usage:
  {{TraverseParentsUses . | trimTrailingWhitespaces}}{{if .HasAvailableSubCommands}} <command> 

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}`)

	/*
			singularityCmd.SetHelpTemplate(
				`{{if .HasParent}}Usage:{{if .Runnable}}
		  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
		  {{.CommandPath}} [options...] <command>{{end}}{{else}}Usage:
		  {{.CommandPath}} [global options...] <command>
		{{end}}{{if gt (len .Aliases) 0}}
		Aliases:
		  {{.NameAndAliases}}
		{{end}}{{if .HasExample}}
		Examples:
		{{.Example}}
		{{end}}{{if .HasAvailableSubCommands}}
		Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
		  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}
		{{end}}{{if .HasAvailableLocalFlags}}
		Flags:
		{{PrintFlagUsages .LocalFlags | trimTrailingWhitespaces}}
		{{end}}{{if .HasAvailableInheritedFlags}}
		Global Flags:
		{{PrintFlagUsages .InheritedFlags | trimTrailingWhitespaces}}
		{{end}}{{if .HasHelpSubCommands}}
		Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
		  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}
		{{end}}{{if .HasAvailableSubCommands}}
		Use "{{.CommandPath}} <command> --help" for more information about a command.{{end}}
		`)

			/*
						singularityCmd.SetHelpTemplate(`
					USAGE: singularity [global options...] <command> [command options...] ...

					GLOBAL OPTIONS:
					    -d|--debug    Print debugging information
					    -h|--help     Display usage summary
					    -s|--silent   Only print errors
					    -q|--quiet    Suppress all normal output
					    --version  Show application version
					    -v|--verbose  Increase verbosity +1
					    -x|--sh-debug Print shell wrapper debugging information

					GENERAL COMMANDS:
					    help       Show additional help for a command or container
					    selftest   Run some self tests for singularity install

					CONTAINER USAGE COMMANDS:
					    exec       Execute a command within container
					    run        Launch a runscript within container
					    shell      Run a Bourne shell within container
					    test       Launch a testscript within container

					CONTAINER MANAGEMENT COMMANDS:
					    apps       List available apps within a container
					    bootstrap  *Deprecated* use build instead
					    build      Build a new Singularity container
					    check      Perform container lint checks
					    inspect    Display container's metadata
					    mount      Mount a Singularity container image
					    pull       Pull a Singularity/Docker container to $PWD

					COMMAND GROUPS:
					    image      Container image command group
					    instance   Persistent instance command group


					CONTAINER USAGE OPTIONS:
					    see singularity help <command>

					For any additional help or support visit the Singularity
					website: http://singularity.lbl.gov/
					`)

					singularityCmd.SetUsageTemplate(`
				USAGE: singularity [global options...] <command> [command options...] ...
				    `)
	*/
}
