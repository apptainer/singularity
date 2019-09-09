// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package env

import (
	"fmt"
	"os"
	"strings"
)

// SetFromList sets environment variables from environ argument list.
func SetFromList(environ []string) error {
	for _, env := range environ {
		splitted := strings.SplitN(env, "=", 2)
		if len(splitted) != 2 {
			return fmt.Errorf("can't process environment variable %s", env)
		}
		if err := os.Setenv(splitted[0], splitted[1]); err != nil {
			return err
		}
	}
	return nil
}
