// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sylabs/singularity/pkg/image"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
	singularitycallback "github.com/sylabs/singularity/pkg/plugin/callback/runtime/engine/singularity"
)

// Allow to use overlay with user namespace on Ubuntu flavors.
var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "github.com/sylabs/singularity/ubuntu-userns-overlay-plugin",
		Author:      "Sylabs Team",
		Version:     "0.1.0",
		Description: "Overlay ubuntu driver with user namespace",
	},
	Callbacks: []pluginapi.Callback{
		(singularitycallback.RegisterImageDriver)(ubuntuOvlRegister),
	},
	Install: setConfiguration,
}

const driverName = "ubuntu-userns-overlay"

type ubuntuOvlDriver struct {
	unprivileged bool
}

func ubuntuOvlRegister(unprivileged bool) error {
	return image.RegisterDriver(driverName, &ubuntuOvlDriver{unprivileged})
}

func (d *ubuntuOvlDriver) Features() image.DriverFeature {
	// if we are running unprivileged we are handling the overlay mount
	if d.unprivileged {
		return image.OverlayFeature
	}
	// privileged run are handled as usual by the singularity runtime
	return 0
}

func (d *ubuntuOvlDriver) Mount(params *image.MountParams, fn image.MountFunc) error {
	return fn(
		params.Source,
		params.Target,
		params.Filesystem,
		params.Flags,
		strings.Join(params.FSOptions, ","),
	)
}

func (d *ubuntuOvlDriver) Start(params *image.DriverParams) error {
	return nil
}

func (d *ubuntuOvlDriver) Stop() error {
	return nil
}

// setConfiguration sets "image driver" and "enable overlay" configuration directives
// during singularity plugin install step.
func setConfiguration(_ string) error {
	cmd := exec.Command("/proc/self/exe", "config", "global", "--set", "image driver", driverName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not set 'image driver = %s' in singularity.conf", driverName)
	}
	cmd = exec.Command("/proc/self/exe", "config", "global", "--set", "enable overlay", "driver")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not set 'enable overlay = driver' in singularity.conf")
	}
	return nil
}
