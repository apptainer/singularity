// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import (
	"github.com/spf13/pflag"
	singularity "github.com/sylabs/singularity/internal/pkg/runtime/engines/singularity/config"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
)

type flagHook struct {
	flag     *pflag.Flag
	callback pluginapi.FlagCallbackFn
}

type flagHooks struct {
	FlagSet *pflag.FlagSet
	Hooks   []flagHook
}

var registeredFlagHooks *flagHooks

func registerStringFlagHook(f pluginapi.StringFlagHook) error {
	sylog.Debugf("Registering string flag %s", f.Flag.Name)
	registeredFlagHooks.FlagSet.StringP(f.Flag.Name, f.Flag.Shorthand, f.Flag.DefValue, f.Flag.Usage)

	hook := flagHook{
		flag:     registeredFlagHooks.FlagSet.Lookup(f.Flag.Name),
		callback: f.Callback,
	}

	registeredFlagHooks.Hooks = append(registeredFlagHooks.Hooks, hook)
	sylog.Debugf("Registered new string flag hook %v\n", hook)
	return nil
}

func registerBoolFlagHook(f pluginapi.BoolFlagHook) error {
	sylog.Debugf("Registering bool flag %s", f.Flag.Name)
	registeredFlagHooks.FlagSet.BoolP(f.Flag.Name, f.Flag.Shorthand, false, f.Flag.Usage)

	hook := flagHook{
		flag:     registeredFlagHooks.FlagSet.Lookup(f.Flag.Name),
		callback: f.Callback,
	}
	registeredFlagHooks.Hooks = append(registeredFlagHooks.Hooks, hook)

	sylog.Debugf("Registered new bool flag hook %v\n", hook)
	return nil
}

func init() {
	registeredFlagHooks = &flagHooks{
		FlagSet: pflag.NewFlagSet("FlagHooksSet", pflag.ExitOnError),
		Hooks:   []flagHook{},
	}

	pluginapi.RegisterStringFlag = registerStringFlagHook
	pluginapi.RegisterBoolFlag = registerBoolFlagHook
}

// AddFlagHooks will add the plugin defined flags to the input FlagSet
func AddFlagHooks(flagSet *pflag.FlagSet) {
	assertInitialized()

	flagSet.AddFlagSet(registeredFlagHooks.FlagSet)
}

// FlagHookCallbacks will run the callback functions for all registered
// flag hooks
func FlagHookCallbacks(c *singularity.EngineConfig) {
	assertInitialized()

	for _, hook := range registeredFlagHooks.Hooks {
		hook.callback(hook.flag, c)
	}
}
