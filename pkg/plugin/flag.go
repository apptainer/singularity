// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"github.com/spf13/pflag"
	singularity "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/config"
)

// RegisterFlag functions are function variables which will be assigned
// before any plugins Init() function is called. These functions can
// only be called during init, and will register a functions specific flag
// hooks with the singularity runtime
var (
	RegisterStringFlag func(StringFlagHook) error
	RegisterBoolFlag   func(BoolFlagHook) error
)

// FlagCallbackFn is the callback function type for flag hooks. It takes two
// arguments:
//
// *pflag.Flag:
//   A pointer to a pflag.Flag object. This object is guaranteed to have been
//   added to the action FlagSet, and will contain the value the flag was
//   set to if it was set.
//
// *singularity.EngineConfig:
//   A pointer to the EngineConfig object which allows the plugin to make
//   modifications to the runtime parameters of the container
//
// This function is guaranteed to be called after the flag has been parsed by
// pflag, and before the starter binary is executed.
type FlagCallbackFn func(*pflag.Flag, *singularity.EngineConfig)

// StringFlagHook provides plugins the ability to add a string flag to the action
// command group. A flag of this type takes one argument as a string. The string
// value which the flag is set to can be then converted to other types, such as int.
type StringFlagHook struct {
	Flag     pflag.Flag
	Callback FlagCallbackFn
}

// BoolFlagHook provides plugins the ability to add a bool flag to the action
// command group. A flag of this type does not take arguments.
type BoolFlagHook struct {
	Flag     pflag.Flag
	Callback FlagCallbackFn
}
