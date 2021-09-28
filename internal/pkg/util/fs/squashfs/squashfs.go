// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package squashfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hpcng/singularity/internal/pkg/buildcfg"
	"github.com/hpcng/singularity/pkg/util/singularityconf"
)

func getConfig() (*singularityconf.File, error) {
	// if the caller has set the current config use it
	// otherwise parse the default configuration file
	cfg := singularityconf.GetCurrentConfig()
	if cfg == nil {
		var err error

		configFile := buildcfg.SINGULARITY_CONF_FILE
		cfg, err = singularityconf.Parse(configFile)
		if err != nil {
			return nil, fmt.Errorf("unable to parse singularity.conf file: %s", err)
		}
	}
	return cfg, nil
}

// GetPath figures out where the mksquashfs binary is
// and return an error is not available or not usable.
func GetPath() (string, error) {
	// Parse singularity configuration file
	c, err := getConfig()
	if err != nil {
		return "", err
	}

	// p is either "" or the string value in the conf file
	p := c.MksquashfsPath

	// If the path contains the binary name use it as is, otherwise add mksquashfs via filepath.Join
	if !strings.HasSuffix(p, "mksquashfs") {
		p = filepath.Join(p, "mksquashfs")
	}

	// exec.LookPath functions on absolute paths (ignoring $PATH) as well
	return exec.LookPath(p)
}

func GetProcs() (uint, error) {
	c, err := getConfig()
	if err != nil {
		return 0, err
	}
	// proc is either "" or the string value in the conf file
	proc := c.MksquashfsProcs

	// let user override via ENV
	procEnv := os.Getenv("SINGULARITY_MKSQUASHFS_PROCS")
	if procEnv != "" {
		procEnvint, err := strconv.Atoi(procEnv)
		if err != nil {
			return 0, fmt.Errorf("failed to convert SINGULARITY_MKSQUASHFS_PROCS env %s to uint: %s", procEnv, err)
		}
		proc = uint(procEnvint)
	}

	return proc, err
}

func GetMem() (string, error) {
	c, err := getConfig()
	if err != nil {
		return "", err
	}
	// mem is either "" or the string value in the conf file
	mem := c.MksquashfsMem

	// let user override via ENV
	memEnv := os.Getenv("SINGULARITY_MKSQUASHFS_MEM")
	if memEnv != "" {
		mem = memEnv
	}

	return mem, err
}
