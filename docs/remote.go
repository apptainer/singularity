// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docs

// Global content for help and man pages
const (
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteUse   string = `remote <subcommand>`
	RemoteShort string = `Manage singularity remote endpoints`
	RemoteLong  string = `
  The 'remote' commands allow you to manage Singularity remote endpoints through 
  its subcommands. These allow you to add, log in, and use endpoints. The remote
  configuration is stored in $HOME/.singularity/remotes.yaml by default.`
	RemoteExample string = `
  All group commands have their own help output:

	$ singularity help remote list
	$ singularity remote list`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote add command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteAddUse   string = `add <remote_name> <remote_URI>`
	RemoteAddShort string = `Create a new singularity remote endpoint`
	RemoteAddLong  string = `
  The 'remote add' command allows you to create a new remote endpoint to be used 
  for Singularity remote services.`
	RemoteAddExample string = `
  $ singularity remote add SylabsCloud cloud.sylabs.io`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote remove command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteRemoveUse   string = `remove <remote_name>`
	RemoteRemoveShort string = `Remove an existing singularity remote endpoint`
	RemoteRemoveLong  string = `
  The 'remote remove' command allows you to remove an existing remote endpoint 
  from the list of potential endpoints to use.`
	RemoteRemoveExample string = `
  $ singularity remote remove SylabsCloud`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote use command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteUseUse   string = `use <remote_name>`
	RemoteUseShort string = `Set a singularity remote endpoint to be actively used`
	RemoteUseLong  string = `
  The 'remote use' command sets the remote to be used by default by any command
  that interacts with Singularity services.`
	RemoteUseExample string = `
  $ singularity remote use SylabsCloud`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote list command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteListUse   string = `list`
	RemoteListShort string = `List all singularity remote endpoints that are configured`
	RemoteListLong  string = `
  The 'remote list' command lists all remote endpoints configured for use. If a remote
  is in use, its name will be encompassed by brackets.`
	RemoteListExample string = `
  $ singularity remote list`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote login command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteLoginUse   string = `login <remote_name>`
	RemoteLoginShort string = `Log into a singularity remote endpoint using an authentication token`
	RemoteLoginLong  string = `
  The 'remote login' command allows you to set an authentication token for a
  specific endpoint. This command will produce a link directing you to the token
  service you can use to generate a valid token. If no endpoint is specified,
  it will try the default remote.`
	RemoteLoginExample string = `
  $ singularity remote login SylabsCloud`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote status command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteStatusUse   string = `status <remote_name>`
	RemoteStatusShort string = `Check the status of the singularity services at an endpoint`
	RemoteStatusLong  string = `
  The 'remote status' command checks the status of the specified remote endpoint
  and reports the availibility of services and their versions. If no endpoint is
  specified, it will check the status of the default remote.`
	RemoteStatusExample string = `
  $ singularity remote status SylabsCloud`
)
