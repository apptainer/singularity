// Copyright (c) 2020, Ctrl-Cmd Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docs

// Global content for help and man pages
const (
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// login command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	LoginUse   string = `login [login options...] <hostname>`
	LoginShort string = `Login to Docker/OCI registries`
	LoginLong  string = `
  The 'login' command allow you to manage Docker/OCI registry login credentials,
  the configuration is stored in $HOME/.docker/config.json.`
	LoginExample string = `$ singularity login -u john -p secret docker.io`

	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	// logout command
	// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
	LogoutUse     string = `logout <hostname>`
	LogoutShort   string = `Logout from Docker/OCI registries`
	LogoutLong    string = `The 'logout' command allow you to remove Docker/OCI registry login credentials`
	LogoutExample string = `$ singularity logout docker.io`
)
