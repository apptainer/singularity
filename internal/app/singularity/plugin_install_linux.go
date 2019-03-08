// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"sort"

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

	_, err = plugin.InstallFromSIF(&fimg, libexecdir)
	if err != nil {
		return err
	}

	return nil
}

func ListPlugins(libexecdir string) error {
	plugins, err := plugin.GetList(libexecdir)
	if err != nil {
		return err
	}

	if len(plugins) == 0 {
		fmt.Println("There are no plugins installed.")
		return nil
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	fmt.Printf("ENABLED  NAME\n")

	for _, p := range plugins {
		enabled := "no"
		if p.Enabled {
			enabled = "yes"
		}
		fmt.Printf("%7s  %s\n", enabled, p.Name)
	}

	return nil
}
