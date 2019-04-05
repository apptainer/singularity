// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
)

// InstanceStartCmd fake command to satisfy actions command
// group flag registration
var InstanceStartCmd *cobra.Command

// initPlatformDefaults customizes the default values for the flags
// to make them appropriate for the build target
func initPlatformDefaults() {
	// TODO: should darwin default to running SyOS? ("syos" flag)
	// hide this flag from the help so that users don't try to turn it off
	actionVMFlag.DefaultValue = true
	actionVMFlag.Hidden = true
}
