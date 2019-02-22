// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

// InstallPlugin takes a plugin located at path and installs it into
// the singularity folder in libexecdir.
//
// Installing a plugin will also automatically enable it.
func InstallPlugin(pluginPath, libexecdir string) error {
	// fimg, err := sif.LoadContainer(pluginPath, true)
	// if err != nil {
	// 	return fmt.Errorf("while opening sif file: %s", err)
	// }

	// if !isPluginFile(&fimg) {
	// 	return fmt.Errorf("sif file is not a plugin")
	// }

	// if err := copyFile(pluginPath, filepath.Join(libexecdir, ".")); err != nil {
	// 	return fmt.Errorf("while copying plugin file to install location: %s", err)
	// }

	return nil
}
