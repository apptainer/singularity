// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docs

// Global content for Runsy CLI help and man pages
const (
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// main Runsy command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunsyUse   string = `runsy [global options...]`
	RunsyShort string = `
Linux container platform based on singularity runtime OCi runtime compliant`
	RunsyLong string = `
Runs containers provide an application virtualization layer enabling
mobility of compute via both application and environment portability.`
	RunsyExample string = `
$ runsy help
    Will print a generalized usage summary and available commands.

$ runsy help <command>
    Additional help for any runs subcommand can be seen by appending
    the subcommand name to the above command.`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Runsy - Spec
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunsySpecUse   string = `spec [command] [options...]`
	RunsySpecShort string = `Cmd tool set for working with OCI (Open Container Initiative) runtime spec`
	RunsySpecLong  string = `
	The specification file includes an args parameter. The args parameter is used
	to specify command(s) that get run when the container is started.`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Runsy - Spec gen
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunsySpecGenUse   string = `gen [options...] </path/to/SIF>`
	RunsySpecGenShort string = `generates a config.json OCI runtime spec`
	RunsySpecGenLong  string = `
	The specification file includes an args parameter. The args parameter is used
	to specify command(s) that get run when the container is started.`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Runsy - Spec add
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunsySpecAddUse   string = `add </path/To/config.json> </path/to/SIF>`
	RunsySpecAddShort string = `adds a target config.json into a SIF`
	RunsySpecAddLong  string = `
	Insert a OCI runtime spec config.json file into a SIF data object as a JSON.generic`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// Runsy - Spec Inspect
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RunsySpecInspectUse   string = `inspect </path/to/SIF>`
	RunsySpecInspectShort string = `seek into a SIF bundle for OCI runtime specs`
	RunsySpecInspectLong  string = `
	seek into a SIF bundle for OCI runtime specs, if found, prints the OCI runtime spec into stoud`
)
