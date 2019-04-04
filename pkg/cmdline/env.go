// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdline

import (
	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// EnvAppend combines command line and environment var into a single argument
func EnvAppend(flag *pflag.Flag, envvar string) {
	if err := flag.Value.Set(envvar); err != nil {
		sylog.Warningf("Unable to set %s to environment variable value %s", flag.Name, envvar)
	} else {
		flag.Changed = true
		sylog.Debugf("Update flag Value to: %s", flag.Value)
	}
}

// EnvBool sets a bool flag if the CLI option is unset and env var is set
func EnvBool(flag *pflag.Flag, envvar string) {
	if flag.Changed || envvar == "" {
		return
	}

	if err := flag.Value.Set(envvar); err != nil {
		sylog.Debugf("Unable to set flag %s to value %s: %s", flag.Name, envvar, err)
		if err := flag.Value.Set("true"); err != nil {
			sylog.Warningf("Unable to set flag %s to value %s: %s", flag.Name, envvar, err)
			return
		}
	}

	flag.Changed = true
	sylog.Debugf("Set %s Value to: %s", flag.Name, flag.Value)
}

// EnvStringNSlice writes to a string or slice flag if CLI option/argument
// string is unset and env var is set
func EnvStringNSlice(flag *pflag.Flag, envvar string) {
	if flag.Changed {
		return
	}

	if err := flag.Value.Set(envvar); err != nil {
		sylog.Warningf("Unable to set flag %s to value %s: %s", flag.Name, envvar, err)
		return
	}

	flag.Changed = true
	sylog.Debugf("Set %s Value to: %s", flag.Name, flag.Value)
}

// EnvHandler ...
type EnvHandler func(*pflag.Flag, string)
