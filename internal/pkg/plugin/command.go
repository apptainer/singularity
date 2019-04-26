// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"github.com/spf13/cobra"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

type commandRegistry struct {
	Commands []*cobra.Command
}

// RegisterCommand registers a CommandHook for adding a new command to the singularity
// binary
func (r *commandRegistry) RegisterCommand(hook pluginapi.CommandHook) error {
	r.Commands = append(r.Commands, hook.Command)

	return nil
}

// AllCommands returns a slice of commands registered by plugins which need to be
// added to the main SingularityCmd. By simply returning the slice of objects, it's
// trivial to handle this from the CLI.
func AllCommands() []*cobra.Command {
	return reg.commandRegistry.Commands
}
