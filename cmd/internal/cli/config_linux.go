// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/sylabs/singularity/docs"
)

// configCmd is the config command
var configCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Use:                   docs.ConfigUse,
	Short:                 docs.ConfigShort,
	Long:                  docs.ConfigLong,
	Example:               docs.ConfigExample,
	SilenceErrors:         true,
}

func init() {
	cmdManager.RegisterCmd(configCmd)

	cmdManager.RegisterSubCmd(configCmd, configFakerootCmd)
}
