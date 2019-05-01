// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"github.com/spf13/cobra"
)

// CommandHook allows a plugin to add new command(s) to singularity by
// defining custom cobra.Command objects.
type CommandHook struct {
	Command *cobra.Command
}
