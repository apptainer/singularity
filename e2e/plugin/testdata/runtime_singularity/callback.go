// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os"
	"syscall"
	"time"

	pluginapi "github.com/sylabs/singularity/pkg/plugin"
	singularitycallback "github.com/sylabs/singularity/pkg/plugin/callback/runtime/engine/singularity"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engine/singularity/config"
)

// Plugin is the only variable which a plugin MUST export.
// This symbol is accessed by the plugin framework to initialize the plugin.
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "github.com/sylabs/singularity/e2e-runtime-plugin",
		Author:      "Sylabs Team",
		Version:     "0.1.0",
		Description: "E2E runtime plugin",
	},
	Callbacks: []pluginapi.Callback{
		(singularitycallback.MonitorContainer)(callbackMonitor),
		(singularitycallback.PostStartProcess)(callbackPostStart),
	},
}

func callbackMonitor(config *config.Common, pid int, signals chan os.Signal) (syscall.WaitStatus, error) {
	var status syscall.WaitStatus

	cfg := config.EngineConfig.(*singularityConfig.EngineConfig)
	if !cfg.GetContain() {
		os.Exit(42)
	} else {
		// sleep until post start process exit
		time.Sleep(10 * time.Second)
	}

	return status, nil
}

func callbackPostStart(config *config.Common, pit int) error {
	cfg := config.EngineConfig.(*singularityConfig.EngineConfig)

	if cfg.GetContain() {
		os.Exit(43)
	}

	return nil
}
