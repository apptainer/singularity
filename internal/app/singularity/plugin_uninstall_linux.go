// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"errors"
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/plugin"
)

var ErrPluginNotFound = errors.New("plugin not found")

// UninstallPlugin removes the named plugin from the system.
func UninstallPlugin(name, libexecdir string) error {
	err := plugin.Uninstall(name, libexecdir)
	if errors.Is(err, os.ErrNotExist) {
		return ErrPluginNotFound
	}
	if err != nil {
		return fmt.Errorf("could not uninstall plugin: %w", err)
	}
	return nil
}
