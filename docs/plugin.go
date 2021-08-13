// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docs

// Plugin command usage.
const (
	PluginUse   string = `plugin [plugin options...]`
	PluginShort string = `Manage Singularity plugins`
	PluginLong  string = `
  The 'plugin' command allows you to manage Singularity plugins which
  provide add-on functionality to the default Singularity installation.`
	PluginExample string = `
  All group commands have their own help output:

  $ singularity help plugin compile
  $ singularity plugin list --help`
)

// Plugin compile command usage.
const (
	PluginCompileUse   string = `compile [compile options...] <host_path>`
	PluginCompileShort string = `Compile a Singularity plugin`
	PluginCompileLong  string = `
  The 'plugin compile' command allows a developer to compile a Singularity 
  plugin in the expected environment. The provided host directory is the 
  location of the plugin's source code. A compiled plugin is packed into a SIF file.`
	PluginCompileExample string = `
  $ singularity plugin compile $HOME/singularity/test-plugin`
)

// Plugin install command usage.
const (
	PluginInstallUse   string = `install <plugin_path>`
	PluginInstallShort string = `Install a compiled Singularity plugin`
	PluginInstallLong  string = `
  The 'plugin install' command installs the compiled plugin found at plugin_path
  into the appropriate directory on the host.`
	PluginInstallExample string = `
  $ singularity plugin install $HOME/singularity/test-plugin/test-plugin.sif`
)

// Plugin uninstall command usage.
const (
	PluginUninstallUse   string = `uninstall <name>`
	PluginUninstallShort string = `Uninstall removes the named plugin from the system`
	PluginUninstallLong  string = `
  The 'plugin uninstall' command removes the named plugin from the system`
	PluginUninstallExample string = `
  $ singularity plugin uninstall example.org/plugin`
)

// Plugin list command usage.
const (
	PluginListUse   string = `list [list options...]`
	PluginListShort string = `List installed Singularity plugins`
	PluginListLong  string = `
  The 'plugin list' command lists the Singularity plugins installed on the host.`
	PluginListExample string = `
  $ singularity plugin list
  ENABLED  NAME
      yes  example.org/plugin`
)

// Plugin enable command usage.
const (
	PluginEnableUse   string = `enable <name>`
	PluginEnableShort string = `Enable an installed Singularity plugin`
	PluginEnableLong  string = `
  The 'plugin enable' command allows a user to enable a plugin that is already
  installed in the system and which has been previously disabled.`
	PluginEnableExample string = `
  $ singularity plugin enable example.org/plugin`
)

// Plugin disable command usage.
const (
	PluginDisableUse   string = `disable <name>`
	PluginDisableShort string = `disable an installed Singularity plugin`
	PluginDisableLong  string = `
  The 'plugin disable' command allows a user to disable a plugin that is already
  installed in the system and which has been previously enabled.`
	PluginDisableExample string = `
  $ singularity plugin disable example.org/plugin`
)

// Plugin inspect command usage.
const (
	PluginInspectUse   string = `inspect (<name>|<image>)`
	PluginInspectShort string = `Inspect a singularity plugin (either an installed one or an image)`
	PluginInspectLong  string = `
  The 'plugin inspect' command allows a user to inspect a plugin that is already
  installed in the system or an image containing a plugin that is yet to be installed.`
	PluginInspectExample string = `
  $ singularity plugin inspect sylabs.io/test-plugin
  Name: sylabs.io/test-plugin
  Description: A test Singularity plugin.
  Author: Sylabs
  Version: 0.1.0`
)

// Plugin create command usage.
const (
	PluginCreateUse   string = `create <host_path> <name>`
	PluginCreateShort string = `Create a plugin skeleton directory`
	PluginCreateLong  string = `
  The 'plugin create' command allows a user to creates a plugin skeleton directory
  structure to start development of a new plugin.`
	PluginCreateExample string = `
  $ singularity plugin create ~/myplugin github.com/username/myplugin
  $ ls -1 ~/myplugin
  go.mod
  main.go
  singularity_source
  `
)
