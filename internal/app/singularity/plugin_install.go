// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/docs"
)

// PluginInstallCmd takes a compiled plugin.sif file and installs it
// in the appropriate location
//
// singularity plugin install <path>
var PluginInstallCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Installing plugin")

		return nil
	},
	DisableFlagsInUseLine: true,
	Args:                  cobra.ExactArgs(1),

	Use:     docs.PluginInstallUse,
	Short:   docs.PluginInstallShort,
	Long:    docs.PluginInstallLong,
	Example: docs.PluginInstallExample,
}

// InstallPlugin takes a plugin located at path and installs it into
// the singularity folder in libexecdir.
//
// Installing a plugin will also automatically enable it.
func InstallPlugin(pluginPath, libexecdir string) error {
	fimg, err := sif.LoadContainer(path, true)
	if err != nil {
		return fmt.Errorf("while opening sif file: %s", err)
	}

	if !isPluginFile(&fimg) {
		return fmt.Errorf("sif file is not a plugin")
	}

	if err := copyFile(pluginPath, filepath.Join(libexecdir, ".")); err != nil {
		return fmt.Errorf("while copying plugin file to install location: %s", err)
	}
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

}

// copyFile copies a file from src -> dst
func copyFile(src, dst string) error {
	// copycmd := exec.Command("cp", src, dst)
}
