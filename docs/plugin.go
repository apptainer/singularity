// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docs

// Global content for help and man pages
const (

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// plugin command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginUse   string = `plugin [plugin options...]`
	PluginShort string = `Manage singularity plugins`
	PluginLong  string = `
  The 'plugin' command allows you to manage `
	PluginExample string = `
  All group commands have their own help output:

  $ singularity help plugin compile
  $ singularity plugin list --help`

	// // ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// // plugin list command
	// // ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// PluginListUse   string = `list [list options...]`
	// PluginListShort string = `List all install plugins`
	// PluginListLong  string = `
	// The 'plugin list' command lists all installed plugins `
	// PluginListExample string = `
	// $ singularity plugin list`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// plugin compile command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginCompileUse   string = `compile [compile options...] <host_path>`
	PluginCompileShort string = `Compile a singularity plugin`
	PluginCompileLong  string = `
	The 'plugin compile' command allows a developer to compile a plugin in the
	expected environment. The host directory specified is the location of the plugins
	source code folder which will be bind mounted into the compile container. Compilation
	of a container must happen in a container due to constraints in the Go plugin package.`
	PluginCompileExample string = `
	$ singularity plugin compile $HOST_PATH`

	// // ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// // plugin install command
	// // ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginInstallUse   string = `install [install options...] <plugin_path>`
	PluginInstallShort string = `Install a singularity plugin`
	PluginInstallLong  string = `
	The 'plugin install' command installs the plugin found at plugin_path into the
	appropriate directory on the host.`
	PluginInstallExample string = `
	$ singularity plugin install $PLUGIN_PATH`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// plugin uninstall command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginUninstallUse   string = `uninstall <name>`
	PluginUninstallShort string = `Uninstall removes the named plugin from the system`
	PluginUninstallLong  string = `
	The 'plugin uninstall' command removes the named plugin from the system`
	PluginUninstallExample string = `
	$ singularity plugin uninstall example.org/plugin`

	// // ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// // plugin list command
	// // ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginListUse   string = `list [list options...]`
	PluginListShort string = `List installed singularity plugins`
	PluginListLong  string = `
	The 'plugin list' command lists the singularity plugins installed on the host.`
	PluginListExample string = `
	$ singularity plugin list`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// plugin enable command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginEnableUse   string = `enable <name>`
	PluginEnableShort string = `Enable an installed singularity plugin`
	PluginEnableLong  string = `
	The 'plugin enable' command allows a user to enable a plugin that is already
	installed in the system and which has been previously disabled.`
	PluginEnableExample string = `
	$ singularity plugin enable sylabs.io/test-plugin`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// plugin disable command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginDisableUse   string = `disable <name>`
	PluginDisableShort string = `disable an installed singularity plugin`
	PluginDisableLong  string = `
	The 'plugin disable' command allows a user to disable a plugin that is already
	installed in the system and which has been previously enabled.`
	PluginDisableExample string = `
	$ singularity plugin disable sylabs.io/test-plugin`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// plugin inspect command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	PluginInspectUse   string = `inspect (<name>|<image>)`
	PluginInspectShort string = `Inspect a singularity plugin (either an installed one or an image)`
	PluginInspectLong  string = `
	The 'plugin inspect' command allows a user to inspect a plugin that is already
	installed in the system or an image containing a plugin that is yet to be installed.`
	PluginInspectExample string = `
	$ singularity plugin inspect sylabs.io/test-plugin
	$ singularity plugin inspect test-plugin.sif`
)
