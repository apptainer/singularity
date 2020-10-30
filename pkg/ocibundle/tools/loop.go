// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package tools

import (
	"fmt"
	"os"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/util/loop"
	"github.com/sylabs/singularity/pkg/util/singularityconf"
)

func getMaxLoopDevices() int {
	// if the caller has set the current config use it
	// otherwise parse the default configuration file
	cfg := singularityconf.GetCurrentConfig()
	if cfg == nil {
		var err error

		configFile := buildcfg.SINGULARITY_CONF_FILE
		cfg, err = singularityconf.Parse(configFile)
		if err != nil {
			return 256
		}
	}
	return int(cfg.MaxLoopDevices)
}

// CreateLoop associates a file to loop device and returns
// path of loop device used
func CreateLoop(file *os.File, offset, size uint64) (string, error) {
	loopDev := &loop.Device{
		MaxLoopDevices: getMaxLoopDevices(),
		Shared:         true,
		Info: &loop.Info64{
			SizeLimit: size,
			Offset:    offset,
			Flags:     loop.FlagsAutoClear | loop.FlagsReadOnly,
		},
	}
	idx := 0
	if err := loopDev.AttachFromFile(file, os.O_RDONLY, &idx); err != nil {
		return "", fmt.Errorf("failed to attach image %s: %s", file.Name(), err)
	}
	return fmt.Sprintf("/dev/loop%d", idx), nil
}
