// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmdline

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// EnvAppend combines command line and environment var into a single argument
func EnvAppend(flag *pflag.Flag, envvar string) error {
	if err := flag.Value.Set(envvar); err != nil {
		return fmt.Errorf("unable to set %s to environment variable value %s", flag.Name, envvar)
	}

	flag.Changed = true
	sylog.Debugf("Updated flag '%s' value to: %s", flag.Name, flag.Value)
	return nil
}

// EnvBool sets a bool flag if the CLI option is unset and env var is set
func EnvBool(flag *pflag.Flag, envvar string) error {
	if flag.Changed || envvar == "" {
		return nil
	}

	if err := flag.Value.Set(envvar); err != nil {
		if err := flag.Value.Set("true"); err != nil {
			return fmt.Errorf("unable to set flag %s to value %s: %s", flag.Name, envvar, err)
		}
	}

	flag.Changed = true
	sylog.Debugf("Updated flag '%s' value to: %s", flag.Name, flag.Value)
	return nil
}

// EnvStringNSlice writes to a string or slice flag if CLI option/argument
// string is unset and env var is set
func EnvStringNSlice(flag *pflag.Flag, envvar string) error {
	if flag.Changed {
		return nil
	}

	if err := flag.Value.Set(envvar); err != nil {
		return fmt.Errorf("unable to set flag %s to value %s: %s", flag.Name, envvar, err)
	}

	flag.Changed = true
	sylog.Debugf("Updated flag '%s' value to: %s", flag.Name, flag.Value)
	return nil
}

// EnvHandler defines an environment handler type to set flag's values
// from environment variables
type EnvHandler func(*pflag.Flag, string) error
