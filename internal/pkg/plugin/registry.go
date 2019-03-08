// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package plugin

import "github.com/spf13/pflag"

type registry struct {
	*flagRegistry
}

var reg registry

func init() {
	reg = registry{
		flagRegistry: &flagRegistry{
			FlagSet: pflag.NewFlagSet("flagRegistrySet", pflag.ExitOnError),
			Hooks:   []flagHook{},
		},
	}
}
