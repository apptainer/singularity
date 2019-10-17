// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"log"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/cgroups"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	singularity "github.com/sylabs/singularity/pkg/runtime/engine/singularity/config"
)

// Plugin is the only variable which a plugin MUST export.
// This symbol is accessed by the plugin framework to initialize the plugin
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "sylabs.io/config-plugin",
		Author:      "Sylabs Team",
		Version:     "0.1.0",
		Description: "This is a short test config plugin for Singularity",
	},

	Initializer: pluginImplementation{},
}

type pluginImplementation struct{}

func (p pluginImplementation) Initialize(r pluginapi.Registry) error {
	r.AddEngineConfigMutator(pluginapi.EngineConfigMutator{
		Mutate: func(common *config.Common) {
			c, ok := common.EngineConfig.(*singularity.EngineConfig)
			if !ok {
				log.Printf("Unexpected engine config")
				return
			}
			cfg := cgroups.Config{
				Devices: nil,
				Memory: &cgroups.LinuxMemory{
					Limit: &[]int64{1024 * 1}[0],
				},
			}

			path, err := filepath.Abs("test-cgroups")
			if err != nil {
				sylog.Errorf("Could not get cgroups path: %s", path)
			}
			err = cgroups.PutConfig(cfg, path)
			if err != nil {
				log.Printf("Put c error: %v", err)
			}
			if path := c.GetCgroupsPath(); path != "" {
				sylog.Infof("Old cgroups path: %s", path)
			}
			sylog.Infof("Setting cgroups path to %s", path)
			c.SetCgroupsPath(path)
		},
	})

	return nil
}
