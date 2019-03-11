// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/plugin"
)

// UninstallPlugin removes the named plugin from the system
func UninstallPlugin(name, sysconfdir, libexecdir string) error {
	err := plugin.Uninstall(name, sysconfdir, libexecdir)
	if err != nil {
		return err
	}

	fmt.Printf("Uninstalled plugin %q.\n", name)

	return nil
}
