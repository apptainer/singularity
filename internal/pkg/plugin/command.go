// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"github.com/spf13/cobra"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

// AddCommands calls all CommandAdder plugins and adds the commands to the
// roootCmd
func AddCommands(rootCmd *cobra.Command) error {
	for _, pl := range loadedPlugins {
		if _pl, ok := (pl.Initializer).(pluginapi.CommandAdder); ok {
			for _, cmd := range _pl.CommandAdd() {
				rootCmd.AddCommand(cmd)
			}
		}
	}

	return nil
}
