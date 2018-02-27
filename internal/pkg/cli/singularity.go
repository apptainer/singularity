/*
Copyright (c) 2018, Sylabs, Inc. All rights reserved.
This software is licensed under a 3-clause BSD license.  Please
consult LICENSE file distributed with the sources of this project regarding
your rights to use or distribute this software.
*/
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// singularity is the base command when called without any subcommands
var singularityCmd = &cobra.Command{Use: "singularity"}

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

var verbose bool

func init() {

	singularityCmd.PersistentFlags().BoolP("debug", "d", false, "")
	singularityCmd.PersistentFlags().BoolP("silent", "s", false, "")
	singularityCmd.PersistentFlags().BoolP("quiet", "q", false, "")
	singularityCmd.PersistentFlags().BoolP("verbose", "v", false, "")

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
}
