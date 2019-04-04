// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

// initPlatformDefaults customizes the default values for the flags in
// actionFlags to make them appropriate for the build target
func initPlatformDefaults() {
	// TODO: should darwin default to running SyOS? ("syos" flag)
	// hide this flag from the help so that users don't try to turn it off
	actionVMFlag.DefaultValue = true
	actionVMFlag.Hidden = true
}
