// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandAdder allows a plugin to add new command(s) to the singularity binary
type CommandAdder interface {
	CommandAdd() []*cobra.Command
}

// RootFlagAdder is the interface for a plugin which wishes to add a flag to the
// root singularity command
type RootFlagAdder interface {
	RootFlagAdd() []*pflag.Flag
}

// ActionFlagAdder is the interface for a plugin which wishes to add a flag to the
// action command group (run, exec, shell, instance)
type ActionFlagAdder interface {
	ActionFlagAdd() []*pflag.Flag
}
