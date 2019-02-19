// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"path/filepath"

	"github.com/sylabs/sif/pkg/sif"
)

// InstallPlugin takes a plugin located at path and installs it into
// the singularity folder in libexecdir.
//
// Installing a plugin will also automatically enable it.
func InstallPlugin(pluginPath, libexecdir string) error {
	fimg, err := sif.LoadContainer(pluginPath, true)
	if err != nil {
		return fmt.Errorf("while opening sif file: %s", err)
	}

	if !isPluginFile(&fimg) {
		return fmt.Errorf("sif file is not a plugin")
	}

	if err := copyFile(pluginPath, filepath.Join(libexecdir, ".")); err != nil {
		return fmt.Errorf("while copying plugin file to install location: %s", err)
	}

	return nil
}

// isPluginFile checks if the sif.FileImage contains the sections which
// make up a valid plugin. A plugin sif file should have the following
// format:
//
// DESCR[0]: Sifplugin
//   - Datatype: sif.DataPartition
//   - Fstype:   sif.FsRaw
//   - Parttype: sif.PartData
// DESCR[1]: Sifmanifest
//   - Datatype: sif.DataGenericJSON
func isPluginFile(fimg *sif.FileImage) bool {
	return false
}

// copyFile copies a file from src -> dst
func copyFile(src, dst string) error {
	// copycmd := exec.Command("cp", src, dst)
	return nil
}
