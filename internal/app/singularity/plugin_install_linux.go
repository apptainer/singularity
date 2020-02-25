// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"github.com/sylabs/singularity/internal/pkg/plugin"
)

// InstallPlugin takes a plugin located at path and installs it into
// the singularity plugin installation directory.
//
// Installing a plugin will also automatically enable it.
func InstallPlugin(pluginPath string) error {
	return plugin.Install(pluginPath)
}
