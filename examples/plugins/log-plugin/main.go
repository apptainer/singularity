// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"log/syslog"
	"os"
	"path/filepath"

	pluginapi "github.com/sylabs/singularity/pkg/plugin"
	singularitycallback "github.com/sylabs/singularity/pkg/plugin/callback/runtime/engine/singularity"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
	singularityConfig "github.com/sylabs/singularity/pkg/runtime/engine/singularity/config"
)

// Plugin is the only variable which a plugin MUST export.
// This symbol is accessed by the plugin framework to initialize the plugin
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "github.com/sylabs/singularity/log-plugin",
		Author:      "Sylabs Team",
		Version:     "0.1.0",
		Description: "Log executed commands to syslog",
	},
	Callbacks: []pluginapi.Callback{
		(singularitycallback.PostStartProcess)(logCommand),
	},
}

func logCommand(common *config.Common, pid int) error {
	cfg := common.EngineConfig.(*singularityConfig.EngineConfig)

	command := "unknown"
	if cfg.OciConfig != nil && cfg.OciConfig.Process != nil {
		if len(cfg.OciConfig.Process.Args) > 0 {
			command = filepath.Base(cfg.OciConfig.Process.Args[0])
		}
	}

	image := cfg.GetImage()
	w, err := syslog.New(syslog.LOG_INFO, "singularity")
	if err != nil {
		return err
	}
	defer w.Close()

	msg := fmt.Sprintf("UID=%d IMAGE=%s COMMAND=%s", os.Getuid(), image, command)
	return w.Info(msg)
}
