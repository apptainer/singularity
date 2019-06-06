// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	pluginapi "github.com/sylabs/singularity/pkg/plugin"
	singularity "github.com/sylabs/singularity/pkg/runtime/engines/singularity/config"
)

var Plugin = pluginapi.Plugin{
	Manifest: pluginapi.Manifest{
		Name:        "sylabs.io/fusecmd",
		Version:     "0.0.1",
		Description: "Singularity plugin adding the --fusecmd option for premounting FUSE filesystems",
	},

	Initializer: impl,
}

type pluginImplementation struct {
}

var impl = pluginImplementation{}

type FuseConfig struct {
	DevFuseFd  int
	MountPoint string
	Program    []string
}

type pluginConfig struct {
	Fuse FuseConfig
}

func fusecmdCallback(f *pflag.Flag, cfg *singularity.EngineConfig) {
	cmd := f.Value.String()

        // This will be called even if the flag was not used.
        // Assume that an empty mount point means the user did not pass
        // the flag and return silently.
	if cmd == "" {
                return
	}

        words := strings.Fields(cmd)
        if len(words) == 1 {
                sylog.Fatalf("No whitespace separators found in command")
	}

        mnt := words[len(words)-1]
	if !strings.HasPrefix(mnt, "/") {
		sylog.Fatalf("Invalid mount point %s.\n", mnt)
	}
        words = words[0:len(words)-1]

	sylog.Verbosef("Mounting FUSE filesystem with %s %s\n",
                strings.Join(words, " "), mnt)

	config := pluginConfig{
                        Fuse: FuseConfig{
                                MountPoint: mnt,
                                Program:    words,
                        },
                    }

	if err := cfg.SetPluginConfig(Plugin.Manifest.Name, config); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot set plugin configuration: %+v\n", err)
		return
	}
}

func (p pluginImplementation) Initialize(r pluginapi.HookRegistration) {
	flag := pluginapi.StringFlagHook{
		Flag: pflag.Flag{
			Name:  "fusecmd",
			Usage: "Command to run inside the container to " +
                               "implement a libfuse3-based filesystem. " +
                               "The last parameter is a mountpoint that " +
                               "will be pre-mounted and replaced with a " +
                               "/dev/fd/NN path to the fuse file descriptor.",
		},
		Callback: fusecmdCallback,
	}

	r.RegisterStringFlag(flag)
}
