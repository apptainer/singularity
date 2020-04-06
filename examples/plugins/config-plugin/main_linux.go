// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"log"
	"path/filepath"

	"github.com/sylabs/singularity/internal/pkg/cgroups"
	"github.com/sylabs/singularity/pkg/sylog"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
	clicallback "github.com/sylabs/singularity/pkg/plugin/callback/cli"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	singularity "github.com/sylabs/singularity/pkg/runtime/engine/singularity/config"
)

// Plugin is the only variable which a plugin MUST export.
// This symbol is accessed by the plugin framework to initialize the plugin
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "github.com/sylabs/singularity/config-example-plugin",
		Author:      "Sylabs Team",
		Version:     "0.1.0",
		Description: "This is a short example config plugin for Singularity",
	},
	Callbacks: []pluginapi.Callback{
		(clicallback.SingularityEngineConfig)(callbackCgroups),
	},
}

func callbackCgroups(common *config.Common) {
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
}
