// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

// HookRegistration exposes functions to a plugin during Init() which allow
// the plugin to register its plugin hooks.
type HookRegistration interface {
	RegisterStringFlag(StringFlagHook) error
	RegisterBoolFlag(BoolFlagHook) error
	RegisterCommand(CommandHook) error
}
