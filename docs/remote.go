// Copyright (c) 2020, Control Command Inc. All rights reserved.
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
	RemoteUse   string = `remote [remote options...]`
	RemoteShort string = `Manage singularity remote endpoints, keyservers and OCI/Docker registry credentials`
	RemoteLong  string = `
  The 'remote' commands allow you to manage Singularity remote endpoints, keyservers
  and OCI/Docker registry credentials through its subcommands. The remote configuration
  is stored in $HOME/.singularity/remotes.yaml by default.`
	RemoteExample string = `
  All group commands have their own help output:

	$ singularity help remote list
	$ singularity remote list`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote add command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteAddUse   string = `add [add options...] <remote_name> <remote_URI>`
	RemoteAddShort string = `Create a new singularity remote endpoint`
	RemoteAddLong  string = `
	The 'remote add' command allows you to create a new remote endpoint to be
	be used for singularity remote services. Authentication with a newly created
	endpoint will occur automatically.`
	RemoteAddExample string = `
  $ singularity remote add SylabsCloud cloud.sylabs.io`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote remove command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteRemoveUse   string = `remove [remove options...] <remote_name>`
	RemoteRemoveShort string = `Remove an existing singularity remote endpoint`
	RemoteRemoveLong  string = `
  The 'remote remove' command allows you to remove an existing remote endpoint 
  from the list of potential endpoints to use.`
	RemoteRemoveExample string = `
  $ singularity remote remove SylabsCloud`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote use command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteUseUse   string = `use [use options...] <remote_name>`
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
	RemoteListShort string = `List all singularity remote endpoints and services that are configured`
	RemoteListLong  string = `
  The 'remote list' command lists all remote endpoints configured for use. If a remote
  is in use, its name will be encompassed by brackets.`
	RemoteListExample string = `
  $ singularity remote list`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote login command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteLoginUse   string = `login [login options...] <remote_name|registry_uri>`
	RemoteLoginShort string = `Log into a singularity remote endpoint, an OCI/Docker registry or a keyserver using credentials`
	RemoteLoginLong  string = `
  The 'remote login' command allows you to set credentials for a specific endpoint,
  an OCI/Docker registry or a keyserver. This command can produce a link directing you to
  the token service you can use to generate a valid token. If no endpoint or registry is
  specified, it will try the default remote endpoint (SylabsCloud).`
	RemoteLoginExample string = `
  To log in to an endpoint:
  $ singularity remote login SylabsCloud

  To login in to a docker/OCI registry:
  $ singularity remote login --username foo --password bar docker://docker.io`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote logout command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteLogoutUse   string = `logout <remote_name|registry_uri>`
	RemoteLogoutShort string = `Log out from a singularity remote endpoint, an OCI/Docker registry or a keyserver`
	RemoteLogoutLong  string = `
  The 'remote logout' command allows you to log out from a singularity specific endpoint,
  an OCI/Docker registry or a keyserver. If no endpoint or service is specified, it will
  try the default remote endpoint (SylabsCloud).`
	RemoteLogoutExample string = `
  To log out from an endpoint
  $ singularity remote logout SylabsCloud

  To log out from a docker/OCI registry
  $ singularity remote logout docker://docker.io`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote status command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteStatusUse   string = `status [remote_name]`
	RemoteStatusShort string = `Check the status of the singularity services at an endpoint, and your authentication token`
	RemoteStatusLong  string = `
  The 'remote status' command checks the status of the specified remote endpoint
  and reports the availability of services and their versions. If no endpoint is
  specified, it will check the status of the default remote (SylabsCloud). If you
  have logged in with an authentication token the validity of that token will be
  checked.`
	RemoteStatusExample string = `
  $ singularity remote status SylabsCloud`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote add-keyserver command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteAddKeyserverUse   string = `add-keyserver [options] [remoteName] <keyserver_url>`
	RemoteAddKeyserverShort string = `Add a keyserver (root user only)`
	RemoteAddKeyserverLong  string = `
  The 'remote add-keyserver' command allows to define additional keyserver. The --order
  option can define the order of the keyserver for all related key operations, therefore
  when specifying '--order 1' the keyserver is becoming the primary keyserver. If no endpoint
  is specified, it will use the default remote endpoint (SylabsCloud).`
	RemoteAddKeyserverExample string = `
  $ singularity remote add-keyserver https://keys.example.com

  To add a keyserver to be used as the primary keyserver for the current endpoint
  $ singularity remote add-keyserver --order 1 https://keys.example.com`
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// remote remove-keyserver command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	RemoteRemoveKeyserverUse   string = `remove-keyserver [remoteName] <keyserver_url>`
	RemoteRemoveKeyserverShort string = `Remove a keyserver (root user only)`
	RemoteRemoveKeyserverLong  string = `
  The 'remote remove-keyserver' command allows to remove a defined keyserver from a specific
  endpoint. If no endpoint is specified, it will use the default remote endpoint (SylabsCloud).`
	RemoteRemoveKeyserverExample string = `
  $ singularity remote remove-keyserver https://keys.example.com`
)
