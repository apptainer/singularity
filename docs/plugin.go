// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docs

// Global content for help and man pages
const (

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// plugin command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginUse   string = `plugin [plugin options...] <subcommand>`
	PluginShort string = `Manage singularity plugins`
	PluginLong  string = `
  The 'plugin' command  allows you to manage `
	PluginExample string = `
  All group commands have their own help output:

  $ singularity help plugin compile
  $ singularity plugin list --help`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// plugin compile command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginCompileUse   string = `compile [compile options...] <host_path>`
	PluginCompileShort string = `Compile a singularity plugin`
	PluginCompileLong  string = `
  The 'plugin compile' command  allows a developer to compile a plugin in the
  expected environment. The host directory specified is the location of the plugins
  source code folder which will be bind mounted into the compile container. Compilation
  of a container must happen in a container due to constraints in the Go plugin package.`
	PluginCompileExample string = `
  $ singularity plugin compile $PLUGIN_PATH`
)
