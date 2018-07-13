// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docs

// Global content for Runs CLI help and man pages
const (
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// main Runs command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunsUse   string = `runs [global options...]`
	RunsShort string = `
Linux container platform based on singularity runtime OCi runtime compliant`
	RunsLong string = `
Runs containers provide an application virtualization layer enabling
mobility of compute via both application and environment portability.`
	RunsExample string = `
$ runs help
    Will print a generalized usage summary and available commands.

$ runs help <command>
    Additional help for any runs subcommand can be seen by appending
    the subcommand name to the above command.`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Runs - create
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunsCreateUse string = `create [command options] <container-id> 
	
Where "<container-id>" is your name for the instance of the container that you
are starting. The name you provide for the container instance must be unique on
your host.`
	RunsCreateShort string = `create a container SIF based`
	RunsCreateLong  string = `
	The create command creates an instance of a container from a SIF bundle. The bundle
	is a SIF with a specification file named "config.json" as a SIF data objet and a root
	filesystem.

	The specification file includes an args parameter. The args parameter is used
	to specify command(s) that get run when the container is started.`
)
