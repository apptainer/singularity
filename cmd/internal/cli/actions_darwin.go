// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
)

// TODO: Let's stick this in another file so that that CLI is just CLI
func execStarter(cobraCmd *cobra.Command, image string, args []string, name string) {
	vmStarter(cobraCmd, image, args)
}
