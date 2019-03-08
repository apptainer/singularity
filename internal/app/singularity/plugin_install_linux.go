// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/plugin"
)

// InstallPlugin takes a plugin located at path and installs it into
// the singularity folder in libexecdir.
//
// Installing a plugin will also automatically enable it.
func InstallPlugin(pluginPath, libexecdir string) error {
	fimg, err := sif.LoadContainer(pluginPath, true)
	if err != nil {
		return err
	}

	defer fimg.UnloadContainer()

	m, err := plugin.InstallFromSIF(&fimg, libexecdir)
	if err != nil {
		return err
	}

	return nil
}
